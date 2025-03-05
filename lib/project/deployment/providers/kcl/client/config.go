package client

import (
	"fmt"

	"cuelang.org/go/cue/cuecontext"
)

// KCLModuleConfig contains the configuration given to a KCL module.
type KCLModuleConfig struct {
	Env       string `json:"env"`
	Instance  string `json:"instance"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Values    any    `json:"values"`
	Version   string `json:"version"`
}

func (k *KCLModuleConfig) ToArgs() ([]string, error) {
	ctx := cuecontext.New()
	v := ctx.Encode(k)
	if v.Err() != nil {
		return nil, fmt.Errorf("failed to encode module: %w", v.Err())
	}

	j, err := v.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal module to JSON: %w", err)
	}

	return []string{
		"-D",
		fmt.Sprintf("deployment=%s", string(j)),
	}, nil
}
