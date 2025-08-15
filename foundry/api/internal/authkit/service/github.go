package service

import (
	"context"
	"errors"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/store"
)

var (
	ErrNotImplemented = errors.New("not implemented")
)

// GHAVerifier validates GitHub OIDC ID tokens and extracts subject claims.
type GHAVerifier interface {
	Verify(ctx context.Context, idToken string) (*GHASubject, error)
}

// GHASubject represents the CI service identity extracted from OIDC claims.
type GHASubject struct {
	Repository  string
	Ref         string
	WorkflowRef string
	SHA         string
	RunID       string
}

// GHAExchangeService handles exchange from OIDC ID token → short-lived access token.
type GHAExchangeService interface {
	Exchange(ctx context.Context, idToken string) (accessToken string, expiresIn int, subj *GHASubject, err error)
}

type ghaExchangeService struct {
	cfg      GHAConfig
	verifier GHAVerifier
	policies store.GithubPolicyStore
	tokens   TokenService
}

// GHAConfig contains the subset of configuration needed for exchange.
type GHAConfig struct {
	Enabled     bool
	ExchangeTTL time.Duration
}

func NewGHAExchangeService(cfg GHAConfig, verifier GHAVerifier, policies store.GithubPolicyStore, tokens TokenService) GHAExchangeService {
	return &ghaExchangeService{cfg: cfg, verifier: verifier, policies: policies, tokens: tokens}
}

func (s *ghaExchangeService) Exchange(ctx context.Context, idToken string) (string, int, *GHASubject, error) {
	if !s.cfg.Enabled {
		return "", 0, nil, ErrNotImplemented
	}
	// Verify incoming ID token and extract subject
	subj, err := s.verifier.Verify(ctx, idToken)
	if err != nil {
		return "", 0, nil, err
	}
	// Resolve roles from policies for repository; simplistic merge for now
	policies, err := s.policies.LookupByRepository(ctx, subj.Repository)
	if err != nil {
		return "", 0, nil, err
	}
	roleSet := map[string]struct{}{}
	for _, p := range policies {
		if !p.Enabled {
			continue
		}
		for _, r := range p.Roles {
			roleSet[r] = struct{}{}
		}
	}
	roles := make([]string, 0, len(roleSet))
	for r := range roleSet {
		roles = append(roles, r)
	}
	// Mint short-lived access token
	expires := s.cfg.ExchangeTTL
	if expires <= 0 {
		expires = 15 * time.Minute
	}
	claims := AccessClaims{Sub: subj.Repository, Email: "", Roles: roles, SessionVersion: 0}
	access, err := s.tokens.SignAccess(ctx, claims)
	if err != nil {
		return "", 0, nil, err
	}
	return access, int(expires.Seconds()), subj, nil
}
