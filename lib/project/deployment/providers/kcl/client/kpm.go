package client

import (
	"bytes"
	"fmt"
	"strings"

	kpm "kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/downloader"
)

// KPMClient is a KCLClient that uses the KPM Go package to run modules.
type KPMClient struct {
	logOutput bytes.Buffer
}

func (k KPMClient) Run(path string, conf KCLModuleConfig) (string, error) {
	client, err := kpm.NewKpmClient()
	if err != nil {
		return "", fmt.Errorf("failed to create KPM client: %w", err)
	}

	client.SetLogWriter(&k.logOutput)
	args, err := conf.ToArgs()
	if err != nil {
		return "", fmt.Errorf("failed to generate KCL arguments: %w", err)
	}

	runArgs := []kpm.RunOption{
		kpm.WithArguments(args),
	}

	if strings.HasPrefix(path, "oci://") {
		runArgs = append(runArgs, kpm.WithRunSourceUrl(path))
	} else {
		src, err := downloader.NewSourceFromStr(path)
		if err != nil {
			return "", fmt.Errorf("failed to create KCL source: %w", err)
		}

		runArgs = append(runArgs, kpm.WithRunSource(src))
	}

	out, err := client.Run(runArgs...)

	if err != nil {
		return "", fmt.Errorf("failed to run KCL module: %w", err)
	}

	return out.GetRawYamlResult(), nil
}

func (k KPMClient) Log() string {
	return k.logOutput.String()
}
