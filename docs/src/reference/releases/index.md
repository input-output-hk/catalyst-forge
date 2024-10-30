# Releases

Releases are the primary mechanism that allow declaring what release automation should occur for projects.
They are defined per project and are aggregated and run in parallel during the CI run.
Releases are optional and are not required to be configured.

There are a growing number of release types, each fulfilling a unique purpose.
Documentation for each release type can be found on their respective pages in this category.
The rest of this page will common the features shared by all release types.

## Configuration

All types share a common set of configuration properies.
These properties are further detailed below.

### `on`

The `on` field allows specifying the events that will trigger a release to run in the CI pipeline.
The field is a map of event names to their respective configurations.
The event name is static and must match one of the supported event types.

The `on` field only specifies when a _full release_ should be created.
When none of the conditions in the `on` field are satisfied, most release types will perform a "dry run" of the release.
This usually consists of running the release target and validating it produces the expected output.
Doing so helps prevent merging a potentially broken release configuration.

The supported events are documented below.

#### `merge` event

| Field    | Description            | Type   | Default                                  |
| -------- | ---------------------- | ------ | ---------------------------------------- |
| `branch` | The target branch name | string | The value of `global.repo.defaultBranch` |

The `merge` event triggers when Forge detects that the current git branch matches the target branch specified in the configuration.

```cue
on: {
    merge: {
        branch: "foo"
    }
}
```

In the above example, the release will trigger when the CI is run for the `foo` branch.
This event type is normally left at its default value to trigger releases when commits are merged to the `main` or `master`
branches.                                                |

#### `tag` event

The `tag` event triggers when Forge detects a git tag in the current context.
This could be because `HEAD` is associated with a tag or a tag is picked up from the GitHub Actions environment.

### `target`

The `target` field specifies which Earthly target should be executed for this release.
Most releases require an artifact of some sort to be produced.
For example, the `docker` release expects a container image and the `github` release expects a set of files to upload.
Refer to the individual release type documentation for details on what it expects.

By default, the `target` option will use the release event type as the name.
For example, the `docker` release type will default to calling a target named `docker` in the `Earthfile`.

### `config`

The `config` field specifies configuration that is custom to the release type.
For more information on how to configure a release type, please refer to the associated documentation.