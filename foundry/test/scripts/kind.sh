#!/usr/bin/env bash

set -eo pipefail

# Create cluster
echo ">>> Creating kind cluster"
kind create cluster --name foundry --config ./manifests/kind.yml

# Configure container registry
echo ">>> Configuring kind registry"
if [ "$(docker inspect -f '{{.State.Running}}' "kind-registry" 2>/dev/null || true)" != 'true' ]; then
  docker run \
    -d --restart=always -p "127.0.0.1:5001:5000" --network bridge --name "kind-registry" \
    registry:2
fi

for node in $(kind get nodes --name foundry); do
  docker exec "${node}" mkdir -p "/etc/containerd/certs.d/localhost:5001"
  cat <<EOF | docker exec -i "${node}" cp /dev/stdin "/etc/containerd/certs.d/localhost:5001/hosts.toml"
[host."http://kind-registry:5000"]
EOF
done

if [ "$(docker inspect -f='{{json .NetworkSettings.Networks.kind}}' "kind-registry")" = 'null' ]; then
  docker network connect "kind" "kind-registry"
fi

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:5001"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF
