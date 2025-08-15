package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
)

// AutoAuthTransport injects Authorization and handles 401 -> refresh -> retry.
type AutoAuthTransport struct {
	Inner          http.RoundTripper
	Tokens         *TokenStore
	BaseURL        string
	CSRFHeaderName string // default: X-CSRF-Token
	refreshPath    string // default: /api/v1/auth/refresh
	csrfCookieName string // default set by server; if empty, we try typical names
	Jar            http.CookieJar

	rf *singleflight
}

// NewAutoAuthTransport creates a transport with sane defaults.
func NewAutoAuthTransport(inner http.RoundTripper, tokens *TokenStore, baseURL string) *AutoAuthTransport {
	if inner == nil {
		inner = http.DefaultTransport
	}
	if tokens == nil {
		tokens = NewTokenStore()
	}
	return &AutoAuthTransport{
		Inner:          inner,
		Tokens:         tokens,
		BaseURL:        baseURL,
		CSRFHeaderName: "X-CSRF-Token",
		refreshPath:    "/api/v1/auth/refresh",
		csrfCookieName: "__Host-csrf_token",
	}
}

func (t *AutoAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Attach bearer if present
	if tok := t.Tokens.Get(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	// Execute
	resp, err := t.Inner.RoundTrip(req)
	if err != nil || resp == nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}
	// Skip if this is the refresh call to avoid loops
	if req.URL.Path == t.refreshPath {
		return resp, err
	}
	// Try refresh
	if rerr := t.refresh(req.Context(), req); rerr != nil {
		return resp, err // return original 401
	}
	// Retry once if body is replayable
	var body io.ReadCloser
	if req.GetBody != nil {
		if rc, gerr := req.GetBody(); gerr == nil {
			body = rc
		}
	}
	// Clone request
	r2 := req.Clone(req.Context())
	if body != nil {
		r2.Body = body
	}
	if tok := t.Tokens.Get(); tok != "" {
		r2.Header.Set("Authorization", "Bearer "+tok)
	}
	return t.Inner.RoundTrip(r2)
}

func (t *AutoAuthTransport) refresh(ctx context.Context, req *http.Request) error {
	key := "global"
	if t.rf == nil {
		t.rf = &singleflight{}
	}
	_, err := t.rf.Do(key, func() (any, error) {
		// Build refresh request
		r, _ := http.NewRequestWithContext(ctx, http.MethodPost, t.BaseURL+t.refreshPath, http.NoBody)
		// Attach CSRF header from cookie jar if available
		if t.Jar != nil {
			for _, c := range t.Jar.Cookies(r.URL) {
				if c.Name == t.csrfCookieName && c.Value != "" {
					r.Header.Set(t.CSRFHeaderName, c.Value)
					break
				}
			}
		}

		// Use a one-off client with the same inner transport and jar so cookies are sent and updated
		hc := &http.Client{Transport: t.Inner, Jar: t.Jar}
		resp, err := hc.Do(r)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, errors.New("refresh failed")
		}

		// Extract new access token from body if API returns it; expects JSON {"access_token":"..."}
		var payload struct {
			AccessToken string `json:"access_token"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		if payload.AccessToken != "" {
			t.Tokens.Set(payload.AccessToken)
		}
		return nil, nil
	})
	return err
}

// singleflight is a tiny single-flight impl for one key.
type singleflight struct {
	mu sync.Mutex
	ch chan struct{}
}

func (s *singleflight) Do(_ string, fn func() (any, error)) (any, error) {
	s.mu.Lock()
	if s.ch != nil {
		ch := s.ch
		s.mu.Unlock()
		<-ch
		return nil, nil
	}
	s.ch = make(chan struct{})
	s.mu.Unlock()
	var v any
	var err error
	v, err = fn()
	s.mu.Lock()
	close(s.ch)
	s.ch = nil
	s.mu.Unlock()
	return v, err
}
