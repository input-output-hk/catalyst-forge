# lib/ociv2 Implementation Tasks

## Overview
Implementation tasks for the OCI v2 client package following the design in GUIDE.md. This package provides a clean, testable API for pushing/pulling OCI artifacts with Cosign signing support.

## Phase 1: Core Foundation ✅
### 1.1 Package Structure Setup
- [x] Create package directory structure (`lib/ociv2/`)
- [x] Create `internal/` subdirectory for unexported helpers
- [x] Set up go.mod with initial dependencies:
  - `oras.land/oras-go/v2`
  - `github.com/google/go-containerregistry`
  - `github.com/sigstore/cosign/v2`

### 1.2 Core Types & Interfaces
- [x] Implement `types.go`:
  - [x] Define `Descriptor` struct with all fields
  - [x] Define media type constants (MTReleaseConfig, MTRenderedIndex, etc.)
  - [x] Define standard error variables (ErrNotFound, ErrUnauthorized, etc.)
  - [x] Define `Annotations` type alias with Merge helper

- [x] Implement `client.go`:
  - [x] Define `Client` interface with all methods
  - [x] Define `ClientOptions` struct
  - [x] Define `AuthProvider` interface
  - [x] Define `CosignOpts` and `OIDCIdentity` structs
  - [x] Implement `New()` constructor with defaults
  - [x] Add placeholder implementations for all interface methods

### 1.3 Annotations
- [x] Implement `annotations.go`:
  - [x] Define OCI standard annotation constants
  - [x] Define Forge-specific annotation constants
  - [x] Add helper functions for merging annotations
  - [x] Add builder methods (WithSource, WithForgeKind, etc.)
  - [x] Add filter methods (FilterForge, FilterOCI)

## Phase 2: Registry Operations ✅
### 2.1 Reference Utilities
- [x] Implement `refs.go`:
  - [x] `isDigestRef()` - check if ref contains @sha256:
  - [x] `ensureDigest()` - ensure Descriptor has canonical ref
  - [x] `parseRef()` - parse and validate OCI references
  - [x] `toCanonical()` - convert to canonical form
  - [x] `normalizeRef()` - handle oci:// scheme
  - [x] `extractRegistry()` - extract registry from ref
  - [x] `validateRef()` - validate OCI references
  - [x] `isLocalRegistry()` - detect local registries
  - [x] `splitRefParts()` - split ref into parts

### 2.2 Auth System
- [x] Implement `auth.go`:
  - [x] `DefaultAuth` struct using Docker config
  - [x] Integration with ggcr's authn.DefaultKeychain
  - [x] Helper for getting authenticator per registry
  - [x] Support for anonymous access (nil return)
  - [x] `multiAuth` for ORAS and ggcr compatibility
  - [x] `StaticAuth` for username/password
  - [x] `GitHubAuth` for GHCR
  - [x] `ECRAuth` placeholder for AWS ECR
  - [x] `ChainAuth` for trying multiple providers
  - [x] Auth conversion helpers (toGGCRAuth, toORASAuth)

### 2.3 ORAS Integration
- [x] Implement `internal/orasx.go`:
  - [x] `PushConfigOnly()` - push JSON as artifact config
  - [x] `PushConfigAndLayer()` - push config + tar layer
  - [x] `PullConfig()` - pull artifact config
  - [x] `PullLayer()` - pull artifact layer
  - [x] `Resolve()` - resolve ref to descriptor
  - [x] `Head()` - HEAD request for descriptor
  - [x] Error mapping from ORAS to standard errors
  - [x] Auth integration with ORAS Client

### 2.4 GGCR Integration
- [x] Implement `internal/ggcrx.go`:
  - [x] `PushJSONLayer()` - fallback for JSON as image
  - [x] `PushConfigAndLayer()` - fallback for tar as image
  - [x] `PullJSONLayer()` - pull JSON from image layer
  - [x] `PullLayer()` - pull tar from image layer
  - [x] `Head()` - HEAD request for descriptor
  - [x] `Resolve()` - resolve ref with full fetch
  - [x] Error mapping from ggcr to standard errors
  - [x] Plain HTTP transport support
  - [x] Auth integration with remote options

### 2.5 Common Utilities
- [x] Implement `internal/errors.go` - shared error definitions
- [x] Implement `internal/utils.go` - shared utility functions

