# KCL Module Library

A robust caching layer for KCL (KCL Configuration Language) modules with OCI registry integration, providing deterministic execution and two-level caching.

## Overview

The KCL module library adds intelligent caching on top of the `lib/ociv2` package to optimize KCL module distribution and execution. It features:

- **Two-Level Caching**: Module cache (by digest) and run cache (by intent hash)
- **Dual Profile Support**: KPM-compatible and Forge-specific profiles
- **Multiple Execution Engines**: Native (CGO) and WASM (sandboxed)
- **Deterministic Execution**: Reproducible results through canonical JSON and intent hashing
- **Cross-Process Safety**: File-based locking for concurrent operations

## Installation

```go
import "github.com/input-output-hk/catalyst-forge/lib/kcl"
```

## Core API

### Publishing Modules

```go
// Publish a KCL module to an OCI registry
digest, err := kcl.Publish(ctx, ociClient, kcl.PublishOptions{
    ModuleRoot: "./my-module",
    Ref:        "registry.example.com/modules/my-module:v1.0.0",
    Profile:    kcl.ProfileStrict,  // or ProfileCompat for KPM compatibility
    Sign:       true,
    SignKeyRef: "cosign.key",
})
```

### Verifying Modules

```go
// Verify a module's integrity and signatures
digest, metadata, err := kcl.Verify(ctx, ociClient, 
    kcl.ModuleRef{
        Repo: "registry.example.com/modules/my-module",
        Tag:  "v1.0.0",
    },
    kcl.VerifyOptions{
        Profile:          kcl.ProfileStrict,
        RequireSignature: true,
    },
)
```

### Pulling Modules

```go
// Pull a module and cache it locally
result, err := kcl.Pull(ctx, ociClient,
    kcl.ModuleRef{
        Repo: "registry.example.com/modules/my-module",
        Tag:  "v1.0.0",
    },
    kcl.PullOptions{
        Profile:          kcl.ProfileStrict,
        RequireSignature: true,
    },
)

// Access the cached module
fmt.Printf("Module cached at: %s\n", result.Path)
fmt.Printf("Module digest: %s\n", result.Digest)
```

### Running Modules

```go
// Execute a KCL module with caching
result, err := kcl.Run(ctx, ociClient,
    kcl.ModuleRef{
        Repo: "registry.example.com/modules/my-module",
        Tag:  "v1.0.0",
    },
    kcl.PullOptions{
        Profile: kcl.ProfileStrict,
    },
    kcl.RunOptions{
        Engine:     kcl.EngineNative,  // or EngineWASM
        ValuesJSON: []byte(`{"env": "production"}`),
        CtxJSON:    []byte(`{"region": "us-west-2"}`),
    },
)

// Access execution results
fmt.Printf("YAML output:\n%s\n", result.YAMLResult)
fmt.Printf("Cache hit: %v\n", result.CacheHit)
```

### Inspecting Modules

```go
// Inspect module metadata without pulling
info, err := kcl.Inspect(ctx, ociClient,
    kcl.ModuleRef{
        Repo: "registry.example.com/modules/my-module",
        Tag:  "v1.0.0",
    },
    kcl.InspectOptions{
        RequireSignature: false,
    },
)

fmt.Printf("Module: %s v%s\n", info.Metadata.Name, info.Metadata.Version)
fmt.Printf("Profile: %s\n", info.Profile)
fmt.Printf("Signed: %v\n", info.Signed)
```

## Cache Management

### Cache Statistics

```go
// Get comprehensive cache statistics
stats, err := kcl.GetCacheStats()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Cache Statistics:\n")
fmt.Printf("  Modules: %d (%.2f MB)\n", 
    stats.ModulesStats.Count,
    float64(stats.ModulesStats.TotalSize)/(1024*1024))
fmt.Printf("  Runs: %d (%.2f MB)\n",
    stats.RunsStats.Count,
    float64(stats.RunsStats.TotalSize)/(1024*1024))
fmt.Printf("  Total: %.2f MB\n",
    float64(stats.TotalSize)/(1024*1024))
```

### Cache Cleanup

