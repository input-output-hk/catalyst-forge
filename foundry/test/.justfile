up: kind-up postgres-up api-up operator-up gitea-up

down:
  kind delete cluster --name foundry
  docker stop kind-registry
  docker rm kind-registry

api-up:
  ./scripts/api.sh

git-up:
  ./scripts/git.sh

gitea-up:
  ./scripts/gitea.sh

kind-up:
  ./scripts/kind.sh

operator-up:
  ./scripts/operator.sh

postgres-up:
  ./scripts/postgres.sh
