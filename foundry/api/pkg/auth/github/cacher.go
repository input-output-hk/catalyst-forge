package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"

	"gopkg.in/square/go-jose.v2"
)

const githubJWKSURL = "https://token.actions.githubusercontent.com/.well-known/jwks"

//go:generate go run github.com/matryer/moq@latest -skip-ensure --pkg mocks --out ./mocks/cacher.go . GitHubJWKSCacher

// JWKSCacher is an interface that provides a way to cache and retrieve JWKS.
type GitHubJWKSCacher interface {
	JWKS() *jose.JSONWebKeySet
	Start(context.Context) error
	Stop()
}

// GitHubJWKSCacher is a JWKSCacher that caches JWKS from the GitHub Actions endpoint.
type DefaultGitHubJWKSCacher struct {
	cachePath string
	ttl       time.Duration
	client    *http.Client
	fs        fs.Filesystem
	jwksURL   string

	mu   sync.RWMutex
	jwks *jose.JSONWebKeySet
	wg   sync.WaitGroup
	ctx  context.Context
	stop context.CancelFunc
}

// GitHubJWKSCacherOption is a function that can be used to configure the GitHubJWKSCacher.
type GitHubJWKSCacherOption func(*DefaultGitHubJWKSCacher)

// WithClient sets the http client to use for the GitHubJWKSCacher.
func WithClient(client *http.Client) GitHubJWKSCacherOption {
	return func(g *DefaultGitHubJWKSCacher) {
		g.client = client
	}
}

// WithFS sets the file system to use for the GitHubJWKSCacher.
func WithFS(fs fs.Filesystem) GitHubJWKSCacherOption {
	return func(g *DefaultGitHubJWKSCacher) {
		g.fs = fs
	}
}

// WithJWKSURL sets the URL to use for the GitHubJWKSCacher.
func WithJWKSURL(jwksURL string) GitHubJWKSCacherOption {
	return func(g *DefaultGitHubJWKSCacher) {
		g.jwksURL = jwksURL
	}
}

// WithTTL sets the TTL for the GitHubJWKSCacher.
func WithTTL(ttl time.Duration) GitHubJWKSCacherOption {
	return func(g *DefaultGitHubJWKSCacher) {
		g.ttl = ttl
	}
}

// Start loads the initial JWKS (from disk or the network) and kicks off the
// refresh loop. It returns an error if *no* valid JWKS can be obtained.
func (g *DefaultGitHubJWKSCacher) Start(parent context.Context) error {
	g.ctx, g.stop = context.WithCancel(parent)

	if err := g.loadFromDisk(); err != nil {
		if err := g.refresh(); err != nil {
			return fmt.Errorf("jwks cacher startup failed: %w", err)
		}
	}

	g.wg.Add(1)
	go g.refresher()

	return nil
}

// Stop signals the goroutine to exit and waits for it to finish.
func (g *DefaultGitHubJWKSCacher) Stop() {
	if g.stop != nil {
		g.stop()
	}
	g.wg.Wait()
}

// JWKS returns the current cached key set (read‑only copy).
func (g *DefaultGitHubJWKSCacher) JWKS() *jose.JSONWebKeySet {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.jwks
}

// refresher polls on a ticker until the context is cancelled.
func (g *DefaultGitHubJWKSCacher) refresher() {
	defer g.wg.Done()

	ticker := time.NewTicker(g.ttl)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = g.refresh() // log inside on failure, keep going
		case <-g.ctx.Done():
			return
		}
	}
}

// refresh downloads the JWKS and, if it parses, stores it to disk + memory.
func (g *DefaultGitHubJWKSCacher) refresh() error {
	req, _ := http.NewRequestWithContext(g.ctx, http.MethodGet, g.jwksURL, nil)
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}

	var ks jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&ks); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}
	if len(ks.Keys) == 0 {
		return fmt.Errorf("jwks empty")
	}

	// Write to disk (best‑effort)
	if data, _ := json.Marshal(&ks); len(data) > 0 {
		_ = g.fs.WriteFile(g.cachePath, data, 0o644)
	}

	g.mu.Lock()
	g.jwks = &ks
	g.mu.Unlock()
	return nil
}

// loadFromDisk attempts to populate g.jwks from the cache file.
func (g *DefaultGitHubJWKSCacher) loadFromDisk() error {
	data, err := g.fs.ReadFile(g.cachePath)
	if err != nil {
		return err
	}
	var ks jose.JSONWebKeySet
	if err := json.Unmarshal(data, &ks); err != nil {
		fmt.Printf("error unmarshalling jwks: %v\n", err)
		return err
	}

	if len(ks.Keys) == 0 {
		return fmt.Errorf("jwks on disk had zero keys")
	}
	g.mu.Lock()
	g.jwks = &ks
	g.mu.Unlock()
	return nil
}

// NewGitHubJWKSCacher creates a new GitHubJWKSCacher.
func NewDefaultGitHubJWKSCacher(
	ctx context.Context,
	cachePath string,
	opts ...GitHubJWKSCacherOption,
) *DefaultGitHubJWKSCacher {
	c := &DefaultGitHubJWKSCacher{
		cachePath: cachePath,
		ttl:       10 * time.Minute,
		client:    &http.Client{Timeout: 30 * time.Second},
		fs:        billy.NewBaseOsFS(),
		jwksURL:   githubJWKSURL,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}
