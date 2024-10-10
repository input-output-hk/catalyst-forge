package injector

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
	tools "github.com/input-output-hk/catalyst-forge/lib/tools/cue"
)

// AttrType represents the type of an attribute
type AttrType string

// BaseAttr represents a base attribute
type BaseAttr struct {
	Name string
	Type AttrType
}

const (
	AttrTypeString AttrType = "string"
	AttrTypeInt    AttrType = "int"
	AttrTypeBool   AttrType = "bool"

	AttrNameKey = "name"
	AttrTypeKey = "type"
)

var (
	ErrNotFound = fmt.Errorf("attribute name not found")
)

type BaseInjector struct {
	attrName     string
	ctx          *cue.Context
	logger       *slog.Logger
	imap         BlueprintInjectorMap
	typeOptional bool
}

func (b *BaseInjector) Inject(bp blueprint.RawBlueprint) blueprint.RawBlueprint {
	rv := bp.Value()

	rv.Walk(func(v cue.Value) bool {
		attr := tools.FindAttr(v, b.attrName)
		if attr == nil {
			return true
		}

		b.logger.Debug("found attribute", "attr", b.attrName, "path", v.Path())

		fillPath := cue.ParsePath(strings.Replace(v.Path().String(), "#Blueprint.", "", -1))
		pAttr, err := b.parseBaseAttr(attr)
		if err != nil {
			rv = rv.FillPath(fillPath, err)
			return true
		}

		b.logger.Debug("parsed attribute", "attr", b.attrName, "name", pAttr.Name, "type", pAttr.Type)

		attrValue, err := b.imap.Get(b.ctx, pAttr.Name, pAttr.Type)
		if errors.Is(err, ErrNotFound) {
			b.logger.Debug("attr name not found", "attr", b.attrName, "name", pAttr.Name)
			return true
		} else if err != nil {
			rv = rv.FillPath(fillPath, err)
			return true
		}

		rv = rv.FillPath(fillPath, attrValue)

		return true
	}, func(v cue.Value) {})

	return blueprint.NewRawBlueprint(rv)
}

// parseBaseAttr parses a base attribute from the given CUE attribute
func (b *BaseInjector) parseBaseAttr(a *cue.Attribute) (BaseAttr, error) {
	var attr BaseAttr

	nameArg, ok, err := a.Lookup(0, AttrNameKey)
	if err != nil {
		return attr, err
	}
	if !ok {
		return attr, fmt.Errorf("missing name key in attribute body '%s'", a.Contents())
	}
	attr.Name = nameArg

	if !b.typeOptional {
		typeArg, ok, err := a.Lookup(0, AttrTypeKey)
		if err != nil {
			return attr, err
		}
		if !ok {
			return attr, fmt.Errorf("missing type key in attribute body '%s'", a.Contents())
		}
		attr.Type = AttrType(typeArg)
	}

	return attr, nil
}
