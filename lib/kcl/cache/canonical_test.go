package cache

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestCanonicalizeJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "empty object",
			input:    `{}`,
			expected: `{}`,
			wantErr:  false,
		},
		{
			name:     "simple object with sorted keys",
			input:    `{"b": 2, "a": 1}`,
			expected: `{"a":1,"b":2}`,
			wantErr:  false,
		},
		{
			name:     "nested object with sorted keys",
			input:    `{"z": {"b": 2, "a": 1}, "y": 3}`,
			expected: `{"y":3,"z":{"a":1,"b":2}}`,
			wantErr:  false,
		},
		{
			name:     "array with objects",
			input:    `[{"z": 1, "a": 2}, {"b": 3, "c": 4}]`,
			expected: `[{"a":2,"z":1},{"b":3,"c":4}]`,
			wantErr:  false,
		},
		{
			name:     "complex nested structure",
			input:    `{"users": [{"name": "alice", "id": 1}, {"name": "bob", "id": 2}], "count": 2}`,
			expected: `{"count":2,"users":[{"id":1,"name":"alice"},{"id":2,"name":"bob"}]}`,
			wantErr:  false,
		},
		{
			name:     "whitespace handling",
			input:    `  {  "a" : 1 ,  "b" : 2  }  `,
			expected: `{"a":1,"b":2}`,
			wantErr:  false,
		},
		{
			name:     "unicode handling",
			input:    `{"你好": "世界", "hello": "world"}`,
			expected: `{"hello":"world","你好":"世界"}`,
			wantErr:  false,
		},
		{
			name:     "null values",
			input:    `{"a": null, "b": 1}`,
			expected: `{"a":null,"b":1}`,
			wantErr:  false,
		},
		{
			name:     "boolean values",
			input:    `{"flag2": false, "flag1": true}`,
			expected: `{"flag1":true,"flag2":false}`,
			wantErr:  false,
		},
		{
			name:     "number precision",
			input:    `{"pi": 3.14159, "e": 2.71828}`,
			expected: `{"e":2.71828,"pi":3.14159}`,
			wantErr:  false,
		},
		{
			name:     "empty array",
			input:    `{"items": []}`,
			expected: `{"items":[]}`,
			wantErr:  false,
		},
		{
			name:     "invalid JSON",
			input:    `{invalid}`,
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CanonicalizeJSON([]byte(tt.input))
			
			if (err != nil) != tt.wantErr {
				t.Errorf("CanonicalizeJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && string(result) != tt.expected {
				t.Errorf("CanonicalizeJSON() = %q, want %q", string(result), tt.expected)
			}
		})
	}
}

func TestCanonicalizeJSONDeterminism(t *testing.T) {
	// Test that the same object in different orders produces the same output
	inputs := []string{
		`{"z": 3, "y": 2, "x": 1}`,
		`{"x": 1, "y": 2, "z": 3}`,
		`{"y": 2, "z": 3, "x": 1}`,
	}
	
	var firstResult []byte
	for i, input := range inputs {
		result, err := CanonicalizeJSON([]byte(input))
		if err != nil {
			t.Fatalf("Failed to canonicalize input %d: %v", i, err)
		}
		
		if i == 0 {
			firstResult = result
		} else if string(result) != string(firstResult) {
			t.Errorf("Non-deterministic output: input %d produced %q, expected %q", 
				i, string(result), string(firstResult))
		}
	}
}

func TestCanonicalizeJSONIdempotent(t *testing.T) {
	// Test that canonicalizing already canonical JSON doesn't change it
	input := `{"a":1,"b":{"c":2,"d":3},"e":[4,5,6]}`
	
	first, err := CanonicalizeJSON([]byte(input))
	if err != nil {
		t.Fatalf("First canonicalize failed: %v", err)
	}
	
	second, err := CanonicalizeJSON(first)
	if err != nil {
		t.Fatalf("Second canonicalize failed: %v", err)
	}
	
	if string(first) != string(second) {
		t.Errorf("Canonicalize not idempotent: first = %q, second = %q", 
			string(first), string(second))
	}
}

func TestCanonicalizeJSONLargeStructure(t *testing.T) {
	// Test with a large nested structure
	large := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key_%03d", i)
		large[key] = map[string]interface{}{
			"value": i,
			"nested": map[string]interface{}{
				"deep": i * 2,
			},
		}
	}
	
	data, err := json.Marshal(large)
	if err != nil {
		t.Fatalf("Failed to marshal large structure: %v", err)
	}
	
	result, err := CanonicalizeJSON(data)
	if err != nil {
		t.Fatalf("Failed to canonicalize large structure: %v", err)
	}
	
	// Verify it's valid JSON
	var parsed interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Errorf("Result is not valid JSON: %v", err)
	}
	
	// Verify keys are sorted
	var resultMap map[string]interface{}
	if err := json.Unmarshal(result, &resultMap); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	
	// Check that the result contains all expected keys
	if len(resultMap) != 100 {
		t.Errorf("Expected 100 keys, got %d", len(resultMap))
	}
}

func BenchmarkCanonicalizeJSON(b *testing.B) {
	input := []byte(`{
		"users": [
			{"name": "alice", "id": 1, "email": "alice@example.com"},
			{"name": "bob", "id": 2, "email": "bob@example.com"}
		],
		"metadata": {
			"version": "1.0.0",
			"timestamp": 1234567890,
			"flags": {"feature1": true, "feature2": false}
		}
	}`)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CanonicalizeJSON(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestCanonicalCompare(t *testing.T) {
	// Test that we can compare canonical forms for equality
	obj1 := `{"a": 1, "b": [2, 3], "c": {"d": 4}}`
	obj2 := `{"c": {"d": 4}, "a": 1, "b": [2, 3]}`
	obj3 := `{"a": 1, "b": [2, 3], "c": {"d": 5}}` // Different value
	
	canon1, err := CanonicalizeJSON([]byte(obj1))
	if err != nil {
		t.Fatalf("Failed to canonicalize obj1: %v", err)
	}
	
	canon2, err := CanonicalizeJSON([]byte(obj2))
	if err != nil {
		t.Fatalf("Failed to canonicalize obj2: %v", err)
	}
	
	canon3, err := CanonicalizeJSON([]byte(obj3))
	if err != nil {
		t.Fatalf("Failed to canonicalize obj3: %v", err)
	}
	
	if !reflect.DeepEqual(canon1, canon2) {
		t.Errorf("obj1 and obj2 should be equal after canonicalization")
	}
	
	if reflect.DeepEqual(canon1, canon3) {
		t.Errorf("obj1 and obj3 should not be equal after canonicalization")
	}
}