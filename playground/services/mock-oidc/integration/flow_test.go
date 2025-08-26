package integration

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	cfgpkg "github.com/input-output-hk/catalyst-forge/playground/services/mock-oidc/config"
	"github.com/input-output-hk/catalyst-forge/playground/services/mock-oidc/server"
)

func httpGetJSON[T any](t *testing.T, url string, out *T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s: status %d: %s", url, resp.StatusCode, string(b))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

func TestDiscoveryAndJWKS(t *testing.T) {
	ts, baseURL := startHermetic(t)
	defer ts.Close()

	var disc map[string]any
	httpGetJSON(t, baseURL+"/default/.well-known/openid-configuration", &disc)
	if disc["issuer"].(string) != baseURL+"/default" {
		t.Fatalf("unexpected issuer: %v", disc["issuer"])
	}
	var jwks map[string]any
	httpGetJSON(t, baseURL+"/default/jwks", &jwks)
	kv := jwks["keys"].([]any)
	if len(kv) == 0 {
		t.Fatalf("no jwk keys")
	}
}

// Previously named TestAuthCodeFlowPlainPKCE; actually tests the S256 flow.
func TestAuthCodeFlowS256(t *testing.T) {
	ts, baseURL := startHermetic(t)
	defer ts.Close()
	// Generate PKCE verifier with valid length (43..128)
	verifier := strings.Repeat("a", 64)
	// Standard: use S256 challenge
	h := sha256.Sum256([]byte(verifier))
	s256 := base64.RawURLEncoding.EncodeToString(h[:])

	authURL := fmt.Sprintf(
		"%s/default/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=openid%%20email%%20profile&state=state12345678&nonce=nonce12345678&code_challenge=%s&code_challenge_method=S256",
		baseURL,
		url.QueryEscape("kratos-client"),
		url.QueryEscape("http://localhost:18080/callback"),
		s256,
	)

	// We expect a 302 with Location containing code=... (do not follow redirects)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(authURL)
	if err != nil {
		t.Fatalf("authorize request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("authorize status: %d body: %s", resp.StatusCode, string(b))
	}
	loc := resp.Header.Get("Location")
	if loc == "" {
		t.Fatalf("missing Location header")
	}
	lu, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}
	code := lu.Query().Get("code")
	if code == "" {
		t.Fatalf("missing code in redirect: %s", loc)
	}

	// Exchange code for tokens for public client (no client auth)
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "http://localhost:18080/callback")
	form.Set("code_verifier", verifier)
	form.Set("client_id", "kratos-client")

	req, err := http.NewRequest(http.MethodPost, baseURL+"/default/token", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	tr, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("token request: %v", err)
	}
	defer tr.Body.Close()
	if tr.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(tr.Body)
		t.Fatalf("token status: %d body: %s", tr.StatusCode, string(b))
	}
	var tok struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(tr.Body).Decode(&tok); err != nil {
		t.Fatalf("decode token: %v", err)
	}
	if tok.AccessToken == "" {
		t.Fatalf("missing access_token: %+v", tok)
	}

	// Call userinfo
	req2, _ := http.NewRequest(http.MethodGet, baseURL+"/default/userinfo", nil)
	req2.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	ui, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("userinfo request: %v", err)
	}
	defer ui.Body.Close()
	if ui.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(ui.Body)
		t.Fatalf("userinfo status: %d body: %s", ui.StatusCode, string(b))
	}
	var claims map[string]any
	if err := json.NewDecoder(ui.Body).Decode(&claims); err != nil {
		t.Fatalf("decode userinfo: %v", err)
	}
	if claims["sub"] == nil || claims["email"] == nil {
		t.Fatalf("missing claims: %+v", claims)
	}
}

