package defaults

import (
	"fmt"

	"cuelang.org/go/cue"
)

// DeploymentModuleSetter sets default values for deployment modules.
type DeploymentModuleSetter struct{}

func (d DeploymentModuleSetter) SetDefault(v cue.Value) (cue.Value, error) {
	projectName, _ := v.LookupPath(cue.ParsePath("project.name")).String()
	registry, _ := v.LookupPath(cue.ParsePath("global.deployment.registries.modules")).String()

	modules := v.LookupPath(cue.ParsePath("project.deployment.modules"))
	if !modules.Exists() || modules.Err() != nil {
		return v, nil
	}

	iter, err := modules.Fields()
	if err != nil {
		return v, fmt.Errorf("failed to iterate deployment modules: %w", err)
	}

	for iter.Next() {
		moduleName := iter.Selector().String()
		module := iter.Value()

		if projectName != "" {
			instance := module.LookupPath(cue.ParsePath("instance"))
			if !instance.Exists() {
				v = v.FillPath(cue.ParsePath(fmt.Sprintf("project.deployment.modules.%s.instance", moduleName)), projectName)
			}
		}

		if registry != "" {
			r := module.LookupPath(cue.ParsePath("registry"))
			if !r.Exists() {
				v = v.FillPath(cue.ParsePath(fmt.Sprintf("project.deployment.modules.%s.registry", moduleName)), registry)
			}
		}
	}

	return v, nil
}
