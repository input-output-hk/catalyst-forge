package defaults

import (
	"fmt"

	"cuelang.org/go/cue"
)

// ReleaseTargetSetter sets default values for deployment modules.
type ReleaseTargetSetter struct{}

func (d ReleaseTargetSetter) SetDefault(v cue.Value) (cue.Value, error) {
	releases := v.LookupPath(cue.ParsePath("project.release"))
	iter, err := releases.Fields()
	if err != nil {
		return v, fmt.Errorf("failed to get releases: %w", err)
	}

	for iter.Next() {
		releaseName := iter.Selector().String()
		release := iter.Value()

		target := release.LookupPath(cue.ParsePath("target"))
		if !target.Exists() {
			v = v.FillPath(cue.ParsePath(fmt.Sprintf("project.release.%s.target", releaseName)), releaseName)
		} else {
			targetName, err := target.String()
			if err != nil {
				return v, fmt.Errorf("failed to get target name: %w", err)
			}

			if targetName == "" {
				v = v.FillPath(cue.ParsePath(fmt.Sprintf("project.release.%s.target", releaseName)), releaseName)
			}
		}
	}

	return v, nil
}
