package deployment

import (
	"fmt"
	"net/url"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
)

// Bundle represents a Timoni bundle file.
type Bundle struct {
	ApiVersion string                    `json:"apiVersion"`
	Name       string                    `json:"name"`
	Instances  map[string]BundleInstance `json:"instances"`
	ctx        *cue.Context
}

// BundleInstance represents a single instance of a module in a Timoni bundle file.
type BundleInstance struct {
	Module    Module    `json:"module"`
	Namespace string    `json:"namespace"`
	Values    cue.Value `json:"values"`
}

// Module represents a single module in a Timoni bundle file.
type Module struct {
	Digest  string `json:"digest"`
	Url     string `json:"url"`
	Version string `json:"version"`
}

// Encode encodes the bundle into CUE syntax.
func (b Bundle) Encode() ([]byte, error) {
	v := b.ctx.CompileString("bundle: {}")
	v = v.FillPath(cue.ParsePath("bundle"), b.ctx.Encode(b))

	if err := v.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate bundle: %w", err)
	}

	src, err := format.Node(v.Syntax())
	if err != nil {
		return nil, fmt.Errorf("failed to encode bundle: %w", err)
	}

	return src, nil
}

// GenerateBundle generates a Timoni bundle file from a project.
func GenerateBundle(project *project.Project) (Bundle, error) {
	ctx := cuecontext.New()
	bp := project.Blueprint
	if bp.Project.Deployment.Modules == nil {
		return Bundle{}, fmt.Errorf("no deployment modules found in project blueprint")
	} else if bp.Global.Deployment.Registry == "" {
		return Bundle{}, fmt.Errorf("no deployment registry found in project blueprint")
	}

	instances := make(map[string]BundleInstance)

	mainInst, err := buildInstance(ctx, bp.Project.Deployment.Modules.Main, project.Blueprint.Global.Deployment.Registry)
	if err != nil {
		return Bundle{}, fmt.Errorf("failed to build main module instance: %w", err)
	}
	instances[project.Blueprint.Project.Name] = mainInst

	if project.Blueprint.Project.Deployment.Modules.Support != nil {
		for name, module := range project.Blueprint.Project.Deployment.Modules.Support {
			instance, err := buildInstance(ctx, module, project.Blueprint.Global.Deployment.Registry)
			if err != nil {
				return Bundle{}, fmt.Errorf("failed to build support module instance %q: %w", name, err)
			}

			instances[name] = instance
		}
	}

	return Bundle{
		ApiVersion: "v1alpha1",
		Name:       project.Blueprint.Project.Name,
		Instances:  instances,
		ctx:        ctx,
	}, nil
}

// buildInstance builds a single instance of a module in a Timoni bundle file.
func buildInstance(ctx *cue.Context, module schema.Module, registry string) (BundleInstance, error) {
	url, err := url.JoinPath("oci://", registry, *module.Container)
	if err != nil {
		return BundleInstance{}, fmt.Errorf("failed to generate module URL: %w", err)
	}

	values := ctx.Encode(module.Values)
	if err := values.Validate(); err != nil {
		return BundleInstance{}, fmt.Errorf("failed to validate module values: %w", err)
	}

	return BundleInstance{
		Module: Module{
			Digest:  "",
			Url:     url,
			Version: module.Version,
		},
		Namespace: module.Namespace,
		Values:    values,
	}, nil
}
