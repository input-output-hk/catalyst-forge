package ociv2

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/input-output-hk/catalyst-forge/lib/ociv2/internal"
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/utils"
)

// SignDigest signs an artifact digest using Cosign
func (c *client) SignDigest(ctx context.Context, refOrDigest string) (Descriptor, error) {
	// If Cosign is not enabled, return without signing
	if !c.opts.Cosign.Enable {
		if c.opts.Logger != nil {
			c.opts.Logger("oci.sign.skipped", "ref", refOrDigest, "reason", "cosign disabled")
		}
		return Descriptor{Ref: NormalizeRef(refOrDigest)}, nil
	}

	// Validate reference
	if err := validateRef(refOrDigest); err != nil {
		return Descriptor{}, err
	}

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
	defer cancel()

	// Normalize reference
	ref := NormalizeRef(refOrDigest)

	// Extract registry
	registry, err := extractRegistry(ref)
	if err != nil {
		return Descriptor{}, fmt.Errorf("failed to extract registry: %w", err)
	}

	// Log operation
	if c.opts.Logger != nil {
		c.opts.Logger("oci.sign", "ref", ref, "registry", registry)
	}

	// If not a digest, resolve to get digest
	if !IsDigestRef(ref) {
		desc, err := c.Resolve(ctx, ref)
		if err != nil {
			return Descriptor{}, fmt.Errorf("failed to resolve ref to digest: %w", err)
		}
		ref = desc.Ref
		if ref == "" || !IsDigestRef(ref) {
			ref = toCanonical(refOrDigest, desc.Digest)
		}
	}

	// Create Cosign signer
	signer := internal.NewCosignSigner(
		c.opts.Cosign.RekorURL,
		c.opts.Cosign.FulcioURL,
		c.opts.Cosign.AllowInsecure,
		c.getCosignAuth(registry),
	)

	// Sign the digest
	sigRef, err := signer.Sign(ctx, ref)
	if err != nil {
		return Descriptor{}, fmt.Errorf("failed to sign: %w", err)
	}

	// Return descriptor for the signature
	return Descriptor{
		Ref:       sigRef,
		MediaType: "application/vnd.dev.cosign.signature.v1+json",
		Annotations: map[string]string{
			utils.AnnForgeSigned:    "true",
			utils.AnnForgeSignedAt:  time.Now().UTC().Format(time.RFC3339),
			utils.AnnForgeSignature: sigRef,
		},
	}, nil
}

// VerifyDigest verifies signatures on an artifact
func (c *client) VerifyDigest(ctx context.Context, refOrDigest string) (*VerificationReport, error) {
	// If Cosign is not enabled, return unsigned status
	if !c.opts.Cosign.Enable {
		if c.opts.Logger != nil {
			c.opts.Logger("oci.verify.skipped", "ref", refOrDigest, "reason", "cosign disabled")
		}
		return &VerificationReport{
			Digest: refOrDigest,
			Signed: false,
			Errors: []string{"cosign verification disabled"},
		}, nil
	}

	// Validate reference
	if err := validateRef(refOrDigest); err != nil {
		return nil, err
	}

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
	defer cancel()

	// Normalize reference
	ref := NormalizeRef(refOrDigest)

	// Extract registry
	registry, err := extractRegistry(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to extract registry: %w", err)
	}

	// Log operation
	if c.opts.Logger != nil {
		c.opts.Logger("oci.verify", "ref", ref, "registry", registry)
	}

	// If not a digest, resolve to get digest
	digest := ref
	if !IsDigestRef(ref) {
		desc, err := c.Resolve(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve ref to digest: %w", err)
		}
		digest = desc.Digest
		if desc.Ref != "" && IsDigestRef(desc.Ref) {
			ref = desc.Ref
		} else {
			ref = toCanonical(refOrDigest, digest)
		}
	} else {
		// Extract digest from ref
		parts := strings.Split(ref, "@")
		if len(parts) == 2 {
			digest = parts[1]
		}
	}

	// Create Cosign verifier
	verifier := internal.NewCosignVerifier(
		c.opts.Cosign.RekorURL,
		c.opts.Cosign.FulcioURL,
		c.opts.Cosign.AllowInsecure,
		c.getCosignAuth(registry),
	)

	// Set identity requirements if provided
	var issuer, subject string
	if c.opts.Cosign.Identity != nil {
		issuer = c.opts.Cosign.Identity.Issuer
		subject = c.opts.Cosign.Identity.Subject
	}

	// Verify the signatures
	signers, bundleVerified, errors, err := verifier.Verify(ctx, ref, issuer, subject)

	// Convert internal signers to our SignerIdentity type
	var identities []SignerIdentity
	for _, s := range signers {
		identities = append(identities, SignerIdentity{
			Issuer:  s.Issuer,
			Subject: s.Subject,
			SANs:    s.SANs,
			Time:    s.Time,
		})
	}

	if err != nil {
		// Even if verification fails, return what we found
		return &VerificationReport{
			Digest:         digest,
			Signed:         false,
			Signers:        identities,
			BundleVerified: bundleVerified,
			Errors:         append(errors, err.Error()),
		}, nil
	}

	return &VerificationReport{
		Digest:         digest,
		Signed:         len(identities) > 0,
		Signers:        identities,
		BundleVerified: bundleVerified,
		Errors:         errors,
	}, nil
}

// getCosignAuth returns an auth function for Cosign operations
func (c *client) getCosignAuth(registry string) internal.CosignAuthFunc {
	return func(ctx context.Context) (string, string, error) {
		if c.auth == nil {
			return "", "", nil
		}

		// Get authenticator for the registry
		authObj, err := c.auth.Authenticator(registry)
		if err != nil {
			return "", "", err
		}

		// Try to extract username/password or token
		if authObj == nil {
			return "", "", nil
		}

		// Check if it's a StaticAuth
		if static, ok := c.auth.(*StaticAuth); ok {
			if static.Token != "" {
				return "", static.Token, nil
			}
			return static.Username, static.Password, nil
		}

		// Check if it's a GitHubAuth
		if gh, ok := c.auth.(*GitHubAuth); ok && gh.Token != "" {
			return "", gh.Token, nil
		}

		// For other auth types, try to get basic auth
		// This is a simplified approach; in production you'd need more sophisticated conversion
		return "", "", nil
	}
}
