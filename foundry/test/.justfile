up: kind-up postgres-up gitea-up api-up operator-up git-up

up-local: kind-up postgres-up gitea-up operator-up-local api-up git-up

down:
  kind delete cluster --name foundry

api:
  earthly --config "" ../api+docker
  docker tag foundry-api:latest localhost:5001/foundry-api:latest
  docker push localhost:5001/foundry-api:latest
  kubectl delete deployment api
  kubectl apply -f manifests/api.yml

api-up:
  ./scripts/api.sh

cleanup-local:
  rm -rf ~/.cache/forge

git-up:
  ./scripts/git.sh

gitea-up:
  ./scripts/gitea.sh

kind-up:
  ./scripts/kind.sh

operator:
  earthly --config "" ../operator+docker
  docker tag foundry-operator:latest localhost:5001/foundry-operator:latest
  docker push localhost:5001/foundry-operator:latest
  kubectl delete deployment operator

  sed "s/GIT_TOKEN/$(cat .token)/g" manifests/operator.yml | kubectl apply -f -
  kubectl wait --for=condition=available --timeout=60s deployment/operator
  kubectl apply -f manifests/operator.yml

operator-local:
  go build -C ../operator -o ../test/bin/operator cmd/main.go
  ./bin/operator --config ./data/local.json

operator-up:
  ./scripts/operator.sh

operator-up-local:
  kubectl apply -f ../operator/config/crd/bases/foundry.projectcatalyst.io_releasedeployments.yaml
  jq -R '{token: .}' .token >./data/auth.json

postgres-up:
  ./scripts/postgres.sh

release:
  ./scripts/release.sh

release-local:
  ./scripts/release.sh --local