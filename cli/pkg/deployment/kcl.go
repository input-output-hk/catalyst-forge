package deployment

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/providers/kcl"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"gopkg.in/yaml.v3"
)

// KCLRunResultModule represents a single module in a KCL run result.
type KCLRunResult struct {
	Manifests string
	Values    string
}

// KCLRunner is used to run KCL commands.
type KCLRunner struct {
	client kcl.KCLClient
	logger *slog.Logger
}

// GetMainValues returns the values (in YAML) for the main module in the project.
func (k *KCLRunner) GetMainValues(p *project.Project, moduleName string) (string, error) {
	if p.Blueprint.Project.Deployment.Modules == nil {
		return "", fmt.Errorf("no deployment modules found in project blueprint")
	} else if p.Blueprint.Global.Deployment.Registries.Modules == "" {
		return "", fmt.Errorf("no module deployment registry found in project blueprint")
	}

	ctx := cuecontext.New()
	module := p.Blueprint.Project.Deployment.Modules[moduleName]

	json, err := encodeValues(ctx, module)
	if err != nil {
		return "", fmt.Errorf("failed to encode module values: %w", err)
	}

	yaml, err := jsonToYaml(json)
	if err != nil {
		return "", fmt.Errorf("failed to convert values to YAML: %w", err)
	}

	return string(yaml), nil
}

// RunDeployment runs the deployment modules in the project and returns the
// combined output.
func (k *KCLRunner) RunDeployment(p *project.Project) (map[string]KCLRunResult, error) {
	ctx := cuecontext.New()
	if p.Blueprint.Project.Deployment.Modules == nil {
		return nil, fmt.Errorf("no deployment modules found in project blueprint")
	} else if p.Blueprint.Global.Deployment.Registries.Modules == "" {
		return nil, fmt.Errorf("no module deployment registry found in project blueprint")
	}

	result := make(map[string]KCLRunResult)
	for name, module := range p.Blueprint.Project.Deployment.Modules {
		json, err := encodeValues(ctx, module)
		if err != nil {
			return nil, fmt.Errorf("failed to encode module values: %w", err)
		}

		container := fmt.Sprintf("%s/%s", strings.TrimSuffix(p.Blueprint.Global.Deployment.Registries.Modules, "/"), module.Name)
		args := kcl.KCLModuleArgs{
			InstanceName: p.Name,
			Module:       container,
			Namespace:    module.Namespace,
			Values:       string(json),
			Version:      module.Version,
		}

		k.logger.Debug("Running KCL module", "module", args.Module, "version", args.Version)
		out, err := k.client.Run(args)
		if err != nil {
			k.logger.Error("Failed to run KCL module", "module", module.Name, "error", err, "log", k.client.Log())
			return nil, fmt.Errorf("failed to run KCL module: %w", err)
		}

		yaml, err := jsonToYaml(json)
		if err != nil {
			return nil, fmt.Errorf("failed to convert values to YAML: %w", err)
		}

		result[name] = KCLRunResult{
			Manifests: out,
			Values:    string(yaml),
		}
	}

	return result, nil
}

// encodeValues encodes the values of a module to JSON.
func encodeValues(ctx *cue.Context, module schema.DeploymentModule) ([]byte, error) {
	v := ctx.Encode(module.Values)
	if err := v.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate module values: %w", err)
	}

	j, err := v.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal module values: %w", err)
	}

	return j, nil
}

// jsonToYaml converts a JSON string to a YAML string.
func jsonToYaml(j []byte) ([]byte, error) {
	var jsonObject map[string]interface{}
	err := json.Unmarshal(j, &jsonObject)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	y, err := yaml.Marshal(jsonObject)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return y, nil
}

// NewKCLRunner creates a new KCLRunner.
func NewKCLRunner(logger *slog.Logger) KCLRunner {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return KCLRunner{
		client: kcl.DefaultKCLClient{},
		logger: logger,
	}
}
