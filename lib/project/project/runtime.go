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
	if project.TagInfo == nil {
		g.logger.Error("No tag info found")
		return data
	}

	data["GIT_TAG_GENERATED"] = project.ctx.CompileString(fmt.Sprintf(`"%s"`, project.TagInfo.Generated))
	data["GIT_IMAGE_TAG"] = data["GIT_TAG_GENERATED"]

	matches, err := project.TagMatches()
	if err != nil {
		g.logger.Error("Failed to check if tag matches", "error", err)
		return data
	}

	if matches {
		var tag string

		if project.TagInfo.Git.IsMono() {
			tag = project.TagInfo.Git.ToMono().Tag
		} else {
			tag = string(project.TagInfo.Git)
		}

		data["GIT_TAG"] = project.ctx.CompileString(fmt.Sprintf(`"%s"`, tag))
		data["GIT_IMAGE_TAG"] = data["GIT_TAG"]
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
