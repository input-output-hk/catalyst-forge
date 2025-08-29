package handlers

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ory/fosite"
	"github.com/ory/fosite/compose"
	"github.com/ory/fosite/handler/openid"
	"github.com/ory/fosite/storage"
	"github.com/ory/fosite/token/jwt"
	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	ListenAddr        string
	IssuerID          string
	BaseURL           string
	ClientID          string
	ClientSecret      string
	RedirectURI       string
	GlobalSecret      string
	DefaultSub        string
	DefaultEmail      string
	DefaultName       string
	AllowPKCEPlain    bool
	ClientPublic      bool
	SigningKeyPEM     string
	SigningKeyPEMPath string
	Override          bool
	IssuerOverride    string
	DefaultAudience   []string
}

type TestUser struct {
	Sub           string
	Email         string
	EmailVerified bool
	Name          string
	GivenName     string
	FamilyName    string
	Picture       string
	Extra         map[string]any
}

type Server struct {
	cfg         *Config
	oauth       fosite.OAuth2Provider
	store       *storage.MemoryStore
	rsaKey      *rsa.PrivateKey
	kid         string
	issuerURL   string
	claimIssuer string
	users       map[string]TestUser
	// Overrides keyed by OAuth2 state; ephemeral, in-memory.
	overrides map[string]TestUser
	// Per-subject overrides used by userinfo after token exchange.
	subjectOverrides map[string]TestUser
}

func Must32Bytes(s string) []byte {
	b := []byte(s)
	if len(b) != 32 {
		panic(fmt.Sprintf("GLOBAL_SECRET must be exactly 32 bytes (got %d)", len(b)))
	}
	return b
}

func KidForKey(pub *rsa.PublicKey) string {
	h := sha256.Sum256(pub.N.Bytes())
	return base64.RawURLEncoding.EncodeToString(h[:8])
}

func NewServerFromConfig(cfg *Config, rsaKey *rsa.PrivateKey, users map[string]TestUser) (*Server, error) {
	store := storage.NewMemoryStore()
	// Hash client secret for confidential clients to match Fosite's default BCrypt hasher.
	secret := []byte(cfg.ClientSecret)
	if !cfg.ClientPublic && len(cfg.ClientSecret) > 0 {
		if hb, err := bcrypt.GenerateFromPassword([]byte(cfg.ClientSecret), bcrypt.DefaultCost); err == nil {
			secret = hb
		}
	}
	store.Clients = map[string]fosite.Client{
		cfg.ClientID: &fosite.DefaultClient{
			ID:            cfg.ClientID,
			Secret:        secret,
			Public:        cfg.ClientPublic,
			RedirectURIs:  []string{cfg.RedirectURI},
			ResponseTypes: []string{"code", "id_token", "token"},
			GrantTypes:    []string{"authorization_code", "refresh_token", "implicit"},
			Scopes:        []string{"openid", "profile", "email", "offline_access"},
		},
	}
	fc := &fosite.Config{
		GlobalSecret:                   Must32Bytes(cfg.GlobalSecret),
		EnablePKCEPlainChallengeMethod: cfg.AllowPKCEPlain,
	}
	oauth := compose.ComposeAllEnabled(fc, store, rsaKey)

	s := &Server{
		cfg:              cfg,
		oauth:            oauth,
		store:            store,
		rsaKey:           rsaKey,
		kid:              KidForKey(&rsaKey.PublicKey),
		issuerURL:        fmt.Sprintf("%s/%s", cfg.BaseURL, cfg.IssuerID),
		claimIssuer:      "",
		users:            users,
		overrides:        make(map[string]TestUser),
		subjectOverrides: make(map[string]TestUser),
	}
	if strings.TrimSpace(cfg.IssuerOverride) != "" {
		s.claimIssuer = cfg.IssuerOverride
	} else {
		s.claimIssuer = s.issuerURL
	}
	return s, nil
}

