#!/usr/bin/env bash

set -eo pipefail

GITEA_USER=root
GITEA_PASS=root
GITEA_EMAIL=root@example.com

# Create cluster
echo ">>> Creating kind cluster"
kind create cluster --name foundry --config ./kind.yml

# Configure container registry
echo ">>> Configuring kind registry"
REGISTRY_NAME='kind-registry'
REGISTRY_DIR="/etc/containerd/certs.d/localhost:5001"

if [ "$(docker inspect -f '{{.State.Running}}' "${REGISTRY_NAME}" 2>/dev/null || true)" != 'true' ]; then
  docker run \
    -d --restart=always -p "127.0.0.1:5001:5000" --network bridge --name "${REGISTRY_NAME}" \
    registry:2
fi

for node in $(kind get nodes --name foundry); do
  docker exec "${node}" mkdir -p "${REGISTRY_DIR}"
  cat <<EOF | docker exec -i "${node}" cp /dev/stdin "${REGISTRY_DIR}/hosts.toml"
[host."http://${REGISTRY_NAME}:5000"]
EOF
done

if [ "$(docker inspect -f='{{json .NetworkSettings.Networks.kind}}' "${REGISTRY_NAME}")" = 'null' ]; then
  docker network connect "kind" "${REGISTRY_NAME}"
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

# Deploy postgres
echo ">>> Deploying postgres"
kubectl apply -f manifests/postgres.yml
kubectl wait --for=condition=available --timeout=60s deployment/postgres

# Deploy API server
echo ">>> Deploying API server"
earthly ../api+docker
docker tag foundry-api:latest localhost:5001/foundry-api:latest
docker push localhost:5001/foundry-api:latest
kubectl apply -f manifests/api.yml
kubectl wait --for=condition=available --timeout=60s deployment/api

# Deploy operator
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

# Deploy Gitea
echo ">>> Deploying Gitea"
kubectl apply -f manifests/gitea.yml
kubectl wait --for=condition=available --timeout=60s deployment/gitea
POD_NAME=$(kubectl get pods -l app=gitea -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it "${POD_NAME}" -- gitea admin user create --admin --username ${GITEA_USER} --password ${GITEA_PASS} --email ${GITEA_EMAIL}
TOKEN=$(kubectl exec -it "${POD_NAME}" -- gitea admin user generate-access-token --raw --username ${GITEA_USER} --token-name "foundry" --scopes "write:user,write:repository")
echo "Gitea Token: ${TOKEN}"

# Create git repos
echo ">>> Creating git repos (sleeping for 5 seconds)"
sleep 5
curl -X POST "localhost:3000/api/v1/user/repos" \
  -H "Content-Type: application/json" \
  -H "Authorization: token ${TOKEN}" \
  -d "{
    \"name\": \"deployment\",
    \"private\": false,
    \"description\": \"Deployment repo\"
  }"
curl -X POST "localhost:3000/api/v1/user/repos" \
  -H "Content-Type: application/json" \
  -H "Authorization: token ${TOKEN}" \
  -d "{
    \"name\": \"source\",
    \"private\": false,
    \"description\": \"Source repo\"
  }"

# Push git repos
echo ">>> Pushing git repos"
if [ -d "./git" ]; then
  rm -rf "./git"
fi
mkdir -p git

cp -r repos/deploy git/deploy
cd git/deploy &&
  git init &&
  git config user.name "${GITEA_USER}" &&
  git config user.email "${GITEA_EMAIL}" &&
  git add . && git commit -m "Initial commit" &&
  git remote add origin "http://${GITEA_USER}:${GITEA_PASS}@localhost:3000/${GITEA_USER}/deployment.git" &&
  git push -u origin master

cp -r repos/source git/source
cd git/source &&
  git init &&
  git config user.name "${GITEA_USER}" &&
  git config user.email "${GITEA_EMAIL}" &&
  git add . && git commit -m "Initial commit" &&
  git remote add origin "http://${GITEA_USER}:${GITEA_PASS}@localhost:3000/${GITEA_USER}/source.git" &&
  git push -u origin master
