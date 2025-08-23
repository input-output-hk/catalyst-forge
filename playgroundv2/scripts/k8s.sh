#!/usr/bin/env bash

# -----------------------------------------------------------------------------
# k8s.sh
#
# Creates/refreshes a MicroK8s single-node Kubernetes cluster inside a
# Multipass VM, enables core addons (DNS, storage, ingress), configures
# MetalLB with a usable Layer 2 address pool, and exports a host-usable
# kubeconfig. Optionally writes a JSON summary for automation.
#
# Requirements on host:
# - multipass (https://multipass.run/)
# - bash, sed, awk
# - kubectl (optional, but recommended to validate the cluster)
#
# Usage:
#   ./k8s.sh [--name NAME] [--cpus N] [--mem SIZE] [--disk SIZE] \
#     [--channel CH] [--metallb-range START-END] [--kubeconfig-out PATH] [--no-ingress] \
#     [--addons "a b c"] [--yes] [--wait-timeout SECONDS] [--output-json PATH]
#
# Notes:
# - If --metallb-range is not provided, a range will be derived from the VM IP
#   as A.B.C.240-A.B.C.250.
# - If a VM with the given name already exists, you'll be prompted to reuse it
#   or delete and recreate it (unless --yes is provided).
# - The exported kubeconfig will have the API server set to the VM IP.
# - Optionally writes a JSON summary with --output-json for downstream automation.
# -----------------------------------------------------------------------------

set -Eeuo pipefail

SCRIPT_NAME="$(basename "$0")"

log() { printf "[INFO] %s\n" "$*"; }
warn() { printf "[WARN] %s\n" "$*" >&2; }
err() { printf "[ERROR] %s\n" "$*" >&2; }

confirm() {
  local prompt="$1"
  local default_yes=${2:-false}
  local reply
  if [[ "${ASSUME_YES}" == "true" ]]; then
    return 0
  fi
  if [[ "$default_yes" == true ]]; then
    read -r -p "$prompt [Y/n]: " reply || true
    reply=${reply:-Y}
  else
    read -r -p "$prompt [y/N]: " reply || true
    reply=${reply:-N}
  fi
  [[ "$reply" == "y" || "$reply" == "Y" ]]
}

usage() {
  cat <<EOF
${SCRIPT_NAME} - Create a MicroK8s cluster in a Multipass VM with MetalLB.

Options:
  --name NAME              VM name (default: microk8s)
  --cpus N                 Number of vCPUs (default: 2)
  --mem SIZE               Memory (default: 4G)
  --disk SIZE              Disk size (default: 20G)
  --channel CH             MicroK8s snap channel (default: 1.30/stable)
  --metallb-range RANGE    MetalLB address range (e.g., 192.168.64.240-192.168.64.250)
  --kubeconfig-out PATH    Path to write kubeconfig (default: ./kubeconfig)
  --no-ingress             Do not enable the ingress addon
  --addons "A B C"          Extra MicroK8s addons to enable (space-separated)
  --yes                    Assume "yes" for prompts (non-interactive)
  --wait-timeout SECONDS   Wait timeout for MicroK8s ready (default: 600)
  --output-json PATH       Write a JSON summary to PATH (default: ./cluster.json)
  -h, --help               Show this help
EOF
}

VM_NAME="microk8s"
VCPUS="2"
MEMORY="4G"
DISK="20G"
CHANNEL="1.33/stable"
METALLB_RANGE=""
KUBECONFIG_OUT="$(pwd)/kubeconfig"
ENABLE_INGRESS="true"
EXTRA_ADDONS=""
ASSUME_YES="false"
WAIT_TIMEOUT="600"
JSON_OUT="$(pwd)/cluster.json"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --name) VM_NAME="$2"; shift 2 ;;
    --cpus) VCPUS="$2"; shift 2 ;;
    --mem) MEMORY="$2"; shift 2 ;;
    --disk) DISK="$2"; shift 2 ;;
    --channel) CHANNEL="$2"; shift 2 ;;
    --metallb-range) METALLB_RANGE="$2"; shift 2 ;;
    --kubeconfig-out) KUBECONFIG_OUT="$2"; shift 2 ;;
    --no-ingress) ENABLE_INGRESS="false"; shift 1 ;;
    --addons) EXTRA_ADDONS="$2"; shift 2 ;;
    --yes) ASSUME_YES="true"; shift 1 ;;
    --wait-timeout) WAIT_TIMEOUT="$2"; shift 2 ;;
    --output-json) JSON_OUT="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) err "Unknown argument: $1"; usage; exit 1 ;;
  esac