func (s *Server) HandleDiscovery(w http.ResponseWriter, r *http.Request) {
	type Disc struct {
		Issuer                           string   `json:"issuer"`
		AuthorizationEndpoint            string   `json:"authorization_endpoint"`
		TokenEndpoint                    string   `json:"token_endpoint"`
		UserinfoEndpoint                 string   `json:"userinfo_endpoint"`
		JwksURI                          string   `json:"jwks_uri"`
		ResponseTypesSupported           []string `json:"response_types_supported"`
		ResponseModesSupported           []string `json:"response_modes_supported"`
		SubjectTypesSupported            []string `json:"subject_types_supported"`
		IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
		ScopesSupported                  []string `json:"scopes_supported"`
		TokenEndpointAuthMethods         []string `json:"token_endpoint_auth_methods_supported"`
		CodeChallengeMethodsSupported    []string `json:"code_challenge_methods_supported"`
	}
	resp := Disc{
		Issuer:                           s.issuerURL,
		AuthorizationEndpoint:            s.issuerURL + "/authorize",
		TokenEndpoint:                    s.issuerURL + "/token",
		UserinfoEndpoint:                 s.issuerURL + "/userinfo",
		JwksURI:                          s.issuerURL + "/jwks",
		ResponseTypesSupported:           []string{"code", "id_token", "token", "none"},
		ResponseModesSupported:           []string{"query", "fragment", "form_post"},
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{"RS256"},
		ScopesSupported:                  []string{"openid", "profile", "email", "offline_access"},
		TokenEndpointAuthMethods: func() []string {
			if s.cfg.ClientPublic {
				return []string{"none"}
			}
			return []string{"client_secret_basic", "client_secret_post"}
		}(),
		CodeChallengeMethodsSupported: func() []string {
			if s.cfg.AllowPKCEPlain {
				return []string{"S256", "plain"}
			}
			return []string{"S256"}
		}(),
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	pub := s.rsaKey.PublicKey
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(bigIntToBytes(pub.E))
	resp := map[string]any{"keys": []map[string]string{{
		"kty": "RSA", "alg": "RS256", "use": "sig", "kid": s.kid, "n": n, "e": e,
	}}}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// If override is enabled, present a lightweight consent/edit page before proceeding.
	if s.cfg.Override {
		// Detect POST from consent form
		if r.Method == http.MethodPost && r.Header.Get("Content-Type") != "application/json" {
			_ = r.ParseForm()
			if r.Form.Get("_action") == "submit_override" {
				state := r.Form.Get("state")
				sub := r.Form.Get("sub")
				claimsJSON := r.Form.Get("claims_json")
				var claims map[string]any
				if err := json.Unmarshal([]byte(claimsJSON), &claims); err != nil {
					http.Error(w, "invalid claims_json", http.StatusBadRequest)
					return
				}
				u := TestUser{Sub: sub, Extra: map[string]any{}}
				// Map common known claims for convenience
				if v, ok := claims["email"].(string); ok {
					u.Email = v
				}
				if v, ok := claims["email_verified"].(bool); ok {
					u.EmailVerified = v
				}
				if v, ok := claims["name"].(string); ok {
					u.Name = v
				}
				if v, ok := claims["given_name"].(string); ok {
					u.GivenName = v
				}
				if v, ok := claims["family_name"].(string); ok {
					u.FamilyName = v
				}
				if v, ok := claims["picture"].(string); ok {
					u.Picture = v
				}
				for k, v := range claims {
					u.Extra[k] = v
				}
				if state == "" {
					http.Error(w, "missing state", http.StatusBadRequest)
					return
				}
				// Store override for this state
				s.overrides[state] = u
				log.Printf("override stored: state=%s sub=%s email=%s hd=%v", state, u.Sub, u.Email, u.Extra["hd"])

				// Reconstruct GET authorize URL with original params and an _consented=1 flag
				vals := url.Values{}
				for k, vv := range r.Form {
					for _, v := range vv {
						vals.Add(k, v)
					}
				}
				vals.Del("_action")
				vals.Set("_consented", "1")
				redir := s.issuerURL + "/authorize?" + vals.Encode()
				if !strings.Contains(redir, "_consented=") {
					if strings.Contains(redir, "?") {
						redir += "&"
					} else {
						redir += "?"
					}
					redir += "_consented=1"
				}
				http.Redirect(w, r, redir, http.StatusSeeOther)
				return
			}
		}
		// If GET and not yet consented, render the form
		if r.Method == http.MethodGet && r.URL.Query().Get("_consented") != "1" {
			// Choose base persona/user
			userKey := r.URL.Query().Get("user")
			if userKey == "" {
				if _, hasDefault := s.users["default"]; hasDefault {
					userKey = "default"
				} else {
					for k := range s.users {
						userKey = k
						break
					}
				}
			}
			u := s.users[userKey]
			claims := map[string]any{
				"email":          u.Email,
				"email_verified": u.EmailVerified,
				"name":           u.Name,
				"given_name":     u.GivenName,
				"family_name":    u.FamilyName,
				"picture":        u.Picture,
			}
			for k, v := range u.Extra {
				claims[k] = v
			}
			b, _ := json.MarshalIndent(claims, "", "  ")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, "<html><head><title>Consent - %s</title></head><body>", html.EscapeString(s.cfg.IssuerID))
			fmt.Fprintf(w, "<h3>Override identity for provider %s</h3>", html.EscapeString(s.cfg.IssuerID))
			fmt.Fprintf(w, "<form method=\"post\" action=\"%s/authorize\">", html.EscapeString(s.issuerURL))
			fmt.Fprintf(w, "<input type=\"hidden\" name=\"_action\" value=\"submit_override\">")
			// Preserve original authorize params
			preserveParams := []string{"client_id", "redirect_uri", "response_type", "scope", "state", "nonce", "code_challenge", "code_challenge_method", "user"}
			for _, k := range preserveParams {
				if v := r.URL.Query().Get(k); v != "" {
					fmt.Fprintf(w, "<input type=\"hidden\" name=\"%s\" value=\"%s\">", html.EscapeString(k), html.EscapeString(v))
				}
			}
			fmt.Fprintf(w, "<div><label>sub</label><br><input name=\"sub\" value=\"%s\" style=\"width: 100%%\"></div>", html.EscapeString(u.Sub))
			fmt.Fprintf(w, "<div><label>claims_json</label><br><textarea name=\"claims_json\" rows=\"20\" style=\"width: 100%%\">%s</textarea></div>", html.EscapeString(string(b)))
			fmt.Fprintf(w, "<div><button type=\"submit\">Continue</button></div>")
			fmt.Fprintf(w, "</form></body></html>")
			return
		}
	}
	ar, err := s.oauth.NewAuthorizeRequest(ctx, r)
	if err != nil {
		s.oauth.WriteAuthorizeError(ctx, w, ar, err)
		return
	}
	// PKCE plain toggle
	if !s.cfg.AllowPKCEPlain && strings.EqualFold(r.URL.Query().Get("code_challenge_method"), "plain") {
		s.oauth.WriteAuthorizeError(ctx, w, ar, fosite.ErrInvalidRequest.WithHint("plain PKCE not allowed"))
		return
	}
	userKey := r.URL.Query().Get("user")
	if userKey == "" {
		if _, hasDefault := s.users["default"]; hasDefault {
			userKey = "default"
		} else {
			for k := range s.users {
				userKey = k
				break
			}
		}
	}
	user, ok := s.users[userKey]
	if !ok {
		http.Error(w, fmt.Sprintf("unknown user key %q", userKey), http.StatusBadRequest)
		return
	}
	// Apply override if present for this state
	if s.cfg.Override {
		state := ar.GetState()
		log.Printf("override applied: state=%s sub=%s email=%s hd=%v", ar.GetState(), user.Sub, user.Email, user.Extra["hd"])
		if ov, has := s.overrides[state]; has {
			user = ov
			// Persist by subject for userinfo lookups
			s.subjectOverrides[user.Sub] = ov
			delete(s.overrides, state)
		} else if r.URL.Query().Get("_consented") == "1" {
			// Fallback: reconstruct override from GET query if not found in memory
			cj := r.URL.Query().Get("claims_json")
			if cj != "" {
				var claims map[string]any
				if err := json.Unmarshal([]byte(cj), &claims); err == nil {
					u2 := TestUser{Sub: r.URL.Query().Get("sub"), Extra: map[string]any{}}
					if v, ok := claims["email"].(string); ok {
						u2.Email = v
					}
					if v, ok := claims["email_verified"].(bool); ok {
						u2.EmailVerified = v
					}
					if v, ok := claims["name"].(string); ok {
						u2.Name = v
					}
					if v, ok := claims["given_name"].(string); ok {
						u2.GivenName = v
					}
					if v, ok := claims["family_name"].(string); ok {
						u2.FamilyName = v
					}
					if v, ok := claims["picture"].(string); ok {
						u2.Picture = v
					}
					for k, v := range claims {
						u2.Extra[k] = v
					}
					user = u2
					s.subjectOverrides[user.Sub] = u2
				}
			}
		}
	}
	now := time.Now()
	extra := map[string]any{
		"email":          user.Email,
		"email_verified": user.EmailVerified,
		"name":           user.Name,
		"given_name":     user.GivenName,
		"family_name":    user.FamilyName,
		"picture":        user.Picture,
	}
	// Merge persona extra arbitrary claims (e.g., hd)
	for k, v := range user.Extra {
		extra[k] = v
	}
	idClaims := &jwt.IDTokenClaims{Issuer: s.claimIssuer, Subject: user.Sub, IssuedAt: now, Nonce: ar.GetRequestForm().Get("nonce"), Audience: s.cfg.DefaultAudience, Extra: extra}
	session := &openid.DefaultSession{
		Claims:    idClaims,
		Headers:   &jwt.Headers{Extra: map[string]any{"kid": s.kid}},
		ExpiresAt: map[fosite.TokenType]time.Time{fosite.AccessToken: now.Add(30 * time.Minute), fosite.RefreshToken: now.Add(24 * time.Hour), fosite.AuthorizeCode: now.Add(10 * time.Minute), fosite.IDToken: now.Add(30 * time.Minute)},
		Subject:   user.Sub, Username: user.Email,
	}
	for _, scope := range ar.GetRequestedScopes() {
		ar.GrantScope(scope)
	}
	resp, err := s.oauth.NewAuthorizeResponse(ctx, ar, session)
	if err != nil {
		s.oauth.WriteAuthorizeError(ctx, w, ar, err)
		return
	}
	s.oauth.WriteAuthorizeResponse(ctx, w, ar, resp)
}

