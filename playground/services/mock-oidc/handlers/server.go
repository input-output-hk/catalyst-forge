package handlers

import (
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
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
	ListenAddr     string
	IssuerID       string
	BaseURL        string
	ClientID       string
	ClientSecret   string
	RedirectURI    string
	GlobalSecret   string
	DefaultSub     string
	DefaultEmail   string
	DefaultName    string
	AllowPKCEPlain bool
	ClientPublic   bool
	SigningKeyPEM  string
	// Future: ExtraIDTokenClaims map[string]any
	// Future: UserinfoClaimsFromScopes map[string][]string
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
	cfg       *Config
	oauth     fosite.OAuth2Provider
	store     *storage.MemoryStore
	rsaKey    *rsa.PrivateKey
	kid       string
	issuerURL string
	users     map[string]TestUser
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
		cfg:       cfg,
		oauth:     oauth,
		store:     store,
		rsaKey:    rsaKey,
		kid:       KidForKey(&rsaKey.PublicKey),
		issuerURL: fmt.Sprintf("%s/%s", cfg.BaseURL, cfg.IssuerID),
		users:     users,
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
	idClaims := &jwt.IDTokenClaims{Issuer: s.issuerURL, Subject: user.Sub, IssuedAt: now, Nonce: ar.GetRequestForm().Get("nonce"), Extra: extra}
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
	for _, candidate := range s.users {
		if candidate.Sub == subject {
			cc := candidate
			u = &cc
			break
		}
	}
	out := map[string]any{"sub": subject}
	if granted["email"] {
		if u != nil {
			out["email"] = u.Email
			out["email_verified"] = u.EmailVerified
		} else {
			out["email"] = claimString(sess, "email")
			out["email_verified"] = claimBool(sess, "email_verified")
		}
	}
	if granted["profile"] {
		if u != nil {
			out["name"] = u.Name
			out["given_name"] = u.GivenName
			out["family_name"] = u.FamilyName
			out["picture"] = u.Picture
			for k, v := range u.Extra {
				if k == "hd" {
					out[k] = v
				}
			}
		} else {
			out["name"] = claimString(sess, "name")
			out["given_name"] = claimString(sess, "given_name")
			out["family_name"] = claimString(sess, "family_name")
			out["picture"] = claimString(sess, "picture")
		}
	}
	writeJSON(w, http.StatusOK, out)
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