done

require_cmd() {
  local name="$1"
  if ! command -v "$name" >/dev/null 2>&1; then
    err "Required command not found: $name"
    case "$name" in
      multipass)
        err "Install Multipass: https://multipass.run/"
        ;;
      *) : ;;
    esac
    exit 1
  fi
}

require_cmd multipass

# Ensure Multipass daemon is running (macOS usually auto-starts it)
if ! multipass list >/dev/null 2>&1; then
  warn "Multipass daemon might not be running. Attempting to start a harmless command..."
  multipass version >/dev/null 2>&1 || true
fi

vm_exists() {
  multipass info "$VM_NAME" >/dev/null 2>&1
}

launch_vm() {
  log "Launching Multipass VM '$VM_NAME' (cpus=$VCPUS mem=$MEMORY disk=$DISK)..."
  multipass launch --name "$VM_NAME" --cpus "$VCPUS" --memory "$MEMORY" --disk "$DISK"
}

ensure_vm() {
  if vm_exists; then
    warn "VM '$VM_NAME' already exists."
    if confirm "Reuse existing VM? Choosing 'n' will delete and recreate." false; then
      log "Reusing existing VM '$VM_NAME'"
    else
      if confirm "Delete existing VM '$VM_NAME'? This is destructive." false; then
        log "Deleting VM '$VM_NAME'..."
        multipass delete "$VM_NAME" || true
        multipass purge || true
        launch_vm
      else
        err "Aborting to avoid destructive action."
        exit 1
      fi
    fi
  else
    launch_vm
  fi
}

vm_exec() {
  multipass exec "$VM_NAME" -- bash -lc "$*"
}

get_vm_ip() {
  # Fetch the first global IPv4 address inside the VM
  vm_exec "ip -4 -o addr show scope global | awk '{print \$4}' | cut -d/ -f1 | head -n1"
}

derive_metallb_range() {
  local ip="$1"
  local prefix
  prefix="${ip%.*}"
  echo "${prefix}.240-${prefix}.250"
}

write_kubeconfig() {
  local vm_ip="$1"
  local tmp_cfg
  tmp_cfg="$(mktemp)"
  # microk8s config uses 127.0.0.1:16443 inside the VM; rewrite to VM IP for host access
  vm_exec "microk8s config" >"${tmp_cfg}"
  sed -E "s#server: https?://127.0.0.1:16443#server: https://${vm_ip}:16443#" "${tmp_cfg}" >"${KUBECONFIG_OUT}"
  rm -f "${tmp_cfg}"
  log "Kubeconfig written to ${KUBECONFIG_OUT}"
}

get_metallb_current_range() {
  local r
  r=$(vm_exec "microk8s kubectl -n metallb-system get ipaddresspools.metallb.io -o jsonpath='{.items[0].spec.addresses[0]}' 2>/dev/null" || true)
  if [[ -n "$r" ]]; then echo "$r"; return 0; fi
  r=$(vm_exec "microk8s kubectl -n metallb-system get configmap config -o jsonpath='{.data.config}' 2>/dev/null | sed -n 's/.*addresses: \[\(.*\)\].*/\1/p'" || true)
  if [[ -n "$r" ]]; then echo "$r"; return 0; fi
  echo ""
}

