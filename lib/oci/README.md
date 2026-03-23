# OCI Registry Client

High-level OCI registry client for pulling and extracting container images with optional signature verification.

## Usage

### Basic Usage
```go
package main

import (
    "github.com/input-output-hk/catalyst-forge/lib/oci"
)

func main() {
    // Uses default in-memory store and OS filesystem for extraction
    client := oci.New()

    err := client.Pull("ghcr.io/example/image:latest", "./output")
    if err != nil {
        // handle error
    }
}
```

### With Custom Store
```go
import (
    "oras.land/oras-go/v2/content/memory"
    "github.com/input-output-hk/catalyst-forge/lib/oci"
)

// Use ORAS memory store
store := memory.New()
client := oci.New(oci.WithStore(store))
err := client.Pull("ghcr.io/example/image:latest", "./output")
```

### With Filesystem-Based Store
```go
import "github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"

// Create filesystem-based store automatically
tempFS := billy.NewInMemoryFs()
client := oci.New(oci.WithFSStore(tempFS, "/temp-storage"))
err := client.Pull("ghcr.io/example/image:latest", "./output")
```

### With Custom Destination Filesystem
```go
// Extract to in-memory filesystem instead of OS filesystem
destFS := billy.NewInMemoryFs()
client := oci.New(oci.WithDestFS(destFS))
err := client.Pull("ghcr.io/example/image:latest", "/extracted-content")
```

### Complete Custom Setup
```go
// Separate filesystems for temporary storage and extraction
tempFS := billy.NewInMemoryFs()  // For temporary image storage
destFS := billy.NewInMemoryFs()  // For extracted content

store, _ := oci.NewFilesystemStore(tempFS, "/temp")

client := oci.New(
    oci.WithContext(ctx),
    oci.WithStore(store),
    oci.WithDestFS(destFS),
)
```

## Signature Verification (Cosign)

The client supports optional cosign signature verification for images signed with keyless OIDC (e.g., GitHub Actions):

### Basic Cosign Verification
```go
// Verify images signed by GitHub Actions
client := oci.New(
    oci.WithCosignVerification(
        "https://token.actions.githubusercontent.com",
        "https://github.com/myorg/myrepo/.github/workflows/release.yml@refs/heads/main",
    ),
)

err := client.Pull("ghcr.io/myorg/signed-image:latest", "./output")
// Will fail if signature verification fails
```

### Advanced Cosign Options
```go
// Custom Rekor instance or skip transparency log
client := oci.New(
    oci.WithCosignVerificationAdvanced(oci.CosignOptions{
        OIDCIssuer:  "https://token.actions.githubusercontent.com",
        OIDCSubject: "https://github.com/myorg/myrepo/.github/workflows/release.yml@refs/heads/main",
        RekorURL:    "https://custom.rekor.example.com", // Custom Rekor
        SkipTLog:    false,                              // Use transparency log
    }),
)
```

### Verification for Different GitHub Refs
```go
// For main branch builds
oidcSubject := "https://github.com/myorg/myrepo/.github/workflows/build.yml@refs/heads/main"

// For tag releases
oidcSubject := "https://github.com/myorg/myrepo/.github/workflows/release.yml@refs/tags/v1.2.3"

// For pull requests
oidcSubject := "https://github.com/myorg/myrepo/.github/workflows/test.yml@refs/pull/123/merge"
```

### Custom Verifier
```go
// Implement your own verification logic
type MyVerifier struct{}

func (v *MyVerifier) Verify(ctx context.Context, imageRef string, manifest ocispec.Descriptor) error {
    // Custom verification logic
    return nil
}

client := oci.New(oci.WithCustomVerifier(&MyVerifier{}))
```