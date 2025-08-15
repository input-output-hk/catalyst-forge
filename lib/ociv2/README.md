# OCI v2 Client Library

A clean, testable Go library for pushing and pulling OCI artifacts with support for signing and verification using Cosign.

## Features

- **Digest-first semantics** - All operations return canonical digest references for immutability
- **Registry compatibility** - Automatic handling of ECR, GHCR, Docker Hub quirks
- **Manifest fallback** - Seamlessly falls back to image manifests when artifact manifests aren't supported
- **Cosign integration** - Built-in support for keyless signing and verification
- **Structured errors** - Categorized errors with retry hints and detailed context
- **Observability** - Structured logging and metrics collection
- **Type-safe builders** - Fluent APIs for annotations and configuration

## Installation

```bash
go get github.com/input-output-hk/catalyst-forge/lib/ociv2
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/input-output-hk/catalyst-forge/lib/ociv2"
)

func main() {
    // Create a client
    client, err := ociv2.New(ociv2.ClientOptions{
        Timeout: 2 * time.Minute,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    ctx := context.Background()
    
    // Push a JSON artifact
    data := []byte(`{"version": "1.0.0", "name": "myapp"}`)
    desc, err := client.PushJSON(ctx, 
        "oci://registry.example.com/myapp:latest",
        "application/vnd.myapp.config+json",
        data,
        ociv2.NewAnnotations().
            WithTitle("My Application").
            WithVersion("1.0.0"))
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Pushed artifact: %s", desc.Ref)
    
    // Pull using the digest reference
    pulledData, _, err := client.PullJSON(ctx, desc.Ref, 
        "application/vnd.myapp.config+json")
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Retrieved: %s", string(pulledData))
}
```

## Configuration Examples

### Basic Configuration

```go
client, err := ociv2.New(ociv2.ClientOptions{
    Timeout:   2 * time.Minute,
    PlainHTTP: false,  // Use HTTPS (default)
})
```

### With Signing Enabled

```go
client, err := ociv2.New(ociv2.ClientOptions{
    Timeout: 2 * time.Minute,
    Cosign: ociv2.CosignOpts{
        Enable: true,
        Identity: &ociv2.OIDCIdentity{
            Issuer:  "https://token.actions.githubusercontent.com",
            Subject: "repo:myorg/myrepo:ref:refs/heads/main",
        },
    },
})
```

### With Custom Logging and Metrics

```go
client, err := ociv2.New(ociv2.ClientOptions{
    StructuredLogger: &MyLogger{},
    EnableMetrics:    true,
    MetricsCallback: func(m *ociv2.Metrics) {
        // Export to your monitoring system
        prometheus.RecordMetrics(m)
    },
})
```

### Registry Compatibility Mode

```go
client, err := ociv2.New(ociv2.ClientOptions{
    PreferArtifactManifest: true,  // Try OCI artifacts first
    FallbackImageManifest:  true,  // Fallback if not supported
})
```

## Authentication Setup

### Docker Config (Default)

The client automatically uses `~/.docker/config.json`:

```bash
# Login to registry
docker login registry.example.com

# Client will use these credentials
```

### GitHub Container Registry (GHCR)

```go
auth := &ociv2.GitHubAuth{
    Token: os.Getenv("GITHUB_TOKEN"),
}

client, err := ociv2.New(ociv2.ClientOptions{
    Auth: auth,
})
```

Required token permissions:
- `read:packages` for pulling
- `write:packages` for pushing
- `delete:packages` for deletion

### Amazon ECR

```go
// Uses AWS SDK credential chain
auth := &ociv2.ECRAuth{}

client, err := ociv2.New(ociv2.ClientOptions{
    Auth: auth,
})
```

Credentials resolved in order:
1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. IAM role (when running on EC2/ECS/Lambda)

### Static Credentials

```go
auth := &ociv2.StaticAuth{
    Username: "myuser",
    Password: os.Getenv("REGISTRY_TOKEN"),
}

client, err := ociv2.New(ociv2.ClientOptions{
    Auth: auth,
})
```

### Multiple Registries

```go
auth := &ociv2.ChainAuth{
    Providers: []ociv2.AuthProvider{
        &ociv2.GitHubAuth{Token: ghToken},     // GHCR
        &ociv2.ECRAuth{},                      // ECR
        &ociv2.DefaultAuth{},                  // Docker config
    },
}

client, err := ociv2.New(ociv2.ClientOptions{
    Auth: auth,
})
```

## Working with Domain Types

### Release Bundles

```go
// Push a release bundle
releaseData := []byte(`{"version": "1.0.0", "components": [...]}`)
desc, err := client.PushReleaseBundle(ctx,
    "oci://registry.example.com/releases/myapp:v1.0.0",
    releaseData,
    ociv2.NewAnnotations().
        WithForgeProject("myapp").
        WithForgeEnv("production"))

// Pull a release bundle
data, desc, err := client.PullReleaseBundle(ctx, desc.Ref)
```

### Rendered Sets

