package git

import (
	"errors"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestTagObjectsToMap(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		tags := []*object.Tag{}
		result := TagObjectsToMap(tags)
		assert.Empty(t, result)
	})

	t.Run("single tag", func(t *testing.T) {
		tag := &object.Tag{
			Name:    "v1.0.0",
			Tagger:  object.Signature{Name: "test", Email: "test@test.com", When: time.Now()},
			Message: "version 1.0.0",
		}
		tags := []*object.Tag{tag}
		result := TagObjectsToMap(tags)

		require.Len(t, result, 1)
		assert.Equal(t, tag, result["v1.0.0"])
	})

	t.Run("multiple tags", func(t *testing.T) {
		tag1 := &object.Tag{
			Name:    "v1.0.0",
			Tagger:  object.Signature{Name: "test", Email: "test@test.com", When: time.Now()},
			Message: "version 1.0.0",
		}
		tag2 := &object.Tag{
			Name:    "v1.1.0",
			Tagger:  object.Signature{Name: "test", Email: "test@test.com", When: time.Now()},
			Message: "version 1.1.0",
		}
		tags := []*object.Tag{tag1, tag2}
		result := TagObjectsToMap(tags)

		require.Len(t, result, 2)
		assert.Equal(t, tag1, result["v1.0.0"])
		assert.Equal(t, tag2, result["v1.1.0"])
	})

	t.Run("duplicate tag names", func(t *testing.T) {
		tag1 := &object.Tag{
			Name:    "v1.0.0",
			Tagger:  object.Signature{Name: "test", Email: "test@test.com", When: time.Now()},
			Message: "version 1.0.0",
		}
		tag2 := &object.Tag{
			Name:    "v1.0.0", // Same name
			Tagger:  object.Signature{Name: "test", Email: "test@test.com", When: time.Now()},
			Message: "version 1.0.0 updated",
		}
		tags := []*object.Tag{tag1, tag2}
		result := TagObjectsToMap(tags)

		// Should only have one entry due to map behavior (last one wins)
		require.Len(t, result, 1)
		assert.Equal(t, tag2, result["v1.0.0"]) // Last one should be the value
	})
}

func TestGetTagCommit(t *testing.T) {
	t.Run("valid tag", func(t *testing.T) {
		// Create a tag with a valid target hash
		tag := &object.Tag{
			Name:    "v1.0.0",
			Tagger:  object.Signature{Name: "test", Email: "test@test.com", When: time.Now()},
			Message: "version 1.0.0",
			Target:  plumbing.NewHash("0000000000000000000000000000000000000000"), // Zero hash for testing
		}

		// This test is limited since we can't easily create a valid tag object
		// with a real commit hash in a unit test context without a full repository
		// The function is a simple wrapper around tag.Commit(), so we test the basic structure
		_, err := GetTagCommit(tag)
		// We expect an error since the target hash is invalid, but this tests the function exists
		assert.Error(t, err)
	})

	t.Run("nil tag", func(t *testing.T) {
		_, err := GetTagCommit(nil)
		assert.Error(t, err)
	})
}
