package ociv2

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/attestation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDSSEEnvelope(t *testing.T) {
	t.Parallel()

	statement := attestation.IntotoStatement{
		Type:          "https://in-toto.io/Statement/v0.1",
		PredicateType: attestation.PredicateSLSAProvenance,
		Subject: []attestation.Subject{
			{
				Name: "test/image:v1",
				Digest: map[string]string{
					"sha256": "abc123",
				},
			},
		},
		Predicate: map[string]interface{}{
			"builder": map[string]string{
				"id": "test-builder",
			},
			"buildType": "test",
		},
	}

	statementBytes, err := json.Marshal(statement)
	require.NoError(t, err)

	envelope := DSSEEnvelope{
		PayloadType: "application/vnd.in-toto+json",
		Payload:     base64.StdEncoding.EncodeToString(statementBytes),
		Signatures: []Signature{
			{
				KeyID: "test-key",
				Sig:   base64.StdEncoding.EncodeToString([]byte("test-signature")),
			},
		},
	}

	// Marshal and unmarshal
	data, err := json.Marshal(envelope)
	require.NoError(t, err)

	var decoded DSSEEnvelope
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, envelope.PayloadType, decoded.PayloadType)
	assert.Equal(t, envelope.Payload, decoded.Payload)
	assert.Len(t, decoded.Signatures, 1)
	assert.Equal(t, "test-key", decoded.Signatures[0].KeyID)

	// Decode and verify statement
	payloadBytes, err := base64.StdEncoding.DecodeString(decoded.Payload)
	require.NoError(t, err)

	var decodedStatement IntotoStatement
	err = json.Unmarshal(payloadBytes, &decodedStatement)
	require.NoError(t, err)

	assert.Equal(t, statement.Type, decodedStatement.Type)
	assert.Equal(t, statement.PredicateType, decodedStatement.PredicateType)
	assert.Len(t, decodedStatement.Subject, 1)
	assert.Equal(t, "test/image:v1", decodedStatement.Subject[0].Name)
}

func TestCreateSLSAProvenance(t *testing.T) {
	t.Parallel()

	materials := []Material{
		{
			URI: "git+https://github.com/example/repo",
			Digest: map[string]string{
				"sha1": "abc123",
			},
		},
		{
			URI: "pkg:github/package-url/purl-spec@244fd47e07d1004f0aed9c",
			Digest: map[string]string{
				"sha256": "def456",
			},
		},
	}

	provenance := CreateSLSAProvenance(
		"https://github.com/actions/runner",
		"https://github.com/slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@v1.2.0",
		materials,
	)

	assert.Equal(t, "https://github.com/actions/runner", provenance.Builder.ID)
	assert.Contains(t, provenance.BuildType, "slsa-github-generator")
	assert.Len(t, provenance.Materials, 2)
	assert.Equal(t, "git+https://github.com/example/repo", provenance.Materials[0].URI)
}

func TestAttestationOptions_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    AttestationOptions
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid_slsa_provenance",
			opts: AttestationOptions{
				PredicateType: PredicateSLSAProvenance,
				Predicate: &SLSAProvenance{
					Builder: Builder{ID: "test-builder"},
					BuildType: "test",
				},
			},
			wantErr: false,
		},
		{
			name: "valid_custom_predicate",
			opts: AttestationOptions{
				PredicateType: PredicateCustom,
				Predicate: map[string]interface{}{
					"custom": "data",
				},
			},
			wantErr: false,
		},
		{
			name: "missing_predicate_type",
			opts: AttestationOptions{
				Predicate: map[string]interface{}{
					"test": "data",
				},
			},
			wantErr: true,
			errMsg:  "predicate type is required",
		},
		{
			name: "missing_predicate",
			opts: AttestationOptions{
				PredicateType: PredicateSLSAProvenance,
			},
			wantErr: true,
			errMsg:  "predicate is required",
		},
	}

	// Create a mock client for testing
	client, err := New(ClientOptions{
		Cosign: CosignOpts{
			Enable: false, // Use mock signing
		},
	})
	require.NoError(t, err)

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a valid reference to test option validation
			_, err := client.Attest(ctx, "test/image:v1", tt.opts)
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				// Will fail trying to resolve the test reference, but options are valid
				assert.Error(t, err)
				// Error will be about failing to get digest, not about options
				assert.NotContains(t, err.Error(), "predicate")
			}
		})
	}
}

