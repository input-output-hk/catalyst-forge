package cmds

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/utils"
	sc "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
	"github.com/input-output-hk/catalyst-forge/lib/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
)

const (
	ConfigFileName   string = "chronos.cue"
	SecretNamePrefix string = "services/bentest"
)

type Get struct {
	Key      string `short:"k" help:"The key inside of the secret to get."`
	Project  string `help:"Path to a project to use for getting secret configuration."`
	Provider string `short:"p" help:"The provider of the secret store." default:"aws"`
	Path     string `kong:"arg,predictor=path" help:"The path to the secret (or path in a project blueprint if --project is specified)."`
}

type Set struct {
	Field    []string `short:"f" help:"A secret field to set."`
	Provider string   `short:"p" help:"The provider of the secret store." default:"aws"`
	Path     string   `kong:"arg,predictor=path" help:"The path to the secret (or path in a project blueprint if --project is specified)."`
	Project  string   `help:"Path to a project to use for getting secret configuration."`
	Value    string   `arg:"" help:"The value to set." default:""`
}

type SecretCmd struct {
	Get *Get `cmd:"" help:"Get a secret."`
	Set *Set `cmd:"" help:"Set a secret."`
}

func (c *Get) Run(ctx run.RunContext) error {
	var path, provider string
	var maps map[string]string

	if c.Project != "" {
		exists, err := fs.Exists(c.Project)
		if err != nil {
			return fmt.Errorf("could not check if project exists: %w", err)
		} else if !exists {
			return fmt.Errorf("project does not exist: %s", c.Project)
		}

		project, err := ctx.ProjectLoader.Load(c.Project)
		if err != nil {
			return fmt.Errorf("could not load project: %w", err)
		}

		var secret sc.Secret
		if err := project.Raw().DecodePath(c.Path, &secret); err != nil {
			return fmt.Errorf("could not decode secret: %w", err)
		}

		path = secret.Path
		provider = secret.Provider

		if len(secret.Maps) > 0 {
			maps = secret.Maps
		} else {
			maps = make(map[string]string)
		}
	} else {
		path = c.Path
		provider = c.Provider
		maps = make(map[string]string)
	}

	client, err := ctx.SecretStore.NewClient(ctx.Logger, secrets.Provider(provider))
	if err != nil {
		ctx.Logger.Error("Unable to create secret client.", "err", err)
		return fmt.Errorf("unable to create secret client: %w", err)
	}

	s, err := client.Get(path)
	if err != nil {
		return fmt.Errorf("could not get secret: %w", err)
	}

	if len(maps) > 0 {
		mappedSecret := make(map[string]string)
		m := make(map[string]string)

		if err := json.Unmarshal([]byte(s), &m); err != nil {
			return err
		}

		for k, v := range maps {
			if _, ok := m[v]; !ok {
				return fmt.Errorf("key %s not found in secret at %s", v, path)
			}

			mappedSecret[k] = m[v]
		}

		if c.Key != "" {
			if _, ok := mappedSecret[c.Key]; !ok {
				return fmt.Errorf("key %s not found in mapped secret at %s", c.Key, path)
			}

			fmt.Println(mappedSecret[c.Key])
			return nil
		} else {
			utils.PrintJson(mappedSecret, false)
			return nil
		}
	}

	if c.Key != "" {
		m := make(map[string]string)

		if err := json.Unmarshal([]byte(s), &m); err != nil {
			return err
		}

		if _, ok := m[c.Key]; !ok {
			return fmt.Errorf("key %s not found in secret at %s", c.Key, path)
		}

		fmt.Println(m[c.Key])
	} else {
		fmt.Println(s)
	}
	return nil
}

func (c *Set) Run(ctx run.RunContext) error {
	var path, provider string

	if c.Project != "" {
		exists, err := fs.Exists(c.Project)
		if err != nil {
			return fmt.Errorf("could not check if project exists: %w", err)
		} else if !exists {
			return fmt.Errorf("project does not exist: %s", c.Project)
		}

		project, err := ctx.ProjectLoader.Load(c.Project)
		if err != nil {
			return fmt.Errorf("could not load project: %w", err)
		}

		var secret sc.Secret
		if err := project.Raw().DecodePath(c.Path, &secret); err != nil {
			return fmt.Errorf("could not decode secret: %w", err)
		}

		path = secret.Path
		provider = secret.Provider
	} else {
		path = c.Path
		provider = c.Provider
	}

	client, err := ctx.SecretStore.NewClient(ctx.Logger, secrets.Provider(provider))
	if err != nil {
		ctx.Logger.Error("Unable to create secret client.", "err", err)
		return fmt.Errorf("unable to create secret client: %w", err)
	}

	var data []byte
	if len(c.Field) > 0 {
		fields := make(map[string]string)
		for _, f := range c.Field {
			kv := strings.Split(f, "=")
			if len(kv) != 2 {
				return fmt.Errorf("invalid field format: %s: must be in the format of key=value", f)
			}

			fields[kv[0]] = kv[1]
		}

		data, err = json.Marshal(&fields)
		if err != nil {
			return err
		}
	} else {
		data = []byte(c.Value)
	}

	id, err := client.Set(path, string(data))
	if err != nil {
		ctx.Logger.Error("could not set secret", "err", err)
		return err
	}

	ctx.Logger.Info("Successfully set secret in AWS Secretsmanager.", "id", id)

	return nil
}
