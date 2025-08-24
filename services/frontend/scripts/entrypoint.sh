#!/usr/bin/env sh

#
# Catalyst Forge Frontend Entrypoint
#
# Renders a runtime config file /app/config.js from environment variables, then
# starts Caddy to serve the static SPA. This allows the same image to be used
# across environments by providing API and Kratos URLs at container start.
#
# Env vars:
#   CF_API_BASE_URL           -> window.__CF_CONFIG__.apiBaseUrl
#   CF_KRATOS_PUBLIC_URL      -> window.__CF_CONFIG__.kratosPublicUrl
#
# Usage: Provide env vars and run the container. Optionally mount a custom
# /app/config.js to override everything rendered here.
#
set -eu

log() { echo "[entrypoint] $*"; }

# Render config.js unless explicitly skipped
if [ "${CF_SKIP_RENDER:-0}" != "1" ]; then
  API_URL=${CF_API_BASE_URL:-}
  KRATOS_URL=${CF_KRATOS_PUBLIC_URL:-}

  log "Rendering /app/config.js (apiBaseUrl='${API_URL:-}', kratosPublicUrl='${KRATOS_URL:-}')"
  cat >/app/config.js <<EOF
// Generated at container start by entrypoint.sh
window.__CF_CONFIG__ = {
  apiBaseUrl: "${API_URL}",
  kratosPublicUrl: "${KRATOS_URL}",
};
EOF
else
  log "CF_SKIP_RENDER=1 set; skipping config render"
fi

exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile


