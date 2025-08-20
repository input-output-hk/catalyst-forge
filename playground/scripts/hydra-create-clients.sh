#!/usr/bin/env bash
set -euo pipefail

# Where the Hydra Admin API is reachable from the hydra container.
# We exec into the hydra container, so localhost:4445 works inside it.
HYDRA_ADMIN_ENDPOINT="${HYDRA_ADMIN_ENDPOINT:-http://127.0.0.1:4445}"

OUT_DIR="${OUT_DIR:-.secrets/hydra-clients}"
mkdir -p "$OUT_DIR"

exec_in_hydra() {
  docker exec -e HYDRA_ADMIN_URL="$HYDRA_ADMIN_ENDPOINT" hydra hydra "$@"
}

json_field() {
  jq -r "$1"
}

echo "==> Creating SPA (PKCE) client"
SPA_JSON=$(exec_in_hydra create client \
  --endpoint "$HYDRA_ADMIN_ENDPOINT" \
  --format json \
  --name "Forge SPA (local)" \
  --grant-type authorization_code --grant-type refresh_token \
  --response-type code \
  --scope openid --scope offline_access --scope forge.api \
  --token-endpoint-auth-method none \
  --redirect-uri https://forge-test.projectcatalyst.io/oidc/callback \
  --post-logout-callback https://forge-test.projectcatalyst.io/ \
  --allowed-cors-origin https://forge-test.projectcatalyst.io
)
echo "$SPA_JSON" | tee "$OUT_DIR/client-spa.json" >/dev/null
SPA_ID=$(echo "$SPA_JSON" | json_field '.client_id')

echo "==> Creating CLI (Device Code) client (public)"
set +e
CLI_DEV_JSON=$(exec_in_hydra create client \
  --endpoint "$HYDRA_ADMIN_ENDPOINT" \
  --format json \
  --name "Forge CLI (device code, local)" \
  --grant-type urn:ietf:params:oauth:grant-type:device_code --grant-type refresh_token \
  --scope openid --scope offline_access --scope forge.cli --scope forge.api \
  --token-endpoint-auth-method none
) ; EC=$?
set -e
if [ $EC -eq 0 ]; then
  echo "$CLI_DEV_JSON" | tee "$OUT_DIR/client-cli-device.json" >/dev/null
  CLI_DEVICE_ID=$(echo "$CLI_DEV_JSON" | json_field '.client_id')
  echo "    ✔ Device Code client created: $CLI_DEVICE_ID"
else
  echo "    ! Device Code grant not supported by this Hydra build."
  echo "    -> Creating PKCE loopback CLI client as a fallback."
  CLI_DEV_JSON=$(exec_in_hydra create client \
    --endpoint "$HYDRA_ADMIN_ENDPOINT" \
    --format json \
    --name "Forge CLI (PKCE loopback, local)" \
    --grant-type authorization_code --grant-type refresh_token \
    --response-type code \
    --scope openid --scope offline_access --scope forge.cli --scope forge.api \
    --token-endpoint-auth-method none \
    --redirect-uri http://127.0.0.1:53682/callback \
    --redirect-uri http://localhost:53682/callback
  )
  echo "$CLI_DEV_JSON" | tee "$OUT_DIR/client-cli-pkce.json" >/dev/null
  CLI_DEVICE_ID=$(echo "$CLI_DEV_JSON" | json_field '.client_id')
fi

echo "==> Creating Machine (client_credentials) client"
M2M_JSON=$(exec_in_hydra create client \
  --endpoint "$HYDRA_ADMIN_ENDPOINT" \
  --format json \
  --name "Forge M2M (local)" \
  --grant-type client_credentials \
  --response-type token \
  --scope forge.api --scope forge.deploy \
  --token-endpoint-auth-method client_secret_post
)
echo "$M2M_JSON" | tee "$OUT_DIR/client-m2m.json" >/dev/null
M2M_ID=$(echo "$M2M_JSON" | json_field '.client_id')
M2M_SECRET=$(echo "$M2M_JSON" | json_field '.client_secret')

echo
echo "==> Summary"
echo "SPA client_id:         $SPA_ID"
echo "CLI client_id:         $CLI_DEVICE_ID"
echo "M2M client_id:         $M2M_ID"
echo "M2M client_secret:     $M2M_SECRET"
echo
echo "Files:"
ls -l "$OUT_DIR"
