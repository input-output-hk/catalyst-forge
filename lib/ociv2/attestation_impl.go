package ociv2

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/attestation"
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/observability"
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/signing"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Attest attaches a DSSE attestation to an artifact
func (c *client) Attest(ctx context.Context, refOrDigest string, opts attestation.AttestationOptions) (Descriptor, error) {
	operation := "attest"

	// Validate input
	if refOrDigest == "" {
		return Descriptor{}, observability.NewValidationError(operation, refOrDigest, fmt.Errorf("reference cannot be empty"))
	}

	if opts.PredicateType == "" {
		return Descriptor{}, observability.NewValidationError(operation, refOrDigest, fmt.Errorf("predicate type is required"))
	}

	if opts.Predicate == nil {
		return Descriptor{}, observability.NewValidationError(operation, refOrDigest, fmt.Errorf("predicate is required"))
	}

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
	defer cancel()

	// Normalize reference
	ref := NormalizeRef(refOrDigest)

	// Get the subject digest
	subjectDigest, err := c.getDigest(ctx, ref)
	if err != nil {
		return Descriptor{}, fmt.Errorf("failed to get subject digest: %w", err)
	}

	// Create in-toto statement
	statement := attestation.IntotoStatement{
		Type:          "https://in-toto.io/Statement/v0.1",
		PredicateType: opts.PredicateType,
		Subject: []attestation.Subject{
			{
				Name: ref,
				Digest: map[string]string{
					"sha256": strings.TrimPrefix(subjectDigest, "sha256:"),
				},
			},
		},
		Predicate: opts.Predicate,
	}

	// Marshal statement
	statementBytes, err := json.Marshal(statement)
	if err != nil {
		return Descriptor{}, fmt.Errorf("failed to marshal statement: %w", err)
	}

	// Create DSSE envelope
	envelope := attestation.DSSEEnvelope{
		PayloadType: "application/vnd.in-toto+json",
		Payload:     base64.StdEncoding.EncodeToString(statementBytes),
		Signatures:  []attestation.Signature{}, // Will be populated by signing
	}

	// Sign the envelope
	if c.opts.Cosign.Enable {
		envelope, err = c.signDSSE(ctx, envelope, opts.SigningKey)
		if err != nil {
			return Descriptor{}, fmt.Errorf("failed to sign DSSE: %w", err)
		}
	} else {
		// Create a mock signature for testing
		envelope.Signatures = append(envelope.Signatures, attestation.Signature{
			KeyID: "test-key",
			Sig:   base64.StdEncoding.EncodeToString([]byte("mock-signature")),
		})
	}

	// Marshal envelope
	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return Descriptor{}, fmt.Errorf("failed to marshal envelope: %w", err)
	}

	// Push attestation as OCI artifact
	// Remove any existing tag or digest from the reference
	baseRef := strings.Split(ref, "@")[0] // Remove digest if present
	if idx := strings.LastIndex(baseRef, ":"); idx > 0 {
		// Check if this is a tag (not a port)
		afterColon := baseRef[idx+1:]
		if !strings.Contains(afterColon, "/") {
			baseRef = baseRef[:idx]
		}
	}
	attestRef := fmt.Sprintf("%s:sha256-%s.att", baseRef, strings.TrimPrefix(subjectDigest, "sha256:"))

	// Create attestation descriptor
	desc, err := c.pushAttestation(ctx, attestRef, envelopeBytes, opts.Annotations)
	if err != nil {
		return Descriptor{}, fmt.Errorf("failed to push attestation: %w", err)
	}

	return desc, nil
}

