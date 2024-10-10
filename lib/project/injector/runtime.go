package injector

import (
	"log/slog"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
)

type BlueprintRuntimeInjector struct {
	base BaseInjector
}

func (b *BlueprintRuntimeInjector) Inject(bp blueprint.RawBlueprint) blueprint.RawBlueprint {
	return b.base.Inject(bp)
}

type BlueprintInjectorRuntimeMap struct {
	runtimeValues map[string]cue.Value
}

func (b BlueprintInjectorRuntimeMap) Get(ctx *cue.Context, name string, attrType AttrType) (cue.Value, error) {
	value, exists := b.runtimeValues[name]
	if !exists {
		return cue.Value{}, ErrNotFound
	}

	return value, nil
}

func NewBlueprintRuntimeInjector(
	runtimeValues map[string]cue.Value,
	logger *slog.Logger,
) *BlueprintRuntimeInjector {
	return &BlueprintRuntimeInjector{
		base: BaseInjector{
			attrName: "forge",
			logger:   logger,
			imap: BlueprintInjectorRuntimeMap{
				runtimeValues: runtimeValues,
			},
			typeOptional: true,
		},
	}
}
