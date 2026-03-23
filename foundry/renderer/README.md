# Catalyst Forge Renderer Service

The Renderer Service is a gRPC microservice that converts deployment bundles into rendered YAML manifests using the Catalyst Forge deployment system.

## Overview

This service provides a gRPC API that takes deployment bundles as input and returns rendered YAML manifests using the existing `lib/deployment` generator logic. It supports multiple manifest generators (KCL, Helm, Git) and can merge environment data with deployment bundles.

## Features

- **gRPC API**: Efficient binary protocol for high-performance rendering
- **Multi-provider Support**: Supports KCL, Helm, and Git manifest generators
- **KCL OCI Caching**: Automatic caching of KCL OCI modules for improved performance
- **Environment Merging**: Merge environment-specific data with deployment bundles
- **Health Checking**: Built-in health check endpoint
- **Structured Logging**: JSON and text logging with configurable levels
- **Graceful Shutdown**: Proper signal handling and graceful shutdown

## API

### RendererService

#### RenderManifests
Renders deployment bundles into YAML manifests.

**Request:**
- `bundle` (ModuleBundle): The deployment bundle to render
- `env_data` (bytes): Optional environment data to merge with the bundle

**Response:**
- `manifests` (map[string]bytes): Rendered YAML manifests keyed by module name
- `bundle_data` (bytes): Raw bundle data that was processed
- `error` (string): Error message if rendering failed

#### HealthCheck
Provides service health status.

**Request:** Empty

**Response:**
- `status` (string): Service status ("ok")
- `timestamp` (int64): Unix timestamp of the health check

## Usage

### Building

```bash
# Build the service binary
earthly +build

# Or build locally
go build ./cmd/renderer
```

### Running

```bash
# Start the server with default settings
./renderer serve

# Start with custom port and debug logging
./renderer serve --port 9090 --debug --log-json

# Check version
./renderer version

# Get help
./renderer --help
./renderer serve --help
```

### Commands

- `serve`: Start the gRPC renderer service
  - `--port, -p`: gRPC server port (default: 8080)
  - `--debug, -d`: Enable debug logging
  - `--log-json`: Enable JSON logging format
  - `--cache-path, -c`: Path to cache directory for KCL OCI modules (default: `/tmp/renderer-cache`)
- `version`: Print version information

### Docker

```bash
# Build Docker image
earthly +image

# Run container
docker run -p 8080:8080 renderer:latest
```

## Development

### Project Structure

```
foundry/renderer/
├── cmd/renderer/           # Main application entrypoint
├── internal/
│   ├── server/            # gRPC server implementation
│   └── service/           # Business logic and service implementation
├── pkg/proto/             # Generated gRPC code
├── proto/                 # Protocol buffer definitions
└── scripts/               # Build and generation scripts
```

### Generating Protocol Buffers

```bash
# Generate gRPC code
earthly +proto-gen
```

### Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

### Dependencies

The service uses local dependencies from the Catalyst Forge monorepo:
- `lib/deployment`: Core deployment generation logic
- `lib/schema`: Protocol buffer definitions
- `lib/tools`: Utility functions and test helpers

## Configuration

The service is configured via command-line flags. Environment variables are not currently supported but could be added in the future.

### KCL OCI Module Caching

The renderer service supports automatic caching of KCL OCI modules to improve performance for subsequent renderings. When a KCL module is specified with an `oci://` URL, the service will:

1. **Download Once**: Download the OCI module to the local cache directory on first use
2. **Reuse Cached**: Use the cached version for subsequent requests with the same module
3. **Hash-based Storage**: Store modules in subdirectories based on the OCI URL hash to avoid conflicts

#### Cache Configuration

- **Cache Path**: Configure with `--cache-path` flag (default: `/tmp/renderer-cache`)
- **Directory Creation**: Cache directory is automatically created with proper permissions
- **Validation**: Service validates cache directory exists and is writable on startup

#### Cache Benefits

- **Performance**: Significant speed improvement for repeated module usage
- **Reliability**: Reduces network dependencies after initial download
- **Storage Efficiency**: Shared cache across multiple rendering requests

## Logging

The service uses structured logging with support for both text and JSON formats. Log levels can be configured with the `-debug` flag.

Example log output:
```json
{
  "time": "2025-01-01T12:00:00Z",
  "level": "INFO",
  "msg": "Starting gRPC server",
  "component": "grpc-server",
  "address": ":8080"
}
```

## Error Handling

The service follows gRPC best practices for error handling:
- Business logic errors are returned in the response `error` field
- System errors are returned as gRPC status codes
- All errors are logged with appropriate context

## Performance

The service is designed for high throughput:
- Uses efficient protocol buffers for serialization
- Leverages existing optimized deployment generation logic
- Supports concurrent request processing
- Minimal memory allocations in hot paths

## Security

- No authentication/authorization is currently implemented
- Service should be deployed behind a reverse proxy or service mesh
- Input validation is performed on all requests
- No sensitive information is logged

## Monitoring

The service exposes:
- Health check endpoint for liveness probes
- Structured logs for observability
- gRPC server metrics (if enabled)

Future improvements could include:
- Prometheus metrics
- OpenTelemetry tracing
- Performance profiling endpoints