func TestAuthorize_PlainPKCE_Disabled(t *testing.T) {
	ts, baseURL := startHermetic(t)
	defer ts.Close()
	// verifier and challenge are identical for plain
	verifier := strings.Repeat("p", 64)
	authURL := fmt.Sprintf(
		"%s/default/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=openid&state=statePLAIN0&nonce=noncePLAIN0&code_challenge=%s&code_challenge_method=plain",
		baseURL,
		url.QueryEscape("kratos-client"),
		url.QueryEscape("http://localhost:18080/callback"),
		url.QueryEscape(verifier),
	)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(authURL)
	if err != nil {
		t.Fatalf("authorize request: %v", err)
	}
	defer resp.Body.Close()
	// Expect redirect with OAuth error in Location query
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 302/303, got %d body=%s", resp.StatusCode, string(b))
	}
	loc := resp.Header.Get("Location")
	if loc == "" {
		t.Fatalf("missing Location header")
	}
	lu, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}
	if got := lu.Query().Get("error"); strings.ToLower(got) != "invalid_request" {
		t.Fatalf("expected error=invalid_request in redirect, got %q", got)
	}
}

func TestAuthCodeFlow_PlainPKCE_Enabled(t *testing.T) {
	ts, baseURL := startHermeticPlainEnabled(t)
	defer ts.Close()
	verifier := strings.Repeat("q", 64)
	// For plain, challenge = verifier
	authURL := fmt.Sprintf(
		"%s/default/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=openid&state=statePLAIN1&nonce=noncePLAIN1&code_challenge=%s&code_challenge_method=plain",
		baseURL,
		url.QueryEscape("kratos-client"),
		url.QueryEscape("http://localhost:18080/callback"),
		url.QueryEscape(verifier),
	)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(authURL)
	if err != nil {
		t.Fatalf("authorize request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("authorize status: %d body: %s", resp.StatusCode, string(b))
	}
	lu, _ := url.Parse(resp.Header.Get("Location"))
	code := lu.Query().Get("code")
	if code == "" {
		t.Fatalf("missing code in redirect")
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "http://localhost:18080/callback")
	form.Set("code_verifier", verifier)
	form.Set("client_id", "kratos-client")
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/default/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tr, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("token request: %v", err)
	}
	defer tr.Body.Close()
	if tr.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(tr.Body)
		t.Fatalf("token status: %d body: %s", tr.StatusCode, string(b))
	}
}

// startHermetic creates an in-process server with a minimal multi-issuer config
// matching the example config semantics, but using the test server URL as BaseURL.
func startHermetic(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	fc := &cfgpkg.FileConfig{
		ListenAddr:   ":0",
		BaseURL:      "http://example.invalid", // placeholder, replaced after server URL known
		GlobalSecret: "0123456789abcdef0123456789abcdef",
		Providers: []cfgpkg.ProviderConfig{
			{
				ID:     "default",
				Public: true,
				Client: cfgpkg.ProviderClientConfig{ID: "kratos-client", Secret: "kratos-secret", RedirectURIs: []string{"http://localhost:18080/callback"}},
				Personas: map[string]cfgpkg.PersonaConfig{
					"default": {Sub: "00000000-0000-0000-0000-000000000001", Claims: map[string]any{"email": "test@example.com", "email_verified": true, "name": "Test User", "hd": "example.com"}},
				},
			},
			{
				ID:     "google",
				Public: false,
				Client: cfgpkg.ProviderClientConfig{ID: "google-client", Secret: "google-secret", RedirectURIs: []string{"http://localhost:18080/callback"}},
				Personas: map[string]cfgpkg.PersonaConfig{
					"acme": {Sub: "11111111-1111-1111-1111-111111111111", Claims: map[string]any{"email": "user@acme.com", "email_verified": true, "name": "G Suite User", "hd": "acme.com"}},
				},
			},
		},
	}
	mux, err := server.NewMuxFromFileConfig(fc)
	if err != nil {
		t.Fatalf("build mux: %v", err)
	}
	ts := httptest.NewServer(mux)
	// Rebuild mux with BaseURL set to test server URL for correct discovery URLs.
	fc.BaseURL = ts.URL
	mux2, err := server.NewMuxFromFileConfig(fc)
	if err != nil {
		t.Fatalf("rebuild mux: %v", err)
	}
	ts.Config.Handler = mux2
	return ts, ts.URL
}

