package providers

import (
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	sb "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	sg "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global"
	spr "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global/providers"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimoniReleaserRelease(t *testing.T) {
	newProject := func(
		name string,
		registries []string,
	) project.Project {
		return project.Project{
			Name: name,
			Blueprint: sb.Blueprint{
				Global: &sg.Global{
					Ci: &sg.CI{
						Providers: &spr.Providers{
							Timoni: &spr.Timoni{
								Registries: registries,
							},
						},
					},
					Repo: &sg.Repo{
						Name: "repo",
					},
				},
				Project: &sp.Project{},
			},
		}
	}

	tests := []struct {
		name     string
		project  project.Project
		release  sp.Release
		config   TimoniReleaserConfig
		firing   bool
		force    bool
		failOn   string
		validate func(t *testing.T, calls []string, err error)
	}{
		{
			name:    "full",
			project: newProject("test", []string{"test.com"}),
			release: sp.Release{},
			config: TimoniReleaserConfig{
				Container: "test",
				Tag:       "test",
			},
			firing: true,
			force:  false,
			failOn: "",
			validate: func(t *testing.T, calls []string, err error) {
				require.NoError(t, err)
				assert.Contains(t, calls, "mod push --version test --latest=false . oci://test.com/test")
			},
		},
		{
			name:    "with v prefix",
			project: newProject("test", []string{"test.com"}),
			release: sp.Release{},
			config: TimoniReleaserConfig{
				Container: "test",
				Tag:       "v1.0.0",
			},
			firing: true,
			force:  false,
			failOn: "",
			validate: func(t *testing.T, calls []string, err error) {
				require.NoError(t, err)
				assert.Contains(t, calls, "mod push --version 1.0.0 --latest=false . oci://test.com/test")
			},
		},
		{
			name:    "no container",
			project: newProject("test", []string{"test.com"}),
			release: sp.Release{},
			config: TimoniReleaserConfig{
				Tag: "test",
			},
			firing: true,
			force:  false,
			failOn: "",
			validate: func(t *testing.T, calls []string, err error) {
				require.NoError(t, err)
				assert.Contains(t, calls, "mod push --version test --latest=false . oci://test.com/test-deployment")
			},
		},
		{
			name:    "not firing",
			project: newProject("test", []string{"test.com"}),
			firing:  false,
			force:   false,
			failOn:  "",
			validate: func(t *testing.T, calls []string, err error) {
				require.NoError(t, err)
				assert.Len(t, calls, 0)
			},
		},
		{
			name:    "forced",
			project: newProject("test", []string{"test.com"}),
			release: sp.Release{},
			config: TimoniReleaserConfig{
				Container: "test",
				Tag:       "test",
			},
			firing: false,
			force:  true,
			failOn: "",
			validate: func(t *testing.T, calls []string, err error) {
				require.NoError(t, err)
				assert.Contains(t, calls, "mod push --version test --latest=false . oci://test.com/test")
			},
		},
		{
			name:    "push fails",
			project: newProject("test", []string{"test.com"}),
			release: sp.Release{},
			config: TimoniReleaserConfig{
				Container: "test",
				Tag:       "test",
			},
			firing: true,
			force:  false,
			failOn: "mod push",
			validate: func(t *testing.T, calls []string, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []string
			timoni := TimoniReleaser{
				config:  tt.config,
				force:   tt.force,
				handler: newReleaseEventHandlerMock(tt.firing),
				logger:  testutils.NewNoopLogger(),
				project: tt.project,
				release: tt.release,
				timoni:  newWrappedExecuterMock(&calls, tt.failOn),
			}

			err := timoni.Release()

			tt.validate(t, calls, err)
		})
	}
}