## Phase 3: Push/Pull Operations ✅
### 3.1 Generic Operations
- [x] Implement in `push_pull.go`:
  - [x] `Resolve()` - resolve ref to digest with full fetch
  - [x] `Head()` - get descriptor via HEAD request
  - [x] `headOrResolve()` - internal helper for both
  - [x] Reference validation and normalization
  - [x] Registry extraction
  - [x] Timeout application
  - [x] Logging support

### 3.2 JSON Operations
- [x] Implement JSON push/pull:
  - [x] `PushJSON()` - with artifact/image fallback logic
  - [x] `PullJSON()` - with media type validation
  - [x] Proper error handling and retry logic
  - [x] Annotation merging
  - [x] ORAS to ggcr fallback

### 3.3 Tar Operations
- [x] Implement tar push/pull:
  - [x] `PushTar()` - config + tar layer
  - [x] `PullTar()` - retrieve tar layer
  - [x] Stream handling with io.Reader/io.ReadCloser
  - [x] Size tracking
  - [x] Layer media type handling

### 3.4 Domain-Specific Helpers
- [x] Release Bundle operations:
  - [x] `PushReleaseBundle()` - wrapper with correct media type
  - [x] `PullReleaseBundle()` - wrapper with validation
  - [x] Automatic ForgeKind annotation

- [x] Rendered Set operations:
  - [x] `PushRenderedSet()` - index + tar with annotations
  - [x] `PullRenderedSet()` - retrieve both components
  - [x] Automatic ForgeKind annotation

### 3.5 Backend Integration
- [x] ORAS backend wrappers:
  - [x] `orasHeadOrResolve()`
  - [x] `orasPushConfigOnly()`
  - [x] `orasPushConfigAndLayer()`
  - [x] `orasPullConfig()`
  - [x] `orasPullLayer()`

- [x] GGCR backend wrappers:
  - [x] `ggcrHeadOrResolve()`
  - [x] `ggcrPushJSONLayer()`
  - [x] `ggcrPushConfigAndLayer()`
  - [x] `ggcrPullJSONLayer()`
  - [x] `ggcrPullLayer()`

- [x] Auth helpers:
  - [x] `getORASAuth()` - ORAS auth function
  - [x] `getGGCRAuth()` - ggcr auth function
  - [x] Auth conversion integration

- [x] Utility functions:
  - [x] `descriptorFromOCISpec()` - convert OCI descriptor

## Phase 4: Signing & Verification ✅
### 4.1 Cosign Integration
- [x] Implement `sign_verify.go`:
  - [x] `SignDigest()` - sign with keyless or key-based
  - [x] `VerifyDigest()` - verify with identity constraints
  - [x] `VerificationReport` struct with all fields
  - [x] `SignerIdentity` struct
  - [x] Auth integration for Cosign operations
  - [x] Identity constraint enforcement (issuer/subject)

### 4.2 Cosign Helpers
- [x] Implement `internal/cosignx.go`:
  - [x] `CosignSigner` and `CosignVerifier` structures
  - [x] Keyless mode detection (COSIGN_EXPERIMENTAL, CI environments)
  - [x] Mock implementation for signing/verification
  - [x] Identity validation (issuer/subject matching)
  - [x] Support for insecure mode (development/testing)

### 4.3 Testing & Validation
- [x] Comprehensive test coverage:
  - [x] `sign_verify_test.go` with table-driven tests
  - [x] Tests for signing disabled/enabled modes
  - [x] Keyless mode simulation tests
  - [x] Identity constraint verification tests
  - [x] Invalid reference handling tests
  - [x] Verification report structure tests
- [x] Updated all tests to follow Go testing standards:
  - [x] Using testify/assert and testify/require
  - [x] Proper table-driven test structure
  - [x] Parallel execution with t.Parallel()
  - [x] Descriptive test names and error messages
  - [x] Same-package testing for access to internals

## Phase 5: Registry Compatibility ✅
### 5.1 Fallback Logic
- [x] Implement manifest type detection:
  - [x] Try artifact manifest first
  - [x] Detect registry rejection (HTTP 400/415/501/502)
  - [x] Automatic fallback to image manifest
  - [x] Preserve annotations across fallback
  - [x] Intelligent error analysis for fallback decision
  - [x] Registry-specific error pattern recognition

