package cache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// Canonicalize converts JSON data to a canonical form with sorted keys.
// This ensures that identical data always produces the same JSON output,
// which is critical for cache key generation.
func Canonicalize(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return []byte("null"), nil
	}

	// Parse JSON into generic interface
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Canonicalize the value
	canonical := canonicalizeValue(v)

	// Marshal back to JSON (compact form, no whitespace)
	result, err := json.Marshal(canonical)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal canonical JSON: %w", err)
	}

	return result, nil
}

// CanonicalizeString canonicalizes a JSON string.
func CanonicalizeString(s string) (string, error) {
	result, err := Canonicalize([]byte(s))
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// canonicalizeValue recursively canonicalizes a value.
func canonicalizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return canonicalizeMap(val)
	case []interface{}:
		return canonicalizeSlice(val)
	case float64:
		// JSON numbers are always float64 when unmarshaled
		// Keep as-is for consistency
		return val
	default:
		// Primitives (string, bool, null) are already canonical
		return val
	}
}

// canonicalizeMap canonicalizes a map by sorting its keys.
func canonicalizeMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(m))

	// Get sorted keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Process in sorted order
	for _, k := range keys {
		result[k] = canonicalizeValue(m[k])
	}

	return result
}

// canonicalizeSlice canonicalizes a slice by canonicalizing each element.
func canonicalizeSlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))
	for i, v := range s {
		result[i] = canonicalizeValue(v)
	}
	return result
}

// Hash computes the SHA256 hash of canonical JSON data.
func Hash(data []byte) (string, error) {
	canonical, err := Canonicalize(data)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(canonical)
	return hex.EncodeToString(hash[:]), nil
}

// HashString computes the SHA256 hash of a canonical JSON string.
func HashString(s string) (string, error) {
	return Hash([]byte(s))
}

// HashObject computes the SHA256 hash of an object after canonical JSON encoding.
func HashObject(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal object: %w", err)
	}
	return Hash(data)
}

// Equal checks if two JSON values are equal after canonicalization.
func Equal(a, b []byte) (bool, error) {
	canonA, err := Canonicalize(a)
	if err != nil {
		return false, fmt.Errorf("failed to canonicalize first value: %w", err)
	}

	canonB, err := Canonicalize(b)
	if err != nil {
		return false, fmt.Errorf("failed to canonicalize second value: %w", err)
	}

	return bytes.Equal(canonA, canonB), nil
}

// Merge merges multiple JSON objects into one, with later values overriding earlier ones.
// The result is canonicalized.
func Merge(jsons ...[]byte) ([]byte, error) {
	result := make(map[string]interface{})

	for _, data := range jsons {
		if len(data) == 0 {
			continue
		}

		var obj map[string]interface{}
		if err := json.Unmarshal(data, &obj); err != nil {
			// Not an object, skip
			continue
		}

		// Merge keys
		for k, v := range obj {
			result[k] = v
		}
	}

	// Canonicalize the merged result
	canonical := canonicalizeMap(result)
	return json.Marshal(canonical)
}

// Diff compares two canonical JSON values and returns the differences.
type Diff struct {
	Added   map[string]interface{} `json:"added,omitempty"`
	Removed map[string]interface{} `json:"removed,omitempty"`
	Changed map[string]Change      `json:"changed,omitempty"`
}

type Change struct {
	Old interface{} `json:"old"`
	New interface{} `json:"new"`
}

// Compare compares two JSON values and returns their differences.
func Compare(a, b []byte) (*Diff, error) {
	var aObj, bObj map[string]interface{}

	if err := json.Unmarshal(a, &aObj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal first value: %w", err)
	}

	if err := json.Unmarshal(b, &bObj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal second value: %w", err)
	}

	diff := &Diff{
		Added:   make(map[string]interface{}),
		Removed: make(map[string]interface{}),
		Changed: make(map[string]Change),
	}

	// Find removed and changed keys
	for k, aVal := range aObj {
		if bVal, exists := bObj[k]; exists {
			// Key exists in both, check if changed
			aCanon := canonicalizeValue(aVal)
			bCanon := canonicalizeValue(bVal)

			aJSON, _ := json.Marshal(aCanon)
			bJSON, _ := json.Marshal(bCanon)

			if !bytes.Equal(aJSON, bJSON) {
				diff.Changed[k] = Change{Old: aVal, New: bVal}
			}
		} else {
			// Key removed
			diff.Removed[k] = aVal
		}
	}

	// Find added keys
	for k, bVal := range bObj {
		if _, exists := aObj[k]; !exists {
			diff.Added[k] = bVal
		}
	}

	return diff, nil
}

// Normalize ensures JSON data is in canonical form.
// Unlike Canonicalize, this preserves the structure but ensures consistent formatting.
func Normalize(data []byte) ([]byte, error) {
	// Parse and re-encode to ensure consistent formatting
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}

	// Use standard marshaling with sorted keys
	return json.Marshal(v)
}

// CompactCanonical removes all whitespace from canonical JSON.
func CompactCanonical(data []byte) ([]byte, error) {
	canonical, err := Canonicalize(data)
	if err != nil {
		return nil, err
	}

	// Already compact from Canonicalize, but ensure
	var buf bytes.Buffer
	if err := json.Compact(&buf, canonical); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// PrettyCanonical formats canonical JSON with indentation.
func PrettyCanonical(data []byte) ([]byte, error) {
	canonical, err := Canonicalize(data)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := json.Indent(&buf, canonical, "", "  "); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// IsCanonical checks if JSON data is already in canonical form.
func IsCanonical(data []byte) bool {
	canonical, err := Canonicalize(data)
	if err != nil {
		return false
	}
	return bytes.Equal(data, canonical)
}

// Must* helpers are provided only for tests (see canonical_testhelpers.go).