// VerifyAttestations queries and verifies attestations for a subject
func (c *client) VerifyAttestations(ctx context.Context, refOrDigest string, predicateType string) (*attestation.AttestationReport, error) {
	operation := "verify_attestations"

	// Validate input
	if refOrDigest == "" {
		return nil, observability.NewValidationError(operation, refOrDigest, fmt.Errorf("reference cannot be empty"))
	}

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
	defer cancel()

	// Normalize reference
	ref := NormalizeRef(refOrDigest)

	// Get the subject descriptor
	subjectDesc, err := c.Resolve(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve subject: %w", err)
	}

	// Query attestations
	attestations, err := c.queryAttestations(ctx, ref, subjectDesc.Digest, predicateType)
	if err != nil {
		return nil, fmt.Errorf("failed to query attestations: %w", err)
	}

	report := &attestation.AttestationReport{
		Subject:      subjectDesc,
		Attestations: attestations,
		Verified:     true,
		Errors:       []error{},
	}

	// Verify each attestation
	for i := range report.Attestations {
		att := &report.Attestations[i]

		if c.opts.Cosign.Enable {
			verified, signerIdentity, err := c.verifyDSSE(ctx, att.Envelope)
			att.Verified = verified
			att.SignerIdentity = signerIdentity

			if err != nil {
				report.Errors = append(report.Errors, fmt.Errorf("attestation %d: %w", i, err))
				report.Verified = false
			}
		} else {
			// Mock verification for testing
			att.Verified = true
			att.SignerIdentity = &signing.SignerIdentity{
				Issuer:  "test-issuer",
				Subject: "test-subject",
			}
		}
	}

	return report, nil
}

// pushAttestation pushes an attestation as an OCI artifact
func (c *client) pushAttestation(ctx context.Context, ref string, data []byte, annotations map[string]string) (Descriptor, error) {
	// Parse reference
	nameRef, err := name.ParseReference(ref)
	if err != nil {
		return Descriptor{}, fmt.Errorf("failed to parse reference: %w", err)
	}

	// Create attestation layer
	layer := static.NewLayer(data, types.MediaType(attestation.MediaTypeDSSE))

	// Create empty image
	img := empty.Image

	// Add layer
	img, err = mutate.AppendLayers(img, layer)
	if err != nil {
		return Descriptor{}, fmt.Errorf("failed to append layer: %w", err)
	}

	// Add annotations
	if len(annotations) > 0 {
		img = mutate.Annotations(img, annotations).(v1.Image)
	}

	// Get auth
	authFunc := c.getGGCRAuthFor(nameRef.Context().RegistryStr())
	remoteOpts := []remote.Option{
		remote.WithContext(ctx),
		remote.WithUserAgent(c.opts.UserAgent),
	}
	if authFunc != nil {
		auth, err := authFunc()
		if err == nil && auth != nil {
			remoteOpts = append(remoteOpts, remote.WithAuth(auth))
		}
	}

	// Push image
	if err := remote.Write(nameRef, img, remoteOpts...); err != nil {
		return Descriptor{}, fmt.Errorf("failed to push attestation: %w", err)
	}

	// Get digest
	dgst, err := img.Digest()
	if err != nil {
		return Descriptor{}, fmt.Errorf("failed to get digest: %w", err)
	}

	// Get size
	manifest, err := img.Manifest()
	if err != nil {
		return Descriptor{}, fmt.Errorf("failed to get manifest: %w", err)
	}

	size := int64(0)
	for _, layer := range manifest.Layers {
		size += layer.Size
	}

	return Descriptor{
		Ref:       fmt.Sprintf("%s@%s", ref, dgst),
		Digest:    dgst.String(),
		MediaType: string(types.OCIManifestSchema1),
		Size:      size,
	}, nil
}

