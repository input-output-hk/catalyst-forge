package deployment

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"gopkg.in/yaml.v3"
)

// KCLModuleArgs contains the arguments to pass to the KCL module.
type KCLModuleArgs struct {
	// InstanceName is the name to use for the deployment instance.
	InstanceName string

	// Namespace is the namespace to deploy the module to.
	Namespace string

	// Values contains the values to pass to the module.
	Values string

	// Version is the version of the module to deploy.
	Version string
}

// Serialize serializes the KCLModuleArgs to a list of arguments.
func (k *KCLModuleArgs) Serialize() []string {
	return []string{
		"-D",
		fmt.Sprintf("name=%s", k.InstanceName),
		"-D",
		fmt.Sprintf("namespace=%s", k.Namespace),
		"-D",
		fmt.Sprintf("values=%s", k.Values),
		"-D",
		k.Version,
	}
}

// KCLRunResultModule represents a single module in a KCL run result.
type KCLRunResult struct {
	Manifests string
	Values    string
}

// KCLRunner is used to run KCL commands.
type KCLRunner struct {
	kcl    executor.WrappedExecuter
	logger *slog.Logger
}

// GetMainValues returns the values (in YAML) for the main module in the project.
func (k *KCLRunner) GetMainValues(p *project.Project) (string, error) {
	if p.Blueprint.Project.Deployment.Modules == nil {
		return "", fmt.Errorf("no deployment modules found in project blueprint")
	} else if p.Blueprint.Global.Deployment.Registry == "" {
		return "", fmt.Errorf("no deployment registry found in project blueprint")
	}

	ctx := cuecontext.New()
	module := p.Blueprint.Project.Deployment.Modules.Main

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
	} else if p.Blueprint.Global.Deployment.Registry == "" {
		return nil, fmt.Errorf("no deployment registry found in project blueprint")
	}

	modules := map[string]schema.Module{"main": p.Blueprint.Project.Deployment.Modules.Main}
	for k, v := range p.Blueprint.Project.Deployment.Modules.Support {
		modules[k] = v
	}

	result := make(map[string]KCLRunResult)
	for name, module := range modules {
		json, err := encodeValues(ctx, module)
		if err != nil {
			return nil, fmt.Errorf("failed to encode module values: %w", err)
		}

		args := KCLModuleArgs{
			InstanceName: p.Blueprint.Project.Name,
			Namespace:    module.Namespace,
			Values:       string(json),
			Version:      module.Version,
		}

		container := fmt.Sprintf("%s/%s", strings.TrimSuffix(p.Blueprint.Global.Deployment.Registry, "/"), module.Module)
		out, err := k.run(container, args)
		if err != nil {
			k.logger.Error("Failed to run KCL module", "module", module.Module, "error", err, "output", string(out))
			return nil, fmt.Errorf("failed to run KCL module: %w", err)
		}

		yaml, err := jsonToYaml(json)
		if err != nil {
			return nil, fmt.Errorf("failed to convert values to YAML: %w", err)
		}

		result[name] = KCLRunResult{
			Manifests: string(out),
			Values:    string(yaml),
		}
	}

	return result, nil
}

// encodeValues encodes the values of a module to JSON.
func encodeValues(ctx *cue.Context, module schema.Module) ([]byte, error) {
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

// run runs a KCL module with the given module container and arguments.
func (k *KCLRunner) run(container string, moduleArgs KCLModuleArgs) ([]byte, error) {
	args := []string{"run", "-q"}
	args = append(args, moduleArgs.Serialize()...)
	args = append(args, fmt.Sprintf("oci://%s", container))

	k.logger.Debug("Running KCL module", "container", container, "args", args)
	return k.kcl.Execute(args...)
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

	kcl := executor.NewLocalWrappedExecutor(executor.NewLocalExecutor(logger), "kcl")
	return KCLRunner{
		kcl:    kcl,
		logger: logger,
	}
}
