package earthfile

import (
	"context"
	"testing"

	"github.com/earthly/earthly/ast/spec"
	"github.com/input-output-hk/catalyst-forge/forge/cli/internal/testutils/mocks"
)

func TestEarthfileTargets(t *testing.T) {
	earthfile := Earthfile{
		spec: spec.Earthfile{
			Targets: []spec.Target{
				{Name: "target1"},
				{Name: "target2"},
			},
		},
	}

	targets := earthfile.Targets()
	if len(targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(targets))
	}

	if targets[0] != "target1" {
		t.Errorf("expected target1, got %s", targets[0])
	}

	if targets[1] != "target2" {
		t.Errorf("expected target2, got %s", targets[1])
	}
}

func TestEarthfileFilterTargets(t *testing.T) {
	earthfile := Earthfile{
		spec: spec.Earthfile{
			Targets: []spec.Target{
				{Name: "target1"},
				{Name: "target2"},
			},
		},
	}

	targets := earthfile.FilterTargets(func(target string) bool {
		return target == "target1"
	})

	if len(targets) != 1 {
		t.Errorf("expected 1 target, got %d", len(targets))
	}

	if targets[0] != "target1" {
		t.Errorf("expected target1, got %s", targets[0])
	}
}

func TestParseEarthfile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		hasError bool
	}{
		{
			name:     "valid earthfile",
			hasError: false,
			content: `
VERSION 0.7

foo:
  LET foo = bar
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseEarthfile(context.Background(), mocks.NewMockFileSeeker(test.content))
			if test.hasError && err == nil {
				t.Error("expected error, got nil")
			}

			if !test.hasError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}
