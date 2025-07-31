package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2"
)

func TestDefaultGitHubJWKSCacher(t *testing.T) {
	tests := []struct {
		name     string
		response interface{}
		files    map[string]interface{}
		validate func(t *testing.T, cacher *DefaultGitHubJWKSCacher, err error)
	}{
		{
			name: "valid response",
			response: jose.JSONWebKeySet{
				Keys: []jose.JSONWebKey{
					{
						KeyID: "test-key-1",
						Key:   []byte("test-key-data"),
					},
				},
			},
			files: map[string]interface{}{},
			validate: func(t *testing.T, cacher *DefaultGitHubJWKSCacher, err error) {
				require.NoError(t, err)
				jwks := cacher.JWKS()
				require.NotNil(t, jwks)
				require.Len(t, jwks.Keys, 1)
				require.Equal(t, "test-key-1", jwks.Keys[0].KeyID)
			},
		},
		{
			name: "with existing cache file",
			files: map[string]interface{}{
				"/test/jwks.json": jose.JSONWebKeySet{
					Keys: []jose.JSONWebKey{
						{
							KeyID: "test-key-1",
							Key:   []byte("test-key-data"),
						},
					},
				},
			},
			validate: func(t *testing.T, cacher *DefaultGitHubJWKSCacher, err error) {
				require.NoError(t, err)
				jwks := cacher.JWKS()
				require.NotNil(t, jwks)
				require.Len(t, jwks.Keys, 1)
				require.Equal(t, "test-key-1", jwks.Keys[0].KeyID)
			},
		},
		{
			name: "with no keys in cache",
			files: map[string]interface{}{
				"/test/jwks.json": jose.JSONWebKeySet{},
			},
			validate: func(t *testing.T, cacher *DefaultGitHubJWKSCacher, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "with invalid json in cache",
			files: map[string]interface{}{
				"/test/jwks.json": "invalid json",
			},
			validate: func(t *testing.T, cacher *DefaultGitHubJWKSCacher, err error) {
				require.Error(t, err)
			},
		},
		{
			name:     "with invalid json response",
			response: "invalid json",
			validate: func(t *testing.T, cacher *DefaultGitHubJWKSCacher, err error) {
				require.Error(t, err)
			},
		},
		{
			name:     "with no keys in response",
			response: jose.JSONWebKeySet{},
			validate: func(t *testing.T, cacher *DefaultGitHubJWKSCacher, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			fs := billy.NewInMemoryFs()
			for path, content := range tt.files {
				bytes, err := json.Marshal(content)
				require.NoError(t, err)

				err = fs.WriteFile(path, bytes, 0644)
				require.NoError(t, err)
			}

			gc := DefaultGitHubJWKSCacher{
				cachePath: "/test/jwks.json",
				ttl:       1 * time.Hour,
				client:    &http.Client{Timeout: 10 * time.Second},
				fs:        fs,
				jwksURL:   server.URL,
			}
			defer gc.Stop()

			err := gc.Start(context.Background())
			tt.validate(t, &gc, err)
		})
	}
}
