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
