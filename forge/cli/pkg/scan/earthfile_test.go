package scan

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/input-output-hk/catalyst-forge/tools/pkg/testutils"
	"github.com/input-output-hk/catalyst-forge/tools/pkg/walker"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
)

type MockFileSeeker struct {
	*strings.Reader
}

func (MockFileSeeker) Stat() (fs.FileInfo, error) {
	return MockFileInfo{}, nil
}

func (MockFileSeeker) Close() error {
	return nil
}

type MockFileInfo struct {
	fs.FileInfo
}

func (MockFileInfo) Name() string {
	return "Earthfile"
}

func NewMockFileSeeker(s string) MockFileSeeker {
	return MockFileSeeker{strings.NewReader(s)}
}

func TestScanEarthfiles(t *testing.T) {
	tests := []struct {
		name         string
		rootPath     string
		files        map[string]string
		expectedKeys []string
		expectErr    bool
	}{
		{
			name:     "single Earthfile",
			rootPath: "/tmp1",
			files: map[string]string{
				"/tmp1/Earthfile": "VERSION 0.8",
			},
			expectedKeys: []string{"/tmp1"},
			expectErr:    false,
		},
		{
			name:     "multiple Earthfiles",
			rootPath: "/",
			files: map[string]string{
				"/tmp1/Earthfile": "VERSION 0.8",
				"/tmp2/Earthfile": "VERSION 0.8",
			},
			expectedKeys: []string{"/tmp1", "/tmp2"},
			expectErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			testutils.SetupFS(t, fs, tt.files)

			walker := walker.NewCustomDefaultFSWalker(fs, nil)
			got, err := ScanEarthfiles(tt.rootPath, &walker, testutils.NewNoopLogger())
			if testutils.AssertError(t, err, tt.expectErr, "") {
				return
			}

			assert.Equal(t, maps.Keys(got), tt.expectedKeys)
		})
	}
}
