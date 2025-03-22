#!/usr/bin/env bash

set -eo pipefail

echo ">>> Deploying Gitea"
kubectl apply -f manifests/gitea.yml
kubectl wait --for=condition=available --timeout=60s deployment/gitea

echo ">>> Creating Gitea user"
sleep 2
POD_NAME=$(kubectl get pods -l app=gitea -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it "${POD_NAME}" -- gitea admin user create --admin --username root --password root --email root@example.com
TOKEN=$(kubectl exec -it "${POD_NAME}" -- gitea admin user generate-access-token --raw --username root --token-name "foundry" --scopes "write:user,write:repository")
TOKEN=$(echo "${TOKEN}" | tr -d '\r')
TOKEN=$(echo "${TOKEN}" | tr -d '\n')
echo -n "${TOKEN}" >.token
