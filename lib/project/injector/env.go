package injector

import (
	"log/slog"
	"os"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
)

type BlueprintEnvInjector struct {
	base BaseInjector
}

func (b *BlueprintEnvInjector) Inject(bp blueprint.RawBlueprint) blueprint.RawBlueprint {
	return b.base.Inject(bp)
}

type BlueprintInjectorEnvMap struct{}

func (b BlueprintInjectorEnvMap) Get(ctx *cue.Context, name string, attrType AttrType, concrete bool) (cue.Value, error) {
	value, exists := os.LookupEnv(name)
	if !exists {
		return cue.Value{}, ErrNotFound
	}

	return compileWithConcrete(ctx, attrType, value, concrete)
}

func NewBlueprintEnvInjector(ctx *cue.Context, logger *slog.Logger) *BlueprintEnvInjector {
	return &BlueprintEnvInjector{
		base: BaseInjector{
			attrName:     "env",
			ctx:          ctx,
			logger:       logger,
			imap:         BlueprintInjectorEnvMap{},
			typeOptional: false,
		},
	}
}