// startHermeticPlainEnabled is like startHermetic, but enables plain PKCE for the default provider.
func startHermeticPlainEnabled(t *testing.T) (*httptest.Server, string) {
	ts, _ := startHermetic(t)
	// Build a new mux with AllowPKCEPlain=true for default provider
	fc := &cfgpkg.FileConfig{
		ListenAddr:   ":0",
		BaseURL:      ts.URL,
		GlobalSecret: "0123456789abcdef0123456789abcdef",
		Providers: []cfgpkg.ProviderConfig{
			{
				ID:     "default",
				Public: true,
				Client: cfgpkg.ProviderClientConfig{ID: "kratos-client", Secret: "kratos-secret", RedirectURIs: []string{"http://localhost:18080/callback"}},
				Personas: map[string]cfgpkg.PersonaConfig{
					"default": {Sub: "00000000-0000-0000-0000-000000000001", Claims: map[string]any{"email": "test@example.com", "email_verified": true, "name": "Test User"}},
				},
				AllowPKCEPlain: func() *bool { b := true; return &b }(),
			},
			{
				ID:     "google",
				Public: false,
				Client: cfgpkg.ProviderClientConfig{ID: "google-client", Secret: "google-secret", RedirectURIs: []string{"http://localhost:18080/callback"}},
				Personas: map[string]cfgpkg.PersonaConfig{
					"acme": {Sub: "11111111-1111-1111-1111-111111111111", Claims: map[string]any{"email": "user@acme.com", "email_verified": true, "name": "G Suite User"}},
				},
			},
		},
	}
	mux, err := server.NewMuxFromFileConfig(fc)
	if err != nil {
		t.Fatalf("rebuild mux with plain enabled: %v", err)
	}
	ts.Config.Handler = mux
	return ts, ts.URL
}

func TestToken_Confidential_NoAuth(t *testing.T) {
	ts, baseURL := startHermetic(t)
	defer ts.Close()
	issuer := "google"
	clientID := "google-client"
	verifier := strings.Repeat("n", 64)
	h := sha256.Sum256([]byte(verifier))
	s256 := base64.RawURLEncoding.EncodeToString(h[:])
	authURL := fmt.Sprintf(
		"%s/%s/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=openid&state=stateNOAUTH&nonce=nonceNOAUTH&code_challenge=%s&code_challenge_method=S256",
		baseURL, issuer,
		url.QueryEscape(clientID),
		url.QueryEscape("http://localhost:18080/callback"),
		s256,
	)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(authURL)
	if err != nil {
		t.Fatalf("authorize request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("authorize status: %d body: %s", resp.StatusCode, string(b))
	}
	lu, _ := url.Parse(resp.Header.Get("Location"))
	code := lu.Query().Get("code")
	if code == "" {
		t.Fatalf("missing code")
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "http://localhost:18080/callback")
	form.Set("code_verifier", verifier)
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s/token", baseURL, issuer), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tr, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("token request: %v", err)
	}
	defer tr.Body.Close()
	if tr.StatusCode != http.StatusBadRequest { // library returns 400 invalid_request when auth missing
		b, _ := io.ReadAll(tr.Body)
		t.Fatalf("expected 400 invalid_request, got %d body=%s", tr.StatusCode, string(b))
	}
	b, _ := io.ReadAll(tr.Body)
	if !strings.Contains(strings.ToLower(string(b)), "invalid_request") {
		t.Fatalf("expected invalid_request error, got: %s", string(b))
	}
}

func TestToken_Confidential_BadSecret(t *testing.T) {
	ts, baseURL := startHermetic(t)
	defer ts.Close()
	issuer := "google"
	clientID := "google-client"
	verifier := strings.Repeat("x", 64)
	h := sha256.Sum256([]byte(verifier))
	s256 := base64.RawURLEncoding.EncodeToString(h[:])
	authURL := fmt.Sprintf(
		"%s/%s/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=openid&state=stateBADSEC&nonce=nonceBADSEC&code_challenge=%s&code_challenge_method=S256",
		baseURL, issuer,
		url.QueryEscape(clientID),
		url.QueryEscape("http://localhost:18080/callback"),
		s256,
	)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(authURL)
	if err != nil {
		t.Fatalf("authorize request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("authorize status: %d body: %s", resp.StatusCode, string(b))
	}
	lu, _ := url.Parse(resp.Header.Get("Location"))
	code := lu.Query().Get("code")
	if code == "" {
		t.Fatalf("missing code")
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "http://localhost:18080/callback")
	form.Set("code_verifier", verifier)
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s/token", baseURL, issuer), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, "wrong-secret")
	tr, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("token request: %v", err)
	}
	defer tr.Body.Close()
	if tr.StatusCode != http.StatusUnauthorized {
		b, _ := io.ReadAll(tr.Body)
		t.Fatalf("expected 401 unauthorized, got %d body=%s", tr.StatusCode, string(b))
	}
	b, _ := io.ReadAll(tr.Body)
	if !strings.Contains(strings.ToLower(string(b)), "invalid_client") {
		t.Fatalf("expected invalid_client error, got: %s", string(b))
	}
}

