package module

import (
	"fmt"
	"os"
	"strings"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
)

type TemplateCmd struct {
	Path string `arg:"" help:"The path to the module (or project)." kong:"arg,predictor=path"`
}

func (c *TemplateCmd) Run(ctx run.RunContext) error {
	stat, err := os.Stat(c.Path)
	if err != nil {
		return fmt.Errorf("could not stat path: %w", err)
	}

	var bundle schema.DeploymentModuleBundle
	if stat.IsDir() {
		project, err := ctx.ProjectLoader.Load(c.Path)
		if err != nil {
			return fmt.Errorf("could not load project: %w", err)
		}

		bundle = project.Blueprint.Project.Deployment.Modules
	} else {
		src, err := os.ReadFile(c.Path)
		if err != nil {
			return fmt.Errorf("could not read file: %w", err)
		}

		bundle, err = deployment.ParseBundle(src)
		if err != nil {
			return fmt.Errorf("could not parse module file: %w", err)
		}
	}

	result, err := ctx.DeploymentGenerator.GenerateBundle(bundle)
	if err != nil {
		return fmt.Errorf("failed to generate manifests: %w", err)
	}

	var out string
	for _, manifest := range result.Manifests {
		out += fmt.Sprintf("%s---\n", manifest)
	}

	fmt.Print(strings.TrimSuffix(out, "---\n"))

	return nil
}