// queryAttestations queries attestations for a subject
func (c *client) queryAttestations(ctx context.Context, ref, subjectDigest, predicateType string) ([]attestation.AttestationEntry, error) {
	// For attestations, we follow the cosign convention of using .att suffix
	// Remove any existing tag or digest from the reference
	baseRef := strings.Split(ref, "@")[0] // Remove digest if present
	if idx := strings.LastIndex(baseRef, ":"); idx > 0 {
		// Check if this is a tag (not a port)
		afterColon := baseRef[idx+1:]
		if !strings.Contains(afterColon, "/") {
			baseRef = baseRef[:idx]
		}
	}
	attestRef := fmt.Sprintf("%s:sha256-%s.att", baseRef, strings.TrimPrefix(subjectDigest, "sha256:"))

	// Try to pull the attestation
	entries := []attestation.AttestationEntry{}

	// Parse reference
	nameRef, err := name.ParseReference(attestRef)
	if err != nil {
		// No attestations found
		return entries, nil
	}

	// Get auth
	authFunc := c.getGGCRAuthFor(nameRef.Context().RegistryStr())
	remoteOpts := []remote.Option{
		remote.WithContext(ctx),
		remote.WithUserAgent(c.opts.UserAgent),
	}
	if authFunc != nil {
		auth, err := authFunc()
		if err == nil && auth != nil {
			remoteOpts = append(remoteOpts, remote.WithAuth(auth))
		}
	}

	// Try to get the attestation
	img, err := remote.Image(nameRef, remoteOpts...)
	if err != nil {
		// No attestations found
		return entries, nil
	}

	// Get layers
	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("failed to get layers: %w", err)
	}

	// Process each layer as potential attestation
	for _, layer := range layers {
		mt, err := layer.MediaType()
		if err != nil {
			continue
		}

		// Check if it's a DSSE envelope
		if string(mt) != attestation.MediaTypeDSSE {
			continue
		}

		// Read layer content
		rc, err := layer.Compressed()
		if err != nil {
			continue
		}
		defer func() { _ = rc.Close() }()

		data, err := io.ReadAll(rc)
		if err != nil {
			continue
		}

		// Parse DSSE envelope
		var envelope attestation.DSSEEnvelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			continue
		}

		// Decode and parse statement
		payloadBytes, err := base64.StdEncoding.DecodeString(envelope.Payload)
		if err != nil {
			continue
		}

		var statement attestation.IntotoStatement
		if err := json.Unmarshal(payloadBytes, &statement); err != nil {
			continue
		}

		// Filter by predicate type if specified
		if predicateType != "" && statement.PredicateType != predicateType {
			continue
		}

		// Get layer digest
		lgst, err := layer.Digest()
		if err != nil {
			lgst = v1.Hash{}
		}

		// Get layer size
		size, err := layer.Size()
		if err != nil {
			size = 0
		}

		entry := attestation.AttestationEntry{
			Envelope:  envelope,
			Statement: statement,
			Descriptor: ocispec.Descriptor{
				MediaType: string(mt),
				Digest:    digest.Digest(lgst.String()),
				Size:      size,
			},
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// signDSSE signs a DSSE envelope
func (c *client) signDSSE(ctx context.Context, envelope attestation.DSSEEnvelope, signingKey []byte) (attestation.DSSEEnvelope, error) {
	// For now, we'll create a mock signature
	// In a real implementation, this would use sigstore/cosign libraries

	// Create PAE (Pre-Authentication Encoding)
	pae := fmt.Sprintf("DSSEv1 %d %s %d %s",
		len(envelope.PayloadType), envelope.PayloadType,
		len(envelope.Payload), envelope.Payload)

	// Hash the PAE
	hash := sha256.Sum256([]byte(pae))

	// Create signature (mock for now)
	sig := attestation.Signature{
		KeyID: "keyless",
		Sig:   base64.StdEncoding.EncodeToString(hash[:]),
	}

	envelope.Signatures = append(envelope.Signatures, sig)
	return envelope, nil
}

// verifyDSSE verifies a DSSE envelope
func (c *client) verifyDSSE(ctx context.Context, envelope attestation.DSSEEnvelope) (bool, *signing.SignerIdentity, error) {
	// For now, return mock verification
	// In a real implementation, this would use sigstore/cosign libraries

	if len(envelope.Signatures) == 0 {
		return false, nil, fmt.Errorf("no signatures found")
	}

	// Mock verification
	identity := &signing.SignerIdentity{
		Issuer:  "https://token.actions.githubusercontent.com",
		Subject: "repo:example/repo:ref:refs/heads/main",
	}

	return true, identity, nil
}

// getDigest gets the digest for a reference
func (c *client) getDigest(ctx context.Context, ref string) (string, error) {
	// If already a digest reference, extract it
	if IsDigestRef(ref) {
		parts := strings.Split(ref, "@")
		if len(parts) == 2 {
			return parts[1], nil
		}
	}

	// Otherwise resolve to get digest
	desc, err := c.Resolve(ctx, ref)
	if err != nil {
		return "", err
	}

	return desc.Digest, nil
}

// ValidateArtifact validates an artifact against expectations
func (c *client) ValidateArtifact(ctx context.Context, ref string, spec attestation.ArtifactValidationSpec) error {
	// This will be implemented in Phase 10.4
	return fmt.Errorf("not implemented")
}
