#!/usr/bin/env bash

# -----------------------------------------------------------------------------
# up.sh
#
# One-shot bring-up of the local environment:
# - Boots/refreshes MicroK8s in Multipass and configures MetalLB
# - Picks a stable IP from the MetalLB range and deploys Envoy Gateway (Helm via OpenTofu)
# - Generates mkcert wildcard certificate for *.localhost
# - Creates/updates the Kubernetes TLS secret for Envoy Gateway HTTPS listener
# - Adds / updates a hosts entry for the chosen domain -> Envoy IP (hostctl if available)
#
# Usage:
#   ./up.sh [--domain DOMAIN] [--profile NAME]
#
# Requirements:
# - jq, mkcert, kubectl, tofu (OpenTofu), hostctl (optional)
# -----------------------------------------------------------------------------

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
BASE_DOMAIN="local.io"
DOMAIN="gateway.${BASE_DOMAIN}"
PROFILE="envoy"
KCFG="${ROOT_DIR}/kubeconfig"
CERT_DIR="${ROOT_DIR}/certs"

# Canonical domains we will register
FORGE_DOMAIN="forge.${BASE_DOMAIN}"
REGISTRY_DOMAIN="registry.${BASE_DOMAIN}"
DOMAINS=("${DOMAIN}" "${REGISTRY_DOMAIN}" "${FORGE_DOMAIN}")

while [[ $# -gt 0 ]]; do
  case "$1" in
    --domain) DOMAIN="$2"; shift 2 ;;
    --profile) PROFILE="$2"; shift 2 ;;
    *) echo "Unknown argument: $1" >&2; exit 1 ;;
  esac
done

need() { command -v "$1" >/dev/null 2>&1 || { echo "Missing dependency: $1" >&2; exit 1; }; }

need jq
need kubectl
need tofu

# 1) Create/refresh cluster (idempotent) and write cluster.json
"${SCRIPT_DIR}/k8s.sh" --yes --output-json "${ROOT_DIR}/cluster.json" --kubeconfig-out "${KCFG}"

# 2) Derive MetalLB range and pick a stable IP
RANGE=$(jq -r .metallb.desired_range "${ROOT_DIR}/cluster.json")
if [[ -z "${RANGE}" || "${RANGE}" == "null" ]]; then
  echo "Failed to read metallb.desired_range from cluster.json" >&2
  exit 1
