package providers

import (
	"testing"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/publish/providers/common"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	sb "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	sg "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global"
	spr "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global/providers"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimoniPublisherPublish(t *testing.T) {
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
		name      string
		project   project.Project
		publisher sp.Publisher
		config    TimoniPublisherConfig
		firing    bool
		force     bool
		failOn    string
		validate  func(t *testing.T, calls []string, err error)
	}{
		{
			name:      "full",
			project:   newProject("test", []string{"test.com"}),
			publisher: sp.Publisher{},
			config: TimoniPublisherConfig{
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
			name:      "with v prefix",
			project:   newProject("test", []string{"test.com"}),
			publisher: sp.Publisher{},
			config: TimoniPublisherConfig{
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
			name:      "no container",
			project:   newProject("test", []string{"test.com"}),
			publisher: sp.Publisher{},
			config: TimoniPublisherConfig{
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
			name:      "forced",
			project:   newProject("test", []string{"test.com"}),
			publisher: sp.Publisher{},
			config: TimoniPublisherConfig{
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
			name:      "push fails",
			project:   newProject("test", []string{"test.com"}),
			publisher: sp.Publisher{},
			config: TimoniPublisherConfig{
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
			timoni := TimoniPublisher{
				config:    tt.config,
				force:     tt.force,
				handler:   common.NewPublisherEventHandlerMock(tt.firing),
				logger:    testutils.NewNoopLogger(),
				project:   tt.project,
				publisher: tt.publisher,
				timoni:    newWrappedExecuterMock(&calls, tt.failOn),
			}

			err := timoni.Publish()

			tt.validate(t, calls, err)
		})
	}
}
