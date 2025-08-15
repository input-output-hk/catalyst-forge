//go:build !wasm
// +build !wasm

package engine

import (
	"testing"
)

func TestNativeEngineVersion(t *testing.T) {
	engine := NewNativeEngine()
	
	// Check engine version
	version := engine.Version()
	if version == "" {
		t.Error("Engine version should not be empty")
	}
	if version != "native-v1.0.0+cgo" {
		t.Errorf("Expected engine version 'native-v1.0.0+cgo', got %s", version)
	}
	
	// Check KCL version
	kclVersion := engine.KCLVersion()
	if kclVersion == "" {
		t.Error("KCL version should not be empty")
	}
	if kclVersion == "unknown" {
		t.Log("Warning: Could not retrieve actual KCL version")
	} else {
		t.Logf("KCL version: %s", kclVersion)
	}
	
	// Validate engine
	if err := engine.Validate(); err != nil {
		t.Errorf("Engine validation failed: %v", err)
	}
	
	// Clean up
	if err := engine.Close(); err != nil {
		t.Errorf("Engine close failed: %v", err)
	}
}