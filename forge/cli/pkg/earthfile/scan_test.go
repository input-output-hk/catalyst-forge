package earthfile

import (
	"fmt"
	"io"
	"testing"

	"log/slog"

	"github.com/input-output-hk/catalyst-forge/forge/cli/internal/testutils/mocks"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/walker"
)

func TestScanEarthfiles(t *testing.T) {
	tests := []struct {
		callbackErr    error
		walkErr        error
		files          map[string]string
		expectedResult map[string][]string
		name           string
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
				"/tmp1/Earthfile": {"foo1", "foo2"},
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
				"/tmp1/Earthfile": {"foo1"},
				"/tmp2/Earthfile": {"foo2"},
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
			walker := &walker.WalkerMock{
				WalkFunc: func(rootPath string, callback walker.WalkerCallback) error {
					for path, content := range tt.files {
						err := callback(path, walker.FileTypeFile, func() (walker.FileSeeker, error) {
							return mocks.NewMockFileSeeker(content), tt.callbackErr
						})

						if err != nil {
							return err
						}
					}

					return tt.walkErr
				},
			}
			result, err := ScanEarthfiles("", walker, slog.New(slog.NewTextHandler(io.Discard, nil)))

			if tt.callbackErr != nil && err == nil {
				t.Error("expected error, got nil")
			} else if tt.walkErr != nil && err == nil {
				t.Error("expected error, got nil")
			} else if tt.callbackErr == nil && tt.walkErr == nil && err != nil {
				t.Errorf("expected no error, got %v", err)
			} else {
				if err != nil {
					return
				}
			}

			if len(result) != len(tt.expectedResult) {
				t.Errorf("expected %d earthfiles, got %d", len(tt.expectedResult), len(result))
			}

			for path, targets := range tt.expectedResult {
				if len(result[path].Targets()) != len(targets) {
					t.Errorf("expected %d targets for %s, got %d", len(targets), path, len(result[path].Targets()))
				}

				for i, target := range targets {
					if result[path].Targets()[i] != target {
						t.Errorf("expected target %s at index %d, got %s", target, i, result[path].Targets()[i])
					}
				}
			}
		})
	}
}
