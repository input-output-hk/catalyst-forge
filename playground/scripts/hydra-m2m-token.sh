#!/usr/bin/env bash
set -euo pipefail

HYDRA_PUBLIC="${HYDRA_PUBLIC:-https://forge-test.projectcatalyst.io/.ory/hydra/public}"
CLIENT_ID="${1:?Usage: $0 <client_id> <client_secret> [scope] [audience?]}"
CLIENT_SECRET="${2:?Usage: $0 <client_id> <client_secret> [scope] [audience?]}"
SCOPE="${3:-forge.api}"
AUD="${4:-}"

data=(
  -d grant_type=client_credentials
  --data-urlencode "client_id=$CLIENT_ID"
  --data-urlencode "client_secret=$CLIENT_SECRET"
  --data-urlencode "scope=$SCOPE"
)

# Optional audience parameter if you use audience restrictions later
if [ -n "$AUD" ]; then
  data+=( --data-urlencode "audience=$AUD" )
fi

curl -sS -k "${data[@]}" \
  "$HYDRA_PUBLIC/oauth2/token" | jq -r '.access_token'
