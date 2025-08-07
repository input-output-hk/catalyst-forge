up:
    earthly --config "" +docker && docker compose up -d certs kcl-publisher renderer

down:
    docker compose down -v

update:
    earthly --config "" +docker && docker compose up -d --no-deps renderer

docker:
    earthly --config "" +docker

docker-test:
    earthly --config "" +docker-test

logs:
    docker compose logs renderer

test:
    docker compose up renderer-test