fi
START=${RANGE%-*}
END=${RANGE#*-}
IFS='.' read -r S1 S2 S3 S4 <<<"${START}"
IFS='.' read -r E1 E2 E3 E4 <<<"${END}"
OCT=$(( S4 + 5 ))
if (( OCT <= E4 )); then
  LB_IP="${S1}.${S2}.${S3}.${OCT}"
else
  LB_IP="${START}"
fi
echo "Using MetalLB IP: ${LB_IP} from range ${RANGE}"

# 3) Terraform init/apply Envoy Gateway with pinned IP and registry
tofu -chdir="${ROOT_DIR}/terraform" init -upgrade
tofu -chdir="${ROOT_DIR}/terraform" apply -auto-approve -var "load_balancer_ip=${LB_IP}" -var "registry_host=registry.${BASE_DOMAIN}"

# After apply, always discover the actual EXTERNAL-IP assigned to the Envoy LoadBalancer
REAL_IP=""
for i in $(seq 1 24); do
  REAL_IP=$(kubectl --kubeconfig "${KCFG}" -n envoy-gateway-system \
    get svc -l 'app.kubernetes.io/name=envoy' \
    -o jsonpath='{.items[?(@.spec.type=="LoadBalancer")].status.loadBalancer.ingress[0].ip}' 2>/dev/null || true)
  if [[ -n "${REAL_IP}" ]]; then break; fi
  sleep 5
done
if [[ -z "${REAL_IP}" ]]; then
  echo "Failed to discover Envoy EXTERNAL-IP after apply" >&2
  exit 1
fi
echo "Envoy EXTERNAL-IP: ${REAL_IP}"
LB_IP="${REAL_IP}"

# 4) Generate mkcert wildcard certs and create TLS secret
need mkcert
mkdir -p "${CERT_DIR}"
pushd "${CERT_DIR}" >/dev/null

# Generate wildcard cert for base domain and a cert for the Gateway domain
WILDCARD="*.${BASE_DOMAIN}"
SAFE_WILDCARD="${WILDCARD//\*/_wildcard}"
WILDCARD_CERT="${CERT_DIR}/${SAFE_WILDCARD}.pem"
WILDCARD_KEY="${CERT_DIR}/${SAFE_WILDCARD}-key.pem"
mkcert -cert-file "${WILDCARD_CERT}" -key-file "${WILDCARD_KEY}" "${WILDCARD}"

FILENAME_SAFE_DOMAIN="${DOMAIN//\*/_wildcard}"
CERT_FILE="${CERT_DIR}/${FILENAME_SAFE_DOMAIN}.pem"
KEY_FILE="${CERT_DIR}/${FILENAME_SAFE_DOMAIN}-key.pem"
mkcert -cert-file "${CERT_FILE}" -key-file "${KEY_FILE}" "${DOMAIN}"
popd >/dev/null

if [[ ! -f "${WILDCARD_CERT}" || ! -f "${WILDCARD_KEY}" ]]; then
  echo "Expected wildcard cert files not found: ${WILDCARD_CERT} / ${WILDCARD_KEY}" >&2
  echo "Available files in ${CERT_DIR}:" >&2
  ls -la "${CERT_DIR}" >&2 || true
  exit 1
fi

# Use wildcard cert for the Gateway TLS so all subdomains are covered
kubectl --kubeconfig "${KCFG}" -n envoy-gateway-system create secret tls envoy-gateway-tls \
  --cert="${WILDCARD_CERT}" --key="${WILDCARD_KEY}" \
  --dry-run=client -o yaml | kubectl --kubeconfig "${KCFG}" -n envoy-gateway-system apply -f -

# 5) Install mkcert CA into MicroK8s VM so containerd trusts our Gateway certs
CAROOT=$(mkcert -CAROOT)
multipass transfer "${CAROOT}/rootCA.pem" microk8s:/tmp/mkcert-rootCA.crt
multipass exec microk8s -- sudo bash -lc 'install -m 0644 /tmp/mkcert-rootCA.crt /usr/local/share/ca-certificates/mkcert-rootCA.crt && update-ca-certificates && snap restart microk8s.daemon-containerd'

# 6) Add hosts entries inside the MicroK8s VM so kubelet/containerd resolve our domains
for d in "${DOMAINS[@]}"; do
  multipass exec microk8s -- sudo bash -lc "grep -q ' $d$' /etc/hosts || echo '${LB_IP} $d' | tee -a /etc/hosts >/dev/null"
done

# 7) Map domains -> LB_IP on the host (idempotent)
if command -v hostctl >/dev/null 2>&1; then
  NEED_UPDATE=false
  PROFILE_LINES=$(hostctl list -o json 2>/dev/null | jq -r --arg p "$PROFILE" '.[] | select(.Profile==$p) | "\(.Host)|\(.IP)|\(.Status)"' || echo "")
  for d in "${DOMAINS[@]}"; do
    match=$(printf '%s\n' "$PROFILE_LINES" | awk -F'|' -v h="$d" -v ip="$LB_IP" '$1==h && $2==ip {print $0}')
    if [[ -z "$match" ]]; then NEED_UPDATE=true; break; fi
  done
  enabled=$(printf '%s\n' "$PROFILE_LINES" | awk -F'|' '$3=="on"{print;exit}')
  if [[ -z "$enabled" ]]; then NEED_UPDATE=true; fi

  if [[ "$NEED_UPDATE" == "true" ]]; then
    sudo hostctl remove "${PROFILE}" >/dev/null 2>&1 || true
    sudo hostctl add domains "${PROFILE}" --ip "${LB_IP}" "${DOMAINS[@]}"
    sudo hostctl enable "${PROFILE}"
    echo "Updated hostctl profile '${PROFILE}' => ${LB_IP} ${DOMAINS[*]}"
  else
    echo "hostctl profile '${PROFILE}' already up-to-date"
  fi
else
  echo "hostctl not found; falling back to /etc/hosts update"
  for d in "${DOMAINS[@]}"; do
    sudo sed -i.bak "/[[:space:]]${d}$/d" /etc/hosts || true
  done
  echo "${LB_IP} ${DOMAINS[*]}" | sudo tee -a /etc/hosts >/dev/null
fi

echo "Environment is up. Test with:"
echo "  curl -I --resolve ${DOMAIN}:443:${LB_IP} https://${DOMAIN}/"
echo "  docker push ${REGISTRY_DOMAIN}/your/image:tag"