func TestDiscovery_TokenEndpointAuthMethods_Confidential(t *testing.T) {
	ts, baseURL := startHermetic(t)
	defer ts.Close()
	var disc map[string]any
	httpGetJSON(t, baseURL+"/google/.well-known/openid-configuration", &disc)
	v, ok := disc["token_endpoint_auth_methods_supported"].([]any)
	if !ok {
		t.Fatalf("missing token_endpoint_auth_methods_supported in discovery")
	}
	for _, m := range v {
		if s, ok := m.(string); ok && strings.EqualFold(s, "none") {
			t.Fatalf("confidential provider must not advertise 'none'")
		}
	}
}

func TestIDToken_VerifySignatureAndClaims(t *testing.T) {
	ts, baseURL := startHermetic(t)
	defer ts.Close()
	// Use implicit flow to receive id_token directly in redirect fragment
	state := "stateIDT"
	nonce := "nonceIDT"
	authURL := fmt.Sprintf("%s/default/authorize?client_id=%s&redirect_uri=%s&response_type=id_token&scope=openid&state=%s&nonce=%s",
		baseURL, url.QueryEscape("kratos-client"), url.QueryEscape("http://localhost:18080/callback"), url.QueryEscape(state), url.QueryEscape(nonce))
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(authURL)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("authorize status: %d body: %s", resp.StatusCode, string(b))
	}
	loc := resp.Header.Get("Location")
	lu, _ := url.Parse(loc)
	frag, _ := url.ParseQuery(strings.TrimPrefix(lu.Fragment, ""))
	idToken := frag.Get("id_token")
	if idToken == "" {
		t.Fatalf("missing id_token in fragment: %s", lu.Fragment)
	}
	if frag.Get("state") != state {
		t.Fatalf("state mismatch: %s", frag.Get("state"))
	}

	// Fetch JWKS and verify signature
	var jwks struct {
		Keys []struct{ Kty, Alg, Use, Kid, N, E string } `json:"keys"`
	}
	httpGetJSON(t, baseURL+"/default/jwks", &jwks)
	if len(jwks.Keys) == 0 {
		t.Fatalf("no jwks")
	}

	hdr, claims, sig, signingInput, err := parseJWS(idToken)
	if err != nil {
		t.Fatalf("parse jws: %v", err)
	}
	alg, _ := hdr["alg"].(string)
	if !strings.EqualFold(alg, "RS256") {
		t.Fatalf("unexpected alg: %v", alg)
	}
	kid, _ := hdr["kid"].(string)
	var nBytes, eBytes []byte
	for _, k := range jwks.Keys {
		if k.Kid == kid {
			var err error
			nBytes, err = base64.RawURLEncoding.DecodeString(k.N)
			if err != nil {
				t.Fatalf("decode n: %v", err)
			}
			eBytes, err = base64.RawURLEncoding.DecodeString(k.E)
			if err != nil {
				t.Fatalf("decode e: %v", err)
			}
			break
		}
	}
	if nBytes == nil {
		t.Fatalf("kid %s not found in jwks", kid)
	}
	eInt := new(big.Int).SetBytes(eBytes).Int64()
	pub := &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: int(eInt)}
	h2 := sha256.Sum256(signingInput)
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, h2[:], sig); err != nil {
		t.Fatalf("verify signature: %v", err)
	}
	if claims["iss"] != baseURL+"/default" {
		t.Fatalf("iss mismatch: %v", claims["iss"])
	}
	if !containsStringClaim(claims["aud"], "kratos-client") {
		t.Fatalf("aud mismatch: %v", claims["aud"])
	}
	if claims["nonce"] != nonce {
		t.Fatalf("nonce mismatch: %v", claims["nonce"])
	}
	// Extra persona claim should be present
	if claims["hd"] != "example.com" {
		t.Fatalf("hd missing/mismatch: %v", claims["hd"])
	}
}

