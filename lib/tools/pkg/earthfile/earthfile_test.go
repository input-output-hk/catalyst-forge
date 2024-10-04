package earthfile

import (
	"context"
	"io/fs"
	"strings"
	"testing"

	"github.com/earthly/earthly/ast/spec"
	"github.com/input-output-hk/catalyst-forge/lib/tools/pkg/testutils"
	"github.com/stretchr/testify/assert"
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

func TestEarthfileTargets(t *testing.T) {
	earthfile := Earthfile{
		spec: spec.Earthfile{
			Targets: []spec.Target{
				{Name: "target1"},
				{Name: "target2"},
			},
		},
	}

	targets := earthfile.Targets()
	assert.Equal(t, 2, len(targets), "expected 2 targets")
	assert.Equal(t, "target1", targets[0], "expected target1")
	assert.Equal(t, "target2", targets[1], "expected target2")
}

func TestEarthfileFilterTargets(t *testing.T) {
	earthfile := Earthfile{
		spec: spec.Earthfile{
			Targets: []spec.Target{
				{Name: "target1"},
				{Name: "target2"},
			},
		},
	}

	targets := earthfile.FilterTargets(func(target string) bool {
		return target == "target1"
	})

	assert.Equal(t, 1, len(targets), "expected 1 target")
	assert.Equal(t, "target1", targets[0], "expected target1")
}

func TestParseEarthfile(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		expectErr bool
	}{
		{
			name: "valid earthfile",
			content: `
VERSION 0.7

foo:
  LET foo = bar
`,
			expectErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseEarthfile(context.Background(), NewMockFileSeeker(test.content))
			testutils.AssertError(t, err, test.expectErr, "")
		})
	}
}