```go
// Clean expired and least-recently-used cache entries
result, err := kcl.CleanCache(false)  // false = not a dry run
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Cleaned: %d modules, %d runs\n",
    result.ModulesRemoved, result.RunsRemoved)
fmt.Printf("Space freed: %.2f MB\n",
    float64(result.SpaceFreed)/(1024*1024))
```

## Configuration

### Environment Variables

```bash
# Cache directory (default: ~/.forge/kcl)
export FORGE_KCL_CACHE_DIR=/custom/cache/path

# Module cache size limit (default: 10GB)
export FORGE_KCL_MODULE_CACHE_MAX_BYTES=5G

# Run cache size limit (default: 1GB) 
export FORGE_KCL_RUN_CACHE_MAX_BYTES=500M

# Cache TTL in days (default: 30)
export FORGE_KCL_TTL_DAYS=7

# Enable blob cache for raw artifacts (default: false)
export FORGE_KCL_ENABLE_BLOB_CACHE=true

# Logging configuration
export FORGE_KCL_LOG_LEVEL=debug  # debug, info, warn, error
export FORGE_KCL_LOG_FORMAT=text   # text or json
```

### Cache Options

```go
// Programmatic cache configuration
opts := kcl.CacheOptions{
    Dir:             "/custom/cache",
    ModulesMaxBytes: 5 * 1024 * 1024 * 1024,  // 5GB
    RunsMaxBytes:    500 * 1024 * 1024,       // 500MB
    TTLDays:         7,
    EnableBlobCache: true,
}

// Apply options (must be done before any operations)
kcl.SetCacheOptions(opts)
```

## Profiles

### ProfileCompat (KPM-Compatible)

- Single tar layer with standard OCI media type
- Metadata derived from manifest annotations
- Compatible with existing KPM tools
- Suitable for public module distribution

### ProfileStrict (Forge-Specific)

- Separate tar and metadata JSON layers
- Rich metadata with CUE schema validation
- Enhanced security and provenance tracking
- Recommended for enterprise deployments

## Execution Engines

### Native Engine (CGO)

- Uses `kcl-go` library directly
- Best performance for trusted modules
- Requires CGO support
- Access to full KCL runtime features

Build with: `go build -tags kcl_native`

### WASM Engine (Sandboxed)

- Uses `wasmer-go` for sandboxed execution
- Enhanced security for untrusted modules
- Platform-independent bytecode
- Some runtime limitations

Build with: `go build -tags kcl_wasm`

## Module Metadata

### Module Structure (kcl.mod)

```toml
name = "my-module"
version = "1.0.0"
description = "Example KCL module"
authors = ["alice@example.com", "bob@example.com"]
license = "Apache-2.0"
repository = "https://github.com/example/my-module"
homepage = "https://example.com/my-module"
```

### Metadata Schema

```go
type ModuleMeta struct {
    Name        string            `json:"name"`
    Version     string            `json:"version"`
    Description string            `json:"description,omitempty"`
    Authors     []string          `json:"authors,omitempty"`
    License     string            `json:"license,omitempty"`
    Repository  string            `json:"repository,omitempty"`
    Homepage    string            `json:"homepage,omitempty"`
    Sum         string            `json:"sum,omitempty"`
    Entry       string            `json:"entry,omitempty"`
    Annotations map[string]string `json:"annotations,omitempty"`
}
```

## Caching Architecture

### Module Cache

- **Key**: SHA-256 digest of module content
- **Location**: `~/.forge/kcl/modules/<digest>/`
- **Contents**: Extracted module files + `.meta.json`
- **Eviction**: LRU when size limit exceeded

### Run Cache

- **Key**: Intent hash (module + inputs + engine)
- **Location**: `~/.forge/kcl/runs/<intent>.yaml`
- **Contents**: KCL execution output
- **Eviction**: TTL-based and LRU

### Intent Hash Computation

```
intent = SHA256(
    moduleDigest +
    canonical(valuesJSON) +
    canonical(contextJSON) +
    engineKind +
    engineVersion
)
```

## Error Handling

Common errors and their meanings:

