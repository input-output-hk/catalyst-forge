package helm

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/providers/helm/downloader/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/project/utils"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/require"
)

func TestHelmManifestGenerator(t *testing.T) {
	m := mocks.ChartDownloaderMock{
		DownloadFunc: func(registry, name, version string) (*bytes.Buffer, error) {
			return archive("./testdata")
		},
	}

	gen := HelmManifestGenerator{
		downloader: &m,
		logger:     testutils.NewNoopLogger(),
	}

	mod := schema.DeploymentModule{
		Instance:  "test",
		Name:      utils.StringPtr("test"),
		Namespace: "default",
		Registry:  utils.StringPtr("https://charts.test.com/repo"),
		Values: map[string]interface{}{
			"image": map[string]interface{}{
				"tag": "1.27.0",
			},
		},
		Version: utils.StringPtr("1.0.0"),
	}

	result, err := gen.Generate(mod)
	require.NoError(t, err)

	golden, err := os.ReadFile("./testdata/golden.yaml")
	require.NoError(t, err)

	require.Equal(t, string(golden), string(result))
}

func archive(dirPath string) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	err := filepath.Walk(dirPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dirPath, filePath)
		if err != nil {
			return err
		}

		header.Name = filepath.Join(filepath.Base(dirPath), relPath)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := io.Copy(tw, file); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}

	return &buf, nil
}
