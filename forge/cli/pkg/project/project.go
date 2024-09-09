package project

import (
	gg "github.com/go-git/go-git/v5"
	"github.com/input-output-hk/catalyst-forge/blueprint/pkg/blueprint"
	"github.com/input-output-hk/catalyst-forge/blueprint/schema"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/earthfile"
)

// Project represents a project
type Project struct {
	Blueprint    schema.Blueprint
	Earthfile    *earthfile.Earthfile
	Name         string
	Path         string
	Repo         *gg.Repository
	RepoRoot     string
	rawBlueprint blueprint.RawBlueprint
}

// Raw returns the raw blueprint.
func (p *Project) Raw() blueprint.RawBlueprint {
	return p.rawBlueprint
}