type captureWriter struct {
	h    http.ResponseWriter
	buf  []byte
	code int
}

func (cw *captureWriter) Header() http.Header        { return cw.h.Header() }
func (cw *captureWriter) WriteHeader(statusCode int) { cw.code = statusCode }
func (cw *captureWriter) Write(p []byte) (int, error) {
	cw.buf = append(cw.buf, p...)
	return len(p), nil
}

func (s *Server) HandleToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	session := new(openid.DefaultSession)
	accessReq, err := s.oauth.NewAccessRequest(ctx, r, session)
	if err != nil {
		s.oauth.WriteAccessError(ctx, w, accessReq, err)
		return
	}
	accessResp, err := s.oauth.NewAccessResponse(ctx, accessReq)
	if err != nil {
		s.oauth.WriteAccessError(ctx, w, accessReq, err)
		return
	}
	writeNoStore(w)
	cw := &captureWriter{h: w}
	s.oauth.WriteAccessResponse(ctx, cw, accessReq, accessResp)
	// Enforce refresh token policy post-write: only for confidential clients with offline_access.
	stripRefresh := false
	if c := accessReq.GetClient(); c == nil || c.IsPublic() {
		stripRefresh = true
	}
	if !hasGrantedScope(accessReq, "offline_access") {
		stripRefresh = true
	}
	if stripRefresh && len(cw.buf) > 0 {
		var out map[string]any
		if err := json.Unmarshal(cw.buf, &out); err == nil {
			delete(out, "refresh_token")
			res, _ := json.Marshal(out)
			w.Header().Set("Content-Type", "application/json")
			if cw.code != 0 {
				w.WriteHeader(cw.code)
			}
			_, _ = w.Write(res)
			return
		}
	}
	// Fallback: write original
	for k, v := range cw.Header() {
		w.Header()[k] = v
	}
	if cw.code != 0 {
		w.WriteHeader(cw.code)
	}
	_, _ = w.Write(cw.buf)
}

