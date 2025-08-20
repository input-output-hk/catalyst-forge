#!/usr/bin/env bash

set -euo pipefail

rm -rf .secrets/hydra-clients
./scripts/hydra-create-clients.sh
./scripts/keto-bootstrap.sh ory/keto/tuples/bootstrap.json

export M2M_ID=$(jq -r '.client_id' .secrets/hydra-clients/client-m2m.json)
export M2M_SECRET=$(jq -r '.client_secret' .secrets/hydra-clients/client-m2m.json)
export M2M_TOKEN=$(scripts/hydra-m2m-token.sh "$M2M_ID" "$M2M_SECRET" "forge.api forge.deploy")

echo "M2M_TOKEN: $M2M_TOKEN"