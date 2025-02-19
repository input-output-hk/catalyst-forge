package module

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/deployer"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/generator"
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

	var bundle deployment.ModuleBundle
	if stat.IsDir() {
		project, err := ctx.ProjectLoader.Load(c.Path)
		if err != nil {
			return fmt.Errorf("could not load project: %w", err)
		}

		bundle = deployment.NewModuleBundle(&project)
	} else {
		src, err := os.ReadFile(c.Path)
		if err != nil {
			return fmt.Errorf("could not read file: %w", err)
		}

		bundle, err = deployment.ParseBundle(ctx.CueCtx, src)
		if err != nil {
			return fmt.Errorf("could not parse module file: %w", err)
		}
	}

	env, err := loadEnv(c.Path, ctx.CueCtx, ctx.Logger)
	if err != nil {
		return fmt.Errorf("could not load environment file: %w", err)
	}

	manifests := make(map[string][]byte)
	gen := generator.NewGenerator(ctx.ManifestGeneratorStore, ctx.Logger)
	if c.Module != "" {
		mod, ok := bundle.Bundle[c.Module]
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
				mod, ok := bundle.Bundle[name]
				if !ok {
					return fmt.Errorf("module %q not found", name)
				}

				mod.Path = path
				bundle.Bundle[name] = mod
			}
		}

		out, err := gen.GenerateBundle(bundle, env)
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

func loadEnv(path string, ctx *cue.Context, logger *slog.Logger) (cue.Value, error) {
	var env cue.Value
	var envPath string

	filename := deployer.ENV_FILE
	stat, err := os.Stat(path)
	if err != nil {
		return cue.Value{}, fmt.Errorf("could not stat path: %w", err)
	}

	if stat.IsDir() {
		envPath = filepath.Join(path, filename)
	} else {
		envPath = filepath.Join(filepath.Dir(path), filename)
	}

	if _, err := os.Stat(envPath); err == nil {
		logger.Info("loading environment file", "path", envPath)
		contents, err := os.ReadFile(envPath)
		if err != nil {
			return cue.Value{}, fmt.Errorf("could not read environment file: %w", err)
		}

		env = ctx.CompileBytes(contents)
		if env.Err() != nil {
			return cue.Value{}, fmt.Errorf("could not compile environment file: %w", env.Err())
		}
	}

	return env, nil
}

func writeManifests(path string, manifests map[string][]byte) error {
	for name, manifest := range manifests {
		if err := os.WriteFile(filepath.Join(path, name), manifest, 0644); err != nil {
			return fmt.Errorf("could not write manifest: %w", err)
		}
	}

	return nil
}