func hasGrantedScope(ar fosite.AccessRequester, scope string) bool {
	for _, s := range ar.GetGrantedScopes() {
		if s == scope {
			return true
		}
	}
	return false
}

func (s *Server) HandleUserinfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	proto := new(openid.DefaultSession)
	_, ar, err := s.oauth.IntrospectToken(ctx, fosite.AccessTokenFromRequest(r), fosite.AccessToken, proto, "openid")
	if err != nil {
		http.Error(w, "invalid or missing access token", http.StatusUnauthorized)
		return
	}
	sess, _ := ar.GetSession().(*openid.DefaultSession)
	if sess == nil {
		sess = proto
	}
	granted := map[string]bool{}
	for _, sc := range ar.GetGrantedScopes() {
		granted[sc] = true
	}
	subject := sess.Subject
	var u *TestUser
	if ov, ok := s.subjectOverrides[subject]; ok {
		cc := ov
		u = &cc
	} else {
		for _, candidate := range s.users {
			if candidate.Sub == subject {
				cc := candidate
				u = &cc
				break
			}
		}
	}
	out := map[string]any{"sub": subject}
	if granted["email"] {
		// Prefer session claims (edited) over persona defaults
		email := claimString(sess, "email")
		emailVerified := claimBool(sess, "email_verified")
		if email == "" && u != nil {
			email = u.Email
		}
		if !emailVerified && u != nil {
			emailVerified = u.EmailVerified
		}
		if email != "" {
			out["email"] = email
		}
		out["email_verified"] = emailVerified
	}
	if granted["profile"] {
		name := claimString(sess, "name")
		given := claimString(sess, "given_name")
		family := claimString(sess, "family_name")
		picture := claimString(sess, "picture")
		if name == "" && u != nil {
			name = u.Name
		}
		if given == "" && u != nil {
			given = u.GivenName
		}
		if family == "" && u != nil {
			family = u.FamilyName
		}
		if picture == "" && u != nil {
			picture = u.Picture
		}
		if name != "" {
			out["name"] = name
		}
		if given != "" {
			out["given_name"] = given
		}
		if family != "" {
			out["family_name"] = family
		}
		if picture != "" {
			out["picture"] = picture
		}
		// Include selected extras like hd; prefer session extras
		hd := claimString(sess, "hd")
		if hd == "" && u != nil {
			if v, ok := u.Extra["hd"].(string); ok {
				hd = v
			}
		}
		if hd != "" {
			out["hd"] = hd
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// HandleActionsToken mints a GitHub Actions-like JWT using the provider's signing key.
// POST JSON: {"repository":"org/repo","ref":"refs/heads/main","sha":"...","actor":"...","environment":"dev","aud":["..."]}
// Response: {"token":"<jwt>"}
func (s *Server) HandleActionsToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var in struct {
		Repository  string   `json:"repository"`
		Ref         string   `json:"ref"`
		Sha         string   `json:"sha"`
		Actor       string   `json:"actor"`
		Environment string   `json:"environment"`
		Aud         []string `json:"aud"`
		Sub         string   `json:"sub"`
		Exp         int64    `json:"exp"`
		TTLSeconds  int64    `json:"ttl_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if len(in.Aud) == 0 {
		in.Aud = s.cfg.DefaultAudience
	}
	if in.Sub == "" && in.Repository != "" && in.Ref != "" {
		in.Sub = "repo:" + in.Repository + ":ref:" + in.Ref
	}
	now := time.Now()
	// Compute expiration: prefer explicit exp (seconds), then ttl_seconds, default 5m
	var expTime time.Time
	if in.Exp > 0 {
		expTime = time.Unix(in.Exp, 0)
	} else if in.TTLSeconds > 0 {
		expTime = now.Add(time.Duration(in.TTLSeconds) * time.Second)
	} else {
		expTime = now.Add(5 * time.Minute)
	}
	// Generate a random jti (JWT ID)
	var jtiBytes [16]byte
	_, _ = rand.Read(jtiBytes[:])
	jti := base64.RawURLEncoding.EncodeToString(jtiBytes[:])

	claims := map[string]any{
		"iss":         s.claimIssuer,
		"sub":         in.Sub,
		"aud":         in.Aud,
		"iat":         now.Unix(),
		"exp":         expTime.Unix(),
		"jti":         jti,
		"repository":  in.Repository,
		"ref":         in.Ref,
		"sha":         in.Sha,
		"actor":       in.Actor,
		"environment": in.Environment,
	}
	header := map[string]any{
		"alg": "RS256",
		"typ": "JWT",
		"kid": s.kid,
	}
	enc := func(v any) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	unsigned := enc(header) + "." + enc(claims)
	h := sha256.Sum256([]byte(unsigned))
	sig, err := rsa.SignPKCS1v15(nil, s.rsaKey, crypto.SHA256, h[:])
	if err != nil {
		http.Error(w, "sign error", http.StatusInternalServerError)
		return
	}
	token := unsigned + "." + base64.RawURLEncoding.EncodeToString(sig)
	log.Printf("actions/token minted kid=%s aud=%v sub=%s", s.kid, in.Aud, in.Sub)
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func writeNoStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}

func bigIntToBytes(e int) []byte {
	if e == 0 {
		return []byte{0}
	}
	var bytes []byte
	for x := e; x > 0; x >>= 8 {
		bytes = append([]byte{byte(x & 0xff)}, bytes...)
	}
	return bytes
}

func claimString(s *openid.DefaultSession, key string) string {
	if s == nil || s.Claims == nil || s.Claims.Extra == nil {
		return ""
	}
	v, _ := s.Claims.Extra[key].(string)
	return v
}
func claimBool(s *openid.DefaultSession, key string) bool {
	if s == nil || s.Claims == nil || s.Claims.Extra == nil {
		return false
	}
	v, _ := s.Claims.Extra[key].(bool)
	return v
}
