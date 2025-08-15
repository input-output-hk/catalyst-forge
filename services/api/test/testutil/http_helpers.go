//go:build integration

package testutil

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

// DoJSON performs a JSON HTTP request with optional client and headers.
// If client is nil, uses http.DefaultClient.
// This centralizes HTTP request handling for all integration tests.
func DoJSON(client *http.Client, method, urlStr string, headers map[string]string, body interface{}, out interface{}) (*http.Response, error) {
    if client == nil {
        client = http.DefaultClient
    }

    var reqBody io.Reader
    if body != nil {
        jsonBytes, err := json.Marshal(body)
        if err != nil {
            return nil, fmt.Errorf("failed to marshal request body: %w", err)
        }
        reqBody = bytes.NewReader(jsonBytes)
    }

    req, err := http.NewRequest(method, urlStr, reqBody)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    // Set headers
    if body != nil {
        req.Header.Set("Content-Type", "application/json")
    }
    for k, v := range headers {
        req.Header.Set(k, v)
    }

    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }

    // Read response body
    respBody, err := io.ReadAll(resp.Body)
    resp.Body.Close()
    if err != nil {
        return resp, fmt.Errorf("failed to read response body: %w", err)
    }

    // Only try to unmarshal if out is provided and response has content
    if out != nil && len(respBody) > 0 {
        if err := json.Unmarshal(respBody, out); err != nil {
            // Return the response even if unmarshal fails, for debugging
            return resp, fmt.Errorf("failed to unmarshal response: %w (body: %s)", err, string(respBody))
        }
    }

    // Check for HTTP errors
    if resp.StatusCode >= 400 {
        return resp, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
    }

    return resp, nil
}