```go
// Push a rendered set (index + tar)
indexData := []byte(`{"manifests": [...]}`)
tarReader := getTarReader() // io.Reader with tar.gz content
tarSize := int64(1024 * 1024) // Size in bytes

desc, err := client.PushRenderedSet(ctx,
    "oci://registry.example.com/rendered/myapp:v1.0.0",
    indexData,
    tarReader,
    tarSize,
    ociv2.NewAnnotations().
        WithForgeProject("myapp"))

// Pull a rendered set
index, tarReader, desc, err := client.PullRenderedSet(ctx, ref)
defer tarReader.Close()

// Process the tar content
data, err := io.ReadAll(tarReader)
```

## Signing and Verification

### Keyless Signing (GitHub Actions)

```yaml
# .github/workflows/release.yml
env:
  COSIGN_EXPERIMENTAL: 1

steps:
  - name: Push and sign artifact
    run: |
      # Client will automatically use GitHub OIDC token
      myapp push --sign
```

### Verify with Identity Constraints

```go
client, err := ociv2.New(ociv2.ClientOptions{
    Cosign: ociv2.CosignOpts{
        Enable: true,
        Identity: &ociv2.OIDCIdentity{
            Issuer:  "https://token.actions.githubusercontent.com",
            Subject: "repo:myorg/myrepo:*", // Wildcard supported
        },
    },
})

report, err := client.VerifyDigest(ctx, artifactRef)
if err != nil {
    log.Fatal("Verification failed:", err)
}

if report.Signed {
    log.Printf("Signed by: %s", report.SignerIdentity.Subject)
    log.Printf("Issuer: %s", report.SignerIdentity.Issuer)
}
```

## Troubleshooting

### Registry Doesn't Support Artifact Manifests

**Problem**: Registry returns 400/415 errors when pushing artifacts.

**Solution**: Enable fallback mode:
```go
client, err := ociv2.New(ociv2.ClientOptions{
    FallbackImageManifest: true,  // Automatically use image manifests
})
```

### Authentication Failures

**Problem**: Getting 401 Unauthorized errors.

**Debug steps**:
1. Check Docker config:
   ```bash
   cat ~/.docker/config.json | jq '.auths'
   ```

2. Test with docker CLI:
   ```bash
   docker pull registry.example.com/test
   ```

3. Enable debug logging:
   ```go
   client, err := ociv2.New(ociv2.ClientOptions{
       Logger: func(format string, args ...interface{}) {
           log.Printf("[DEBUG] "+format, args...)
       },
   })
   ```

### Signing Fails in CI

**Problem**: Cosign signing fails with "no provider" error.

**Solution for GitHub Actions**:
```yaml
permissions:
  id-token: write  # Required for OIDC
  packages: write  # Required for GHCR

env:
  COSIGN_EXPERIMENTAL: 1
```

**Solution for local development**:
```go
// Use insecure mode for testing only
client, err := ociv2.New(ociv2.ClientOptions{
    Cosign: ociv2.CosignOpts{
        Enable:        true,
        AllowInsecure: true,  // Testing only!
    },
})
```

### Timeout Errors

**Problem**: Operations timeout with large artifacts.

**Solution**: Increase timeout:
```go
client, err := ociv2.New(ociv2.ClientOptions{
    Timeout: 10 * time.Minute,  // Increase for large files
})
```

### ECR Cross-Region Issues

**Problem**: Cannot push to ECR in different region.

**Solution**: Set AWS region:
```bash
export AWS_REGION=us-west-2
# or
export AWS_DEFAULT_REGION=us-west-2
```

### Digest Mismatch Errors

**Problem**: Getting "digest mismatch" errors.

**Possible causes**:
1. Content modified during transfer
2. Registry corruption
3. Network issues

**Solution**: Enable retries and validate locally:
```go
// Compute expected digest
digest := sha256.Sum256(data)
expected := "sha256:" + hex.EncodeToString(digest[:])

// Compare with returned digest
if desc.Digest != expected {
    log.Fatal("Digest mismatch!")
}
```

## Error Handling

The library provides structured errors with categories:

```go
desc, err := client.PushJSON(ctx, ref, mediaType, data, nil)
if err != nil {
    var ociErr *ociv2.OCIError
    if errors.As(err, &ociErr) {
        switch ociErr.Category {
        case ociv2.ErrorCategoryAuth:
            // Handle authentication error
            refreshCredentials()
        case ociv2.ErrorCategoryNetwork:
            if ociErr.IsRetryable() {
                // Retry with backoff
                time.Sleep(time.Second)
                retry()
            }
        case ociv2.ErrorCategoryRegistry:
            // Registry-specific issue
            log.Printf("Registry error: %s", ociErr.RegistryError)
        }
    }
}
```

## Performance Tips

1. **Reuse clients** - Create once, use many times
2. **Use digest references** - Avoid unnecessary resolves
3. **Enable metrics** - Monitor operation latency
4. **Set appropriate timeouts** - Based on artifact size
5. **Use streaming** - For large tar files

## API Stability

This library follows semantic versioning. The public API includes:
- All exported types in the root package
- All exported methods on those types
- Media type constants
- Error variables

Internal packages (`internal/`) are not part of the public API and may change.

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for development setup and guidelines.

## License

This project is licensed under the Apache License 2.0. See [LICENSE](../../LICENSE) for details.