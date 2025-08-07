package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		setupFunc func(*testing.T) string
		validate  func(*testing.T, *Server, error)
	}{
		{
			name: "server_without_cache",
			config: Config{
				Port:   8080,
				Logger: testutils.NewNoopLogger(),
			},
			validate: func(t *testing.T, server *Server, err error) {
				require.NoError(t, err)
				assert.NotNil(t, server)
				assert.Equal(t, 8080, server.config.Port)
			},
		},
		{
			name: "server_with_valid_cache_path",
			setupFunc: func(t *testing.T) string {
				tmpDir, err := os.MkdirTemp("", "renderer-cache-test")
				require.NoError(t, err)
				t.Cleanup(func() { os.RemoveAll(tmpDir) })
				return tmpDir
			},
			validate: func(t *testing.T, server *Server, err error) {
				require.NoError(t, err)
				assert.NotNil(t, server)
			},
		},
		{
			name: "server_with_nonexistent_cache_path",
			setupFunc: func(t *testing.T) string {
				tmpDir, err := os.MkdirTemp("", "renderer-cache-parent")
				require.NoError(t, err)
				t.Cleanup(func() { os.RemoveAll(tmpDir) })
				return filepath.Join(tmpDir, "cache", "subdir")
			},
			validate: func(t *testing.T, server *Server, err error) {
				require.NoError(t, err)
				assert.NotNil(t, server)
			},
		},
		{
			name: "server_with_invalid_cache_path_file_exists",
			setupFunc: func(t *testing.T) string {
				tmpDir, err := os.MkdirTemp("", "renderer-cache-test")
				require.NoError(t, err)
				t.Cleanup(func() { os.RemoveAll(tmpDir) })
				
				// Create a file instead of directory
				filePath := filepath.Join(tmpDir, "not-a-directory")
				f, err := os.Create(filePath)
				require.NoError(t, err)
				f.Close()
				
				return filePath
			},
			validate: func(t *testing.T, server *Server, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "exists but is not a directory")
				assert.Nil(t, server)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.config
			config.Logger = testutils.NewNoopLogger()
			
			if tt.setupFunc != nil {
				config.CachePath = tt.setupFunc(t)
			}

			server, err := NewServer(config)
			tt.validate(t, server, err)
		})
	}
}

func TestInitializeCacheDirectory(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*testing.T) string
		validate  func(*testing.T, string, error)
	}{
		{
			name: "create_new_directory",
			setupFunc: func(t *testing.T) string {
				tmpDir, err := os.MkdirTemp("", "renderer-cache-parent")
				require.NoError(t, err)
				t.Cleanup(func() { os.RemoveAll(tmpDir) })
				return filepath.Join(tmpDir, "new-cache-dir")
			},
			validate: func(t *testing.T, cachePath string, err error) {
				require.NoError(t, err)
				
				// Verify directory was created
				info, statErr := os.Stat(cachePath)
				require.NoError(t, statErr)
				assert.True(t, info.IsDir())
			},
		},
		{
			name: "use_existing_directory",
			setupFunc: func(t *testing.T) string {
				tmpDir, err := os.MkdirTemp("", "renderer-cache-existing")
				require.NoError(t, err)
				t.Cleanup(func() { os.RemoveAll(tmpDir) })
				return tmpDir
			},
			validate: func(t *testing.T, cachePath string, err error) {
				require.NoError(t, err)
				
				// Verify directory still exists
				info, statErr := os.Stat(cachePath)
				require.NoError(t, statErr)
				assert.True(t, info.IsDir())
			},
		},
		{
			name: "fail_on_file_instead_of_directory",
			setupFunc: func(t *testing.T) string {
				tmpDir, err := os.MkdirTemp("", "renderer-cache-test")
				require.NoError(t, err)
				t.Cleanup(func() { os.RemoveAll(tmpDir) })
				
				// Create a file instead of directory
				filePath := filepath.Join(tmpDir, "not-a-directory")
				f, err := os.Create(filePath)
				require.NoError(t, err)
				f.Close()
				
				return filePath
			},
			validate: func(t *testing.T, cachePath string, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "exists but is not a directory")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cachePath := tt.setupFunc(t)
			logger := testutils.NewNoopLogger()
			
			err := initializeCacheDirectory(cachePath, logger)
			tt.validate(t, cachePath, err)
		})
	}
}