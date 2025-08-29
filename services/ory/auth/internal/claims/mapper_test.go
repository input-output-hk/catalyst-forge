package claims

import "testing"

func TestMapKratosTraitsToTokens_Empty(t *testing.T) {
	id, ext := MapKratosTraitsToTokens(nil)
	if len(id) != 0 || len(ext) != 0 {
		t.Fatalf("expected empty maps, got id=%v ext=%v", id, ext)
	}
}

func TestMapKratosTraitsToTokens_Populates(t *testing.T) {
	traits := map[string]any{
		"email":  "user@iohk.io",
		"domain": "iohk.io",
	}
	id, ext := MapKratosTraitsToTokens(traits)
	if id["email"] != "user@iohk.io" || id["domain"] != "iohk.io" {
		t.Fatalf("unexpected id token mapping: %v", id)
	}
	if ext["email"] != "user@iohk.io" || ext["domain"] != "iohk.io" {
		t.Fatalf("unexpected access ext mapping: %v", ext)
	}
}
