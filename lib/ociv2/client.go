package ociv2

import (
	"context"
	"io"
	"net/http"
	"time"
)

// AuthProvider provides authentication for registry operations
type AuthProvider interface {
	// Authenticator returns an authenticator for the given registry.
	// Returns nil for anonymous access.
	// The returned value should be compatible with both ORAS and ggcr.
	Authenticator(registry string) (any, error)
}

// CosignOpts configures Cosign signing and verification
type CosignOpts struct {
	Enable        bool          // Enable Cosign operations
	RekorURL      string        // Rekor transparency log URL (optional, uses default if empty)
	FulcioURL     string        // Fulcio CA URL (optional, uses default if empty)
	Identity      *OIDCIdentity // Optional: enforce issuer/subject on verify
	AllowInsecure bool          // Allow insecure operations (local dev only)
}

// ClientOptions configures the OCI client
type ClientOptions struct {
	// Networking
	PlainHTTP      bool            // Use HTTP instead of HTTPS (local registries only)
	Timeout        time.Duration   // Request timeout (default: 2 minutes)
	UserAgent      string          // User-Agent header for requests
	HTTPTransport  *http.Transport // Custom HTTP transport for connection pooling
	MaxConcurrency int             // Max concurrent operations (default: 10)

	// Auth & signing
	Auth   AuthProvider // Authentication provider (nil for anonymous)
	Cosign CosignOpts   // Cosign signing/verification options

	// Registry compatibility
	PreferArtifactManifest bool // Try OCI artifact manifest first
	FallbackImageManifest  bool // Fallback to image manifest if registry rejects artifacts

	// Observability
	Logger           func(msg string, kv ...any) // Optional structured logger (legacy)
	StructuredLogger Logger                      // Enhanced structured logger
	EnableMetrics    bool                        // Enable operation metrics collection
	MetricsCallback  func(*Metrics)              // Callback for metrics reporting

	// Resource limits
	MaxBlobSize      int64 // Maximum blob size allowed (default: 5GB)
	StreamBufferSize int   // Buffer size for streaming operations (default: 32KB)
}

// Client provides operations for OCI artifacts
type Client interface {
	// -------- Generic primitives --------

	// Resolve fetches the manifest and returns a complete descriptor
	Resolve(ctx context.Context, ref string) (Descriptor, error)

	// Head performs a HEAD request to get descriptor metadata
	Head(ctx context.Context, ref string) (Descriptor, error)

	// -------- JSON operations --------

	// PushJSON pushes a JSON blob as an artifact
	PushJSON(ctx context.Context, ref string, mediaType string, payload []byte, ann Annotations) (Descriptor, error)

	// PullJSON pulls a JSON blob artifact
	PullJSON(ctx context.Context, ref string, wantMediaType string) ([]byte, Descriptor, error)

	// -------- TAR operations --------

	// PushTar pushes a tar stream with a JSON config
	PushTar(ctx context.Context, ref string, cfg []byte, cfgMT, layerMT string, tar io.Reader, size int64, ann Annotations) (Descriptor, error)

	// PullTar pulls a tar layer from an artifact
	PullTar(ctx context.Context, ref string, layerMT string) (io.ReadCloser, Descriptor, error)

	// -------- Release Bundle helpers --------

	// PushReleaseBundle pushes a Release Bundle JSON
	PushReleaseBundle(ctx context.Context, ref string, releaseJSON []byte, ann Annotations) (Descriptor, error)

	// PullReleaseBundle pulls a Release Bundle JSON
	PullReleaseBundle(ctx context.Context, ref string) ([]byte, Descriptor, error)

	// -------- Rendered Set helpers --------

	// PushRenderedSet pushes a Rendered Set (index + tar)
	PushRenderedSet(ctx context.Context, ref string, indexJSON []byte, tar io.Reader, size int64, ann Annotations) (Descriptor, error)

	// PullRenderedSet pulls a Rendered Set (returns tar stream, index JSON, and descriptor)
	PullRenderedSet(ctx context.Context, ref string) (io.ReadCloser, []byte, Descriptor, error)

	// -------- Multi-layer operations --------

	// PushArtifact pushes a multi-layer artifact with the specified options
	PushArtifact(ctx context.Context, ref string, opts PackOptions) (Descriptor, error)

	// PullArtifact pulls a complete artifact with all layers
	PullArtifact(ctx context.Context, ref string) (*PullResult, error)

	// -------- Tag management --------

	// ListTags lists all tags for a repository
	ListTags(ctx context.Context, repo string) ([]string, error)

	// LatestSemverTag returns the latest semantic version tag
	LatestSemverTag(ctx context.Context, repo string, includePrerelease bool) (string, error)

	// -------- Signing & verification --------

	// SignDigest signs an artifact digest using Cosign
	SignDigest(ctx context.Context, refOrDigest string) (Descriptor, error)

	// VerifyDigest verifies signatures on an artifact
	VerifyDigest(ctx context.Context, refOrDigest string) (*VerificationReport, error)

	// VerifyArtifact pulls and verifies an artifact with configurable validation
	VerifyArtifact(ctx context.Context, ref string, opts VerifyOptions) (*PullResult, *ValidationReport, error)

	// -------- Attestations --------

	// Attest attaches a DSSE attestation to an artifact
	Attest(ctx context.Context, refOrDigest string, opts AttestationOptions) (Descriptor, error)

	// VerifyAttestations queries and verifies attestations for a subject
	VerifyAttestations(ctx context.Context, refOrDigest string, predicateType string) (*AttestationReport, error)
}

// client implements the Client interface
type client struct {
	opts      ClientOptions
	auth      AuthProvider
	transport *http.Transport
	semaphore chan struct{} // For concurrency limiting
}

// New creates a new OCI client with the given options
func New(opts ClientOptions) (Client, error) {
	// Set defaults
	if opts.Timeout == 0 {
		opts.Timeout = 2 * time.Minute
	}

	if opts.UserAgent == "" {
		opts.UserAgent = "forge-oci/1.0"
	}

	// Default to preferring artifact manifests with image fallback
	if !opts.PreferArtifactManifest && !opts.FallbackImageManifest {
		opts.PreferArtifactManifest = true
		opts.FallbackImageManifest = true
	}

	// Set resource limits
	if opts.MaxBlobSize == 0 {
		opts.MaxBlobSize = 5 * 1024 * 1024 * 1024 // 5GB default
	}

	if opts.StreamBufferSize == 0 {
		opts.StreamBufferSize = 32 * 1024 // 32KB default
	}

	if opts.MaxConcurrency == 0 {
		opts.MaxConcurrency = 10 // Default max concurrent operations
	}

	// Setup HTTP transport with connection pooling
	transport := opts.HTTPTransport
	if transport == nil {
		transport = &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			MaxConnsPerHost:     10,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
			ForceAttemptHTTP2:   true,
		}
	}

	c := &client{
		opts:      opts,
		auth:      opts.Auth,
		transport: transport,
		semaphore: make(chan struct{}, opts.MaxConcurrency),
	}

	// Use default auth if none provided
	if c.auth == nil {
		c.auth = &DefaultAuth{}
	}

	return c, nil
}
