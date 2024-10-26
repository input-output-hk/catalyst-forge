# Projects

The primary component of Catalyst Forge is the _project_.
Forge is designed to interact with monorepos, where each repository contains one or more projects.
A project can be classified as a discrete deliverable within a repository: it has own its dedicated process for validating,
building, testing, and potentially deploying.

## Organizing Projects

!!! tip
    While there's no hierarchical order enforced by Forge, it's against best practices to have projects live at the root of a
    repository.
    The only exception to this case is where the repository only has a single deliverable.
    Since the global blueprint at the root of a repository is always unified with project blueprints, trying to configure a project
    in the global blueprint will result in overlapping values and will cause the parsing process to fail.

Catalyst Forge does not enforce projects live in any particular folder within the repository.
Developers are encouraged to organize the repository in whatever way makes sense for them.
The discovery mechanisms used by Forge will ensure that projects are found no matter where they live.

In some cases, projects may have dependencies on each other.
For example, one project may have a dependency on one or more projects that provide language libraries.
Whether or not these are treated as separate projects is up to developers.
If a library is used by more than one project, or is consumed externally, it's recommended to treat it as a separate project.

## Project Components

Forge discovers projects within a repository using a specific set of rules.
Namely, a valid project is any folder within the repository that contains a blueprint (`blueprint.cue`).
This is the _only_ requirement for forge to classify that directory as a project.
While a project may consist of one or more _other_ files or directories, the blueprint should always exist at the root of the
project folder.

Optionally, a project may also contain an `Earthfile` that contains definitions for the common targets used by the
[CI system](./ci.md).
The CI system automatically checks for the existence of this file after it discovers a project.
However, it's important to recognize the existence of an `Earthfile` _does not_ define a project according to Forge.

## Blueprints

A blueprint file contains the configuration for a project.
By convention, the blueprint file is named `blueprint.cue` and is placed at the root of the project folder.
A blueprint contains several options for configuring a project.
Please refer to the [reference documentation](../reference/blueprint.md) for more information.

In addition to project blueprint files, a _global_ blueprint file can also be provided at the root of the repository.
This blueprint configures global options that impact every project in the repository.
The final configuration always consists of a unification of the project and global blueprints.

## Tagging

When tagging a project, it's recommended to use the following format:

```
<project_name>/<version>
```

Various systems within Forge are configured to automatically detect and parse this tag structure.
For example, the `tag` event when configuring releases only triggers when a tag matching the current project name is found.
This structure also ensures that projects are versioned separately and are easy to identify when examining tags.