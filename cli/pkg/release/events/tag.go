package events

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/lib/project/project"
)

// TagEvent fires when a tag is pushed.
type TagEvent struct {
	logger  *slog.Logger
	project *project.Project
}

func (t *TagEvent) Firing() (bool, error) {
	if t.project.TagInfo == nil {
		return false, fmt.Errorf("tag info not available")
	}

	if t.project.TagInfo.Git == "" {
		t.logger.Debug("no git tag found")
		return false, nil
	}

	matches, err := t.project.TagMatches()
	if err != nil {
		return false, fmt.Errorf("failed to check tag matches: %w", err)
	}

	return matches, nil
}