func TestIntegrationAttestation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Parallel()

	// Set up in-memory registry
	registryServer := httptest.NewServer(registry.New())
	defer registryServer.Close()

	registryHost := strings.TrimPrefix(registryServer.URL, "http://")

	// Create client
	client, err := New(ClientOptions{
		PlainHTTP: true,
		Cosign: CosignOpts{
			Enable: false, // Use mock signing for tests
		},
	})
	require.NoError(t, err)

	ctx := context.Background()

	// First, push a test artifact
	ref := fmt.Sprintf("%s/test/attestation:v1", registryHost)
	testData := []byte(`{"test": "artifact"}`)
	
	desc, err := client.PushJSON(ctx, ref, "application/json", testData, Annotations{
		"test.type": "attestation-subject",
	})
	require.NoError(t, err)

	t.Run("AttachSLSAProvenance", func(t *testing.T) {
		// Create SLSA provenance
		provenance := CreateSLSAProvenance(
			"https://github.com/actions/runner",
			"https://github.com/slsa-framework/slsa-github-generator",
			[]Material{
				{
					URI: "git+https://github.com/example/repo",
					Digest: map[string]string{
						"sha1": "abc123",
					},
				},
			},
		)

		// Attach attestation
		attestOpts := AttestationOptions{
			PredicateType: PredicateSLSAProvenance,
			Predicate:     provenance,
			Annotations: map[string]string{
				"attestation.type": "slsa-provenance",
			},
		}

		attestDesc, err := client.Attest(ctx, desc.Ref, attestOpts)
		require.NoError(t, err)
		assert.NotEmpty(t, attestDesc.Digest)
		assert.Contains(t, attestDesc.Ref, ".att")
	})

	t.Run("QueryAttestations", func(t *testing.T) {
		// Query attestations
		report, err := client.VerifyAttestations(ctx, desc.Ref, PredicateSLSAProvenance)
		require.NoError(t, err)
		
		// Should have at least one attestation
		assert.GreaterOrEqual(t, len(report.Attestations), 1)
		
		if len(report.Attestations) > 0 {
			att := report.Attestations[0]
			assert.Equal(t, PredicateSLSAProvenance, att.Statement.PredicateType)
			assert.True(t, att.Verified) // Mock verification always passes
			assert.NotNil(t, att.SignerIdentity)
		}
	})

	t.Run("FilterByPredicateType", func(t *testing.T) {
		// Attach a different type of attestation
		customOpts := AttestationOptions{
			PredicateType: PredicateCustom,
			Predicate: map[string]interface{}{
				"custom": "test-data",
			},
		}

		_, err := client.Attest(ctx, desc.Ref, customOpts)
		require.NoError(t, err)

		// Query only SLSA attestations
		report, err := client.VerifyAttestations(ctx, desc.Ref, PredicateSLSAProvenance)
		require.NoError(t, err)

		// Should only have SLSA attestations
		for _, att := range report.Attestations {
			assert.Equal(t, PredicateSLSAProvenance, att.Statement.PredicateType)
		}

		// Query all attestations
		allReport, err := client.VerifyAttestations(ctx, desc.Ref, "")
		require.NoError(t, err)

		// Should have more attestations
		assert.GreaterOrEqual(t, len(allReport.Attestations), len(report.Attestations))
	})
}