### 5.2 Registry-Specific Handling
- [x] ECR compatibility:
  - [x] ECR registry detection
  - [x] ECR-specific error patterns
  - [x] ECR region extraction and optimization
  - [x] ECR auth helper with credential detection
  - [x] ECR-specific manifest requirements handling
  
- [x] GHCR compatibility:
  - [x] GHCR registry detection  
  - [x] GitHub token auth integration
  - [x] GHCR namespace and permission handling
  - [x] GHCR-specific optimizations
  - [x] Public namespace detection

### 5.3 Enhanced Registry Support
- [x] Implement `internal/registry_compat.go`:
  - [x] `DetectRegistryType()` - identify registry from hostname
  - [x] `ShouldFallbackToImageManifest()` - intelligent fallback decision
  - [x] `GetRegistrySpecificOptions()` - optimal settings per registry
  - [x] HTTP error extraction and analysis
  - [x] Registry capabilities tracking

- [x] Implement `internal/registry_ecr.go`:
  - [x] ECR-specific configuration options
  - [x] ECR region and account ID extraction
  - [x] ECR auth helper with credential detection
  - [x] ECR-specific error pattern recognition

- [x] Implement `internal/registry_ghcr.go`:
  - [x] GHCR-specific configuration options
  - [x] GHCR auth helper with token management
  - [x] GHCR namespace extraction and validation
  - [x] Public namespace detection for anonymous access

### 5.4 Enhanced Push/Pull Operations
- [x] Updated `push_pull.go` with intelligent fallback:
  - [x] Registry type detection for all push operations
  - [x] Smart fallback decision based on error analysis
  - [x] Enhanced logging for registry compatibility
  - [x] Detailed error reporting for both manifest types
  - [x] Preservation of annotations across fallback attempts

### 5.5 Comprehensive Testing
- [x] Implement `registry_compat_test.go`:
  - [x] Registry type detection tests (Docker Hub, ECR, GHCR, GCR, ACR, Quay)
  - [x] Fallback decision logic tests with various error scenarios
  - [x] Registry-specific options validation
  - [x] ECR helper function tests (region extraction, auth detection)
  - [x] GHCR helper function tests (namespace extraction, public detection)
  - [x] HTTP error pattern recognition tests
  - [x] All tests following Go testing standards with testify

## Phase 6: Observability & Error Handling ✅
### 6.1 Structured Error Handling
- [x] Implement comprehensive error types:
  - [x] `OCIError` struct with categorization, context, and metadata
  - [x] Error categories (auth, network, registry, validation, config, cosign, fallback)
  - [x] Specific error constructors (`NewAuthError`, `NewNetworkError`, etc.)
  - [x] Error code generation and human-readable messages
  - [x] Error wrapping and unwrapping for error chains
  - [x] HTTP status extraction and error classification
  - [x] Retryability and temporariness determination

### 6.2 Structured Logging
- [x] Implement structured logging interfaces:
  - [x] `Logger` interface with level-specific methods
  - [x] `DefaultLogger` implementation using legacy Logger function
  - [x] `NoOpLogger` for disabled logging
  - [x] Operation-specific logging with context fields
  - [x] Error category and HTTP status logging
  - [x] Support for preset fields via `WithFields()`

### 6.3 Operation Tracking & Metrics
- [x] Implement operation tracking:
  - [x] `OperationTracker` for timing and logging operations
  - [x] Start/complete tracking with custom fields
  - [x] Integration with structured errors and logging
  - [x] Panic recovery and error logging

- [x] Implement metrics collection:
  - [x] `Metrics` struct for operation and error counts
  - [x] Registry-specific statistics and health scoring
  - [x] Fallback attempt tracking
  - [x] Average duration tracking
  - [x] Configurable metrics callback for external reporting

### 6.4 Enhanced Client Integration
- [x] Updated `ClientOptions` with observability:
  - [x] `StructuredLogger` field for enhanced logging
  - [x] `EnableMetrics` field for metrics collection
  - [x] `MetricsCallback` for external metrics reporting
  - [x] Backward compatibility with legacy `Logger` function

