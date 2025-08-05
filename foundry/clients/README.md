# Foundry API Clients

This directory contains auto-generated API clients for the Foundry API.

## Structure

```
clients/
├── python/     # Python client
├── typescript/ # TypeScript client
└── justfile    # Build commands
```

## Generating Clients

All clients are generated from the OpenAPI specification located at `../api/docs/swagger.yaml`.

### Generate all clients:
```bash
just generate
```

### Generate specific client:
```bash
just go-client
```

### Clean generated files:
```bash
just clean
```

### Test compilation:
```bash
just test
```

## Go Client

The Go client is generated using OpenAPI Generator v7.10.0. The generated code includes:

- API clients for all endpoints
- Model definitions
- Documentation

### Usage Example

```go
package main

import (
    "context"
    "fmt"
    foundryclient "github.com/input-output-hk/catalyst-forge/foundry/clients/go"
)

func main() {
    cfg := foundryclient.NewConfiguration()
    cfg.Servers = foundryclient.ServerConfigurations{
        {URL: "http://localhost:5050"},
    }

    client := foundryclient.NewAPIClient(cfg)

    // Example: Get health status
    resp, _, err := client.HealthAPI.HealthzGet(context.Background()).Execute()
    if err != nil {
        panic(err)
    }
    fmt.Printf("Health: %v\n", resp)
}
```

## CI Integration

The `just check` command can be used in CI to verify that generated files are up-to-date:

```bash
just check
```

This will clean, regenerate, and check if there are any uncommitted changes.

## Adding New Clients

To add support for a new language:

1. Create a new directory (e.g., `python/`)
2. Add a generation configuration file (e.g., `python-config.yaml`)
3. Update the `justfile` with a new recipe
4. Add appropriate `.gitignore` and `.openapi-generator-ignore` files

## Configuration

Client generation is configured via YAML files:
- `go/go-config.yaml` - Go client configuration