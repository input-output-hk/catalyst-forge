package mapping

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

// FileConfig represents the YAML configuration for claim mappings.
type FileConfig struct {
	Mappings struct {
		Consent   Mapping `yaml:"consent"`
		TokenHook Mapping `yaml:"token_hook"`
	} `yaml:"mappings"`
	Policy struct {
		OnError       string `yaml:"on_error"`       // "deny" | "warn_passthrough"
		MergeStrategy string `yaml:"merge_strategy"` // "deep" | "replace" (future)
	} `yaml:"policy"`
}

// Mapping defines expressions for producing token claims.
// Expressions are CEL strings evaluated against provided inputs.
type Mapping struct {
	Requirements []string          `yaml:"requirements"`
	IDToken      map[string]string `yaml:"id_token"`
	AccessToken  struct {
		Ext map[string]string `yaml:"ext"`
	} `yaml:"access_token"`
}

// LoadFileConfig loads a mapping configuration from a YAML file.
func LoadFileConfig(path string) (*FileConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fc FileConfig
	if err := yaml.Unmarshal(b, &fc); err != nil {
		return nil, err
	}
	// defaults
	if fc.Policy.OnError == "" {
		fc.Policy.OnError = "deny"
	}
	if fc.Policy.MergeStrategy == "" {
		fc.Policy.MergeStrategy = "deep"
	}
	// minimal validation
	if fc.Mappings.Consent.IDToken == nil && (fc.Mappings.Consent.AccessToken.Ext == nil) &&
		fc.Mappings.TokenHook.AccessToken.Ext == nil {
		return nil, errors.New("mapping config has no outputs defined")
	}
	return &fc, nil
}
