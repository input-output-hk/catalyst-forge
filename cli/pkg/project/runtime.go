package project

import (
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/lib/blueprint/schema"
	"github.com/input-output-hk/catalyst-forge/lib/tools/pkg/git"
)

// RuntimeData is an interface for runtime data loaders.
type RuntimeData interface {
	Load(project *Project) map[string]string
}

// GitRuntime is a runtime data loader for git related data.
type GitRuntime struct {
	logger *slog.Logger
}

func (g *GitRuntime) Load(project *Project) map[string]string {
	g.logger.Debug("Loading git runtime data")

	var err error
	var strategy string
	var aliases map[string]string

	if project.Raw().Get("global.ci.tagging.strategy").Exists() {
		strategy, err = project.Raw().Get("global.ci.tagging.strategy").String()
		if err != nil {
			g.logger.Error("Failed to get tag strategy", "error", err)
		}
	}

	if project.Raw().Get("global.ci.tagging.aliases").Exists() {
		err = project.Raw().DecodePath("global.ci.tagging.aliases", &aliases)
		if err != nil {
			g.logger.Error("Failed to get tag aliases", "error", err)
		}
	}

	project.Blueprint = schema.Blueprint{
		Global: schema.Global{
			CI: schema.GlobalCI{
				Tagging: schema.Tagging{
					Aliases:  aliases,
					Strategy: schema.TagStrategy(strategy),
				},
			},
		},
	}

	data := make(map[string]string)
	tagger := NewTagger(project, git.InCI(), true, g.logger)

	generatedTag, err := tagger.GenerateTag()
	if err != nil {
		g.logger.Error("Failed to get git tag", "error", err)
	} else if generatedTag != "" {
		data["GIT_TAG_GENERATED"] = generatedTag
	}

	gitTag, err := tagger.GetGitTag()
	if err != nil {
		g.logger.Error("Failed to get git tag", "error", err)
	} else if gitTag != "" {
		data["GIT_TAG"] = gitTag
	}

	if gitTag != "" {
		data["GIT_IMAGE_TAG"] = gitTag
	} else {
		data["GIT_IMAGE_TAG"] = generatedTag
	}

	return data
}

// NewGitRuntime creates a new GitRuntime.
func NewGitRuntime(logger *slog.Logger) *GitRuntime {
	return &GitRuntime{
		logger: logger,
	}
}

// GetDefaultRuntimes returns the default runtime data loaders.
func GetDefaultRuntimes(logger *slog.Logger) []RuntimeData {
	return []RuntimeData{
		NewGitRuntime(logger),
	}
}
