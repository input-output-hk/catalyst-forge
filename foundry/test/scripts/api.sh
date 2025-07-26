#!/usr/bin/env bash

set -eo pipefail

echo ">>> Deploying API server"
earthly --config "" ../api+docker
docker tag foundry-api:latest localhost:5001/foundry-api:latest
docker push localhost:5001/foundry-api:latest

echo '>>> Building local version'
go build -C ../api -o ../test/bin/api cmd/api/main.go

echo '>>> Generating certificates'
mkdir -p ./.secrets
./bin/api auth init --output-dir ./.secrets

echo '>>> Generating admin token'
./bin/api auth generate -a -k ./.secrets/private.pem > ./.secrets/jwt.txt

echo '>>> Creating Kubernetes secret with certificates'
kubectl create secret generic api-auth-keys \
  --from-file=public.pem=./.secrets/public.pem \
  --from-file=private.pem=./.secrets/private.pem \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -f manifests/api.yml
kubectl wait --for=condition=available --timeout=60s deployment/api
