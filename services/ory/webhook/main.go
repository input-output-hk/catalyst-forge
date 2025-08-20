package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	Org               string // GitHub org to enforce
	KratosAdminURL    string // e.g. http://kratos-admin:4434
	KratosAdminAPIKey string // optional: if you protect admin with an auth proxy
	GitHubAPIBase     string // default https://api.github.com
	AllowOnAPIFailure bool   // optional safety valve
}

func loadConfig() Config {
	cfg := Config{
		Org:               os.Getenv("GITHUB_ALLOWED_ORG"),
		KratosAdminURL:    os.Getenv("KRATOS_ADMIN_URL"),
		KratosAdminAPIKey: os.Getenv("KRATOS_ADMIN_API_KEY"),
		GitHubAPIBase:     os.Getenv("GITHUB_API_BASE"),
	}
	if cfg.GitHubAPIBase == "" {
		cfg.GitHubAPIBase = "https://api.github.com"
	}
	if strings.ToLower(os.Getenv("ALLOW_ON_API_FAILURE")) == "true" {
		cfg.AllowOnAPIFailure = true
	}
	return cfg
}

type ActionRequest struct {
	Identity struct {
		ID string `json:"id"`
	} `json:"identity"`

	// Some Kratos OIDC actions include the upstream token context in the payload.
	// We try a few common shapes defensively.
	Context map[string]any `json:"context"`
}

type Identity struct {
	ID          string `json:"id"`
	Credentials map[string]struct {
		Type        string         `json:"type"`
		Identifiers []string       `json:"identifiers"`
		Config      map[string]any `json:"config"`
	} `json:"credentials"`
}

// Try to pull a GitHub access token out of either the Action payload context
// (best case) or the identity's OIDC credential record (fallback).
func extractGitHubTokenFromAction(a ActionRequest) (string, bool) {
	// Common places it might show up depending on Kratos version/config:
	paths := [][]string{
		{"context", "oidc", "access_token"},
		{"context", "oauth2", "access_token"},
		{"context", "oidc", "token", "access_token"},
	}
	for _, p := range paths {
		if tok, ok := digString(a, p...); ok {
			return tok, true
		}
	}
	return "", false
}

func extractGitHubTokenFromIdentity(i Identity) (string, bool) {
	cred, ok := i.Credentials["oidc"]
	if !ok {
		return "", false
	}
	// Known keys used by Kratos for initial tokens (names vary by version/provider):
	candidates := []string{
		"access_token", "token", "oauth2_access_token", "provider_access_token",
	}
	// tokens may be stored per provider under "providers": [{"provider":"github","initial_access_token": "..."}]
	if provs, ok := cred.Config["providers"].([]any); ok {
		for _, p := range provs {
			if m, ok := p.(map[string]any); ok {
				// Require provider == github
				if provName, _ := m["provider"].(string); provName == "github" {
					for _, k := range []string{"initial_access_token", "access_token"} {
						if v, ok := m[k].(string); ok && v != "" {
							return v, true
						}
					}
				}
			}
		}
	}
	for _, k := range candidates {
		if v, ok := cred.Config[k].(string); ok && v != "" {
			return v, true
		}
	}
	return "", false
}

// tiny helper for dynamic payloads
func digString(a ActionRequest, path ...string) (string, bool) {
	var cur any = map[string]any{"context": a.Context}
	for _, k := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return "", false
		}
		cur, ok = m[k]
		if !ok {
			return "", false
		}
	}
	s, ok := cur.(string)
	return s, ok && s != ""
}

type OrgMembership struct {
	State string `json:"state"` // "active" when member (or "pending")
	Role  string `json:"role"`  // "member" or "admin"
}

func isMember(ctx context.Context, apiBase, token, org string) (bool, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/user/memberships/orgs/%s", strings.TrimRight(apiBase, "/"), org), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// 404 means not a member
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("github api %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var m OrgMembership
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return false, err
	}
	return strings.EqualFold(m.State, "active"), nil
}

func fetchIdentity(ctx context.Context, adminURL, apiKey, id string) (Identity, error) {
	var out Identity
	url := fmt.Sprintf("%s/admin/identities/%s?include_credential=oidc", strings.TrimRight(adminURL, "/"), id)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		return out, fmt.Errorf("kratos admin %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	err = json.NewDecoder(resp.Body).Decode(&out)
	return out, err
}

type actionAllow struct {
	Action string `json:"action"` // "continue"
}

type actionAbort struct {
	Action string       `json:"action"` // "abort"
	Error  actionErrMsg `json:"error"`
}
type actionErrMsg struct {
	Message string `json:"message"`
}

func respondAllow(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(actionAllow{Action: "continue"})
}

func respondAbort(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(actionAbort{
		Action: "abort",
		Error:  actionErrMsg{Message: msg},
	})
}

func checkOrgHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var payload ActionRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			log.Printf("bad payload: %v", err)
			respondAbort(w, "Invalid request.")
			return
		}
		if cfg.Org == "" {
			log.Printf("GITHUB_ALLOWED_ORG not set")
			respondAbort(w, "Server not configured.")
			return
		}

		// 1) Try to get GitHub token from the action context
		if tok, ok := extractGitHubTokenFromAction(payload); ok {
			okMember, err := isMember(ctx, cfg.GitHubAPIBase, tok, cfg.Org)
			if err != nil {
				log.Printf("github check (ctx token) error: %v", err)
				if cfg.AllowOnAPIFailure {
					respondAllow(w)
				} else {
					respondAbort(w, "Unable to verify organization membership right now. Please try again.")
				}
				return
			}
			if !okMember {
				respondAbort(w, fmt.Sprintf("You must be an active member of the %q GitHub organization.", cfg.Org))
				return
			}
			respondAllow(w)
			return
		}

		// 2) Fallback: fetch identity with OIDC credential details and try to read a token
		if payload.Identity.ID == "" {
			log.Printf("no identity id in action payload")
			respondAbort(w, "Missing identity.")
			return
		}
		ident, err := fetchIdentity(ctx, cfg.KratosAdminURL, cfg.KratosAdminAPIKey, payload.Identity.ID)
		if err != nil {
			log.Printf("fetch identity error: %v", err)
			if cfg.AllowOnAPIFailure {
				respondAllow(w)
			} else {
				respondAbort(w, "Unable to verify identity.")
			}
			return
		}
		tok, ok := extractGitHubTokenFromIdentity(ident)
		if !ok || tok == "" {
			log.Printf("no github token on identity %s", ident.ID)
			if cfg.AllowOnAPIFailure {
				respondAllow(w)
			} else {
				respondAbort(w, "Unable to verify organization membership.")
			}
			return
		}

		okMember, err := isMember(ctx, cfg.GitHubAPIBase, tok, cfg.Org)
		if err != nil {
			log.Printf("github check (identity token) error: %v", err)
			if cfg.AllowOnAPIFailure {
				respondAllow(w)
			} else {
				respondAbort(w, "Unable to verify organization membership right now. Please try again.")
			}
			return
		}
		if !okMember {
			respondAbort(w, fmt.Sprintf("You must be an active member of the %q GitHub organization.", cfg.Org))
			return
		}
		respondAllow(w)
	}
}

func main() {
	cfg := loadConfig()
	if cfg.Org == "" || cfg.KratosAdminURL == "" {
		log.Fatal("Set env: GITHUB_ALLOWED_ORG and KRATOS_ADMIN_URL")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/check-github-org", checkOrgHandler(cfg))

	addr := ":8080"
	log.Printf("kratos action webhook listening on %s (org=%s)", addr, cfg.Org)
	if err := http.ListenAndServe(addr, mux); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
