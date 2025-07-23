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
			satellite: credentials: {
				provider: "aws"
				path:     "path/to/secret"
			}
			version: "latest"
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

1. **AWS Provider Setup** (if configured):
   - Authenticate to AWS using OIDC with the configured role
   - Login to Amazon ECR if a registry is specified

2. **Docker Provider Setup** (if configured):
   - Login to Docker Hub using credentials from the configured secret

3. **GitHub Provider Setup** (if configured):
   - Login to GitHub Container Registry (ghcr.io) using the GitHub token

4. **Earthly Provider Setup** (if configured):
   - Install Earthly CLI (latest or specified version)
   - Configure remote Earthly satellite authentication if credentials are provided

5. **Timoni Provider Setup** (if configured):
   - Install Timoni CLI with the specified version

6. **CUE Provider Setup** (if configured):
   - Install CUE CLI with the specified version

7. **KCL Provider Setup** (if configured):
   - Install KCL CLI with the specified version

8. **Tailscale Provider Setup** (if configured):
   - Install and configure Tailscale using OAuth2 credentials
   - Apply specified tags to the Tailscale node

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
   - `ca_certificate`: Base64-encoded string containing the common CA certificate for mTLS
   - `certificate`: Base64 encoded string containing the (signed) client certificate used to authenticate with the satellite
   - `private_key`: Base64 encoded string containing the private key used to authenticate with the satellite
   - `host`: The address of the remote satellite in the form of `tcp://hostname:8372`
1. Tailscale
   - `client_id`: The OAuth2 client ID used to authenticate with the Tailscale API
   - `client_secret`: The OAuth2 secret key used to authenticate with the Tailscale API
1. GitHub
   - `token`: The access token used to authenticate with GitHub

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

| Name                   | Description                                                          | Required | Default                 |
| ---------------------- | -------------------------------------------------------------------- | -------- | ----------------------- |
| github_token           | The GitHub token used for authentication                             | No       | `"${{ github.token }}"` |
| skip_aws               | If true, skip authenticating with AWS and configuring ECR            | No       | `"false"`               |
| skip_cue               | If true, skips installing CUE CLI if the provider is configured      | No       | `"false"`               |
| skip_docker            | If true, skip authenticating to DockerHub                            | No       | `"false"`               |
| skip_earthly_install   | If true, skip installing Earthly                                     | No       | `"false"`               |
| skip_earthly_satellite | If true, skip adding authentication for the remote Earthly satellite | No       | `"false"`               |
| skip_github            | If true, skip authenticating to GitHub Container Registry            | No       | `"false"`               |
| skip_kcl               | If true, skips installing KCL CLI if the provider is configured      | No       | `"false"`               |
| skip_tailscale         | If true, skips installing and authenticating with skip_tailscale     | No       | `"false"`               |
| skip_timoni            | If true, skips installing Timoni CLI if the provider is configured   | No       | `"false"`               |