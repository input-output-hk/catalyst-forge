#!/usr/bin/env bash

set -eo pipefail

echo ">>> Deploying API server"
earthly ../api+docker
docker tag foundry-api:latest localhost:5001/foundry-api:latest
docker push localhost:5001/foundry-api:latest
kubectl apply -f manifests/api.yml
kubectl wait --for=condition=available --timeout=60s deployment/api
