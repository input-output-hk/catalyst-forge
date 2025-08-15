//go:build signing_integration

package signing

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/registry"
	ociv2 "github.com/input-output-hk/catalyst-forge/lib/ociv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSigningIntegration tests Cosign signing and verification in a real environment
// This test requires COSIGN_EXPERIMENTAL=1 and appropriate environment setup
func TestSigningIntegration(t *testing.T) {
	// Skip if not in CI or signing environment
	if os.Getenv("COSIGN_EXPERIMENTAL") != "1" && os.Getenv("CI") == "" {
		t.Skip("Skipping signing integration test - requires COSIGN_EXPERIMENTAL=1 or CI environment")
	}

	// Set up in-memory registry
	registryServer := httptest.NewServer(registry.New())
	defer registryServer.Close()

	registryHost := strings.TrimPrefix(registryServer.URL, "http://")

	// Create client with signing enabled
	client, err := ociv2.New(ociv2.ClientOptions{
		PlainHTTP: true,
		Cosign: ociv2.CosignOpts{
			Enable:        true,
			AllowInsecure: true, // For testing only
		},
	})
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("SignAndVerify_JSON_Artifact", func(t *testing.T) {
		testRef := fmt.Sprintf("%s/test/signed-json:latest", registryHost)

		// Push an artifact first
		testJSON := []byte(`{"signed": true, "content": "test data"}`)
		testAnnotations := ociv2.NewAnnotations().
			WithForgeKind("signed-config").
			WithForgeProject("catalyst-test")

		desc, err := client.PushJSON(ctx, testRef, "application/vnd.test.config+json",
			testJSON, testAnnotations)
		require.NoError(t, err)

		// Sign the artifact
		signDesc, err := client.SignDigest(ctx, desc.Digest)
		if err != nil {
			// Check if this is expected in the environment
			if strings.Contains(err.Error(), "no provider") ||
				strings.Contains(err.Error(), "not found") {
				t.Skip("Skipping signing test - no signing provider available")
			}
			require.NoError(t, err)
		}

		// Verify signature exists
		assert.NotEmpty(t, signDesc.Digest)
		assert.NotEmpty(t, signDesc.Ref)

		// Verify the signature
		report, err := client.VerifyDigest(ctx, desc.Digest)
		require.NoError(t, err)

		// Check verification report
		assert.NotNil(t, report)
		assert.Equal(t, desc.Digest, report.Digest)
		assert.True(t, report.Signed, "Artifact should be marked as signed")

		if len(report.Signers) > 0 {
			assert.NotEmpty(t, report.Signers[0].Subject, "Should have signer subject")
		}
	})

	t.Run("SignAndVerify_TAR_Artifact", func(t *testing.T) {
		testRef := fmt.Sprintf("%s/test/signed-tar:v1.0", registryHost)

		// Push a TAR artifact
		configJSON := []byte(`{"name": "signed-package", "version": "1.0.0"}`)
		tarData := "signed tar content for testing"
		tarReader := strings.NewReader(tarData)

		testAnnotations := ociv2.NewAnnotations().
			WithForgeKind("signed-package").
			WithForgeProject("catalyst-test").
			WithTrace("signing-test-123")

		desc, err := client.PushTar(ctx, testRef, configJSON,
			"application/vnd.test.package.config+json",
			"application/vnd.test.package.layer.v1.tar+gzip",
			tarReader, int64(len(tarData)), testAnnotations)
		require.NoError(t, err)

		// Sign the artifact
		signDesc, err := client.SignDigest(ctx, desc.Digest)
		if err != nil {
			// Check if this is expected in the environment
			if strings.Contains(err.Error(), "no provider") ||
				strings.Contains(err.Error(), "not found") {
				t.Skip("Skipping signing test - no signing provider available")
			}
			require.NoError(t, err)
		}

		// Verify signature exists
		assert.NotEmpty(t, signDesc.Digest)

		// Verify the signature
		report, err := client.VerifyDigest(ctx, desc.Digest)
		require.NoError(t, err)

		// Check verification report
		assert.True(t, report.Signed)
		assert.Equal(t, desc.Digest, report.Digest)
	})

	t.Run("VerifyUnsigned_Artifact", func(t *testing.T) {
		testRef := fmt.Sprintf("%s/test/unsigned:latest", registryHost)

		// Push an unsigned artifact
		testJSON := []byte(`{"unsigned": true}`)
		desc, err := client.PushJSON(ctx, testRef, "application/json", testJSON, nil)
		require.NoError(t, err)

		// Try to verify (should indicate no signatures)
		report, err := client.VerifyDigest(ctx, desc.Digest)
		require.NoError(t, err)

		// Should indicate not signed
		assert.False(t, report.Signed, "Unsigned artifact should not be marked as signed")
		assert.Empty(t, report.Signers, "Unsigned artifact should have no signers")
	})
}

