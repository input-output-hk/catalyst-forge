package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
	"github.com/stretchr/testify/require"
)

type mockVerifier struct {
	subj *GHASubject
	err  error
}

func (m mockVerifier) Verify(ctx context.Context, idToken string) (*GHASubject, error) {
	return m.subj, m.err
}

type mockPolicyStore struct {
	policies []*store.GithubPolicy
	err      error
}

func (m mockPolicyStore) Create(ctx context.Context, p *store.GithubPolicy) error { return nil }
func (m mockPolicyStore) Update(ctx context.Context, p *store.GithubPolicy) error { return nil }
func (m mockPolicyStore) Delete(ctx context.Context, id uuid.UUID) error          { return nil }
func (m mockPolicyStore) GetByID(ctx context.Context, id uuid.UUID) (*store.GithubPolicy, error) {
	return nil, nil
}
func (m mockPolicyStore) List(ctx context.Context) ([]*store.GithubPolicy, error) { return nil, nil }
func (m mockPolicyStore) LookupByRepository(ctx context.Context, repository string) ([]*store.GithubPolicy, error) {
	return m.policies, m.err
}

type mockTokens struct {
	token string
	err   error
}

func (m mockTokens) SignAccess(ctx context.Context, claims AccessClaims) (string, error) {
	return m.token, m.err
}
func (m mockTokens) ParseAccess(ctx context.Context, token string) (*AccessClaims, error) {
	return nil, nil
}
func (m mockTokens) JWKS() interface{} { return nil }

func TestGHAExchangeService_Success(t *testing.T) {
	t.Parallel()
	svc := &ghaExchangeService{
		cfg:      GHAConfig{Enabled: true, ExchangeTTL: time.Minute},
		verifier: mockVerifier{subj: &GHASubject{Repository: "org/repo"}},
		policies: mockPolicyStore{policies: []*store.GithubPolicy{{Repository: "org/repo", Roles: []string{"deployer", "deployer", "reader"}, Enabled: true}}},
		tokens:   mockTokens{token: "abc"},
	}
	tok, exp, subj, err := svc.Exchange(context.Background(), "idtoken")
	require.NoError(t, err)
	require.Equal(t, "abc", tok)
	require.Equal(t, 60, exp)
	require.NotNil(t, subj)
}

func TestGHAExchangeService_Disabled(t *testing.T) {
	t.Parallel()
	svc := &ghaExchangeService{cfg: GHAConfig{Enabled: false}}
	_, _, _, err := svc.Exchange(context.Background(), "idtoken")
	require.Error(t, err)
}

func TestGHAExchangeService_VerifyError(t *testing.T) {
	t.Parallel()
	svc := &ghaExchangeService{cfg: GHAConfig{Enabled: true}, verifier: mockVerifier{err: errors.New("bad token")}}
	_, _, _, err := svc.Exchange(context.Background(), "idtoken")
	require.Error(t, err)
}