func containsStringClaim(v any, target string) bool {
	if s, ok := v.(string); ok {
		return s == target
	}
	if arr, ok := v.([]any); ok {
		for _, x := range arr {
			if sx, ok := x.(string); ok && sx == target {
				return true
			}
		}
	}
	return false
}

func parseJWS(token string) (map[string]any, map[string]any, []byte, []byte, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, nil, nil, nil, fmt.Errorf("invalid JWS format")
	}
	headB, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("decode header: %w", err)
	}
	payloadB, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("decode payload: %w", err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("decode sig: %w", err)
	}
	var hdr map[string]any
	if err := json.Unmarshal(headB, &hdr); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("unmarshal header: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payloadB, &claims); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("unmarshal claims: %w", err)
	}
	return hdr, claims, sig, []byte(parts[0] + "." + parts[1]), nil
}

func TestToken_NonCacheableHeaders(t *testing.T) {
	ts, baseURL := startHermetic(t)
	defer ts.Close()
	verifier := strings.Repeat("h", 64)
	h := sha256.Sum256([]byte(verifier))
	s256 := base64.RawURLEncoding.EncodeToString(h[:])
	authURL := fmt.Sprintf("%s/default/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=openid&state=stateHDR&nonce=nonceHDR&code_challenge=%s&code_challenge_method=S256", baseURL, url.QueryEscape("kratos-client"), url.QueryEscape("http://localhost:18080/callback"), s256)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(authURL)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	lu, _ := url.Parse(resp.Header.Get("Location"))
	code := lu.Query().Get("code")
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "http://localhost:18080/callback")
	form.Set("code_verifier", verifier)
	form.Set("client_id", "kratos-client")
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/default/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tr, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	defer tr.Body.Close()
	if cc := tr.Header.Get("Cache-Control"); !strings.Contains(strings.ToLower(cc), "no-store") {
		t.Fatalf("missing no-store: %q", cc)
	}
	if pr := tr.Header.Get("Pragma"); !strings.Contains(strings.ToLower(pr), "no-cache") {
		t.Fatalf("missing pragma no-cache: %q", pr)
	}
}

func TestAuthorize_StateEcho(t *testing.T) {
	ts, baseURL := startHermetic(t)
	defer ts.Close()
	verifier := strings.Repeat("s", 64)
	h := sha256.Sum256([]byte(verifier))
	s256 := base64.RawURLEncoding.EncodeToString(h[:])
	state := "stateECHO123"
	authURL := fmt.Sprintf("%s/default/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=openid&state=%s&nonce=nonceECHO&code_challenge=%s&code_challenge_method=S256", baseURL, url.QueryEscape("kratos-client"), url.QueryEscape("http://localhost:18080/callback"), url.QueryEscape(state), s256)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(authURL)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	lu, _ := url.Parse(resp.Header.Get("Location"))
	if got := lu.Query().Get("state"); got != state {
		t.Fatalf("state mismatch: %s", got)
	}
}

