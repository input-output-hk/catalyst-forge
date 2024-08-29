package cmds

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/secrets"
)

const (
	ConfigFileName   string = "chronos.cue"
	SecretNamePrefix string = "services/bentest"
)

type Get struct {
	Provider string `short:"p" help:"The provider of the secret store." default:"aws"`
	Path     string `arg:"" help:"The path of the secret."`
	Key      string `arg:"" help:"The key inside of the secret to get."`
}

type Set struct {
	Field    []string `short:"f" help:"A secret field to set."`
	Provider string   `short:"p" help:"The provider of the secret store." default:"aws"`
	Path     string   `arg:"" help:"The path of the secret."`
}

type SecretCmd struct {
	Get *Get `cmd:"" help:"Get a secret."`
	Set *Set `cmd:"" help:"Set a secret."`
}

func (c *Get) Run(logger *slog.Logger) error {
	store := secrets.NewDefaultSecretStore()
	client, err := store.NewClient(logger, secrets.Provider(c.Provider))
	if err != nil {
		logger.Error("Unable to create secret client.", "err", err)
		return fmt.Errorf("unable to create secret client: %w", err)
	}

	s, err := client.Get(c.Path)
	if err != nil {
		return fmt.Errorf("could not get secret: %w", err)
	}

	m := make(map[string]string)

	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return err
	}

	if _, ok := m[c.Key]; !ok {
		return fmt.Errorf("key %s not found in secret at %s", c.Key, c.Path)
	}

	fmt.Println(m[c.Key])
	return nil
}

func (c *Set) Run(logger *slog.Logger) error {
	store := secrets.NewDefaultSecretStore()
	client, err := store.NewClient(logger, secrets.Provider(secrets.ProviderAWS))
	if err != nil {
		logger.Error("Unable to create secret client.", "err", err)
		return fmt.Errorf("unable to create secret client: %w", err)
	}

	fields := make(map[string]string)
	for _, f := range c.Field {
		kv := strings.Split(f, "=")
		if len(kv) != 2 {
			return fmt.Errorf("invalid field format: %s: must be in the format of key=value", f)
		}

		fields[kv[0]] = kv[1]
	}

	b, err := json.Marshal(&fields)
	if err != nil {
		return err
	}

	id, err := client.Set(c.Path, string(b))
	if err != nil {
		logger.Error("could not set secret", "err", err)
		return err
	}

	logger.Info("Successfully set secret in AWS Secretsmanager.", "id", id)

	return nil
}
