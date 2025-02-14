package module

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/generator"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
)

type TemplateCmd struct {
	Module  string            `short:"m" help:"The specific module to template."`
	Out     string            `short:"o" help:"The output directory to write manifests to."`
	Path    string            `arg:"" help:"The path to the module (or project)." kong:"arg,predictor=path"`
	SetPath map[string]string `help:"Overrides the path for a given module (format: module=path)."`
}

func (c *TemplateCmd) Run(ctx run.RunContext) error {
	stat, err := os.Stat(c.Path)
	if err != nil {
		return fmt.Errorf("could not stat path: %w", err)
	}

	var bundle sp.ModuleBundle
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

	manifests := make(map[string][]byte)
	gen := generator.NewGenerator(ctx.ManifestGeneratorStore, ctx.Logger)
	if c.Module != "" {
		mod, ok := bundle[c.Module]
		if !ok {
			return fmt.Errorf("module %q not found", c.Module)
		}

		if path, ok := c.SetPath[c.Module]; ok {
			ctx.Logger.Info("overriding path for module", "module", c.Module, "path", path)
			mod.Path = path
		}

		out, err := gen.Generate(mod)
		if err != nil {
			return fmt.Errorf("failed to generate manifest: %w", err)
		}

		filename := fmt.Sprintf("%s.yaml", c.Module)
		manifests[filename] = out
	} else {
		if c.SetPath != nil {
			for name, path := range c.SetPath {
				mod, ok := bundle[name]
				if !ok {
					return fmt.Errorf("module %q not found", name)
				}

				mod.Path = path
				bundle[name] = mod
			}
		}

		out, err := gen.GenerateBundle(bundle)
		if err != nil {
			return fmt.Errorf("failed to generate manifests: %w", err)
		}

		for name, manifest := range out.Manifests {
			filename := fmt.Sprintf("%s.yaml", name)
			manifests[filename] = manifest
		}
	}

	if c.Out != "" {
		if err := writeManifests(c.Out, manifests); err != nil {
			return fmt.Errorf("could not write manifests: %w", err)
		}
	} else {
		var output string
		for _, manifest := range manifests {
			output += fmt.Sprintf("%s\n---\n", strings.TrimSuffix(string(manifest), "---\n"))
		}

		fmt.Print(strings.TrimSuffix(output, "---\n"))
	}

	return nil
}

func writeManifests(path string, manifests map[string][]byte) error {
	for name, manifest := range manifests {
		if err := os.WriteFile(filepath.Join(path, name), manifest, 0644); err != nil {
			return fmt.Errorf("could not write manifest: %w", err)
		}
	}

	return nil
}
