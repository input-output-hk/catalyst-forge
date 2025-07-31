.PHONY: swagger-init swagger-gen swagger-serve clean

# Initialize swag (run once to install swag)
swagger-init:
	go install github.com/swaggo/swag/cmd/swag@latest

# Generate swagger documentation
swagger-gen:
	swag init -g cmd/api/main.go -o docs

# Serve swagger documentation (requires the API to be running)
swagger-serve:
	@echo "Swagger UI will be available at: http://localhost:8080/swagger/index.html"
	@echo "Make sure the API server is running first!"

# Clean generated files
clean:
	rm -rf docs/docs.go docs/swagger.json docs/swagger.yaml

# Install dependencies
deps:
	go mod tidy
	go mod download

# Build the application
build: deps
	go build -o bin/foundry-api cmd/api/main.go

# Run the application
run: build
	./bin/foundry-api run

# Generate docs and run
dev: swagger-gen run