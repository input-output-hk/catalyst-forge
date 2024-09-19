package git

import (
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/tools/pkg/testutils"
	"github.com/input-output-hk/catalyst-forge/lib/tools/pkg/walker"
	"github.com/input-output-hk/catalyst-forge/lib/tools/pkg/walker/mocks"
	"github.com/stretchr/testify/assert"
)

func TestFindGitRoot(t *testing.T) {
	tests := []struct {
		name      string
		start     string
		dirs      []string
		want      string
		expectErr bool
	}{
		{
			name:  "simple",
			start: "/tmp/test1/test2/test3",
			dirs: []string{
				"/tmp/test1/test2",
				"/tmp/test1",
				"/tmp/.git",
				"/",
			},
			want:      "/tmp",
			expectErr: false,
		},
		{
			name:  "no git root",
			start: "/tmp/test1/test2/test3",
			dirs: []string{
				"/tmp/test1/test2",
				"/tmp/test1",
				"/",
			},
			want:      "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lastPath string
			w := &mocks.ReverseWalkerMock{
				WalkFunc: func(startPath string, endPath string, callback walker.WalkerCallback) error {
					for _, dir := range tt.dirs {
						err := callback(dir, walker.FileTypeDir, func() (walker.FileSeeker, error) {
							return nil, nil
						})

						if errors.Is(err, io.EOF) {
							lastPath = dir
							return nil
						} else if err != nil {
							return err
						}
					}
					return nil
				},
			}

			got, err := FindGitRoot(tt.start, w)
			if testutils.AssertError(t, err, tt.expectErr, "") {
				return
			}
			assert.Equal(t, tt.want, got)
			assert.Equal(t, lastPath, filepath.Join(tt.want, ".git"))
		})
	}
}
