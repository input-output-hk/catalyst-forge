#!/usr/bin/env bash

set -eo pipefail

echo ">>> Deploying operator"
kubectl apply -f ../operator/config/crd/bases/foundry.projectcatalyst.io_releasedeployments.yaml
kubectl apply -f ../operator/config/rbac/role.yaml
kubectl apply -f ../operator/config/rbac/role_binding.yaml
kubectl apply -f ../operator/config/rbac/service_account.yaml
earthly ../operator+docker
docker tag foundry-operator:latest localhost:5001/foundry-operator:latest
docker push localhost:5001/foundry-operator:latest
kubectl apply -f manifests/operator.yml
kubectl wait --for=condition=available --timeout=60s deployment/operator
