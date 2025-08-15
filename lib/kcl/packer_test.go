package kcl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateModuleStructure_MissingKclMod(t *testing.T) {
	dir := t.TempDir()
	if err := ValidateModuleStructure(dir); err == nil {
		t.Fatalf("expected error for missing kcl.mod")
	}
}

func TestValidateModuleStructure_Valid(t *testing.T) {
	dir := t.TempDir()
	// Write minimal kcl.mod
	content := []byte("name = \"foo\"\nversion = \"1.2.3\"\n")
	if err := os.WriteFile(filepath.Join(dir, "kcl.mod"), content, 0644); err != nil {
		t.Fatalf("write kcl.mod: %v", err)
	}
	if err := ValidateModuleStructure(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateModuleStructure_InvalidMissingVersion(t *testing.T) {
	dir := t.TempDir()
	content := []byte("name = \"foo\"\n")
	if err := os.WriteFile(filepath.Join(dir, "kcl.mod"), content, 0644); err != nil {
		t.Fatalf("write kcl.mod: %v", err)
	}
	if err := ValidateModuleStructure(dir); err == nil {
		t.Fatalf("expected error for missing version")
	}
}
