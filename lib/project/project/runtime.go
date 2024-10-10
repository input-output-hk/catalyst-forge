package project

import (
	"fmt"
	"log/slog"

	"cuelang.org/go/cue"
)

// RuntimeData is an interface for runtime data loaders.
type RuntimeData interface {
	Load(project *Project) map[string]cue.Value
}

// GitRuntime is a runtime data loader for git related data.
type GitRuntime struct {
	logger *slog.Logger
}

func (g *GitRuntime) Load(project *Project) map[string]cue.Value {
	g.logger.Debug("Loading git runtime data")

	data := make(map[string]cue.Value)
	ctx := project.Raw().Context()

	data["GIT_TAG_GENERATED"] = ctx.CompileString(fmt.Sprintf(`"%s"`, project.TagInfo.Generated))
	if project.TagInfo.Git != "" {
		v := ctx.CompileString(fmt.Sprintf(`"%s"`, project.TagInfo.Git))
		data["GIT_TAG"] = v
		data["GIT_IMAGE_TAG"] = v
	} else {
		data["GIT_IMAGE_TAG"] = data["GIT_TAG_GENERATED"]
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
