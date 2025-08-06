package helm

import (
	"os"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/lib/external/helm"
	"github.com/input-output-hk/catalyst-forge/lib/external/helm/mocks"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/require"
)

func TestHelmManifestGenerator(t *testing.T) {
	golden, err := os.ReadFile("./testdata/golden.yaml")
	require.NoError(t, err)

	m := &mocks.ClientMock{
		TemplateFunc: func(config helm.TemplateConfig) (string, error) {
			return string(golden), nil
		},
	}

	gen := &HelmManifestGenerator{
		client: m,
		logger: testutils.NewNoopLogger(),
	}

	mod := sp.Module{
		Instance:  "test",
		Name:      "test",
		Namespace: "default",
		Registry:  "https://charts.test.com/repo",
		Values: map[string]interface{}{
			"image": map[string]interface{}{
				"tag": "1.27.0",
			},
		},
		Version: "1.0.0",
	}

	result, err := gen.Generate(mod, getRaw(mod), "test")
	require.NoError(t, err)

	require.Equal(t, string(golden), string(result))
}


func getRaw(m sp.Module) cue.Value {
	ctx := cuecontext.New()
	return ctx.Encode(m)
}
