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

## Project Configuration

A project's configuration is defined within the blueprint file contained at the root of the project folder.
How this file is configured is outside the scope of this document.
See the [blueprints](./blueprints.md) section for more details.