enable_addons() {
  local metallb_range="$1"
  log "Enabling MicroK8s addons..."
  vm_exec "sudo microk8s enable dns storage || true"
  if [[ "${ENABLE_INGRESS}" == "true" ]]; then
    vm_exec "sudo microk8s enable ingress || true"
  fi
  if [[ -n "${EXTRA_ADDONS}" ]]; then
    vm_exec "sudo microk8s enable ${EXTRA_ADDONS} || true"
  fi
  local current_range
  current_range="$(get_metallb_current_range)"
  if [[ -z "${current_range}" ]]; then
    log "Configuring MetalLB with range ${metallb_range} (no existing config detected)"
    vm_exec "sudo microk8s enable metallb:${metallb_range}"
  elif [[ "${current_range}" != "${metallb_range}" ]]; then
    warn "MetalLB range differs (current: '${current_range}' desired: '${metallb_range}'). Updating..."
    vm_exec "sudo microk8s enable metallb:${metallb_range}"
  else
    log "MetalLB already configured with desired range: ${metallb_range}"
  fi
}

install_microk8s() {
  if vm_exec "snap list microk8s >/dev/null 2>&1"; then
    log "MicroK8s already installed; refreshing to channel ${CHANNEL} if needed..."
    vm_exec "sudo snap refresh microk8s --channel=${CHANNEL} || true"
  else
    log "Installing MicroK8s (${CHANNEL}) inside VM..."
    vm_exec "sudo snap install microk8s --classic --channel=${CHANNEL}"
  fi
  # Ensure 'ubuntu' user can run microk8s without sudo (useful for future execs)
  vm_exec "sudo usermod -a -G microk8s ubuntu && sudo chown -f -R ubuntu /home/ubuntu/.kube || true"
}

wait_for_ready() {
  local timeout="$1"
  log "Waiting for MicroK8s to be ready (timeout ${timeout}s)..."
  vm_exec "timeout ${timeout} microk8s status --wait-ready" || {
    err "MicroK8s did not become ready within ${timeout}s"
    exit 1
  }
}

validate_cluster() {
  log "Validating cluster nodes and services..."
  vm_exec "microk8s kubectl get nodes -o wide"
  vm_exec "microk8s kubectl get ns"
}

main() {
  ensure_vm

  install_microk8s
  wait_for_ready "${WAIT_TIMEOUT}"

  local vm_ip
  vm_ip="$(get_vm_ip)"
  if [[ -z "${vm_ip}" ]]; then
    err "Failed to determine VM IP address"
    exit 1
  fi
  log "VM IP detected: ${vm_ip}"

  local range
  if [[ -n "${METALLB_RANGE}" ]]; then
    range="${METALLB_RANGE}"
  else
    range="$(derive_metallb_range "${vm_ip}")"
    log "Derived MetalLB range: ${range}"
  fi

  enable_addons "${range}"
  wait_for_ready "${WAIT_TIMEOUT}"

  write_kubeconfig "${vm_ip}"
  validate_cluster || true

  # Write JSON summary
  metallb_current="$(get_metallb_current_range)"
  cat >"${JSON_OUT}" <<JSON
{
  "vm": {
    "name": "${VM_NAME}",
    "ip": "${vm_ip}",
    "cpus": "${VCPUS}",
    "memory": "${MEMORY}",
    "disk": "${DISK}"
  },
  "microk8s": {
    "channel": "${CHANNEL}",
    "addons": {
      "dns": true,
      "storage": true,
      "ingress": ${ENABLE_INGRESS}
    },
    "extra_addons": "${EXTRA_ADDONS}"
  },
  "metallb": {
    "desired_range": "${range}",
    "current_range": "${metallb_current}"
  },
  "kubeconfig": "${KUBECONFIG_OUT}"
}
JSON

  cat <<EON

Cluster is ready.

- VM name: ${VM_NAME}
- VM IP: ${vm_ip}
- MetalLB range: ${range}
- Kubeconfig: ${KUBECONFIG_OUT}
- Summary JSON: ${JSON_OUT}

You can use this kubeconfig with:
  export KUBECONFIG="${KUBECONFIG_OUT}"
  kubectl get nodes

If you deploy an Ingress resource, ensure its Service uses a LoadBalancer type
so MetalLB assigns an external IP from the configured range.
EON
}

main "$@"


