up:
    docker compose up -d auth auth-jwt api postgres

down:
    rm -rf .secret && docker compose down

docker:
    earthly --config "" +docker

login:
    (cd ../../cli && go run cmd/main.go -vvv --api-url "http://localhost:5050" api login "$(cat ../foundry/api/.secret/jwt.txt)")

swagger:
    earthly --config "" +swagger