package ociv2

import (
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/attestation"
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/auth"
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/observability"
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/signing"
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/utils"
)

// Type aliases for backward compatibility

// Auth types (AuthProvider is already in the root package)
type (
	DefaultAuth  = auth.DefaultAuth
	StaticAuth   = auth.StaticAuth
	GitHubAuth   = auth.GitHubAuth
	ECRAuth      = auth.ECRAuth
	ChainAuth    = auth.ChainAuth
)

// Signing types
type (
	SignerIdentity     = signing.SignerIdentity
	OIDCIdentity       = signing.OIDCIdentity
	VerificationReport = signing.VerificationReport
)

// Attestation types
type (
	AttestationOptions     = attestation.AttestationOptions
	AttestationReport      = attestation.AttestationReport
	AttestationEntry       = attestation.AttestationEntry
	DSSEEnvelope           = attestation.DSSEEnvelope
	IntotoStatement        = attestation.IntotoStatement
	SLSAProvenance         = attestation.SLSAProvenance
	ArtifactValidationSpec = attestation.ArtifactValidationSpec
	Signature              = attestation.Signature
	Subject                = attestation.Subject
	Builder                = attestation.Builder
	Material               = attestation.Material
)

// Artifact types are now in the root package

// Observability types
type (
	Logger           = observability.Logger
	LogLevel         = observability.LogLevel
	Metrics          = observability.Metrics
	OCIError         = observability.OCIError
	ErrorCategory    = observability.ErrorCategory
	OperationTracker = observability.OperationTracker
	NoOpLogger       = observability.NoOpLogger
)

// Utils types are internal - not exported

// Constants re-exported
const (
	// Attestation predicate types
	PredicateSLSAProvenance   = attestation.PredicateSLSAProvenance
	PredicateSLSAProvenanceV1 = attestation.PredicateSLSAProvenanceV1
	PredicateSPDX             = attestation.PredicateSPDX
	PredicateCycloneDX        = attestation.PredicateCycloneDX
	PredicateCustom           = attestation.PredicateCustom
	
	// Media types
	MediaTypeDSSE             = attestation.MediaTypeDSSE
	MediaTypeIntotoStatement  = attestation.MediaTypeIntotoStatement
	
	// Error categories
	ErrorCategoryAuth       = observability.ErrorCategoryAuth
	ErrorCategoryNetwork    = observability.ErrorCategoryNetwork
	ErrorCategoryRegistry   = observability.ErrorCategoryRegistry
	ErrorCategoryValidation = observability.ErrorCategoryValidation
	ErrorCategoryConfig     = observability.ErrorCategoryConfig
	ErrorCategoryCosign     = observability.ErrorCategoryCosign
	ErrorCategoryFallback   = observability.ErrorCategoryFallback
	ErrorCategoryUnknown    = observability.ErrorCategoryUnknown
)

// Function aliases for commonly used functions

// CreateSLSAProvenance creates a SLSA provenance predicate
var CreateSLSAProvenance = attestation.CreateSLSAProvenance

// NewAnnotations creates a new Annotations map with standard values
func NewAnnotations() Annotations {
	return Annotations(utils.NewAnnotations())
}

// Standard annotations
var (
	AnnTitle       = utils.AnnTitle
	AnnDescription = utils.AnnDescription
	AnnVersion     = utils.AnnVersion
	AnnCreated     = utils.AnnCreated
	AnnAuthors     = utils.AnnAuthors
	AnnURL         = utils.AnnURL
	AnnSourceRepo  = utils.AnnSourceRepo
	AnnSourceRev   = utils.AnnSourceRev
	AnnVendor      = utils.AnnVendor
	AnnLicenses    = utils.AnnLicenses
	
	// Forge-specific annotations
	AnnForgeVersion     = utils.AnnForgeVersion
	AnnForgeKind        = utils.AnnForgeKind
	AnnForgeBuildID     = utils.AnnForgeBuildID
	AnnForgeBuildNumber = utils.AnnForgeBuildNumber
	AnnForgeBuildURL    = utils.AnnForgeBuildURL
	AnnForgeBuilder     = utils.AnnForgeBuilder
	AnnForgeProject     = utils.AnnForgeProject
	AnnForgeEnv         = utils.AnnForgeEnv
	AnnForgeTrace       = utils.AnnForgeTrace
	AnnForgeRelease     = utils.AnnForgeRelease
	
	// Deployment annotations
	AnnForgeCluster    = utils.AnnForgeCluster
	AnnForgeNamespace  = utils.AnnForgeNamespace
	AnnForgeDeployedBy = utils.AnnForgeDeployedBy
	AnnForgeDeployedAt = utils.AnnForgeDeployedAt
	
	// Git annotations
	AnnForgeGitCommit = utils.AnnForgeGitCommit
	AnnForgeGitBranch = utils.AnnForgeGitBranch
	AnnForgeGitTag    = utils.AnnForgeGitTag
	AnnForgeGitDirty  = utils.AnnForgeGitDirty
	
	// Signature annotations
	AnnForgeSigned    = utils.AnnForgeSigned
	AnnForgeSignedBy  = utils.AnnForgeSignedBy
	AnnForgeSignedAt  = utils.AnnForgeSignedAt
	AnnForgeSignature = utils.AnnForgeSignature
	
	// Additional OCI annotations
	AnnDocumentation = utils.AnnDocumentation
	AnnBaseDigest    = utils.AnnBaseDigest
	AnnBaseName      = utils.AnnBaseName
)

// Error variables
var (
	ErrInsecureRef = observability.ErrInsecureRef
	ErrInvalidRef  = observability.ErrInvalidRef
)

// Observability functions
var (
	NewDefaultLogger     = observability.NewDefaultLogger
	NewOperationTracker  = observability.NewOperationTracker
	NewMetrics           = observability.NewMetrics
	NewAuthError         = observability.NewAuthError
	NewNetworkError      = observability.NewNetworkError
	NewRegistryError     = observability.NewRegistryError
	GetErrorCategory     = observability.GetErrorCategory
)

// LogLevel constants
const (
	LogLevelDebug = observability.LogLevelDebug
	LogLevelInfo  = observability.LogLevelInfo
	LogLevelWarn  = observability.LogLevelWarn
	LogLevelError = observability.LogLevelError
)