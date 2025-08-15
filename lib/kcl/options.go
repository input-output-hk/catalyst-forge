package kcl

// PublishOptions configures module publishing.
type PublishOptions struct {
	Profile     Profile           // Artifact profile (compat or strict)
	Ref         string            // Repository reference (repo + optional tag)
	ModuleRoot  string            // Directory containing kcl.mod
	Tag         string            // Optional semver tag
	Annotations map[string]string // OCI annotations
	Sign        bool              // Enable signing
	SignKeyRef  string            // Signing key reference (e.g., "awskms://alias/forge-ci")
	Attest      bool              // Enable attestation
	AttestBytes []byte            // Optional DSSE predicate JSON
}

// VerifyOptions configures module verification.
type VerifyOptions struct {
	Profile          Profile // Artifact profile
	RequireSignature bool    // Require valid signature
}

// PullOptions configures module pulling.
type PullOptions struct {
	Profile          Profile           // Artifact profile
	RequireSignature bool              // Require valid signature
	SkipVerify       bool              // Skip verification (dev mode)
	Progress         func(string)      // Progress callback
}

// RunOptions configures module execution.
type RunOptions struct {
	Profile          Profile           // Artifact profile
	Engine           EngineKind        // Execution engine (native or wasm)
	Values           []byte            // Input values (will be canonicalized)
	Context          []byte            // Context values (will be canonicalized)
	TimeoutSec       int               // Execution timeout in seconds
	MemoryLimitMB    int               // Memory limit in MB (WASM only)
	UseCache         bool              // Use run cache (default true)
	ForceRecompute   bool              // Bypass run cache but still populate it
	RequireSignature bool              // Require valid signature for module
	SkipVerify       bool              // Skip verification (dev mode)
	Progress         func(string)      // Progress callback
}

// InspectOptions configures module inspection.
type InspectOptions struct {
	Profile          Profile // Artifact profile
	RequireSignature bool    // Require valid signature for inspection
}

// CacheOptions configures cache behavior.
type CacheOptions struct {
	Dir              string // Cache directory (default: ~/.forge/kcl)
	ModulesMaxBytes  int64  // Max size for module cache in bytes
	RunsMaxBytes     int64  // Max size for run cache in bytes
	TTLDays          int    // TTL for cache entries in days
	EnableBlobCache  bool   // Store raw tar blobs
	DisableLocking   bool   // Disable cross-process locking (testing only)
}

// DefaultCacheOptions returns default cache configuration.
func DefaultCacheOptions() CacheOptions {
	return CacheOptions{
		Dir:             "", // Will be set to ~/.forge/kcl if empty
		ModulesMaxBytes: 10 * 1024 * 1024 * 1024, // 10 GiB
		RunsMaxBytes:    5 * 1024 * 1024 * 1024,  // 5 GiB
		TTLDays:         30,
		EnableBlobCache: false,
		DisableLocking:  false,
	}
}