package ociv2

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignDigest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		ref         string
		opts        ClientOptions
		setupEnv    func()
		cleanupEnv  func()
		wantErr     bool
		checkResult func(t *testing.T, desc Descriptor)
	}{
		{
			name: "ok/signing_disabled",
			ref:  "registry.example.com/repo/test@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			opts: ClientOptions{
				Cosign: CosignOpts{
					Enable: false,
				},
			},
			wantErr: false,
			checkResult: func(t *testing.T, desc Descriptor) {
				assert.Equal(t, "registry.example.com/repo/test@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", desc.Ref, "ref should be normalized when signing disabled")
			},
		},
		{
			name: "ok/insecure_mode",
			ref:  "registry.example.com/repo/test@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			opts: ClientOptions{
				Cosign: CosignOpts{
					Enable:        true,
					AllowInsecure: true,
				},
			},
			wantErr: false,
			checkResult: func(t *testing.T, desc Descriptor) {
				if desc.Ref == "" {
					t.Error("SignDigest() returned empty ref")
				}
			},
		},
		{
			name: "keyless mode simulation",
			ref:  "registry.example.com/repo/test@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			opts: ClientOptions{
				Cosign: CosignOpts{
					Enable:        true,
					AllowInsecure: true,
				},
			},
			setupEnv: func() {
				os.Setenv("COSIGN_EXPERIMENTAL", "1")
			},
			cleanupEnv: func() {
				os.Unsetenv("COSIGN_EXPERIMENTAL")
			},
			wantErr: false,
			checkResult: func(t *testing.T, desc Descriptor) {
				if desc.MediaType != "application/vnd.dev.cosign.signature.v1+json" {
					t.Errorf("SignDigest() media type = %v, want signature media type", desc.MediaType)
				}
			},
		},
		{
			name: "error/invalid_reference",
			ref:  "http://insecure.com/image",
			opts: ClientOptions{
				Cosign: CosignOpts{
					Enable: true,
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range var
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupEnv != nil {
				tc.setupEnv()
			}
			if tc.cleanupEnv != nil {
				defer tc.cleanupEnv()
			}

			client, err := New(tc.opts)
			require.NoError(t, err, "Failed to create client for test=%s", tc.name)

			ctx := context.Background()
			desc, err := client.SignDigest(ctx, tc.ref)

			if tc.wantErr {
				require.Error(t, err, "expected error for test=%s ref=%s", tc.name, tc.ref)
				return
			}
			require.NoError(t, err, "unexpected error for test=%s ref=%s", tc.name, tc.ref)

			if tc.checkResult != nil {
				tc.checkResult(t, desc)
			}
		})
	}
}

func TestVerifyDigest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		ref         string
		opts        ClientOptions
		setupEnv    func()
		cleanupEnv  func()
		wantErr     bool
		checkReport func(t *testing.T, report *VerificationReport)
	}{
		{
			name: "ok/verification_disabled",
			ref:  "registry.example.com/repo/test@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			opts: ClientOptions{
				Cosign: CosignOpts{
					Enable: false,
				},
			},
			wantErr: false,
			checkReport: func(t *testing.T, report *VerificationReport) {
				assert.False(t, report.Signed, "should not be signed when cosign disabled")
				assert.NotEmpty(t, report.Errors, "should include error when disabled")
			},
		},
		{
			name: "ok/insecure_mode",
			ref:  "registry.example.com/repo/test@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			opts: ClientOptions{
				Cosign: CosignOpts{
					Enable:        true,
					AllowInsecure: true,
				},
			},
			wantErr: false,
			checkReport: func(t *testing.T, report *VerificationReport) {
				assert.NotEmpty(t, report.Digest, "digest should not be empty")
			},
		},
		{
			name: "ok/with_identity_requirements",
			ref:  "registry.example.com/repo/test@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			opts: ClientOptions{
				Cosign: CosignOpts{
					Enable:        true,
					AllowInsecure: true,
					Identity: &OIDCIdentity{
						Issuer:  "https://token.actions.githubusercontent.com",
						Subject: "repo:example/repo:ref:refs/heads/main",
					},
				},
			},
			wantErr: false,
			checkReport: func(t *testing.T, report *VerificationReport) {
				assert.Equal(t, "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", report.Digest, "digest should match expected value")
			},
		},
		{
			name: "error/invalid_reference",
			ref:  "http://insecure.com/image",
			opts: ClientOptions{
				Cosign: CosignOpts{
					Enable: true,
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range var
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupEnv != nil {
				tc.setupEnv()
			}
			if tc.cleanupEnv != nil {
				defer tc.cleanupEnv()
			}

			client, err := New(tc.opts)
			require.NoError(t, err, "Failed to create client for test=%s", tc.name)

			ctx := context.Background()
			report, err := client.VerifyDigest(ctx, tc.ref)

			if tc.wantErr {
				require.Error(t, err, "expected error for test=%s ref=%s", tc.name, tc.ref)
				return
			}
			require.NoError(t, err, "unexpected error for test=%s ref=%s", tc.name, tc.ref)

			if tc.checkReport != nil {
				tc.checkReport(t, report)
			}
		})
	}
}

func TestVerificationReport(t *testing.T) {
	t.Parallel()

	report := &VerificationReport{
		Digest: "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		Signed: true,
		Signers: []SignerIdentity{
			{
				Issuer:  "https://token.actions.githubusercontent.com",
				Subject: "repo:example/repo:ref:refs/heads/main",
				SANs:    []string{"example@github.com"},
				Time:    time.Now(),
			},
		},
		BundleVerified: true,
		Errors:         []string{},
	}

	assert.Equal(t, "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", report.Digest, "digest should match")
	assert.True(t, report.Signed, "report should show as signed")
	require.Len(t, report.Signers, 1, "should have exactly one signer")
	assert.True(t, report.BundleVerified, "bundle should be verified")

	signer := report.Signers[0]
	assert.Equal(t, "https://token.actions.githubusercontent.com", signer.Issuer, "signer issuer should match GitHub Actions")
	assert.Equal(t, "repo:example/repo:ref:refs/heads/main", signer.Subject, "signer subject should match repo reference")
	assert.Empty(t, report.Errors, "report should have no errors")
}