```go
var (
    ErrModuleNotFound    = errors.New("module not found")
    ErrSignatureInvalid  = errors.New("signature verification failed")
    ErrShapeMismatch     = errors.New("artifact shape mismatch")
    ErrCacheLocked       = errors.New("cache operation locked")
    ErrEngineUnavailable = errors.New("execution engine unavailable")
)
```

## Performance Considerations

1. **Cache Warming**: Pre-fetch frequently used modules
   ```go
   refs := []kcl.ModuleRef{
       {Repo: "registry/module1", Tag: "v1"},
       {Repo: "registry/module2", Tag: "v2"},
   }
   err := kcl.PrefetchModules(ctx, ociClient, refs, pullOpts)
   ```

2. **Batch Operations**: Process multiple modules concurrently
   ```go
   // Modules are automatically deduplicated via singleflight
   ```

3. **Cache Sizing**: Balance memory usage vs performance
   - Module cache: 10-50GB for large deployments
   - Run cache: 1-5GB for typical workloads

## Security

### Signature Verification

All modules can be signed and verified using Cosign:

```go
// Publishing with signature
opts := kcl.PublishOptions{
    Sign:       true,
    SignKeyRef: "cosign.key",
}

// Verification requirement
verifyOpts := kcl.VerifyOptions{
    RequireSignature: true,
}
```

### SLSA Attestations

Generate and attach SLSA provenance:

```go
opts := kcl.PublishOptions{
    Attest:      true,
    AttestBytes: slsaPredicate,  // Or auto-generated
}
```

### Sandboxed Execution

Use WASM engine for untrusted modules:

```go
runOpts := kcl.RunOptions{
    Engine: kcl.EngineWASM,
    // WASM runs in isolated sandbox
}
```

## Monitoring

### Metrics (Prometheus)

The library exposes Prometheus metrics:

- `forge_kcl_module_pulls_total` - Total module pulls
- `forge_kcl_cache_hits_total` - Cache hit count
- `forge_kcl_cache_misses_total` - Cache miss count
- `forge_kcl_cache_modules_size_bytes` - Module cache size
- `forge_kcl_pull_duration_seconds` - Pull operation duration
- `forge_kcl_run_duration_seconds` - Run operation duration

### Logging

Structured logging with configurable levels:

```go
import "github.com/input-output-hk/catalyst-forge/lib/kcl/logging"

logger := logging.GetLogger()
logger.Info("Module pulled", 
    "digest", digest,
    "duration_ms", duration)
```

## Examples

### Complete Workflow

```go
package main

import (
    "context"
    "log"
    
    "github.com/input-output-hk/catalyst-forge/lib/kcl"
    "github.com/input-output-hk/catalyst-forge/lib/ociv2"
)

func main() {
    ctx := context.Background()
    
    // Create OCI client
    ociClient := ociv2.NewClient()
    
    // 1. Publish a module
    digest, err := kcl.Publish(ctx, ociClient, kcl.PublishOptions{
        ModuleRoot: "./my-module",
        Ref:        "registry.example.com/my-module:v1.0.0",
        Profile:    kcl.ProfileStrict,
        Sign:       true,
        SignKeyRef: "cosign.key",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // 2. Verify the published module
    _, metadata, err := kcl.Verify(ctx, ociClient,
        kcl.ModuleRef{
            Repo: "registry.example.com/my-module",
            Tag:  "v1.0.0",
        },
        kcl.VerifyOptions{
            Profile:          kcl.ProfileStrict,
            RequireSignature: true,
        },
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // 3. Run the module (pulls automatically if needed)
    result, err := kcl.Run(ctx, ociClient,
        kcl.ModuleRef{
            Repo: "registry.example.com/my-module",
            Tag:  "v1.0.0",
        },
        kcl.PullOptions{
            Profile: kcl.ProfileStrict,
        },
        kcl.RunOptions{
            Engine:     kcl.EngineNative,
            ValuesJSON: []byte(`{"env": "production"}`),
        },
    )
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Module executed successfully:\n%s", result.YAMLResult)
}
```

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for development guidelines.

## License

Apache 2.0 - See [LICENSE](../../LICENSE) for details.