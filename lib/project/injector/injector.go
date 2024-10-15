package injector

//go:generate go run github.com/matryer/moq@latest -skip-ensure --pkg mocks --out ./mocks/blueprint_injector.go . BlueprintInjector
//go:generate go run github.com/matryer/moq@latest -skip-ensure --pkg mocks --out ./mocks/blueprint_injector_map.go . BlueprintInjectorMap

import (
	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
)

type BlueprintInjector interface {
	Inject(bp blueprint.RawBlueprint) blueprint.RawBlueprint
}

type BlueprintInjectorMap interface {
	Get(ctx *cue.Context, name string, attrType AttrType) (cue.Value, error)
}
