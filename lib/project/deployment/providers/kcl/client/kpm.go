package client

import (
	"bytes"
	"fmt"

	kpm "kcl-lang.io/kpm/pkg/client"
)

// KPMClient is a KCLClient that uses the KPM Go package to run modules.
type KPMClient struct {
	logOutput bytes.Buffer
}

func (k KPMClient) Run(container string, conf KCLModuleConfig) (string, error) {
	client, err := kpm.NewKpmClient()
	if err != nil {
		return "", fmt.Errorf("failed to create KPM client: %w", err)
	}

	client.SetLogWriter(&k.logOutput)
	args, err := conf.ToArgs()
	if err != nil {
		return "", fmt.Errorf("failed to generate KCL arguments: %w", err)
	}

	out, err := client.Run(
		kpm.WithRunSourceUrl(container),
		kpm.WithArguments(args),
	)

	if err != nil {
		return "", fmt.Errorf("failed to run KCL module: %w", err)
	}

	return out.GetRawYamlResult(), nil
}

func (k KPMClient) Log() string {
	return k.logOutput.String()
}
