package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"time"

	cfgpkg "github.com/input-output-hk/catalyst-forge/playground/services/mock-oidc/config"
	"github.com/input-output-hk/catalyst-forge/playground/services/mock-oidc/handlers"
	u "github.com/input-output-hk/catalyst-forge/playground/services/mock-oidc/util"
)

// loadOrGenerateKey returns the RSA private key from PEM if provided, otherwise generates a new one.
func loadOrGenerateKey(cfg *handlers.Config) (*rsa.PrivateKey, error) {
	if cfg.SigningKeyPEM == "" {
		return rsa.GenerateKey(rand.Reader, 2048)
	}
	block, _ := pem.Decode([]byte(cfg.SigningKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse SIGNING_KEY_PEM: no PEM block")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	rsaKey, ok := keyAny.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("SIGNING_KEY_PEM is not RSA private key")
	}
	return rsaKey, nil
}

// logRequests wraps the handler to log simple request lines with latency.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Truncate(time.Millisecond))
	})
}

// NewMuxFromFileConfig builds an http.Handler serving all providers from the given FileConfig.
// It does not start any server or read environment variables, making it ideal for hermetic tests.
func NewMuxFromFileConfig(fc *cfgpkg.FileConfig) (http.Handler, error) {
	if fc == nil {
		return nil, fmt.Errorf("file config is nil")
	}
	if len(fc.Providers) == 0 {
		return nil, fmt.Errorf("config contains no providers")
	}

	mux := http.NewServeMux()
	for _, p := range fc.Providers {
		pc := p
		cfg := &handlers.Config{
			ListenAddr:     u.FirstNonEmpty(fc.ListenAddr, ":8080"),
			IssuerID:       pc.ID,
			BaseURL:        u.FirstNonEmpty(pc.BaseURL, fc.BaseURL),
			ClientID:       pc.Client.ID,
			ClientSecret:   pc.Client.Secret,
			RedirectURI:    u.FirstString(pc.Client.RedirectURIs),
			GlobalSecret:   fc.GlobalSecret,
			DefaultSub:     u.PickPersonaSub(pc.Personas),
			DefaultEmail:   u.PickPersonaClaimString(pc.Personas, "email"),
			DefaultName:    u.PickPersonaClaimString(pc.Personas, "name"),
			AllowPKCEPlain: u.BoolOrDefault(pc.AllowPKCEPlain, fc.AllowPKCEPlain),
			ClientPublic:   pc.Public,
			SigningKeyPEM:  pc.SigningKeyPEM,
		}
		priv, err := loadOrGenerateKey(cfg)
		if err != nil {
			return nil, fmt.Errorf("load signing key for %s: %w", pc.ID, err)
		}
		users := u.BuildUsersFromPersonas(pc.Personas)
		srv, err := handlers.NewServerFromConfig(cfg, priv, users)
		if err != nil {
			return nil, fmt.Errorf("build server for %s: %w", pc.ID, err)
		}

		base := "/" + cfg.IssuerID
		mux.HandleFunc(base+"/.well-known/openid-configuration", srv.HandleDiscovery)
		mux.HandleFunc(base+"/jwks", srv.HandleJWKS)
		mux.HandleFunc(base+"/authorize", srv.HandleAuthorize)
		mux.HandleFunc(base+"/token", srv.HandleToken)
		mux.HandleFunc(base+"/userinfo", srv.HandleUserinfo)
	}

	// Global redirects for convenience
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/"+fc.Providers[0].ID+"/jwks", http.StatusFound)
	})
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/"+fc.Providers[0].ID+"/.well-known/openid-configuration", http.StatusFound)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	return logRequests(mux), nil
}
