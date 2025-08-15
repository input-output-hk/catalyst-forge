package client

import (
	"bytes"
	"encoding/json"
	"io"
)

// EncodeJSON encodes v into a replayable JSON body suitable for WithBody methods.
// It returns content type and an io.ReadSeeker for safe retry.
func EncodeJSON(v any) (string, io.ReadSeeker) {
	b, _ := json.Marshal(v)
	return "application/json", bytes.NewReader(b)
}
