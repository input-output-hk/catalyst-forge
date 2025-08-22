#!/usr/bin/env bash

# ----------------------------------------------------------------------------
# deploy.sh
#
# Description:
#   Production-grade deployment helper for Catalyst Forge Playground v2.
#   Builds and pushes the container for a service using Earthly, then renders
#   a deployment template via the CLI generator, and applies it with kubectl
#   using the kubeconfig generated during setup.
#   Requires a single argument:
#   the service name (e.g., "api").
#
#   This script will:
#     1) Verify required tools and files exist.
#     2) Build and push the service image with Earthly.
#     3) Generate module template from the service using the CLI.
#     4) Clean up temporary files on exit.
#
# Usage:
#   ./deploy.sh <service-name>
#   ./deploy.sh -h | --help
#
# Non-interactive mode:
#   Set DEBUG=1 to enable command tracing.
# ----------------------------------------------------------------------------

set -euo pipefail

if [[ "${DEBUG:-0}" == "1" ]]; then
  set -x
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PLAY_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
ROOT_DIR="$(cd "${PLAY_DIR}/.." && pwd)"

print_usage() {
  cat <<EOF
Usage: $(basename "$0") <service-name>

Builds and pushes the specified service with Earthly, then renders the module
template using the project CLI. Only one argument is accepted: the service name.

Environment variables:
  DEBUG=1       Enable verbose command tracing
  KUBECONFIG    Override kubeconfig path (defaults to playgroundv2/kubeconfig)
EOF
}

log() { printf "[deploy] %s\n" "$*"; }
err() { printf "[deploy] ERROR: %s\n" "$*" >&2; }
warn() { printf "[deploy] WARN: %s\n" "$*" >&2; }

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    err "Required command '$1' is not installed or not in PATH."
    exit 1
  fi
}

if [[ $# -eq 0 ]]; then
  print_usage
  exit 1
fi
if [[ "$1" == "-h" || "$1" == "--help" ]]; then
  print_usage
  exit 0
fi

SERVICE_NAME="$1"

# Validate dependencies
require_cmd earthly
require_cmd go
require_cmd mktemp
require_cmd kubectl

EARTHLY_CONFIG="${PLAY_DIR}/config/earthly.yml"
CLI_MAIN="${ROOT_DIR}/cli/cmd/main.go"
SERVICE_DIR="${ROOT_DIR}/services/${SERVICE_NAME}"
OVERRIDE_FILE="${PLAY_DIR}/config/overrides/${SERVICE_NAME}.cue"

KUBECONFIG_DEFAULT="${PLAY_DIR}/kubeconfig"
KCFG="${KUBECONFIG:-${KUBECONFIG_DEFAULT}}"

if [[ ! -f "${EARTHLY_CONFIG}" ]]; then
  err "Earthly config not found at '${EARTHLY_CONFIG}'."
  exit 1
fi

if [[ ! -f "${CLI_MAIN}" ]]; then
  err "CLI entrypoint not found at '${CLI_MAIN}'."
  exit 1
fi

if [[ ! -d "${SERVICE_DIR}" ]]; then
  err "Service directory does not exist: '${SERVICE_DIR}'."
  exit 1
fi

if [[ ! -f "${OVERRIDE_FILE}" ]]; then
  err "Override file not found: '${OVERRIDE_FILE}'."
  exit 1
fi

if [[ ! -f "${KCFG}" ]]; then
  err "Kubeconfig not found at '${KCFG}'. Run '${PLAY_DIR}/scripts/up.sh' first."
  exit 1
fi

TMP_DIR="$(mktemp -d)"
cleanup() {
  if [[ -n "${TMP_DIR:-}" && -d "${TMP_DIR}" ]]; then
    rm -rf "${TMP_DIR}"
  fi
}
trap cleanup EXIT INT TERM

log "Service: ${SERVICE_NAME}"
log "Root dir: ${ROOT_DIR}"
log "Playground dir: ${PLAY_DIR}"
log "Temp dir: ${TMP_DIR}"

log "Running Earthly build and push..."
earthly --config "${EARTHLY_CONFIG}" --push +"${SERVICE_NAME}"

log "Generating module CUE from service directory..."
(cd "${ROOT_DIR}/cli" && go run cmd/main.go mod dump "${SERVICE_DIR}" >"${TMP_DIR}/mod.cue")

log "Copying env override '${OVERRIDE_FILE}' to temporary module..."
cp "${OVERRIDE_FILE}" "${TMP_DIR}/env.mod.cue"

log "Rendering module template..."
(cd "${ROOT_DIR}/cli" && go run cmd/main.go mod template -o "${TMP_DIR}" "${TMP_DIR}/mod.cue")

if ! compgen -G "${TMP_DIR}"/*.yaml >/dev/null && ! compgen -G "${TMP_DIR}"/*.yml >/dev/null; then
  err "No Kubernetes manifest files (*.yaml|*.yml) found in ${TMP_DIR}."
  exit 1
fi

log "Applying Kubernetes manifests from ${TMP_DIR} (context: $(kubectl --kubeconfig "${KCFG}" config current-context 2>/dev/null || echo 'unknown'))..."
kubectl --kubeconfig "${KCFG}" apply -f "${TMP_DIR}"

log "Done. Temporary files cleaned up."