package github_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth/github"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth/github/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

func TestDefaultGithubActionsOIDCClient_Verify(t *testing.T) {
	baseAud := "expected-aud"

	tests := []struct {
		name         string
		audience     string
		tokenMutator func(*TestArtifacts)
		jwksMutator  func(*TestArtifacts)
		expectErr    bool
		errContains  string
	}{
		{
			name:     "happy-path",
			audience: baseAud,
		},
		{
			name:         "empty-token",
			audience:     baseAud,
			tokenMutator: func(ta *TestArtifacts) { ta.Token = "" },
			expectErr:    true,
			errContains:  "empty token",
		},
		{
			name:        "wrong audience",
			audience:    "other-aud",
			expectErr:   true,
			errContains: "audience",
		},
		{
			name:     "jwks cache empty",
			audience: baseAud,
			jwksMutator: func(ta *TestArtifacts) {
				ta.JWKS.Keys = nil
			},
			expectErr:   true,
			errContains: "jwks cache is empty",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			art := NewTestArtifacts(t, baseAud)

			if tc.tokenMutator != nil {
				tc.tokenMutator(art)
			}
			if tc.jwksMutator != nil {
				tc.jwksMutator(art)
			}

			cacher := newJWKSReturningCacher(art.JWKS)

			client, err := github.NewDefaultGithubActionsOIDCClient(context.Background(), "", github.WithCacher(cacher))
			require.NoError(t, err)

			ti, err := client.Verify(art.Token, tc.audience)

			if tc.expectErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				// should not return a TokenInfo on failure
				assert.Nil(t, ti)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, ti)

			// spot-check a few important fields
			assert.Equal(t, art.Claims.Subject, ti.Subject)
			assert.Equal(t, art.Claims.Repository, ti.Repository)
			assert.Equal(t, art.Claims.RunID, ti.RunID)
			assert.ElementsMatch(t, art.Claims.Audience, ti.Aud)
		})
	}
}

type TestArtifacts struct {
	Token  string
	JWKS   *jose.JSONWebKeySet
	Claims github.GitHubActionsTokenClaims
	KID    string
}

// NewTestArtifacts creates a fresh key-pair, JWKS and signed token.
func NewTestArtifacts(tb testing.TB, aud string) *TestArtifacts {
	tb.Helper()

	// 1. generate key-pair
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		tb.Fatalf("generate key: %v", err)
	}

	kid := uuid.NewString()

	// 2. build the JWKS containing the public key
	pubJWK := jose.JSONWebKey{
		Key:       &privKey.PublicKey,
		KeyID:     kid,
		Algorithm: string(jose.ES256),
		Use:       "sig",
	}
	jwks := &jose.JSONWebKeySet{Keys: []jose.JSONWebKey{pubJWK}}

	// 3. create a signer using the private key
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.ES256, Key: privKey},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", kid),
	)
	if err != nil {
		tb.Fatalf("new signer: %v", err)
	}

	// 4. craft realistic claims
	now := time.Now().UTC()
	claims := github.GitHubActionsTokenClaims{
		Claims: jwt.Claims{
			Issuer:   "https://token.actions.githubusercontent.com",
			Subject:  "repo:owner/repo:environment:Production",
			Audience: jwt.Audience{aud},
			IssuedAt: jwt.NewNumericDate(now),
			Expiry:   jwt.NewNumericDate(now.Add(10 * time.Minute)),
		},
		Repository:        "owner/repo",
		RepositoryID:      "123456",
		RepositoryOwner:   "owner",
		RepositoryOwnerID: "654321",
		Ref:               "refs/heads/main",
		SHA:               "deadbeefcafebabefeedface0123456789abcd",
		Workflow:          "CI",
		JobWorkflowRef:    "owner/repo/.github/workflows/ci.yml@refs/heads/main",
		RunID:             "424242",
		RunnerEnvironment: "github-hosted",
		Environment:       "Production",
	}

	token, err := jwt.Signed(signer).Claims(claims).CompactSerialize()
	if err != nil {
		tb.Fatalf("sign token: %v", err)
	}

	return &TestArtifacts{
		Token:  token,
		JWKS:   jwks,
		Claims: claims,
		KID:    kid,
	}
}

// newJWKSReturningCacher creates a mock cacher that returns the given JWKS.
func newJWKSReturningCacher(jwks *jose.JSONWebKeySet) *mocks.GitHubJWKSCacherMock {
	return &mocks.GitHubJWKSCacherMock{
		JWKSFunc: func() *jose.JSONWebKeySet {
			return jwks
		},
		StartFunc: func(_ context.Context) error {
			return nil
		},
		StopFunc: func() {
		},
	}
}
