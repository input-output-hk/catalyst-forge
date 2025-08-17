#!/usr/bin/env bash

# Sync the built Forge TS client into the frontend's vendored bundle
#
# Usage:
#   ./scripts/sync-vendored-client.sh [--build] [--no-install]
#
# Options:
#   --build        Build the TypeScript client before syncing (runs `npm run build`)
#   --no-install   Skip `npm ci` in the client project (assumes deps are installed)
#
# Behavior:
# - Verifies required tools (node, npm)
# - Builds the client (optional) located at services/clients/ts
# - Copies dist/index.mjs to services/frontend/vendor/forge-client/index.mjs
# - Copies dist/index.d.ts and src/api/schema.d.ts for type support
# - Prints the client version that was synced

set -euo pipefail

log() { echo "[sync-vendored-client] $*"; }
err() { echo "[sync-vendored-client] ERROR: $*" >&2; }

NEEDS_BUILD=false
SKIP_INSTALL=false
for arg in "$@"; do
  case "$arg" in
    --build) NEEDS_BUILD=true ;;
    --no-install) SKIP_INSTALL=true ;;
    -h|--help)
      sed -n '1,30p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *) err "Unknown arg: $arg"; exit 2 ;;
  esac
done

# Resolve directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FRONTEND_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$FRONTEND_DIR/../.." && pwd)"
CLIENT_DIR="$REPO_ROOT/services/clients/ts"
VENDOR_DIR="$FRONTEND_DIR/vendor/forge-client"

# Preconditions
command -v node >/dev/null 2>&1 || { err "node is required"; exit 1; }
command -v npm  >/dev/null 2>&1 || { err "npm is required"; exit 1; }

[[ -d "$CLIENT_DIR" ]] || { err "Client directory not found: $CLIENT_DIR"; exit 1; }

# Read client version for log output
CLIENT_PKG_JSON="$CLIENT_DIR/package.json"
CLIENT_VERSION="unknown"
if [[ -f "$CLIENT_PKG_JSON" ]]; then
  CLIENT_VERSION=$(node -e "console.log(require(process.argv[1]).version)" "$CLIENT_PKG_JSON" 2>/dev/null || echo "unknown")
fi

log "Client project: $CLIENT_DIR (version: $CLIENT_VERSION)"

pushd "$CLIENT_DIR" >/dev/null

if [[ "$SKIP_INSTALL" != "true" ]]; then
  if [[ ! -d node_modules ]]; then
    log "Installing client dependencies (npm ci)"
    npm ci --no-audit --no-fund
  else
    log "Dependencies present; skipping install (use --no-install to skip check)"
  fi
fi

if [[ "$NEEDS_BUILD" == "true" ]]; then
  log "Building client (npm run build)"
  npm run --silent build
else
  log "Skipping build (use --build to force)"
fi

DIST_MJS="$CLIENT_DIR/dist/index.mjs"
DIST_DTS="$CLIENT_DIR/dist/index.d.ts"
SCHEMA_DTS="$CLIENT_DIR/src/api/schema.d.ts"
[[ -f "$DIST_MJS" ]] || { err "Built file not found: $DIST_MJS. Try re-running with --build"; exit 1; }

mkdir -p "$VENDOR_DIR"
cp "$DIST_MJS" "$VENDOR_DIR/index.mjs"

# Optional: copy types if present
if [[ -f "$DIST_DTS" ]]; then
  cp "$DIST_DTS" "$VENDOR_DIR/index.d.ts"
  log "Copied types: $DIST_DTS -> $VENDOR_DIR/index.d.ts"
fi

if [[ -f "$SCHEMA_DTS" ]]; then
  cp "$SCHEMA_DTS" "$VENDOR_DIR/schema.d.ts"
  log "Copied schema types: $SCHEMA_DTS -> $VENDOR_DIR/schema.d.ts"
fi

# Minimal package.json for type resolution
cat > "$VENDOR_DIR/package.json" <<'JSON'
{
  "type": "module",
  "name": "forge-client",
  "private": true,
  "types": "./index.d.ts"
}
JSON

# Ensure index.d.ts re-exports schema types for paths
if [[ -f "$VENDOR_DIR/index.d.ts" ]]; then
  if ! grep -q 'export type { paths } from "./schema";' "$VENDOR_DIR/index.d.ts" 2>/dev/null; then
    echo 'export type { paths } from "./schema";' >> "$VENDOR_DIR/index.d.ts"
  fi
fi

popd >/dev/null

log "Synced: $DIST_MJS -> $VENDOR_DIR/index.mjs (and types if available)"
log "Done."


