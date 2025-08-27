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
		"email":  "user@example.com",
		"name":   map[string]any{"given": "Alice", "family": "Doe"},
		"org_id": "org-1",
		"roles":  []string{"admin"},
		"tenant": "tenant-1",
	}
	id, ext := MapKratosTraitsToTokens(traits)
	if id["email"] != "user@example.com" || id["given_name"] != "Alice" || id["family_name"] != "Doe" {
		t.Fatalf("unexpected id token mapping: %v", id)
	}
	if ext["org_id"] != "org-1" || ext["tenant"] != "tenant-1" {
		t.Fatalf("unexpected access ext mapping: %v", ext)
	}
}
