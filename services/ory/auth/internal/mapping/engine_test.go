package mapping

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "mapping.yaml")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return p
}

func TestEvaluateConsent_Success(t *testing.T) {
	cfg := `
mappings:
  consent:
    requirements: [ has(kratos.identity.traits.email), has(kratos.identity.traits.domain) ]
    id_token:
      email: kratos.identity.traits.email
      domain: lower(kratos.identity.traits.domain)
    access_token:
      ext:
        email: kratos.identity.traits.email
        domain: lower(kratos.identity.traits.domain)
policy:
  on_error: deny
  merge_strategy: deep
`
	path := writeTempConfig(t, cfg)
	eng, err := NewEngine(nil, path)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	id, ext, err := eng.EvaluateConsent(ConsentInput{
		Kratos: map[string]any{"identity": map[string]any{"traits": map[string]any{"email": "a@b.c", "domain": "EXAMPLE.com"}}},
		Hydra:  map[string]any{},
		Req:    map[string]any{},
	})
	if err != nil {
		t.Fatalf("evaluate consent: %v", err)
	}
	if id["email"] != "a@b.c" || id["domain"] != "example.com" {
		t.Fatalf("unexpected id token: %v", id)
	}
	if ext["email"] != "a@b.c" || ext["domain"] != "example.com" {
		t.Fatalf("unexpected access ext: %v", ext)
	}
}

func TestEvaluateConsent_FailRequirement(t *testing.T) {
	cfg := `
mappings:
  consent:
    requirements: [ has(kratos.identity.traits.domain) ]
    id_token:
      email: kratos.identity.traits.email
    access_token:
      ext:
        email: kratos.identity.traits.email
policy:
  on_error: deny
`
	path := writeTempConfig(t, cfg)
	eng, err := NewEngine(nil, path)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	_, _, err = eng.EvaluateConsent(ConsentInput{
		Kratos: map[string]any{"identity": map[string]any{"traits": map[string]any{"email": "a@b.c"}}},
		Hydra:  map[string]any{},
		Req:    map[string]any{},
	})
	if err == nil {
		t.Fatalf("expected requirement failure, got nil error")
	}
}

func TestEvaluateTokenHook_Success(t *testing.T) {
	cfg := `
mappings:
  token_hook:
    requirements: [ has(jwt.repository), has(jwt.ref), has(jwt.sha) ]
    access_token:
      ext:
        gh_repository: jwt.repository
        gh_ref: jwt.ref
        gh_sha: jwt.sha
policy:
  on_error: deny
`
	path := writeTempConfig(t, cfg)
	eng, err := NewEngine(nil, path)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	ext, err := eng.EvaluateTokenHook(TokenHookInput{
		JWT: map[string]any{"repository": "org/repo", "ref": "refs/heads/main", "sha": "abc"},
		Req: map[string]any{"client_id": "ci"},
	})
	if err != nil {
		t.Fatalf("evaluate token hook: %v", err)
	}
	if ext["gh_repository"] != "org/repo" || ext["gh_ref"] != "refs/heads/main" || ext["gh_sha"] != "abc" {
		t.Fatalf("unexpected ext: %v", ext)
	}
}

func TestEvaluateTokenHook_RequirementFail_PolicyDeny(t *testing.T) {
	cfg := `
mappings:
  token_hook:
    requirements: [ has(jwt.repository) ]
    access_token:
      ext:
        gh_repository: jwt.repository
policy:
  on_error: deny
`
	path := writeTempConfig(t, cfg)
	eng, err := NewEngine(nil, path)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	if _, err := eng.EvaluateTokenHook(TokenHookInput{JWT: map[string]any{}}); err == nil {
		t.Fatalf("expected error on requirement fail")
	}
}

func TestEvaluateTokenHook_RequirementFail_PolicyWarnPassthrough(t *testing.T) {
	cfg := `
mappings:
  token_hook:
    requirements: [ has(jwt.repository) ]
    access_token:
      ext:
        gh_repository: jwt.repository
policy:
  on_error: warn_passthrough
`
	path := writeTempConfig(t, cfg)
	eng, err := NewEngine(nil, path)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	// With warn_passthrough, EvaluateTokenHook should still error since engine enforces requirements.
	// The policy is enforced by handlers, not the engine. So we still expect error here.
	if _, err := eng.EvaluateTokenHook(TokenHookInput{JWT: map[string]any{}}); err == nil {
		t.Fatalf("expected error on requirement fail")
	}
}
