package hydra

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGetLoginRequest_OK(t *testing.T) {
	var expectedHost string
	sr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assert host, method, path and query
		if r.Host != expectedHost {
			t.Fatalf("unexpected host: %s (expected %s)", r.Host, expectedHost)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/oauth2/auth/requests/login" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("login_challenge"); got != "abc" {
			t.Fatalf("unexpected login_challenge: %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"skip": true, "subject": "user:1"})
	}))
	defer sr.Close()

	u, _ := url.Parse(sr.URL)
	expectedHost = u.Host
	fmt.Printf("expectedHost: %s", expectedHost)
	c := NewAdminClient(sr.URL, sr.Client())

	lr, err := c.GetLoginRequest("abc")
	if err != nil || lr == nil || !lr.Skip || lr.Subject != "user:1" {
		t.Fatalf("unexpected response: lr=%v err=%v", lr, err)
	}
}

func TestAcceptLoginRequest_OK(t *testing.T) {
	var expectedHost string
	sr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assert host, method, path and query
		if r.Host != expectedHost {
			t.Fatalf("unexpected host: %s (expected %s)", r.Host, expectedHost)
		}
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/oauth2/auth/requests/login/accept" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("login_challenge"); got != "abc" {
			t.Fatalf("unexpected login_challenge: %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"redirect_to": "https://hydra/redirect"})
	}))
	defer sr.Close()
	u, _ := url.Parse(sr.URL)
	expectedHost = u.Host
	c := NewAdminClient(sr.URL, sr.Client())
	url, err := c.AcceptLoginRequest("abc", "user:1", true, 300)
	if err != nil || url == "" {
		t.Fatalf("unexpected: url=%q err=%v", url, err)
	}
}

func TestBuildAdminEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		segments []string
		query    map[string]string
		wantHost string
		wantPath string
		wantQKey string
		wantQVal string
	}{
		{
			name:     "simple base no path",
			base:     "http://hydra-admin:4445",
			segments: []string{"oauth2", "auth", "requests", "login"},
			query:    map[string]string{"login_challenge": "abc"},
			wantHost: "hydra-admin:4445",
			wantPath: "/oauth2/auth/requests/login",
			wantQKey: "login_challenge",
			wantQVal: "abc",
		},
		{
			name:     "base with trailing slash",
			base:     "http://hydra-admin:4445/",
			segments: []string{"oauth2", "auth", "requests", "login"},
			query:    map[string]string{"login_challenge": "abc"},
			wantHost: "hydra-admin:4445",
			wantPath: "/oauth2/auth/requests/login",
			wantQKey: "login_challenge",
			wantQVal: "abc",
		},
		{
			name:     "base with existing path",
			base:     "http://example/base",
			segments: []string{"oauth2", "auth", "requests", "login"},
			query:    map[string]string{"login_challenge": "abc"},
			wantHost: "example",
			wantPath: "/base/oauth2/auth/requests/login",
			wantQKey: "login_challenge",
			wantQVal: "abc",
		},
		{
			name:     "base with existing path slash",
			base:     "http://example/base/",
			segments: []string{"oauth2", "auth", "requests", "login"},
			query:    map[string]string{"login_challenge": "abc"},
			wantHost: "example",
			wantPath: "/base/oauth2/auth/requests/login",
			wantQKey: "login_challenge",
			wantQVal: "abc",
		},
		{
			name:     "https scheme and admin prefix",
			base:     "https://hydra-admin:4445/admin",
			segments: []string{"oauth2", "auth", "requests", "login"},
			query:    map[string]string{"login_challenge": "abc"},
			wantHost: "hydra-admin:4445",
			wantPath: "/admin/oauth2/auth/requests/login",
			wantQKey: "login_challenge",
			wantQVal: "abc",
		},
		{
			name:     "query encoding",
			base:     "http://hydra-admin:4445",
			segments: []string{"oauth2", "auth", "requests", "login"},
			query:    map[string]string{"login_challenge": "a b/c?d"},
			wantHost: "hydra-admin:4445",
			wantPath: "/oauth2/auth/requests/login",
			wantQKey: "login_challenge",
			wantQVal: "a b/c?d",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildAdminEndpoint(tc.base, tc.segments, tc.query)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			u, err := url.Parse(got)
			if err != nil {
				t.Fatalf("parse got: %v", err)
			}
			if u.Host != tc.wantHost {
				t.Fatalf("host: got=%q want=%q", u.Host, tc.wantHost)
			}
			if u.Path != tc.wantPath {
				t.Fatalf("path: got=%q want=%q", u.Path, tc.wantPath)
			}
			if val := u.Query().Get(tc.wantQKey); val != tc.wantQVal {
				t.Fatalf("query %q: got=%q want=%q", tc.wantQKey, val, tc.wantQVal)
			}
		})
	}
}
