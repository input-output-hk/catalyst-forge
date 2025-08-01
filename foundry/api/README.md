# Catalyst Foundry API

This is the API server for the Catalyst Foundry system, providing endpoints for managing releases and deployments.

## API Documentation

The API documentation is generated using Swagger/OpenAPI and is available in two formats:

1. **Interactive Swagger UI**: Available at `/swagger/index.html` when the server is running
2. **OpenAPI JSON**: Available at `/swagger/doc.json` when the server is running

## Getting Started

### Prerequisites

- Go 1.24.2 or later
- PostgreSQL database
- Kubernetes cluster (optional, for deployment features)

### Installation

1. Install dependencies:
   ```bash
   make deps
   ```

2. Install Swagger tools (one-time setup):
   ```bash
   make swagger-init
   ```

3. Generate API documentation:
   ```bash
   make swagger-gen
   ```

4. Build and run the API:
   ```bash
   make run
   ```

### Development

For development with auto-generated documentation:

```bash
make dev
```

This will generate the documentation and start the server.

## API Endpoints

### Health Check
- `GET /healthz` - Check API health status

### GitHub Actions Authentication
- `POST /gha/validate` - Validate GitHub Actions OIDC token
- `POST /gha/auth` - Create GHA authentication configuration
- `GET /gha/auth` - List GHA authentication configurations
- `GET /gha/auth/:id` - Get specific GHA authentication configuration
- `GET /gha/auth/repository/:repository` - Get GHA auth by repository
- `PUT /gha/auth/:id` - Update GHA authentication configuration
- `DELETE /gha/auth/:id` - Delete GHA authentication configuration

### Releases
- `POST /release` - Create a new release
- `GET /release/:id` - Get a specific release
- `PUT /release/:id` - Update a release
- `GET /releases` - List all releases

### Release Aliases
- `GET /release/alias/:name` - Get release by alias
- `POST /release/alias/:name` - Create an alias for a release
- `DELETE /release/alias/:name` - Delete an alias
- `GET /release/:id/aliases` - List aliases for a release

### Deployments
- `POST /release/:id/deploy` - Create a deployment for a release
- `GET /release/:id/deploy/:deployId` - Get a specific deployment
- `PUT /release/:id/deploy/:deployId` - Update a deployment
- `GET /release/:id/deployments` - List deployments for a release
- `GET /release/:id/deploy/latest` - Get the latest deployment

### Deployment Events
- `POST /release/:id/deploy/:deployId/events` - Add an event to a deployment
- `GET /release/:id/deploy/:deployId/events` - Get events for a deployment

## Authentication

The API uses JWT tokens for authentication. Most endpoints require authentication with the following permissions:

- `PermReleaseRead` - Read access to releases
- `PermReleaseWrite` - Write access to releases
- `PermDeploymentRead` - Read access to deployments
- `PermDeploymentWrite` - Write access to deployments
- `PermDeploymentEventRead` - Read access to deployment events
- `PermDeploymentEventWrite` - Write access to deployment events
- `PermGHAAuthRead` - Read access to GHA authentication
- `PermGHAAuthWrite` - Write access to GHA authentication

## Configuration

The API can be configured using environment variables or command-line flags. See the main application help for details:

```bash
./bin/foundry-api --help
```

## Documentation Generation

To regenerate the API documentation after making changes:

```bash
make swagger-gen
```

This will update the `docs/` directory with the latest API documentation.

## Testing

Run the tests:

```bash
go test ./...
```

## License

This project is licensed under the Apache License 2.0.