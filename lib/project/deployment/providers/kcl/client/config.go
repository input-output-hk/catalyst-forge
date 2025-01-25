package client

import (
	"fmt"

	"cuelang.org/go/cue/cuecontext"
)

// KCLModuleConfig contains the configuration given to a KCL module.
type KCLModuleConfig struct {
	InstanceName string
	Namespace    string
	Values       any
}

func (k *KCLModuleConfig) ToArgs() ([]string, error) {
	ctx := cuecontext.New()
	v := ctx.Encode(k.Values)
	if v.Err() != nil {
		return nil, fmt.Errorf("failed to encode values: %w", v.Err())
	}

	j, err := v.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal values to JSON: %w", err)
	}

	return []string{
		"-D",
		fmt.Sprintf("name=%s", k.InstanceName),
		"-D",
		fmt.Sprintf("namespace=%s", k.Namespace),
		"-D",
		fmt.Sprintf("values=%s", string(j)),
	}, nil
}
