package satellite

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/input-output-hk/catalyst-forge/cli/internal/testutils"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	sb "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	sc "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
	sg "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global/providers"
	"github.com/input-output-hk/catalyst-forge/lib/secrets"
	smocks "github.com/input-output-hk/catalyst-forge/lib/secrets/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/stretchr/testify/require"
)

func TestEarthlySatelliteConfigure(t *testing.T) {
	type earthlySecret struct {
		Host string `json:"host"`
		Ca   string `json:"ca_certificate"`
		Cert string `json:"certificate"`
		Key  string `json:"private_key"`
	}
	tests := []struct {
		name     string
		path     string
		secret   earthlySecret
		validate func(t *testing.T, fs fs.Filesystem, err error)
	}{
		{
			name: "success",
			path: "/tmp/earthly",
			secret: earthlySecret{
				Host: "tcp://localhost:1234",
				Ca:   "ca",
				Cert: "cert",
				Key:  "key",
			},
			validate: func(t *testing.T, fs fs.Filesystem, err error) {
				require.NoError(t, err)
				exists, err := fs.Exists("/tmp/earthly/config.yml")
				require.NoError(t, err)
				require.True(t, exists)

				content, err := fs.ReadFile("/tmp/earthly/config.yml")
				require.NoError(t, err)
				require.Equal(t, `global:
    buildkit_host: tcp://localhost:1234
    tlsca: ca.pem
    tlscert: cert.pem
    tlskey: key.pem
`, string(content))

				exists, err = fs.Exists("/tmp/earthly/ca.pem")
				require.NoError(t, err)
				require.True(t, exists)

				exists, err = fs.Exists("/tmp/earthly/cert.pem")
				require.NoError(t, err)
				require.True(t, exists)

				exists, err = fs.Exists("/tmp/earthly/key.pem")
				require.NoError(t, err)
				require.True(t, exists)

				ca, err := fs.ReadFile("/tmp/earthly/ca.pem")
				require.NoError(t, err)
				require.Equal(t, "ca", string(ca))

				cert, err := fs.ReadFile("/tmp/earthly/cert.pem")
				require.NoError(t, err)
				require.Equal(t, "cert", string(cert))

				key, err := fs.ReadFile("/tmp/earthly/key.pem")
				require.NoError(t, err)
				require.Equal(t, "key", string(key))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := billy.NewInMemoryFs()

			tt.secret.Ca = base64.StdEncoding.EncodeToString([]byte(tt.secret.Ca))
			tt.secret.Cert = base64.StdEncoding.EncodeToString([]byte(tt.secret.Cert))
			tt.secret.Key = base64.StdEncoding.EncodeToString([]byte(tt.secret.Key))

			secret, err := json.Marshal(tt.secret)
			require.NoError(t, err)

			ss := &smocks.SecretProviderMock{
				GetFunc: func(path string) (string, error) {
					return string(secret), nil
				},
			}
			store := secrets.NewSecretStore(map[secrets.Provider]func(*slog.Logger) (secrets.SecretProvider, error){
				secrets.Provider("mock"): func(logger *slog.Logger) (secrets.SecretProvider, error) {
					return ss, nil
				},
			})

			sat := EarthlySatellite{
				ci:     false,
				fs:     fs,
				logger: testutils.NewNoopLogger(),
				path:   tt.path,
				project: &project.Project{
					Blueprint: sb.Blueprint{
						Global: &sg.Global{
							Ci: &sg.CI{
								Providers: &sp.Providers{
									Earthly: &sp.Earthly{
										Satellite: &sp.EarthlySatellite{
											Credentials: &sc.Secret{
												Provider: "mock",
												Path:     "foo/bar",
											},
										},
									},
								},
							},
						},
					},
				},
				secretStore: store,
			}

			tt.validate(t, fs, sat.Configure())
		})
	}
}