func TestUserinfo_ScopeGating(t *testing.T) {
	ts, baseURL := startHermetic(t)
	defer ts.Close()
	// Helper to run a flow with scopes and return userinfo claims
	run := func(scopes string) map[string]any {
		verifier := strings.Repeat("g", 64)
		h := sha256.Sum256([]byte(verifier))
		s256 := base64.RawURLEncoding.EncodeToString(h[:])
		authURL := fmt.Sprintf("%s/default/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=stateGATE&nonce=nonceGATE&code_challenge=%s&code_challenge_method=S256", baseURL, url.QueryEscape("kratos-client"), url.QueryEscape("http://localhost:18080/callback"), url.QueryEscape(scopes), s256)
		client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
		resp, err := client.Get(authURL)
		if err != nil {
			t.Fatalf("authorize: %v", err)
		}
		lu, _ := url.Parse(resp.Header.Get("Location"))
		code := lu.Query().Get("code")
		form := url.Values{}
		form.Set("grant_type", "authorization_code")
		form.Set("code", code)
		form.Set("redirect_uri", "http://localhost:18080/callback")
		form.Set("code_verifier", verifier)
		form.Set("client_id", "kratos-client")
		req, _ := http.NewRequest(http.MethodPost, baseURL+"/default/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		tr, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("token: %v", err)
		}
		defer tr.Body.Close()
		var tok struct {
			AccessToken string `json:"access_token"`
		}
		if err := json.NewDecoder(tr.Body).Decode(&tok); err != nil {
			t.Fatalf("decode token: %v", err)
		}
		req2, _ := http.NewRequest(http.MethodGet, baseURL+"/default/userinfo", nil)
		req2.Header.Set("Authorization", "Bearer "+tok.AccessToken)
		ui, err := http.DefaultClient.Do(req2)
		if err != nil {
			t.Fatalf("userinfo: %v", err)
		}
		defer ui.Body.Close()
		var claims map[string]any
		if err := json.NewDecoder(ui.Body).Decode(&claims); err != nil {
			t.Fatalf("decode userinfo: %v", err)
		}
		return claims
	}
	claims := run("openid")
	if _, ok := claims["email"]; ok {
		t.Fatalf("email present without scope: %v", claims)
	}
	if _, ok := claims["name"]; ok {
		t.Fatalf("profile present without scope: %v", claims)
	}
	if _, ok := claims["hd"]; ok {
		t.Fatalf("hd present without profile scope: %v", claims)
	}
	claims = run("openid email")
	if _, ok := claims["email"]; !ok {
		t.Fatalf("email missing with scope: %v", claims)
	}
	if _, ok := claims["name"]; ok {
		t.Fatalf("profile should be absent: %v", claims)
	}
	if _, ok := claims["hd"]; ok {
		t.Fatalf("hd should be absent without profile scope: %v", claims)
	}
	claims = run("openid profile")
	if _, ok := claims["name"]; !ok {
		t.Fatalf("profile missing with scope: %v", claims)
	}
	if claims["hd"] != "example.com" {
		t.Fatalf("hd missing/mismatch with profile scope: %v", claims["hd"])
	}
}

func TestToken_CodeReuse_InvalidGrant(t *testing.T) {
	ts, baseURL := startHermetic(t)
	defer ts.Close()
	verifier := strings.Repeat("r", 64)
	h := sha256.Sum256([]byte(verifier))
	s256 := base64.RawURLEncoding.EncodeToString(h[:])
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	authURL := fmt.Sprintf("%s/default/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=openid&state=stateREUSE&nonce=nonceREUSE&code_challenge=%s&code_challenge_method=S256", baseURL, url.QueryEscape("kratos-client"), url.QueryEscape("http://localhost:18080/callback"), s256)
	resp, err := client.Get(authURL)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	lu, _ := url.Parse(resp.Header.Get("Location"))
	code := lu.Query().Get("code")
	// First exchange succeeds
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "http://localhost:18080/callback")
	form.Set("code_verifier", verifier)
	form.Set("client_id", "kratos-client")
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/default/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tr, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("token1: %v", err)
	}
	tr.Body.Close()
	// Second exchange should fail invalid_grant
	req2, _ := http.NewRequest(http.MethodPost, baseURL+"/default/token", strings.NewReader(form.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tr2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("token2: %v", err)
	}
	defer tr2.Body.Close()
	if tr2.StatusCode == http.StatusOK {
		t.Fatalf("expected failure on code reuse")
	}
	b, _ := io.ReadAll(tr2.Body)
	if !strings.Contains(strings.ToLower(string(b)), "invalid_grant") {
		t.Fatalf("expected invalid_grant, got: %s", string(b))
	}
}

