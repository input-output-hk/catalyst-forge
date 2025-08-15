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

### Bootstrap Process for Initial Admin Setup

For first-time deployment or local testing, you can use the bootstrap process to create the initial admin account:

#### 1. Configure Bootstrap Token

Set a secure bootstrap token (minimum 32 characters) when starting the API:

```bash
export BOOTSTRAP_TOKEN="your-secure-random-token-here-min-32-chars"
./foundry-api run
```

#### 2. Create Initial Admin Invite

Make a single POST request to create an admin invite (this can only be done once):

```bash
curl -X POST https://api.example.com/auth/bootstrap \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "bootstrap_token": "your-secure-random-token-here-min-32-chars"
  }'
```

Response:
```json
{
  "id": 1,
  "token": "invite-token-here"
}
```

#### 3. Complete Registration

Use the returned invite token to complete the normal device registration flow:

1. **Initialize device registration:**
   ```bash
   curl -X POST https://api.example.com/auth/devices/init \
     -H "Content-Type: application/json" \
     -d '{
       "token": "invite-token-here",
       "invite_id": 1
     }'
   ```

2. **Complete device registration** with your browser's ECDSA P-256 key and device proof.

3. **Continue with normal authentication** using the issued access and refresh tokens.

#### Important Notes

- **One-time use**: The bootstrap token can only be used once and is tracked in the database
- **Admin role creation**: The bootstrap process automatically creates an "admin" role with all permissions
- **Security**: Remove the `BOOTSTRAP_TOKEN` environment variable after initial setup
- **Local testing**: This same process works for local development and testing environments

For detailed authentication flows, see the files `1.md` and `2.md` in this directory.

## Configuration

The API can be configured using environment variables or command-line flags. See the main application help for details:

```bash
./bin/foundry-api --help
```

### Key Environment Variables

- `BOOTSTRAP_TOKEN` - One-time bootstrap token for creating initial admin invite (min 32 chars, optional)
- `AUTH_PRIVATE_KEY` - Path to private key for JWT authentication
- `AUTH_PUBLIC_KEY` - Path to public key for JWT authentication  
- `INVITE_HASH_SECRET` - Secret for HMAC invite token hashing
- `REFRESH_HASH_SECRET` - Secret for HMAC refresh token validation
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` - Database connection parameters
- `PUBLIC_BASE_URL` - Public base URL for generating links (e.g., https://api.example.com)

For a complete list of configuration options, use `--help` or see the config struct in `internal/config/config.go`.

## Documentation Generation

To regenerate the API documentation after making changes:

```bash
make swagger-gen
```

This will update the `docs/` directory with the latest API documentation.

## Testing

The project has two types of tests:

### Unit Tests

Run unit tests only (excludes integration tests):

```bash
go test ./...
```

Or using the justfile:

```bash
just test
```

### Integration Tests

Integration tests use Testcontainers and require Docker. They are tagged with `//go:build integration` to prevent them from running with regular `go test ./...` commands.

Run integration tests only:

```bash
go test -tags=integration ./test -v
```

Or using the justfile:

```bash
just test-integration
```

### All Tests

Run both unit and integration tests:

```bash
just test-all
```

**Note**: Integration tests are excluded from `go test ./...` by design to keep regular test runs fast. Use the appropriate command or justfile target to run them when needed.

## Docker Development Environment

The included `docker-compose.yml` provides a complete local development environment using the modern bootstrap system:

### Quick Start

1. **Build the API image:**
   ```bash
   docker build -t foundry-api:latest .
   ```

2. **Start the development stack:**
   ```bash
   docker-compose up -d
   ```

3. **Bootstrap the initial admin (optional):**
   ```bash
   docker-compose --profile bootstrap up bootstrap-admin
   ```

### Services Included

- **api**: Main API server with bootstrap authentication
- **postgres**: PostgreSQL database with automatic schema migration
- **pgadmin**: Web-based PostgreSQL administration (accessible at `http://localhost:5051`)
- **auth-init**: One-time JWT key generation service
- **bootstrap-admin**: Optional admin user creation (profile: `bootstrap`)
- **mockdata**: Optional test data population (profile: `mockdata`)

### Admin Access

The Docker environment uses the same bootstrap system as production:

1. The bootstrap service creates an admin invite using the `/auth/bootstrap` endpoint
2. Use the device registration flow with the created invite to get access tokens
3. For programmatic access, use the `testutil.BootstrapAdmin` function from the test utilities

### Configuration

The compose environment uses development-safe defaults:
- Bootstrap token: `dev-bootstrap-token-change-in-production`
- Admin email: `admin@foundry.dev`
- Database: `foundry/foundry` with user `foundry:changeme`
- API endpoints: `http://localhost:5050`

**Important:** Change the bootstrap token in production environments!

## License

This project is licensed under the Apache License 2.0.