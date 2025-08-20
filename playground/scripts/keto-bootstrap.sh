#!/usr/bin/env bash
set -euo pipefail

# Bootstrap Ory Keto relation tuples by exec'ing into the running keto container
# and calling the write API on 127.0.0.1:4467. This avoids relying on host port exposure.
#
# Usage:
#   ./scripts/keto-bootstrap.sh [path/to/tuples.json]
#
# Env vars:
#   KETO_CONTAINER   Name of the keto container (default: keto)
#   FILE             Path to JSON array of tuples (default: ory/keto/tuples/bootstrap.json)

KETO_CONTAINER="${KETO_CONTAINER:-keto}"
FILE="${1:-ory/keto/tuples/bootstrap.json}"

require() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Error: required command '$1' not found in PATH" >&2
    exit 1
  fi
}

require jq
require docker

if [ ! -f "$FILE" ]; then
  echo "Error: tuples file not found: $FILE" >&2
  exit 1
fi

# Ensure container is running
if ! docker inspect -f '{{.State.Running}}' "$KETO_CONTAINER" >/dev/null 2>&1; then
  echo "Error: container '$KETO_CONTAINER' not found or not running" >&2
  exit 1
fi

echo "==> Bootstrapping tuples into '$KETO_CONTAINER' from: $FILE"

# Prefer the official keto CLI inside the container for correctness
if docker exec "$KETO_CONTAINER" sh -lc 'command -v keto >/dev/null 2>&1'; then
  # keto expects a JSON array file path or '-' for stdin; we can stream the file as-is
  cat "$FILE" | docker exec -i "$KETO_CONTAINER" sh -lc 'keto relation-tuple create - \
    --write-remote 127.0.0.1:4467 \
    --insecure-disable-transport-security \
    --block \
    >/dev/null'
  echo "Done."
  exit 0
fi

# Fallback to HTTP if keto CLI is unavailable (should not happen with official image)
jq -c '.[]' "$FILE" | while read -r tuple; do
  echo "PUT $tuple"
  printf '%s' "$tuple" | docker exec -i "$KETO_CONTAINER" sh -lc '
    if command -v curl >/dev/null 2>&1; then
      curl -sS -X PUT -H "Content-Type: application/json" --data-binary @- \
        "http://127.0.0.1:4467/admin/relation-tuples" >/dev/null
    elif command -v wget >/dev/null 2>&1; then
      # Busybox wget lacks PUT; emulate via HTTP method override header if supported by server (not required here)
      echo "Error: wget fallback does not support PUT; please ensure curl is present or keto CLI is available" >&2
      exit 1
    else
      echo "Error: neither curl nor wget found in container" >&2
      exit 1
    fi
  '
done

echo "Done."
