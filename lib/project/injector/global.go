package injector

import (
	"fmt"
	"log/slog"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
)

type BlueprintGlobalInjector struct {
	base BaseInjector
}

func (b *BlueprintGlobalInjector) Inject(bp blueprint.RawBlueprint) blueprint.RawBlueprint {
	b.base.imap = BlueprintGlobalInjectorMap{
		rbp: bp,
	}

	return b.base.Inject(bp)
}

type BlueprintGlobalInjectorMap struct {
	rbp blueprint.RawBlueprint
}

func (b BlueprintGlobalInjectorMap) Get(ctx *cue.Context, name string, attrType AttrType) (cue.Value, error) {
	path := fmt.Sprintf("global.%s", name)
	v := b.rbp.Get(path)
	if v.Err() != nil || v.IsNull() || !v.Exists() {
		return cue.Value{}, ErrNotFound
	}

	return v, nil
}

func NewBlueprintGlobalInjector(ctx *cue.Context, logger *slog.Logger) *BlueprintGlobalInjector {
	return &BlueprintGlobalInjector{
		base: BaseInjector{
			attrName:     "global",
			ctx:          ctx,
			logger:       logger,
			typeOptional: true,
		},
	}
}
