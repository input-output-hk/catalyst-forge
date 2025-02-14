package project

import (
	"testing"

	sb "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	sg "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global"
	"github.com/stretchr/testify/assert"
)

func TestGenerateContainerName(t *testing.T) {
	tests := []struct {
		name          string
		projectName   string
		containerName string
		repoName      string
		registry      string
		validate      func(*testing.T, string)
	}{
		{
			name:          "full",
			projectName:   "test",
			containerName: "test-container",
			repoName:      "test/repo",
			registry:      "test-registry",
			validate: func(t *testing.T, container string) {
				assert.Equal(t, "test-registry/repo/test-container", container)
			},
		},
		{
			name:          "partial repo",
			projectName:   "test",
			containerName: "test-container",
			repoName:      "repo",
			registry:      "test-registry",
			validate: func(t *testing.T, container string) {
				assert.Equal(t, "test-registry/repo/test-container", container)
			},
		},
		{
			name:          "no container name",
			projectName:   "test",
			containerName: "",
			repoName:      "test/repo",
			registry:      "test-registry",
			validate: func(t *testing.T, container string) {
				assert.Equal(t, "test-registry/repo/test", container)
			},
		},
		{
			name:          "no registry",
			projectName:   "test",
			containerName: "test-container",
			repoName:      "test/repo",
			validate: func(t *testing.T, container string) {
				assert.Equal(t, "test-container", container)
			},
		},
		{
			name:          "GHCR registry",
			projectName:   "test",
			containerName: "test-container",
			repoName:      "test/repo",
			registry:      "ghcr.io/org/repo",
			validate: func(t *testing.T, container string) {
				assert.Equal(t, "ghcr.io/org/repo/test-container", container)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Project{
				Name: tt.projectName,
				Blueprint: sb.Blueprint{
					Global: &sg.Global{
						Repo: &sg.Repo{
							Name: tt.repoName,
						},
					},
				},
			}

			container := GenerateContainerName(p, tt.containerName, tt.registry)
			tt.validate(t, container)
		})
	}
}
