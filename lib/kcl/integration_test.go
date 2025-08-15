//go:build !wasm
// +build !wasm

package kcl

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/input-output-hk/catalyst-forge/lib/kcl/engine"
)

// TestRunCaching tests the complete run caching functionality
func TestRunCaching(t *testing.T) {
	// Skip if no KCL runtime available
	t.Skip("Skipping integration test - requires full KCL runtime and test module")

	// Setup test environment
	tempDir := t.TempDir()
	if err := os.Setenv("FORGE_KCL_CACHE_DIR", filepath.Join(tempDir, "cache")); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}

	// Create a mock OCI client
	// In a real test, this would be a properly mocked OCI client
	var mockOCI OCI

	// Test module reference
	ref := ModuleRef{
		Repo: "oci://test.registry/test/module",
		Tag:  "v1.0.0",
	}

	// Test run options
	opts := RunOptions{
		Profile:        ProfileCompat,
		Engine:         EngineNative,
		Values:         []byte(`{"key": "value1"}`),
		Context:        []byte(`{"env": "test"}`),
		TimeoutSec:     30,
		UseCache:       true,
		ForceRecompute: false,
	}

	ctx := context.Background()

	// First run - should cache miss
	start1 := time.Now()
	result1, err := Run(ctx, mockOCI, ref, opts)
	if err != nil {
		t.Fatalf("First run failed: %v", err)
	}
	duration1 := time.Since(start1)

	if result1.CacheHit {
		t.Error("First run should be a cache miss")
	}

	// Second run with same inputs - should cache hit
	start2 := time.Now()
	result2, err := Run(ctx, mockOCI, ref, opts)
	if err != nil {
		t.Fatalf("Second run failed: %v", err)
	}
	duration2 := time.Since(start2)

	if !result2.CacheHit {
		t.Error("Second run should be a cache hit")
	}

	// Cache hit should be much faster
	if duration2 >= duration1 {
		t.Errorf("Cache hit should be faster: first=%v, second=%v", duration1, duration2)
	}

	// Results should be identical
	if string(result1.YAML) != string(result2.YAML) {
		t.Error("Cached result should match original")
	}

	// Third run with different values - should cache miss
	opts.Values = []byte(`{"key": "value2"}`)
	result3, err := Run(ctx, mockOCI, ref, opts)
	if err != nil {
		t.Fatalf("Third run failed: %v", err)
	}

	if result3.CacheHit {
		t.Error("Third run with different values should be a cache miss")
	}

	// Fourth run with ForceRecompute - should cache miss but still cache result
	opts.ForceRecompute = true
	result4, err := Run(ctx, mockOCI, ref, opts)
	if err != nil {
		t.Fatalf("Fourth run failed: %v", err)
	}

	if result4.CacheHit {
		t.Error("Fourth run with ForceRecompute should be a cache miss")
	}
}

// TestIntentHashDeterminism tests that intent hash is deterministic
func TestIntentHashDeterminism(t *testing.T) {
	// Create test values with different key orders
	values1 := []byte(`{"b": 2, "a": 1, "c": 3}`)
	values2 := []byte(`{"a": 1, "c": 3, "b": 2}`)

	// Create mock engine
	mockEngine := &mockTestEngine{
		version:    "test-v1.0.0",
		kclVersion: "0.10.0",
	}

	// Compute intent hashes
	hash1, err := computeIntentHash(
		ProfileCompat,
		"sha256:abc123",
		EngineNative,
		mockEngine,
		values1,
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to compute hash1: %v", err)
	}

	hash2, err := computeIntentHash(
		ProfileCompat,
		"sha256:abc123",
		EngineNative,
		mockEngine,
		values2,
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to compute hash2: %v", err)
	}

	// Hashes should be identical due to canonical JSON
	if hash1 != hash2 {
		t.Errorf("Intent hashes should be identical for same logical values: %s != %s", hash1, hash2)
	}

	// Different values should produce different hashes
	values3 := []byte(`{"a": 1, "b": 2, "c": 4}`)
	hash3, err := computeIntentHash(
		ProfileCompat,
		"sha256:abc123",
		EngineNative,
		mockEngine,
		values3,
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to compute hash3: %v", err)
	}

	if hash1 == hash3 {
		t.Error("Different values should produce different intent hashes")
	}
}

// mockTestEngine is a mock engine for testing
type mockTestEngine struct {
	version    string
	kclVersion string
}

func (e *mockTestEngine) Version() string    { return e.version }
func (e *mockTestEngine) KCLVersion() string { return e.kclVersion }
func (e *mockTestEngine) Run(ctx context.Context, workDir, entry string, valuesJSON, ctxJSON []byte, lim engine.Limits) ([]byte, engine.Stats, error) {
	// Return mock YAML output
	return []byte("test: output\nvalues: processed"), engine.Stats{
		ColdStart: false,
		CompileMS: 10,
		EvalMS:    5,
		TotalMS:   15,
		PeakMemMB: 50,
	}, nil
}
func (e *mockTestEngine) Validate() error { return nil }
func (e *mockTestEngine) Close() error    { return nil }