func TestAttestationReport(t *testing.T) {
	t.Parallel()

	report := &AttestationReport{
		Subject: Descriptor{
			Ref:    "test/image:v1@sha256:abc123",
			Digest: "sha256:abc123",
		},
		Attestations: []AttestationEntry{
			{
				Statement: IntotoStatement{
					PredicateType: PredicateSLSAProvenance,
				},
				Verified: true,
				SignerIdentity: &SignerIdentity{
					Issuer:  "test-issuer",
					Subject: "test-subject",
				},
			},
			{
				Statement: IntotoStatement{
					PredicateType: PredicateCustom,
				},
				Verified: false,
			},
		},
		Verified: false,
		Errors: []error{
			fmt.Errorf("attestation 1: signature verification failed"),
		},
	}

	// Check report properties
	// Subject is stored as interface{}, should be a Descriptor
	if subjectDesc, ok := report.Subject.(Descriptor); ok {
		assert.Equal(t, "sha256:abc123", subjectDesc.Digest)
	}
	assert.Len(t, report.Attestations, 2)
	assert.False(t, report.Verified)
	assert.Len(t, report.Errors, 1)

	// Check first attestation
	att1 := report.Attestations[0]
	assert.Equal(t, PredicateSLSAProvenance, att1.Statement.PredicateType)
	assert.True(t, att1.Verified)
	assert.NotNil(t, att1.SignerIdentity)
	assert.Equal(t, "test-issuer", att1.SignerIdentity.Issuer)

	// Check second attestation
	att2 := report.Attestations[1]
	assert.Equal(t, PredicateCustom, att2.Statement.PredicateType)
	assert.False(t, att2.Verified)
	assert.Nil(t, att2.SignerIdentity)
}

func TestSubjectDigestHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		subjects []Subject
		want     string
	}{
		{
			name: "single_subject",
			subjects: []Subject{
				{
					Name: "test/image:v1",
					Digest: map[string]string{
						"sha256": "abc123",
					},
				},
			},
			want: "abc123",
		},
		{
			name: "multiple_digests",
			subjects: []Subject{
				{
					Name: "test/image:v1",
					Digest: map[string]string{
						"sha256": "abc123",
						"sha512": "def456",
					},
				},
			},
			want: "abc123",
		},
		{
			name: "multiple_subjects",
			subjects: []Subject{
				{
					Name: "test/image1:v1",
					Digest: map[string]string{
						"sha256": "abc123",
					},
				},
				{
					Name: "test/image2:v1",
					Digest: map[string]string{
						"sha256": "def456",
					},
				},
			},
			want: "abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statement := IntotoStatement{
				Type:          "https://in-toto.io/Statement/v0.1",
				PredicateType: PredicateSLSAProvenance,
				Subject:       tt.subjects,
				Predicate:     map[string]interface{}{},
			}

			// Check first subject's sha256
			if len(statement.Subject) > 0 {
				sha256Digest := statement.Subject[0].Digest["sha256"]
				assert.Equal(t, tt.want, sha256Digest)
			}
		})
	}
}

func TestMockDSSESignature(t *testing.T) {
	t.Parallel()

	cl, err := New(ClientOptions{
		Cosign: CosignOpts{
			Enable: false, // Use mock signing
		},
	})
	require.NoError(t, err)

	ctx := context.Background()

	envelope := DSSEEnvelope{
		PayloadType: "application/vnd.in-toto+json",
		Payload:     base64.StdEncoding.EncodeToString([]byte(`{"test": "data"}`)),
	}

	// Sign with mock
	c := cl.(*client)
	signed, err := c.signDSSE(ctx, envelope, nil)
	require.NoError(t, err)

	// Should have a signature
	assert.Len(t, signed.Signatures, 1)
	assert.NotEmpty(t, signed.Signatures[0].Sig)
	assert.NotEmpty(t, signed.Signatures[0].KeyID)

	// Verify with mock
	verified, identity, err := c.verifyDSSE(ctx, signed)
	require.NoError(t, err)
	assert.True(t, verified)
	assert.NotNil(t, identity)
	assert.NotEmpty(t, identity.Issuer)
	assert.NotEmpty(t, identity.Subject)
}