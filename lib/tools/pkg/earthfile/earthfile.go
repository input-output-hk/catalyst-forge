package earthfile

import (
	"context"
	"fmt"
	"strings"

	"github.com/earthly/earthly/ast"
	"github.com/earthly/earthly/ast/spec"
	"github.com/input-output-hk/catalyst-forge/lib/tools/pkg/walker"
)

// Earthfile represents a parsed Earthfile.
type Earthfile struct {
	spec spec.Earthfile
}

// EarthfileRef represents a reference to an Earthfile and a target.
type EarthfileRef struct {
	Path   string
	Target string
}

// Targets returns the names of the targets in the Earthfile.
func (e Earthfile) Targets() []string {
	var targetNames []string
	for _, target := range e.spec.Targets {
		targetNames = append(targetNames, target.Name)
	}

	return targetNames
}

// FilterTargets returns the names of the targets in the Earthfile that pass the given filter.
func (e Earthfile) FilterTargets(filter func(string) bool) []string {
	var targetNames []string
	for _, target := range e.spec.Targets {
		if filter(target.Name) {
			targetNames = append(targetNames, target.Name)
		}
	}

	return targetNames
}

// ParseEarthfile parses an Earthfile from the given FileSeeker.
func ParseEarthfile(ctx context.Context, earthfile walker.FileSeeker) (Earthfile, error) {
	nr, err := newNamedReader(earthfile)
	if err != nil {
		return Earthfile{}, err
	}

	ef, err := ast.ParseOpts(ctx, ast.FromReader(nr))
	if err != nil {
		return Earthfile{}, err
	}

	return Earthfile{
		spec: ef,
	}, nil
}

// ParseEarthfileRef parses an Earthfile+Target pair.
func ParseEarthfileRef(ref string) (EarthfileRef, error) {
	parts := strings.Split(ref, "+")

	if len(parts) != 2 {
		return EarthfileRef{}, fmt.Errorf("invalid Earthfile+Target pair: %s", ref)
	}

	return EarthfileRef{
		Path:   parts[0],
		Target: parts[1],
	}, nil
}

// namedReader is a FileSeeker that also provides the name of the file.
// This is used to interopt with the Earthly AST parser.
type namedReader struct {
	walker.FileSeeker
	name string
}

// Name returns the name of the file.
func (n namedReader) Name() string {
	return n.name
}

// newNamedReader wraps a FileSeeker in a namedReader.
func newNamedReader(reader walker.FileSeeker) (namedReader, error) {
	stat, err := reader.Stat()
	if err != nil {
		return namedReader{}, err
	}

	return namedReader{
		FileSeeker: reader,
		name:       stat.Name(),
	}, nil
}
