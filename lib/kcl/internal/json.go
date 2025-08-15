package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ReadJSON reads and unmarshals a JSON file.
func ReadJSON(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

// WriteJSON marshals and writes data to a JSON file.
func WriteJSON(path string, v interface{}, indent bool) error {
	var data []byte
	var err error

	if indent {
		data, err = json.MarshalIndent(v, "", "  ")
	} else {
		data, err = json.Marshal(v)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// PrettyJSON formats JSON data with indentation.
func PrettyJSON(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return nil, fmt.Errorf("failed to format JSON: %w", err)
	}
	return buf.Bytes(), nil
}

// CompactJSON removes whitespace from JSON data.
func CompactJSON(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to compact JSON: %w", err)
	}
	return buf.Bytes(), nil
}

// ValidateJSON checks if data is valid JSON.
func ValidateJSON(data []byte) error {
	var v interface{}
	return json.Unmarshal(data, &v)
}

// MergeJSON merges multiple JSON objects into one.
func MergeJSON(objects ...[]byte) ([]byte, error) {
	result := make(map[string]interface{})

	for _, data := range objects {
		if len(data) == 0 {
			continue
		}

		var obj map[string]interface{}
		if err := json.Unmarshal(data, &obj); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		// Merge keys (later values override)
		for k, v := range obj {
			result[k] = v
		}
	}

	return json.Marshal(result)
}

// StreamJSON reads JSON from a reader and unmarshals it.
func StreamJSON(r io.Reader, v interface{}) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(v)
}

// StreamWriteJSON writes JSON to a writer.
func StreamWriteJSON(w io.Writer, v interface{}, indent bool) error {
	encoder := json.NewEncoder(w)
	if indent {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(v)
}

// CloneJSON creates a deep copy of a JSON-serializable value.
func CloneJSON(src, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("failed to marshal source: %w", err)
	}

	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("failed to unmarshal to destination: %w", err)
	}

	return nil
}

// GetJSONField extracts a field from JSON data without full unmarshaling.
func GetJSONField(data []byte, field string) (json.RawMessage, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	value, exists := obj[field]
	if !exists {
		return nil, fmt.Errorf("field %s not found", field)
	}

	return value, nil
}

// SetJSONField sets a field in JSON data.
func SetJSONField(data []byte, field string, value interface{}) ([]byte, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		// If not an object, create new one
		obj = make(map[string]interface{})
	}

	obj[field] = value
	return json.Marshal(obj)
}

// RemoveJSONField removes a field from JSON data.
func RemoveJSONField(data []byte, field string) ([]byte, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	delete(obj, field)
	return json.Marshal(obj)
}

// IsJSONObject checks if data is a JSON object.
func IsJSONObject(data []byte) bool {
	data = bytes.TrimSpace(data)
	return len(data) > 0 && data[0] == '{'
}

// IsJSONArray checks if data is a JSON array.
func IsJSONArray(data []byte) bool {
	data = bytes.TrimSpace(data)
	return len(data) > 0 && data[0] == '['
}