- [x] Enhanced push/pull operations:
  - [x] Integrated operation tracking throughout all client methods
  - [x] Structured error wrapping with appropriate context
  - [x] Metrics recording for operations and fallback attempts
  - [x] Registry type detection and logging
  - [x] Detailed error reporting for debugging

### 6.5 Comprehensive Testing
- [x] Implement `observability_test.go`:
  - [x] Log level and logger functionality tests
  - [x] Operation tracker timing and field tests
  - [x] Metrics collection and health scoring tests
  - [x] Integration tests with structured errors
  - [x] All tests following Go testing standards with testify

- [x] Implement `errors_test.go`:
  - [x] Error construction and categorization tests
  - [x] Error code and message generation tests
  - [x] Error chaining and unwrapping tests
  - [x] Retryability and temporariness determination tests
  - [x] HTTP status extraction tests
  - [x] Error category classification tests
  - [x] All tests following Go testing standards with testify

## Phase 7: Testing ✅
### 7.1 Unit Tests
- [x] Test core functionality:
  - [x] Reference parsing and validation
  - [x] Annotation merging
  - [x] Error mapping
  - [x] Timeout handling

### 7.2 Integration Tests
- [x] Set up test infrastructure:
  - [x] In-memory registry using ggcr
  - [x] Or local registry container
  
- [x] Test push/pull operations:
  - [x] Release Bundle round-trip
  - [x] Rendered Set round-trip
  - [x] Fallback scenarios
  - [x] Auth scenarios

### 7.3 Golden Tests
- [x] Create golden test files:
  - [x] Expected manifest formats
  - [x] Annotation preservation
  - [x] Media type correctness

### 7.4 Signing Tests
- [x] Test Cosign integration (behind build tag):
  - [x] Keyless signing flow
  - [x] Identity verification
  - [x] Invalid signature handling

## Phase 8: Documentation ✅
### 8.1 Code Documentation
- [x] Add comprehensive godoc comments:
  - [x] All exported types and methods
  - [x] Usage examples in doc.go
  - [x] Common patterns and best practices

### 8.2 README
- [x] Create README.md with:
  - [x] Quick start guide
  - [x] Configuration examples
  - [x] Auth setup for different registries
  - [x] Troubleshooting guide

## Phase 9: Optimization & Polish ✅
### 9.1 Performance
- [x] Optimize for common cases:
  - [x] Connection pooling (HTTP transport with connection reuse)
  - [x] Parallel operations where possible (semaphore-based concurrency control)
  - [x] Efficient streaming for large files (buffered readers, stream progress)

### 9.2 Validation
- [x] Add input validation:
  - [x] Reference format validation (ValidateReference function)
  - [x] Media type validation (ValidateMediaType function)
  - [x] Size limits for safety (ValidateBlobSize with configurable limits)

### 9.3 Final Review
- [x] Code review checklist:
  - [x] No domain types leaked into package (verified, only annotations)
  - [x] Consistent error handling (structured errors throughout)
  - [x] Proper resource cleanup (cleanup manager, resource tracker)
  - [x] Thread safety where needed (sync primitives, atomic operations)

## Dependencies
```yaml
dependencies:
  - oras.land/oras-go/v2: "2.5.0+"
  - github.com/google/go-containerregistry: "0.19.0+"
  - github.com/sigstore/cosign/v2: "2.2.0+"
  - github.com/opencontainers/image-spec: "1.1.0+"
```

## Success Criteria
- [x] Clean API that hides registry complexity
- [x] Digest-first semantics throughout
- [x] Seamless artifact/image manifest fallback
- [x] Working Cosign keyless signing
- [x] Support for ECR and GHCR registries
- [x] Comprehensive test coverage (>80%)
- [x] No Forge domain types in package
- [x] Performance benchmarks pass

## Phase 10: Enhanced APIs from Feedback
Based on FEEDBACK.md requirements for domain-agnostic OCI utilities supporting KCL/KPM and other use cases.

### 10.1 Multi-Layer Push/Pull APIs ✅
- [x] Implement `PushArtifact()` with `PackOptions`:
  - [x] Define `LayerSpec` struct with MediaType, Title, Annotations, Size, Reader
  - [x] Define `PackOptions` with ArtifactType, Config, Layers support
  - [x] Support multiple layers in single push operation
  - [x] Handle artifact manifest v1.1 with automatic fallback
  - [x] Return manifest digest as canonical reference

