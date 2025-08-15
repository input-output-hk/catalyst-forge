package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	jose "gopkg.in/square/go-jose.v2"
	josejwt "gopkg.in/square/go-jose.v2/jwt"
)

// GHAVerifierConfig controls verification behavior for GitHub OIDC ID tokens.
type GHAVerifierConfig struct {
	Issuer       string
	Audiences    []string
	JWKSCacheTTL time.Duration
}

// NewGHAVerifier returns a GHAVerifier that validates tokens using the issuer JWKS.
func NewGHAVerifier(cfg GHAVerifierConfig) GHAVerifier {
	if cfg.JWKSCacheTTL <= 0 {
		cfg.JWKSCacheTTL = 15 * time.Minute
	}
	return &ghaVerifier{
		cfg:   cfg,
		http:  &http.Client{Timeout: 10 * time.Second},
		cache: &jwksCache{ttl: cfg.JWKSCacheTTL},
	}
}

type ghaVerifier struct {
	cfg   GHAVerifierConfig
	http  *http.Client
	cache *jwksCache
}

type jwksCache struct {
	mu        sync.RWMutex
	keys      *jose.JSONWebKeySet
	fetchedAt time.Time
	ttl       time.Duration
}

func (c *jwksCache) get() (*jose.JSONWebKeySet, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.keys == nil {
		return nil, false
	}
	if time.Since(c.fetchedAt) > c.ttl {
		return nil, false
	}
	return c.keys, true
}

func (c *jwksCache) set(keys *jose.JSONWebKeySet) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys = keys
	c.fetchedAt = time.Now()
}

func (v *ghaVerifier) fetchJWKS(ctx context.Context) (*jose.JSONWebKeySet, error) {
	if ks, ok := v.cache.get(); ok {
		return ks, nil
	}
	// GitHub JWKS lives at <issuer>/.well-known/jwks
	url := strings.TrimRight(v.cfg.Issuer, "/") + "/.well-known/jwks"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := v.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks fetch failed: %s", resp.Status)
	}
	var ks jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&ks); err != nil {
		return nil, err
	}
	v.cache.set(&ks)
	return &ks, nil
}

type ghaAllClaims struct {
	josejwt.Claims
	Repository     string `json:"repository"`
	Ref            string `json:"ref"`
	JobWorkflowRef string `json:"job_workflow_ref"`
	Sha            string `json:"sha"`
	RunID          string `json:"run_id"`
	Environment    string `json:"environment,omitempty"`
}

func (v *ghaVerifier) Verify(ctx context.Context, idToken string) (*GHASubject, error) {
	if idToken == "" {
		return nil, errors.New("empty token")
	}

	parsed, err := josejwt.ParseSigned(idToken)
	if err != nil {
		return nil, err
	}
	if len(parsed.Headers) == 0 {
		return nil, errors.New("missing header")
	}
	kid := parsed.Headers[0].KeyID

	ks, err := v.fetchJWKS(ctx)
	if err != nil {
		return nil, err
	}
	keys := ks.Key(kid)
	if len(keys) == 0 {
		if _, ok := v.cache.get(); ok {
			v.cache.set(nil)
			ks, err = v.fetchJWKS(ctx)
			if err != nil {
				return nil, err
			}
			keys = ks.Key(kid)
		}
	}
	if len(keys) == 0 {
		return nil, errors.New("signing key not found")
	}
	var claims ghaAllClaims
	if err := parsed.Claims(keys[0].Key, &claims); err != nil {
		return nil, err
	}
	expected := josejwt.Expected{Issuer: v.cfg.Issuer, Time: time.Now()}
	if len(v.cfg.Audiences) > 0 {
		expected.Audience = josejwt.Audience(v.cfg.Audiences)
	}
	if err := claims.Validate(expected); err != nil {
		return nil, err
	}
	subj := &GHASubject{Repository: claims.Repository, Ref: claims.Ref, WorkflowRef: claims.JobWorkflowRef, SHA: claims.Sha, RunID: claims.RunID}
	return subj, nil
}
