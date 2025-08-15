package attestation

import (
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/signing"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// DSSE media types
const (
	// MediaTypeDSSE is the media type for DSSE envelopes
	MediaTypeDSSE = "application/vnd.dsse.envelope.v1+json"
	
	// MediaTypeIntotoStatement is the media type for in-toto statements
	MediaTypeIntotoStatement = "application/vnd.in-toto+json"
	
	// Predicate types
	PredicateSLSAProvenance = "https://slsa.dev/provenance/v0.2"
	PredicateSLSAProvenanceV1 = "https://slsa.dev/provenance/v1"
	PredicateSPDX = "https://spdx.dev/Document/v2.3"
	PredicateCycloneDX = "https://cyclonedx.org/bom/v1.4"
	PredicateCustom = "https://example.com/custom/v1"
)

// DSSEEnvelope represents a DSSE envelope
type DSSEEnvelope struct {
	PayloadType string      `json:"payloadType"`
	Payload     string      `json:"payload"` // base64 encoded
	Signatures  []Signature `json:"signatures"`
}

// Signature represents a DSSE signature
type Signature struct {
	KeyID string `json:"keyid,omitempty"`
	Sig   string `json:"sig"` // base64 encoded
}

// IntotoStatement represents an in-toto statement
type IntotoStatement struct {
	Type          string      `json:"_type"`
	PredicateType string      `json:"predicateType"`
	Subject       []Subject   `json:"subject"`
	Predicate     interface{} `json:"predicate"`
}

// Subject represents a subject in an in-toto statement
type Subject struct {
	Name   string            `json:"name"`
	Digest map[string]string `json:"digest"`
}

// AttestationOptions configures attestation creation
type AttestationOptions struct {
	// PredicateType specifies the type of predicate (e.g., SLSA provenance)
	PredicateType string
	
	// Predicate is the actual predicate data (will be JSON encoded)
	Predicate interface{}
	
	// SigningKey is the private key for signing (optional, uses keyless if nil)
	SigningKey []byte
	
	// Replace existing attestations of the same predicate type
	Replace bool
	
	// Annotations to add to the attestation layer
	Annotations map[string]string
}

// AttestationReport contains information about attestations
type AttestationReport struct {
	// Subject is the artifact that has attestations
	Subject interface{} // Will be replaced with proper Descriptor type
	
	// Attestations found for the subject
	Attestations []AttestationEntry
	
	// Verified indicates if all signatures were verified
	Verified bool
	
	// Errors contains any verification errors
	Errors []error
}

// AttestationEntry represents a single attestation
type AttestationEntry struct {
	// Envelope is the DSSE envelope
	Envelope DSSEEnvelope
	
	// Statement is the parsed in-toto statement
	Statement IntotoStatement
	
	// Verified indicates if this attestation was verified
	Verified bool
	
	// SignerIdentity contains information about who signed it
	SignerIdentity *signing.SignerIdentity
	
	// Descriptor is the OCI descriptor for this attestation
	Descriptor ocispec.Descriptor
}


// ArtifactValidationSpec defines validation requirements
type ArtifactValidationSpec struct {
	// RequiredPredicateTypes lists predicate types that must be present
	RequiredPredicateTypes []string
	
	// MinAttestations is the minimum number of attestations required
	MinAttestations int
	
	// RequireVerified indicates if all attestations must be verified
	RequireVerified bool
}