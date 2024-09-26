package defaults

import (
	"fmt"

	"cuelang.org/go/cue"
)

// DeploymentModuleSetter sets default values for deployment modules.
type DeploymentModuleSetter struct{}

func (d DeploymentModuleSetter) SetDefault(v cue.Value) (cue.Value, error) {
	var err error
	main := v.LookupPath(cue.ParsePath("project.deployment.modules.main"))
	if main.Exists() {
		v, err = setMain(main, v)
		if err != nil {
			return v, fmt.Errorf("failed to set defaults for main module: %w", err)
		}
	}

	support := v.LookupPath(cue.ParsePath("project.deployment.modules.support"))
	if support.Exists() {
		v, err = setSupport(support, v)
		if err != nil {
			return v, err
		}
	}

	return v, nil
}

func setMain(main cue.Value, v cue.Value) (cue.Value, error) {
	container := main.LookupPath(cue.ParsePath("container"))
	if !container.Exists() {
		projectName, err := v.LookupPath(cue.ParsePath("project.name")).String()
		if err != nil {
			return v, fmt.Errorf("failed to get project name: %w", err)
		}

		containerName := fmt.Sprintf("%s-deployment", projectName)
		v = v.FillPath(cue.ParsePath("project.deployment.modules.main.container"), containerName)
	}

	return v, nil
}

func setSupport(support cue.Value, v cue.Value) (cue.Value, error) {
	fields, err := support.Fields()
	if err != nil {
		return v, fmt.Errorf("failed to get support modules: %w", err)
	}

	for fields.Next() {
		container := fields.Value().LookupPath(cue.ParsePath("container"))
		if !container.Exists() {
			return v, fmt.Errorf("support module %s does not have a container field", fields.Selector())
		}
	}

	return v, nil
}
