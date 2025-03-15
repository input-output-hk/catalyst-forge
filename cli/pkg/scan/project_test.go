package scan

import (
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/project/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker"
	"github.com/stretchr/testify/assert"
)

func TestScanProjects(t *testing.T) {
	tests := []struct {
		name         string
		rootPath     string
		files        map[string]string
		expectedKeys []string
		expectErr    bool
	}{
		{
			name:     "single project",
			rootPath: "/tmp1",
			files: map[string]string{
				"/tmp1/blueprint.cue": "",
			},
			expectedKeys: []string{"/tmp1"},
			expectErr:    false,
		},
		{
			name:     "multiple projects",
			rootPath: "/",
			files: map[string]string{
				"/tmp1/blueprint.cue": "",
				"/tmp2/blueprint.cue": "",
			},
			expectedKeys: []string{"/tmp1", "/tmp2"},
			expectErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := billy.NewInMemoryFs()
			testutils.SetupFS(t, fs, tt.files)

			walker := walker.NewCustomDefaultFSWalker(fs, nil)
			loader := &mocks.ProjectLoaderMock{
				LoadFunc: func(projectPath string) (project.Project, error) {
					return project.Project{}, nil
				},
			}

			got, err := ScanProjects(tt.rootPath, loader, &walker, testutils.NewNoopLogger())
			if testutils.AssertError(t, err, tt.expectErr, "") {
				return
			}

			for _, key := range tt.expectedKeys {
				assert.Contains(t, got, key)
			}
		})
	}
}