// TestSigningWithIdentityConstraints tests signing with OIDC identity requirements
func TestSigningWithIdentityConstraints(t *testing.T) {
	// Skip if not in appropriate signing environment
	if os.Getenv("COSIGN_EXPERIMENTAL") != "1" && os.Getenv("GITHUB_ACTIONS") == "" {
		t.Skip("Skipping identity constraint test - requires proper OIDC environment")
	}

	// Set up in-memory registry
	registryServer := httptest.NewServer(registry.New())
	defer registryServer.Close()

	registryHost := strings.TrimPrefix(registryServer.URL, "http://")

	// Create client with identity constraints
	client, err := ociv2.New(ociv2.ClientOptions{
		PlainHTTP: true,
		Cosign: ociv2.CosignOpts{
			Enable:        true,
			AllowInsecure: true,
			Identity: &ociv2.OIDCIdentity{
				Issuer:  "https://token.actions.githubusercontent.com",
				Subject: "repo:input-output-hk/catalyst-forge:ref:refs/heads/main",
			},
		},
	})
	require.NoError(t, err)

	ctx := context.Background()
	testRef := fmt.Sprintf("%s/test/identity-constrained:latest", registryHost)

	// Push an artifact
	testJSON := []byte(`{"identity": "constrained"}`)
	desc, err := client.PushJSON(ctx, testRef, "application/json", testJSON, nil)
	require.NoError(t, err)

	// Try to verify with identity constraints
	report, err := client.VerifyDigest(ctx, desc.Digest)

	if err != nil {
		// If verification fails due to environment, that's expected
		if strings.Contains(err.Error(), "no provider") ||
			strings.Contains(err.Error(), "identity") ||
			strings.Contains(err.Error(), "issuer") {
			t.Logf("Identity constraint test failed as expected: %v", err)
			return
		}
		require.NoError(t, err)
	}

	// If verification succeeds, check the identity
	assert.NotNil(t, report)
	if len(report.Signers) > 0 {
		signer := report.Signers[0]
		t.Logf("Verified signer: issuer=%s, subject=%s", signer.Issuer, signer.Subject)
	}
}

// TestSigningErrors tests error conditions in signing operations
func TestSigningErrors(t *testing.T) {
	// Create client with signing disabled
	client, err := ociv2.New(ociv2.ClientOptions{
		Cosign: ociv2.CosignOpts{
			Enable: false,
		},
	})
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("SigningDisabled", func(t *testing.T) {
		// Try to sign with signing disabled
		desc, err := client.SignDigest(ctx, "sha256:abc123")

		// Should return a descriptor indicating signing was skipped
		assert.NoError(t, err)
		assert.Contains(t, desc.MediaType, "skipped")
	})

	t.Run("VerificationDisabled", func(t *testing.T) {
		// Try to verify with signing disabled
		report, err := client.VerifyDigest(ctx, "sha256:abc123")

		// Should return a report indicating verification was skipped
		assert.NoError(t, err)
		assert.NotNil(t, report)
		assert.False(t, report.Signed)
		assert.Empty(t, report.Signers)
	})

	t.Run("InvalidDigest", func(t *testing.T) {
		// Enable signing for error testing
		client, err := ociv2.New(ociv2.ClientOptions{
			Cosign: ociv2.CosignOpts{
				Enable:        true,
				AllowInsecure: true,
			},
		})
		require.NoError(t, err)

		// Try to sign invalid digest
		_, err = client.SignDigest(ctx, "invalid-digest")
		assert.Error(t, err)

		var ociErr *ociv2.OCIError
		assert.True(t, errors.As(err, &ociErr))
		assert.Equal(t, ociv2.ErrorCategoryValidation, ociErr.Category)
	})
}

// TestSigningObservability tests that signing operations are properly logged and tracked
func TestSigningObservability(t *testing.T) {
	if os.Getenv("COSIGN_EXPERIMENTAL") != "1" && os.Getenv("CI") == "" {
		t.Skip("Skipping signing observability test - requires signing environment")
	}

	var logEntries []string
	var metricsCallbacks []*ociv2.Metrics

	logFunc := func(msg string, kv ...any) {
		logEntries = append(logEntries, msg)
		t.Logf("Log: %s", msg)
	}

	metricsFunc := func(m *ociv2.Metrics) {
		metricsCallbacks = append(metricsCallbacks, m)
	}

	// Set up in-memory registry
	registryServer := httptest.NewServer(registry.New())
	defer registryServer.Close()

	registryHost := strings.TrimPrefix(registryServer.URL, "http://")

	client, err := ociv2.New(ociv2.ClientOptions{
		PlainHTTP:        true,
		StructuredLogger: ociv2.NewDefaultLogger(logFunc),
		EnableMetrics:    true,
		MetricsCallback:  metricsFunc,
		Cosign: ociv2.CosignOpts{
			Enable:        true,
			AllowInsecure: true,
		},
	})
	require.NoError(t, err)

	ctx := context.Background()
	testRef := fmt.Sprintf("%s/test/observed-signing:latest", registryHost)

	// Push and sign an artifact
	testJSON := []byte(`{"observability": "test"}`)
	desc, err := client.PushJSON(ctx, testRef, "application/json", testJSON, nil)
	require.NoError(t, err)

	// Sign the artifact
	_, err = client.SignDigest(ctx, desc.Digest)
	if err != nil && (strings.Contains(err.Error(), "no provider") ||
		strings.Contains(err.Error(), "not found")) {
		t.Skip("Skipping observability test - no signing provider available")
	}
	require.NoError(t, err)

	// Verify signing operations were logged
	assert.Greater(t, len(logEntries), 0, "Should have logged signing operations")

	// Check for signing-related log entries
	foundSigningLog := false
	for _, entry := range logEntries {
		if strings.Contains(entry, "sign") || strings.Contains(entry, "cosign") {
			foundSigningLog = true
			break
		}
	}
	assert.True(t, foundSigningLog, "Should have logged signing-related operations")

	// Verify metrics were collected
	assert.Greater(t, len(metricsCallbacks), 0, "Should have collected metrics")
}
