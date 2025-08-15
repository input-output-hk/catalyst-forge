package kcl

import (
	"encoding/json"
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/kcl/engine"
)

func TestComputeIntentHashDeterminism(t *testing.T) {
	// Test that the same inputs produce the same hash
	inputs := []struct {
		digest  string
		values  map[string]interface{}
		ctx     map[string]interface{}
		engine  EngineKind
		mockEng engine.Engine
	}{
		{
			digest: "sha256:abcd1234",
			values: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
			ctx:     map[string]interface{}{"env": "production"},
			engine:  EngineNative,
			mockEng: &mockTestEngine{version: "native-v1.0.0", kclVersion: "0.10.8"},
		},
	}

	for _, input := range inputs {
		valuesJSON, _ := json.Marshal(input.values)
		ctxJSON, _ := json.Marshal(input.ctx)

		// Compute hash multiple times
		hashes := make([]string, 10)
		for i := 0; i < 10; i++ {
			hash, err := computeIntentHash(ProfileCompat, input.digest, input.engine, input.mockEng, valuesJSON, ctxJSON)
			if err != nil {
				t.Fatalf("Failed to compute intent hash: %v", err)
			}
			hashes[i] = hash
		}

		// All hashes should be the same
		firstHash := hashes[0]
		for i, hash := range hashes {
			if hash != firstHash {
				t.Errorf("Non-deterministic hash at iteration %d: got %q, expected %q",
					i, hash, firstHash)
			}
		}
	}
}

func TestComputeIntentHashSensitivity(t *testing.T) {
	// Test that different inputs produce different hashes
	baseDigest := "sha256:abcd1234"
	baseValues := []byte(`{"key": "value"}`)
	baseCtx := []byte(`{"env": "prod"}`)
	mockEng := &mockTestEngine{version: "native-v1.0.0", kclVersion: "0.10.8"}

	baseHash, err := computeIntentHash(ProfileCompat, baseDigest, EngineNative, mockEng, baseValues, baseCtx)
	if err != nil {
		t.Fatalf("Failed to compute base hash: %v", err)
	}

	tests := []struct {
		name   string
		digest string
		values []byte
		ctx    []byte
		engine EngineKind
	}{
		{
			name:   "different digest",
			digest: "sha256:different",
			values: baseValues,
			ctx:    baseCtx,
			engine: EngineNative,
		},
		{
			name:   "different values",
			digest: baseDigest,
			values: []byte(`{"key": "different"}`),
			ctx:    baseCtx,
			engine: EngineNative,
		},
		{
			name:   "different context",
			digest: baseDigest,
			values: baseValues,
			ctx:    []byte(`{"env": "dev"}`),
			engine: EngineNative,
		},
		{
			name:   "different engine",
			digest: baseDigest,
			values: baseValues,
			ctx:    baseCtx,
			engine: EngineWASM,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := computeIntentHash(ProfileCompat, tt.digest, tt.engine, mockEng, tt.values, tt.ctx)
			if err != nil {
				t.Fatalf("Failed to compute hash: %v", err)
			}

			if hash == baseHash {
				t.Errorf("Hash should be different for %s, but got the same: %q", tt.name, hash)
			}
		})
	}
}

func TestComputeIntentHashOrderIndependence(t *testing.T) {
	// Test that JSON field order doesn't affect the hash
	digest := "sha256:abcd1234"
	mockEng := &mockTestEngine{version: "native-v1.0.0", kclVersion: "0.10.8"}

	// Same data, different field order
	values1 := []byte(`{"a": 1, "b": 2, "c": 3}`)
	values2 := []byte(`{"c": 3, "a": 1, "b": 2}`)
	values3 := []byte(`{"b": 2, "c": 3, "a": 1}`)

	ctx := []byte(`{}`)

	hash1, err := computeIntentHash(ProfileCompat, digest, EngineNative, mockEng, values1, ctx)
	if err != nil {
		t.Fatalf("Failed to compute hash1: %v", err)
	}

	hash2, err := computeIntentHash(ProfileCompat, digest, EngineNative, mockEng, values2, ctx)
	if err != nil {
		t.Fatalf("Failed to compute hash2: %v", err)
	}

	hash3, err := computeIntentHash(ProfileCompat, digest, EngineNative, mockEng, values3, ctx)
	if err != nil {
		t.Fatalf("Failed to compute hash3: %v", err)
	}

	if hash1 != hash2 || hash1 != hash3 {
		t.Errorf("Hashes should be the same regardless of field order: %q, %q, %q",
			hash1, hash2, hash3)
	}
}

