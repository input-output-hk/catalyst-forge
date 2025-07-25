package events

import (
	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/providers/github"
)

// PREvent fires when the current operation is within a PR.
type PREvent struct {
	gc github.GithubClient
}

func (m *PREvent) Firing(p *project.Project, config cue.Value) (bool, error) {
	if m.gc.Env().IsPR() {
		return true, nil
	}

	return false, nil
}
