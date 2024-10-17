package events

import (
	"fmt"
	"log/slog"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
)

// TagEvent fires when a tag is pushed.
type TagEvent struct {
	logger *slog.Logger
}

func (t *TagEvent) Firing(p *project.Project, config cue.Value) (bool, error) {
	if p.TagInfo == nil {
		return false, fmt.Errorf("tag info not available")
	}

	if p.TagInfo.Git == "" {
		t.logger.Debug("no git tag found")
		return false, nil
	}

	matches, err := p.TagMatches()
	if err != nil {
		return false, fmt.Errorf("failed to check tag matches: %w", err)
	}

	return matches, nil
}
