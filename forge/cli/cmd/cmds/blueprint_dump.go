package cmds

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/input-output-hk/catalyst-forge/blueprint/pkg/loader"
)

type DumpCmd struct {
	Config string `arg:"" help:"Path to the blueprint file."`
	Pretty bool   `help:"Pretty print JSON output."`
}

func (c *DumpCmd) Run(logger *slog.Logger) error {
	if _, err := os.Stat(c.Config); os.IsNotExist(err) {
		return fmt.Errorf("blueprint file does not exist: %s", c.Config)
	}

	loader := loader.NewDefaultBlueprintLoader(c.Config, logger)
	if err := loader.Load(); err != nil {
		return err
	}

	json, err := loader.Raw().MarshalJSON()
	if err != nil {
		return err
	}

	fmt.Println(string(json))
	return nil
}
