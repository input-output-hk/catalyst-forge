#!/usr/bin/env bash

set -eo pipefail

echo ">>> Deploying postgres"
kubectl apply -f manifests/postgres.yml
kubectl wait --for=condition=available --timeout=60s deployment/postgres
