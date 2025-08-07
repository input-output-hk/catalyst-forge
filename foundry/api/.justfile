up:
    earthly --config "" +docker && docker compose up -d auth auth-jwt api postgres pgadmin step-ca

down:
    rm -rf .auth && docker compose down -v

update:
    earthly --config "" +docker && docker compose up -d --no-deps api

docker:
    earthly --config "" +docker

docker-test:
    earthly --config "" +docker-test

register:
    ./scripts/tests/register.sh

login:
    (cd ../../cli && go run cmd/main.go -vvv --api-url "http://localhost:5050" api login)

login-admin:
    (cd ../../cli && go run cmd/main.go -vvv --api-url "http://localhost:5050" api login --token "$(cat ../foundry/api/.auth/jwt.txt)")

logs:
    docker compose logs api

test:
    docker compose up api-test

swagger:
    earthly --config "" +swagger
