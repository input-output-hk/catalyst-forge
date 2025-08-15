package kcl

import (
	"errors"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Profile defines artifact shape rules.
type Profile string

const (
	// ProfileCompat mirrors KPM: 1 tar layer, artifactType=tar
	ProfileCompat Profile = "compat"
	// ProfileStrict is Forge specific: tar + meta.json, custom artifactType
	ProfileStrict Profile = "strict"
)

// EngineKind defines execution backend.
type EngineKind string

const (
	// EngineNative uses CGO kcl-go
	EngineNative EngineKind = "native"
	// EngineWASM uses WASM/WASI runtime
	EngineWASM EngineKind = "wasm"
)

// ModuleRef identifies an OCI module.
type ModuleRef struct {
	Repo string // "oci://ghcr.io/org/module"
	Tag  string // optional; use digest for enforcement
	Dig  string // "sha256:..."; preferred if set
}

// PullResult contains the result of pulling a module.
type PullResult struct {
	Path     string      // Local path to extracted module
	Digest   string      // Module digest
	Meta     *ModuleMeta // Module metadata
	MetaJSON []byte      // Module metadata as JSON
}

// RunResult contains the output of KCL execution.
type RunResult struct {
	YAML     []byte      // KCL output in YAML format
	Digest   string      // Module digest used
	MetaJSON []byte      // Module metadata (strict) or derived
	Stats    EngineStats // Execution statistics
	CacheHit bool        // True if returned from run cache
}

// EngineStats contains execution performance metrics.
type EngineStats struct {
	ColdStart bool      // True if engine was just initialized
	CompileMS int64     // Compilation time in milliseconds
	EvalMS    int64     // Evaluation time in milliseconds
	PeakMemMB int       // Peak memory usage in MB
	TotalMS   int64     // Total execution time in milliseconds
}

// ModuleMeta contains module metadata.
type ModuleMeta struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description,omitempty"`
	Sum         string            `json:"sum,omitempty"`
	Entry       string            `json:"entry,omitempty"`
	Authors     []string          `json:"authors,omitempty"`
	License     string            `json:"license,omitempty"`
	Repository  string            `json:"repository,omitempty"`
	Homepage    string            `json:"homepage,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	CreatedAt   time.Time         `json:"createdAt,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// CacheMeta contains cache entry metadata.
type CacheMeta struct {
	Digest        string      `json:"digest"`
	Profile       Profile     `json:"profile"`
	Engine        EngineKind  `json:"engine,omitempty"`
	EngineVersion string      `json:"engineVersion,omitempty"`
	KCLVersion    string      `json:"kclVersion,omitempty"`
	CreatedAt     time.Time   `json:"createdAt"`
	LastAccessAt  time.Time   `json:"lastAccessAt"`
	Stats         EngineStats `json:"stats,omitempty"`
}

// Common errors
var (
	ErrSignatureMissing  = errors.New("signature missing")
	ErrSignatureInvalid  = errors.New("signature invalid")
	ErrShapeMismatch     = errors.New("artifact shape mismatch")
	ErrSchemaViolation   = errors.New("metadata schema violation")
	ErrEntryNotFound     = errors.New("entry file not found")
	ErrEngineUnavailable = errors.New("execution engine unavailable")
	ErrCacheCorrupt      = errors.New("cache content corrupt")
	ErrDigestMismatch    = errors.New("digest mismatch")
	ErrProfileUnknown    = errors.New("unknown profile")
)

// OCI client interface (implemented by lib/ociv2 client)
type OCI interface {
	// PushArtifact pushes an artifact to the registry
	PushArtifact(ref string, layers []ocispec.Descriptor, manifest ocispec.Manifest, artifactType string, annotations map[string]string) (string, error)
	
	// PullArtifact pulls an artifact from the registry
	PullArtifact(ref string) (*OCIPullResult, error)
	
	// SignArtifact signs an artifact
	SignArtifact(digest string, keyRef string) error
	
	// AttestArtifact creates an attestation for an artifact
	AttestArtifact(subjectRef string, predicateType string, predicate []byte) (string, error)
	
	// VerifyArtifact verifies an artifact's signature and shape
	VerifyArtifact(ref string, requireSignature bool) (*VerificationReport, error)
}

// OCIPullResult contains the result of pulling an OCI artifact
type OCIPullResult struct {
	Manifest    ocispec.Manifest
	Layers      map[string][]byte // digest -> content
	Digest      string
	Annotations map[string]string
}

// VerificationReport contains verification results
type VerificationReport struct {
	Digest           string
	SignatureValid   bool
	SignatureDetails map[string]interface{}
	ShapeValid       bool
	SchemaValid      bool
	Errors           []string
	Details          map[string]interface{} // Additional verification details
}

// Limits defines resource limits for KCL execution
type Limits struct {
	TimeoutSec    int // Execution timeout in seconds
	MemoryLimitMB int // Memory limit in MB (used by WASM)
}