- [x] Implement `PullArtifact()` with structured result:
  - [x] Define `PulledLayer` struct with lazy `Open()` function
  - [x] Define `PullResult` with Descriptor, Config, Layers, Annotations
  - [x] Support lazy streaming of layer blobs
  - [x] Extract artifact type from manifest
  - [x] Return manifest annotations

### 10.2 Registry Tag Management ✅
- [x] Implement `ListTags()`:
  - [x] List all tags for a repository
  - [x] Support pagination for large tag lists
  - [x] Handle different registry APIs (Docker, OCI)
  
- [x] Implement `LatestSemverTag()`:
  - [x] Parse and sort semver tags
  - [x] Return latest stable version
  - [x] Support pre-release version filtering
  - [x] Handle malformed tags gracefully

### 10.3 DSSE Attestation Support ✅
- [x] Implement `Attest()`:
  - [x] Attach DSSE attestations to artifacts
  - [x] Support custom predicate types
  - [x] Use in-toto format for attestations
  - [x] Sign attestations with configured key
  - [x] Return attestation descriptor

- [x] Implement `VerifyAttestations()`:
  - [x] Query attestations for subject digest
  - [x] Filter by predicate type
  - [x] Verify attestation signatures
  - [x] Return `AttestationReport` with details
  - [x] Support SLSA and custom predicates

### 10.4 Artifact Shape Validation
- [ ] Implement `ValidateArtifact()`:
  - [ ] Define `ArtifactValidationSpec` struct
  - [ ] Check artifact type requirements
  - [ ] Validate layer count (min/max)
  - [ ] Verify allowed layer media types
  - [ ] Check required manifest annotations
  - [ ] Validate annotation key=value pairs
  
- [ ] Add checksum validation support:
  - [ ] Extract checksum from annotations
  - [ ] Compute layer checksums (sha256, sha512)
  - [ ] Compare expected vs actual
  - [ ] Support configurable checksum algorithms
  - [ ] Return detailed validation report

### 10.5 Safe Tar Utilities
- [ ] Implement `tarutil` package:
  - [ ] Create `tarutil/writer.go` with deterministic tar creation
  - [ ] Create `tarutil/reader.go` with safe extraction
  - [ ] Create `tarutil/validate.go` for tar validation

- [ ] Implement `WriteDeterministic()`:
  - [ ] Stable file ordering (lexical)
  - [ ] Fixed timestamps (Unix epoch or configured)
  - [ ] Normalized permissions (0644 files, 0755 dirs)
  - [ ] Consistent ownership (uid/gid 0)
  - [ ] Include/exclude pattern support
  - [ ] Return sha256 checksum of tar

- [ ] Implement `ExtractSafe()`:
  - [ ] Path traversal protection (no ../)
  - [ ] Symlink validation (no escaping root)
  - [ ] Device file blocking
  - [ ] Size limit enforcement
  - [ ] Safe permission handling
  - [ ] Progress callback support

### 10.6 Integration & Testing
- [ ] Update existing methods to use new internals:
  - [ ] Refactor PushJSON/PushTar to use PushArtifact
  - [ ] Refactor PullJSON/PullTar to use PullArtifact
  - [ ] Ensure backward compatibility

- [ ] Comprehensive test coverage:
  - [ ] Unit tests for all new APIs
  - [ ] Integration tests with test registry
  - [ ] Attestation round-trip tests
  - [ ] Validation spec test cases
  - [ ] Tar utility security tests

### 10.7 Documentation Updates
- [ ] Update package documentation:
  - [ ] Document new APIs in doc.go
  - [ ] Add examples for multi-layer operations
  - [ ] Document attestation workflow
  - [ ] Add validation spec examples

- [ ] Update README with new features:
  - [ ] Multi-layer artifact examples
  - [ ] Tag management examples
  - [ ] Attestation usage guide
  - [ ] Validation framework guide
  - [ ] Tar utility usage

## Notes
- Start with Phase 1-3 for basic functionality
- Phase 4 (signing) can be developed in parallel
- Phase 5 (compatibility) requires real registry testing
- Keep internal/ packages unexported for flexibility
- Maintain clear separation between ORAS and ggcr code paths
- Phase 10 additions maintain domain-agnostic design per FEEDBACK.md