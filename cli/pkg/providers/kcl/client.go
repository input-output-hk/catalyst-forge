package kcl

import (
	"bytes"
	"fmt"

	kpm "kcl-lang.io/kpm/pkg/client"
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure -pkg mocks -out mocks/kcl.go . KCLClient

type KCLClient interface {
	Run(KCLModuleArgs) (string, error)
	Log() string
}

type DefaultKCLClient struct {
	logOutput bytes.Buffer
}

func (k DefaultKCLClient) Run(args KCLModuleArgs) (string, error) {
	client, err := kpm.NewKpmClient()
	if err != nil {
		return "", fmt.Errorf("failed to create KPM client: %w", err)
	}

	client.SetLogWriter(&k.logOutput)

	out, err := client.Run(
		kpm.WithRunSourceUrl(fmt.Sprintf("oci://%s?tag=%s", args.Module, args.Version)),
		kpm.WithArguments(args.Serialize()),
	)

	if err != nil {
		return "", fmt.Errorf("failed to run KCL module: %w", err)
	}

	return out.GetRawYamlResult(), nil
}

func (k DefaultKCLClient) Log() string {
	return k.logOutput.String()
}

// KCLModuleArgs contains the arguments to pass to the KCL module.
type KCLModuleArgs struct {
	// InstanceName is the name to use for the deployment instance.
	InstanceName string

	// Module is the name of the OCI module to deploy.
	Module string

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
	}
}
