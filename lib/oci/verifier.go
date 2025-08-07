package oci

import (
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/cosign/v2/pkg/oci/remote"
)

// Verifier defines the interface for signature verification
type Verifier interface {
	Verify(ctx context.Context, imageRef string, manifest ocispec.Descriptor) error
}

// CosignVerifier implements keyless OIDC-based cosign verification
type CosignVerifier struct {
	oidcIssuer  string
	oidcSubject string
	rekorURL    string
	skipTLog    bool
}

// CosignOptions holds configuration for cosign verification
type CosignOptions struct {
	OIDCIssuer  string // Required: OIDC issuer
	OIDCSubject string // Required: OIDC subject
	RekorURL    string // Optional: Rekor URL (defaults to public)
	SkipTLog    bool   // Optional: Skip transparency log
}

// NewCosignVerifier creates a new cosign verifier with the given options
func NewCosignVerifier(opts CosignOptions) (*CosignVerifier, error) {
	if opts.OIDCIssuer == "" {
		return nil, fmt.Errorf("OIDC issuer cannot be empty")
	}
	if opts.OIDCSubject == "" {
		return nil, fmt.Errorf("OIDC subject cannot be empty")
	}

	rekorURL := opts.RekorURL
	if rekorURL == "" {
		rekorURL = "https://rekor.sigstore.dev"
	}

	return &CosignVerifier{
		oidcIssuer:  opts.OIDCIssuer,
		oidcSubject: opts.OIDCSubject,
		rekorURL:    rekorURL,
		skipTLog:    opts.SkipTLog,
	}, nil
}

// Verify verifies the cosign signature for the given image reference
func (v *CosignVerifier) Verify(ctx context.Context, imageRef string, manifest ocispec.Descriptor) error {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return fmt.Errorf("failed to parse image reference %s: %w", imageRef, err)
	}

	checkOpts := &cosign.CheckOpts{
		ClaimVerifier:      cosign.SimpleClaimVerifier,
		RegistryClientOpts: []remote.Option{},
		IgnoreTlog:         v.skipTLog,
		Identities: []cosign.Identity{
			{
				Issuer:  v.oidcIssuer,
				Subject: v.oidcSubject,
			},
		},
	}

	_, bundleVerified, err := cosign.VerifyImageSignatures(ctx, ref, checkOpts)
	if err != nil {
		return fmt.Errorf("cosign signature verification failed for %s: %w", imageRef, err)
	}

	if !bundleVerified && !v.skipTLog {
		return fmt.Errorf("transparency log verification failed for %s", imageRef)
	}

	return nil
}
