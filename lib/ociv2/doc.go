// Package ociv2 provides a clean, testable API for pushing and pulling OCI artifacts
// with support for signing and verification using Cosign.
//
// This package abstracts away registry differences (ECR, GHCR, etc.) and provides
// digest-first semantics for reliable artifact management. It supports both OCI
// artifact manifests and image manifests (with automatic fallback for compatibility).
//
// # Basic Usage
//
//	// Create a client with default settings
//	cli, err := ociv2.New(ociv2.ClientOptions{
//	    Timeout: 2 * time.Minute,
//	})
//	if err != nil {
//	    return err
//	}
//
//	// Push a Release Bundle
//	desc, err := cli.PushReleaseBundle(ctx, "oci://registry.example.com/releases/myapp:v1.0.0", 
//	    releaseJSON, ociv2.NewAnnotations().
//	        WithForgeProject("myapp").
//	        WithForgeEnv("production"))
//	// desc.Ref contains the canonical reference with digest
//
//	// Pull a Release Bundle using the digest reference
//	data, desc, err := cli.PullReleaseBundle(ctx, desc.Ref)
//
// # Advanced Configuration
//
//	cli, err := ociv2.New(ociv2.ClientOptions{
//	    Timeout:                2 * time.Minute,
//	    PlainHTTP:             false,  // Use HTTPS (default)
//	    PreferArtifactManifest: true,   // Try OCI artifacts first (default)
//	    FallbackImageManifest:  true,   // Fallback to image manifest if needed
//	    Cosign: ociv2.CosignOpts{
//	        Enable:        true,
//	        AllowInsecure: false,  // Require proper signing
//	        Identity: &ociv2.OIDCIdentity{
//	            Issuer:  "https://token.actions.githubusercontent.com",
//	            Subject: "repo:myorg/myrepo:ref:refs/heads/main",
//	        },
//	    },
//	    StructuredLogger: myLogger,  // Custom structured logger
//	    EnableMetrics:    true,       // Enable metrics collection
//	})
//
// # Signing and Verification
//
//	// Sign an artifact (requires COSIGN_EXPERIMENTAL=1 or key setup)
//	signDesc, err := cli.SignDigest(ctx, desc.Ref)
//	if err != nil {
//	    log.Printf("Failed to sign: %v", err)
//	}
//
//	// Verify signatures with identity constraints
//	report, err := cli.VerifyDigest(ctx, desc.Ref)
//	if err != nil {
//	    return fmt.Errorf("verification failed: %w", err)
//	}
//	
//	if report.Signed {
//	    log.Printf("Artifact signed by: %s", report.SignerIdentity.Subject)
//	}
//
// # Working with Rendered Sets
//
//	// Push a Rendered Set (index + tar layer)
//	desc, err := cli.PushRenderedSet(ctx, 
//	    "oci://registry.example.com/rendered/myapp:v1.0.0",
//	    indexJSON,      // JSON index of rendered components
//	    tarReader,      // io.Reader for tar.gz content
//	    tarSize,        // Size in bytes
//	    ociv2.NewAnnotations().
//	        WithForgeKind("rendered").
//	        WithForgeProject("myapp"))
//
//	// Pull a Rendered Set
//	index, tarReader, desc, err := cli.PullRenderedSet(ctx, ref)
//	defer tarReader.Close()
//	
//	// Process tar content
//	data, err := io.ReadAll(tarReader)
//
// # Authentication
//
// By default, the client uses Docker config (~/.docker/config.json) for authentication.
// You can provide custom authentication:
//
//	// Static credentials
//	auth := &ociv2.StaticAuth{
//	    Username: "user",
//	    Password: "token",
//	}
//	
//	// GitHub token authentication
//	auth := &ociv2.GitHubAuth{
//	    Token: os.Getenv("GITHUB_TOKEN"),
//	}
//	
//	// Chain multiple auth providers
//	auth := &ociv2.ChainAuth{
//	    Providers: []ociv2.AuthProvider{
//	        &ociv2.GitHubAuth{Token: ghToken},
//	        &ociv2.ECRAuth{},  // AWS ECR using SDK
//	        &ociv2.DefaultAuth{}, // Docker config fallback
//	    },
//	}
//	
//	cli, err := ociv2.New(ociv2.ClientOptions{
//	    Auth: auth,
//	})
//
// # Registry-Specific Behavior
//
// The package automatically handles registry-specific quirks:
//
//   - ECR: Detects ECR registries and handles authentication via AWS SDK
//   - GHCR: Handles GitHub Container Registry namespaces and permissions  
//   - Docker Hub: Proper namespace handling for official images
//   - Fallback: Automatically falls back to image manifests for registries
//     that don't support OCI artifact manifests
//
// # Annotations
//
// Use the Annotations builder for standard OCI and Forge-specific annotations:
//
//	ann := ociv2.NewAnnotations().
//	    WithTitle("My Application").
//	    WithDescription("Production release").
//	    WithVersion("1.0.0").
//	    WithForgeKind("release").
//	    WithForgeProject("myapp").
//	    WithForgeEnv("production").
//	    WithSource("https://github.com/myorg/myrepo", "abc123").
//	    WithBuildInfo("build-123", "456", "https://ci.example.com/build/123").
//	    WithGitInfo("abc123def", "main", "v1.0.0", false)
//
// # Error Handling
//
// The package provides structured errors with categories and context:
//
//	desc, err := cli.PushJSON(ctx, ref, mediaType, data, nil)
//	if err != nil {
//	    var ociErr *ociv2.OCIError
//	    if errors.As(err, &ociErr) {
//	        log.Printf("Category: %s, Code: %s", ociErr.Category, ociErr.Code)
//	        if ociErr.IsRetryable() {
//	            // Retry logic
//	        }
//	    }
//	    
//	    // Check for specific conditions
//	    if errors.Is(err, ociv2.ErrNotFound) {
//	        // Handle not found
//	    } else if errors.Is(err, ociv2.ErrUnauthorized) {
//	        // Handle auth failure
//	    }
//	}
//
// # Observability
//
// The package supports structured logging and metrics:
//
//	// Custom logger implementation
//	type MyLogger struct{}
//	
//	func (l *MyLogger) Debug(msg string, fields map[string]interface{}) {
//	    // Log with your preferred logger
//	}
//	
//	// Metrics callback
//	cli, err := ociv2.New(ociv2.ClientOptions{
//	    StructuredLogger: &MyLogger{},
//	    EnableMetrics: true,
//	    MetricsCallback: func(m *ociv2.Metrics) {
//	        // Export metrics to your monitoring system
//	        prometheus.RecordMetrics(m)
//	    },
//	})
//
// # Best Practices
//
//   - Always use digest references (@ notation) for immutable references
//   - Enable signing in production environments
//   - Set appropriate timeouts for large artifact transfers
//   - Use structured logging for better observability
//   - Handle fallback scenarios for maximum compatibility
//   - Validate media types when pulling artifacts
package ociv2