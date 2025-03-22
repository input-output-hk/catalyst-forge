#!/usr/bin/env bash

set -eo pipefail

# Delete cluster
echo ">>> Deleting kind cluster"
kind delete cluster --name foundry

# Delete kind registry
echo ">>> Deleting kind registry"
REGISTRY_NAME='kind-registry'
if [ "$(docker inspect -f '{{.State.Running}}' "${REGISTRY_NAME}" 2>/dev/null || true)" = 'true' ]; then
    docker stop "${REGISTRY_NAME}"
    docker rm "${REGISTRY_NAME}"
fi
