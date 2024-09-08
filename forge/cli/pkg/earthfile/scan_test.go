package earthfile

import (
	"fmt"
	"io"
	"io/fs"
	"strings"
	"testing"

	"log/slog"

	"github.com/input-output-hk/catalyst-forge/tools/pkg/testutils"
	"github.com/input-output-hk/catalyst-forge/tools/pkg/walker"
	"github.com/input-output-hk/catalyst-forge/tools/pkg/walker/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		name           string
		files          map[string]string
		expectedResult map[string][]string
		callbackErr    error
		walkErr        error
	}{
		{
			name: "one earthfile",
			files: map[string]string{
				"/tmp1/Earthfile": `
VERSION 0.7

foo1:
  LET foo = bar

foo2:
  LET foo = bar
`,
			},
			expectedResult: map[string][]string{
				"/tmp1": {"foo1", "foo2"},
			},
			callbackErr: nil,
			walkErr:     nil,
		},
		{
			name: "multiple earthfiles",
			files: map[string]string{
				"/tmp1/Earthfile": `
VERSION 0.7

foo1:
  LET foo = bar
`,
				"/tmp2/Earthfile": `
VERSION 0.7

foo2:
  LET foo = bar
`,
			},
			expectedResult: map[string][]string{
				"/tmp1": {"foo1"},
				"/tmp2": {"foo2"},
			},
			callbackErr: nil,
			walkErr:     nil,
		},
		{
			name: "callback error",
			files: map[string]string{
				"/tmp1/Earthfile": `
VERSION 0.7

foo1:
  LET foo = bar
`,
			},
			expectedResult: map[string][]string{},
			callbackErr:    fmt.Errorf("callback error"),
			walkErr:        nil,
		},
		{
			name: "walk error",
			files: map[string]string{
				"/tmp1/Earthfile": `
VERSION 0.7

foo1:
  LET foo = bar
`,
			},
			expectedResult: map[string][]string{},
			callbackErr:    nil,
			walkErr:        fmt.Errorf("walk error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			walker := &mocks.WalkerMock{
				WalkFunc: func(rootPath string, callback walker.WalkerCallback) error {
					for path, content := range tt.files {
						err := callback(path, walker.FileTypeFile, func() (walker.FileSeeker, error) {
							return NewMockFileSeeker(content), tt.callbackErr
						})

						if err != nil {
							return err
						}
					}

					return tt.walkErr
				},
			}

			result, err := ScanEarthfiles("/", walker, slog.New(slog.NewTextHandler(io.Discard, nil)))
			if testutils.AssertError(t, err, tt.callbackErr != nil || tt.walkErr != nil, "") {
				return
			}
			require.Equal(t, len(tt.expectedResult), len(result))

			for path, targets := range tt.expectedResult {
				require.Equal(t, len(targets), len(result[path].Targets()))

				for i, target := range targets {
					assert.Equal(t, target, result[path].Targets()[i])
				}
			}
		})
	}
}
