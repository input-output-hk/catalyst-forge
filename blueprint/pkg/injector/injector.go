package injector

//go:generate go run github.com/matryer/moq@latest -skip-ensure --pkg mocks --out ./mocks/injector.go . EnvGetter

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"

	"cuelang.org/go/cue"
)

const (
	EnvAttrName = "env"
	EnvNameKey  = "name"
	EnvTypeKey  = "type"
)

// EnvType represents the type of an environment variable
type envType string

const (
	EnvTypeString envType = "string"
	EnvTypeInt    envType = "int"
	EnvTypeBool   envType = "bool"
)

// envAttr represents a parsed @env() attribute
type envAttr struct {
	name    string
	envType envType
}

// EnvGetter is an interface to get environment variables
type EnvGetter interface {
	Get(key string) (string, bool)
}

// OSEnvGetter is an implementation of EnvGetter that gets environment variables
// from the OS
type OSEnvGetter struct{}

func (OSEnvGetter) Get(key string) (string, bool) {
	return os.LookupEnv(key)
}

// Injector is a struct that injects environment variables into a CUE value
type Injector struct {
	getter EnvGetter
	logger *slog.Logger
}

// InjectEnv injects environment variables into the given CUE value
func (i *Injector) InjectEnv(v cue.Value) cue.Value {
	rv := v

	v.Walk(func(v cue.Value) bool {
		attr := findEnvAttr(v)
		if attr == nil {
			return true
		}

		i.logger.Debug("found @env() attribute", "path", v.Path())

		env, err := parseEnvAttr(attr)
		if err != nil {
			rv = rv.FillPath(v.Path(), err)
			return true
		}

		i.logger.Debug("parsed @env() attribute", "name", env.name, "type", env.envType)

		envValue, ok := i.getter.Get(env.name)
		if !ok {
			i.logger.Warn("environment variable not found", "name", env.name)
			return true
		}

		switch env.envType {
		case EnvTypeString:
			rv = rv.FillPath(v.Path(), envValue)
		case EnvTypeInt:
			n, err := strconv.Atoi(envValue)
			if err != nil {
				rv = rv.FillPath(v.Path(), fmt.Errorf("invalid int value '%s'", envValue))
			}
			rv = rv.FillPath(v.Path(), n)
		case EnvTypeBool:
			rv = rv.FillPath(v.Path(), true)
		default:
			rv = rv.FillPath(v.Path(), fmt.Errorf("invalid type '%s', must be one of: string, int, bool", env.envType))
		}

		return true
	}, func(v cue.Value) {})

	return rv
}

// NewDefaultInjector creates a new Injector with default settings and an
// optional logger.
func NewDefaultInjector(logger *slog.Logger) Injector {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return Injector{
		getter: OSEnvGetter{},
		logger: logger,
	}
}

// NewInjector creates a new Injector
func NewInjector(logger *slog.Logger, getter EnvGetter) Injector {
	return Injector{
		getter: getter,
		logger: logger,
	}
}

// findEnvAttr finds an @env() attribute in the given CUE value
func findEnvAttr(v cue.Value) *cue.Attribute {
	for _, attr := range v.Attributes(cue.FieldAttr) {
		if attr.Name() == EnvAttrName {
			return &attr
		}
	}
	return nil
}

// parseEnvAttr parses an @env() attribute
func parseEnvAttr(a *cue.Attribute) (envAttr, error) {
	var env envAttr

	nameArg, ok, err := a.Lookup(0, EnvNameKey)
	if err != nil {
		return env, err
	}
	if !ok {
		return env, fmt.Errorf("missing name key in attribute body '%s'", a.Contents())
	}
	env.name = nameArg

	typeArg, ok, err := a.Lookup(0, EnvTypeKey)
	if err != nil {
		return env, err
	}
	if !ok {
		return env, fmt.Errorf("missing type key in attribute body '%s'", a.Contents())
	}
	env.envType = envType(typeArg)

	return env, nil
}
