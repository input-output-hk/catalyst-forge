# Github Release

The `github` release type creates a new GitHub release and uploads any artifacts produced from the configured target as release
assets.

## Config

| Field    | Description                                    | Type   | Required | Default |
| -------- | ---------------------------------------------- | ------ | -------- | ------- |
| `name`   | The name to use for the release                | string | yes      | N/A     |
| `prefix` | The prefix to use for naming assets            | string | yes      | N/A     |
| `token`  | The GitHub token to use for creating a release | secret | yes      | N/A     |

## How it Works

The `github` release calls the configured Earthly target and expects artifacts to be generated (using `SAVE ARTIFACT`).
Any number of artifacts may be generated and they do not need to be saved with a specific name.
After the target successfully completes, the release will validate that at least one artifact was produced by the target.
It is an error for the target to produce zero artifacts.
At this point, if there is no currently triggered event, the release will stop.

In the case where a release event is firing, all artifacts are archived and compressed in a `.tar.gz` file.
The file name consists of the configured `prefix` plus the current platform (e.g., `prefix-linux-amd64.tar.gz`).
In the case where multiple platforms were specified in the target configuration, an archive is created and uploaded for each
platform.

Finally, the release uses the GitHub API (using the provided `token`) to create a new release and upload the artifacts gathered in
the previous step.
The name of the release is determined by the `name` field.

## Authentication

The `github` release needs a valid GitHub token with write permissions to the target repository.
The default CI pipeline run by Forge will ensure the generated `GITHUB_TOKEN` has sufficient write permissions.
The `token` field is of the `secret` type and matches the format of secrets used elsewhere in blueprints.
To use the `GITHUB_TOKEN` present in the current context, configure the `token` field as seen below:

```cue
project: {
    release: {
		github: {
			on: tag: {}
			config: {
				token: {
					provider: "env"
					path:     "GITHUB_TOKEN"
				}
			}
		}
	}
}
```

## Triggering

It's recommended to always use the `tag` event type when configuring the release's `on` field.
This is because the release expects a git tag to exist in the current context so that it can properly configure the GitHub release.
If the release fails to find a tag it will stop execution and fail.