func TestComputeIntentHashWhitespaceIndependence(t *testing.T) {
	// Test that JSON whitespace doesn't affect the hash
	digest := "sha256:abcd1234"
	mockEng := &mockTestEngine{version: "native-v1.0.0", kclVersion: "0.10.8"}

	// Same data, different whitespace
	values1 := []byte(`{"key":"value","num":42}`)
	values2 := []byte(`{  "key" : "value" , "num" : 42  }`)
	values3 := []byte(`{
		"key": "value",
		"num": 42
	}`)

	ctx := []byte(`{}`)

	hash1, err := computeIntentHash(ProfileCompat, digest, EngineNative, mockEng, values1, ctx)
	if err != nil {
		t.Fatalf("Failed to compute hash1: %v", err)
	}

	hash2, err := computeIntentHash(ProfileCompat, digest, EngineNative, mockEng, values2, ctx)
	if err != nil {
		t.Fatalf("Failed to compute hash2: %v", err)
	}

	hash3, err := computeIntentHash(ProfileCompat, digest, EngineNative, mockEng, values3, ctx)
	if err != nil {
		t.Fatalf("Failed to compute hash3: %v", err)
	}

	if hash1 != hash2 || hash1 != hash3 {
		t.Errorf("Hashes should be the same regardless of whitespace: %q, %q, %q",
			hash1, hash2, hash3)
	}
}

func TestComputeIntentHashLength(t *testing.T) {
	// Test that hash length is consistent
	mockEng := &mockTestEngine{version: "native-v1.0.0", kclVersion: "1.0.0"}
	hash, err := computeIntentHash(
		ProfileCompat,
		"sha256:test",
		EngineNative,
		mockEng,
		[]byte(`{}`),
		[]byte(`{}`),
	)

	if err != nil {
		t.Fatalf("Failed to compute hash: %v", err)
	}

	// SHA-256 produces 64 character hex strings
	if len(hash) != 64 {
		t.Errorf("Expected hash length of 64, got %d: %q", len(hash), hash)
	}
}

func TestComputeIntentHashNestedStructures(t *testing.T) {
	// Test with complex nested structures
	values := map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{
				"name": "alice",
				"id":   1,
				"tags": []string{"admin", "user"},
			},
			map[string]interface{}{
				"name": "bob",
				"id":   2,
				"tags": []string{"user"},
			},
		},
		"config": map[string]interface{}{
			"feature_flags": map[string]bool{
				"feature1": true,
				"feature2": false,
			},
		},
	}

	valuesJSON, err := json.Marshal(values)
	if err != nil {
		t.Fatalf("Failed to marshal values: %v", err)
	}

	hash1, err := computeIntentHash(
		ProfileCompat,
		"sha256:complex",
		EngineNative,
		&mockTestEngine{version: "native-v1.0.0", kclVersion: "1.0.0"},
		valuesJSON,
		[]byte(`{}`),
	)
	if err != nil {
		t.Fatalf("Failed to compute hash: %v", err)
	}

	// Reorder the nested structure
	values2 := map[string]interface{}{
		"config": map[string]interface{}{
			"feature_flags": map[string]bool{
				"feature2": false,
				"feature1": true,
			},
		},
		"users": []interface{}{
			map[string]interface{}{
				"id":   1,
				"tags": []string{"admin", "user"},
				"name": "alice",
			},
			map[string]interface{}{
				"tags": []string{"user"},
				"id":   2,
				"name": "bob",
			},
		},
	}

	values2JSON, err := json.Marshal(values2)
	if err != nil {
		t.Fatalf("Failed to marshal values2: %v", err)
	}

	hash2, err := computeIntentHash(
		ProfileCompat,
		"sha256:complex",
		EngineNative,
		&mockTestEngine{version: "native-v1.0.0", kclVersion: "1.0.0"},
		values2JSON,
		[]byte(`{}`),
	)
	if err != nil {
		t.Fatalf("Failed to compute hash2: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("Hashes should be the same for reordered nested structures")
	}
}

func BenchmarkComputeIntentHash(b *testing.B) {
	digest := "sha256:benchmark"
	values := []byte(`{"key1": "value1", "key2": 42, "key3": true}`)
	ctx := []byte(`{"env": "production", "region": "us-west-2"}`)
	mockEng := &mockTestEngine{version: "native-v1.0.0", kclVersion: "0.10.8"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := computeIntentHash(ProfileCompat, digest, EngineNative, mockEng, values, ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
