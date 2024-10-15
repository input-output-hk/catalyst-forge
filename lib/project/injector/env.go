package injector

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

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

func (b BlueprintInjectorEnvMap) Get(ctx *cue.Context, name string, attrType AttrType) (cue.Value, error) {
	value, exists := os.LookupEnv(name)
	if !exists {
		return cue.Value{}, ErrNotFound
	}

	switch attrType {
	case AttrTypeString:
		return ctx.CompileString(fmt.Sprintf(`"%s"`, value)), nil
	case AttrTypeInt:
		n, err := strconv.Atoi(value)
		if err != nil {
			return cue.Value{}, fmt.Errorf("invalid int value '%s'", value)
		}
		return ctx.CompileString(fmt.Sprintf("%d", n)), nil
	case AttrTypeBool:
		return ctx.CompileString("true"), nil
	default:
		return cue.Value{}, fmt.Errorf("unsupported attribute type '%s'", attrType)
	}
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
