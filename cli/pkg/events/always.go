package events

import (
	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
)

// AlwaysEvent fires always.
type AlwaysEvent struct{}

func (m *AlwaysEvent) Firing(p *project.Project, config cue.Value) (bool, error) {
	return true, nil
}
