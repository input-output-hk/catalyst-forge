# Setup Action

The setup action can be used to install the Forge CLI and configure various providers by reading from the root blueprint file.
The blueprint schema provides options for configuring a number of third-party providers like AWS, Earthly, etc.
The setup action will automatically interrogate these options and use them to determine which providers to set up.
The action only configures providers that have been specified in the blueprint file.

## Usage

Add a `blueprint.cue` to the root of your repository and add configuration for individual providers.
Here is an example:

```cue
ci: {
	providers: {
		aws: {
			region:   "eu-central-1"
			registry: "123456.dkr.ecr.eu-central-1.amazonaws.com"
			role:     "arn:aws:iam::123456:role/ci"
		}
		earthly: {
			credentials: {
				provider: "aws"
				path:     "path/to/secret"
			}
			org:       "myorg"
			satellite: "sat"
		}
	}
}
```

The above blueprint configures both the AWS and Earthly Cloud providers.
Once in place, simply invoke the setup action in a step:

```yaml
name: Run Setup
on:
  push:

permissions:
  contents: read
  id-token: write

jobs:
  setup:
    runs-on: ubuntu-latest
    steps:
      - name: Setup
        uses: input-output-hk/catalyst-forge/actions/setup@master
```

The action will then perform the following:

1. Install the latest version of the Forge CLI
2. Authenticate to AWS via OIDC
3. Authenticate to Earthly Cloud using the credentials in the AWS Secrets Manager secret stored at `path/to/secret`
4. Set the default Earthly Cloud organization to `myorg`

### Configuring Providers

All providers expect credentials to be passed via a secret.
The format for the secret is the same as used elsewhere in Catalyst Forge.
Notably, the setup action assumes credentials are stored in a common way inside secrets.
The secret must be a JSON string with specific keys mapping to specific credentials.

The below list documents the expected format for each provider:

1. Docker
   - `username`: The username to login with
   - `password`: The password to login with
1. Earthly
  - `token`: The Earthly Cloud token to login with

If the secret uses a different format, the `maps` field of the secret can be used to map them correctly:

```cue
ci: {
	providers: {
		docker: {
			credentials: {
				provider: "aws"
				path:     "path/to/secret"
                maps: {
                    username: "my_username"
                    password: "my_password"
                }
			}
		}
	}
}
```

In the above example, the fields `my_username` and `my_password` are remapped to the expected `username` and `password` fields.

### Local Testing

By default, the setup action installs release versions of the Forge CLI.
The `forge_version` input can be set to `local` in order to build a local version of the CLI.
This is useful for testing changes without needing to perform a release.

Note that this _only_ works when run within the Catalyst Forge repository.

## Inputs

| Name          | Description                              | Required | Default                 |
| ------------- | ---------------------------------------- | -------- | ----------------------- |
| forge_version | The version of the forge CLI to install  | No       | `"latest"`              |
| github_token  | The GitHub token used for authentication | No       | `"${{ github.token }}"` |