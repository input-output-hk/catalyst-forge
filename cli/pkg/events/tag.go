package events

import (
	"log/slog"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
)

// TagEvent fires when a git tag is present.
type TagEvent struct {
	logger *slog.Logger
}

func (t *TagEvent) Firing(p *project.Project, config cue.Value) (bool, error) {
	return p.Tag != nil, nil
}
