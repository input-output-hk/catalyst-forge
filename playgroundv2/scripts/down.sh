#!/usr/bin/env bash

# -----------------------------------------------------------------------------
# down.sh
#
# Fast teardown for the local playground:
# - Removes /etc/hosts mappings via hostctl (if available)
# - Deletes the Multipass VM and purges remnants
# - Clears local Terraform state
# - Removes kubeconfig and cluster.json
# - Optionally removes generated certs
#
# Usage:
#   ./down.sh [--name microk8s] [--profile envoy] [--keep-certs true|false]
# -----------------------------------------------------------------------------

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
NAME="microk8s"
PROFILE="envoy"
KEEP_CERTS="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --name) NAME="$2"; shift 2 ;;
    --profile) PROFILE="$2"; shift 2 ;;
    --keep-certs) KEEP_CERTS="$2"; shift 2 ;;
    -h|--help)
      cat <<EOF
Usage: $0 [--name microk8s] [--profile envoy] [--keep-certs true|false]
EOF
      exit 0
      ;;
    *) echo "Unknown argument: $1" >&2; exit 1 ;;
  esac
done

log() { printf "[INFO] %s\n" "$*"; }
warn() { printf "[WARN] %s\n" "$*" >&2; }

# 1) Remove hosts mapping via hostctl if available
if command -v hostctl >/dev/null 2>&1; then
  log "Removing hostctl profile '${PROFILE}'..."
  sudo hostctl disable "${PROFILE}" >/dev/null 2>&1 || true
  sudo hostctl remove "${PROFILE}" >/dev/null 2>&1 || true
else
  warn "hostctl not found; you may need to manually clean /etc/hosts entries for profile '${PROFILE}'"
fi

# 2) Delete Multipass VM (if present)
if command -v multipass >/dev/null 2>&1; then
  if multipass info "${NAME}" >/dev/null 2>&1; then
    log "Deleting Multipass VM '${NAME}'..."
    multipass delete "${NAME}" || true
    multipass purge || true
  else
    log "Multipass VM '${NAME}' not found; skipping"
  fi
else
  warn "multipass not found; skipping VM deletion"
fi

# 3) Clean local Terraform state
rm -rf "${SCRIPT_DIR}/terraform/.terraform" || true
rm -f  "${SCRIPT_DIR}/terraform/.terraform.lock.hcl" || true
rm -f  "${SCRIPT_DIR}/terraform/terraform.tfstate"* || true

# 4) Remove kubeconfig and cluster summary; optionally remove certs
rm -f "${SCRIPT_DIR}/kubeconfig" || true
rm -f "${SCRIPT_DIR}/cluster.json" || true
if [[ "${KEEP_CERTS}" != "true" ]]; then
  rm -rf "${SCRIPT_DIR}/certs" || true
fi

log "Down complete."


