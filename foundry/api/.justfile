# Build the Foundry API binary
build:
    mkdir -p bin && go build -o bin/foundry-api ./cmd/api

# Run checks on the codebase
check:
    go mod tidy && go fmt ./... && go vet ./... && golangci-lint run

# Start the local development environment
up:
    earthly --config "" +docker && docker compose up -d postgres api pgadmin caddy

# Stop the local development environment
down:
    docker compose down -v

# Update the Foundry API container in the local development environment
update:
    earthly --config "" +docker && docker compose up -d --no-deps api

# Build the Foundry API container
docker:
    earthly --config "" +docker

# Show the logs for the Foundry API container
logs:
    docker compose logs api

# Run unit tests (excludes integration tests)
test:
    if command -v gotestsum >/dev/null 2>&1; then \
        gotestsum --no-summary=output,skipped ./...; \
    else \
        go test ./...; \
    fi

# Run integration tests only
test-integration:
    if command -v gotestsum >/dev/null 2>&1; then \
        gotestsum --no-summary=output,skipped -- -tags=integration -count=1 -p 1 -parallel 1 ./test; \
    else \
        go test -tags=integration -count=1 -p 1 -parallel 1 ./test -v; \
    fi

# Run all tests (unit + integration)
test-all:
    if command -v gotestsum >/dev/null 2>&1; then \
        gotestsum --no-summary=output,skipped ./... && gotestsum --format=standard-verbose -- -tags=integration -count=1 -p 1 -parallel 1 ./test; \
    else \
        go test ./... && go test -tags=integration -count=1 -p 1 -parallel 1 ./test -v; \
    fi

# Generate the Swagger documentation
swagger:
    earthly --config "" +swagger