func TestRedirectURI_Validation(t *testing.T) {
	ts, baseURL := startHermetic(t)
	defer ts.Close()
	// Mismatched redirect at authorize should error
	verifier := strings.Repeat("u", 64)
	h := sha256.Sum256([]byte(verifier))
	s256 := base64.RawURLEncoding.EncodeToString(h[:])
	badRedirect := url.QueryEscape("http://localhost:9999/cb")
	authURL := fmt.Sprintf("%s/default/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=openid&state=stateRU1&nonce=nonceRU1&code_challenge=%s&code_challenge_method=S256", baseURL, url.QueryEscape("kratos-client"), badRedirect, s256)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(authURL)
	if err == nil && (resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusSeeOther) {
		lu, _ := url.Parse(resp.Header.Get("Location"))
		if lu.Query().Get("error") == "" {
			t.Fatalf("expected OAuth error for mismatched redirect_uri")
		}
	}
	// Valid authorize but mismatched at token should fail
	authURL = fmt.Sprintf("%s/default/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=openid&state=stateRU2&nonce=nonceRU2&code_challenge=%s&code_challenge_method=S256", baseURL, url.QueryEscape("kratos-client"), url.QueryEscape("http://localhost:18080/callback"), s256)
	resp2, err := client.Get(authURL)
	if err != nil {
		t.Fatalf("authorize2: %v", err)
	}
	lu2, _ := url.Parse(resp2.Header.Get("Location"))
	code := lu2.Query().Get("code")
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "http://localhost:9999/cb")
	form.Set("code_verifier", verifier)
	form.Set("client_id", "kratos-client")
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/default/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tr, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	defer tr.Body.Close()
	if tr.StatusCode == http.StatusOK {
		b, _ := io.ReadAll(tr.Body)
		t.Fatalf("expected token failure for mismatched redirect_uri, got 200: %s", string(b))
	}
}

func TestGlobalWellKnownRedirects(t *testing.T) {
	ts, baseURL := startHermetic(t)
	defer ts.Close()
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(baseURL + "/.well-known/openid-configuration")
	if err != nil {
		t.Fatalf("global discovery: %v", err)
	}
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if !strings.Contains(loc, "/default/.well-known/openid-configuration") {
		t.Fatalf("unexpected redirect: %s", loc)
	}
	resp2, err := client.Get(baseURL + "/.well-known/jwks.json")
	if err != nil {
		t.Fatalf("global jwks: %v", err)
	}
	if resp2.StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", resp2.StatusCode)
	}
	loc2 := resp2.Header.Get("Location")
	if !strings.Contains(loc2, "/default/jwks") {
		t.Fatalf("unexpected redirect: %s", loc2)
	}
}

func TestRefreshToken_Gating(t *testing.T) {
	t.Skip("refresh token semantics vary by provider/library; track policy in README and wire per-provider toggles before asserting")
	ts, baseURL := startHermetic(t)
	defer ts.Close()

	run := func(issuer, clientID, scopes string, authHeader func(*http.Request)) (map[string]any, int) {
		verifier := strings.Repeat("o", 64)
		h := sha256.Sum256([]byte(verifier))
		s256 := base64.RawURLEncoding.EncodeToString(h[:])
		authURL := fmt.Sprintf("%s/%s/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=stateRT&nonce=nonceRT&code_challenge=%s&code_challenge_method=S256", baseURL, url.PathEscape(issuer), url.QueryEscape(clientID), url.QueryEscape("http://localhost:18080/callback"), url.QueryEscape(scopes), s256)
		client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
		resp, err := client.Get(authURL)
		if err != nil {
			t.Fatalf("authorize: %v", err)
		}
		lu, _ := url.Parse(resp.Header.Get("Location"))
		code := lu.Query().Get("code")
		form := url.Values{}
		form.Set("grant_type", "authorization_code")
		form.Set("code", code)
		form.Set("redirect_uri", "http://localhost:18080/callback")
		form.Set("code_verifier", verifier)
		if authHeader == nil {
			form.Set("client_id", clientID)
		}
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s/token", baseURL, url.PathEscape(issuer)), strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if authHeader != nil {
			authHeader(req)
		}
		tr, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("token: %v", err)
		}
		defer tr.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(tr.Body).Decode(&out)
		return out, tr.StatusCode
	}

	// Confidential client (google): without offline_access -> either success without refresh, or library may reject; both acceptable.
	out, status := run("google", "google-client", "openid", func(r *http.Request) { r.SetBasicAuth("google-client", "google-secret") })
	if status == http.StatusOK {
		if _, ok := out["refresh_token"]; ok {
			t.Fatalf("refresh present without offline_access: %+v", out)
		}
	}

	// Confidential client (google): with offline_access -> success and refresh_token present
	out, status = run("google", "google-client", "openid offline_access", func(r *http.Request) { r.SetBasicAuth("google-client", "google-secret") })
	if status != http.StatusOK {
		t.Fatalf("confidential offline_access should succeed: %d %+v", status, out)
	}
	if _, ok := out["refresh_token"]; !ok {
		t.Fatalf("refresh missing with offline_access for confidential: %+v", out)
	}
}
