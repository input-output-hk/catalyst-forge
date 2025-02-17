# CI

One of the primary components of Catalyst Forge is the integrated CI system.
The standardization of the git repository structure allows for automatic discovery and execution of distributed CI code.
The Forge CI system takes advantage of this and dynamically builds a CI pipeline that changes as the repository changes.
Adding a new project (along with the associated CI code) is all it takes to integrate with the system.

## Architecture

![Image title](./images/pipeline_light.png#only-light)
![Image title](./images/pipeline_dark.png#only-dark)


### Discovery

The CI system is built on a simple discovery mechanism.
When the system starts, it recursively scans from the root of the repository looking for projects.
For each project found, it looks for and parses an associated `Earthfile` and collects all of the defined targets.
It then filters and orders these targets into discrete target groups using a list of reguar expressions predefined by Forge.
The name and dependency order of these groups is hardcoded and does not often change.

Each of these target groups can be considered _phases_ in the overall CI pipeline.
Each phase consists of the associated targets and each phase occurs in dependency order.
The name and order of these phases is hardcoded and does not often change.

### Execution

For each phase, the CI system spawns a series of parallel jobs that executes the current target for all discovered projects.
Each project execution is given its own unique job in GitHub Actions to allow easy identification of failing targets as well as
providing an isolated log stream.
If any target in the group fails, the entire group is considered to be failed, and CI execution stops.

!!! hint

    This means that _any_ failing project in the repository will stop CI execution.
    If a code change in one project causes a dependent project to fail, it's expected that the dependent project is fixed.
    This promotes a "full ownership" philosophy where developers are responsible for ensuring their changes keep CI passing.

Projects are not required to define targets for each phase.
In some cases, a phase may only contain a subset of all projects in the repository.
If a phase ends up with zero targets, the entire job is skipped.
This allows repositories to define a small subset of targets initially and grow as project complexity increases.

Some target executions are limited to running the associated Earthly target and then immediately finishing.
Other targets have additional logic that is executed after the target finishes running.
For more information on supported targets, please see the [reference documentation](../reference/targets.md).

## Extending

Due to the underlying discovery mechanism, creating new jobs within the CI system is as simple as adding targets to the `Earthfile`
of a project.
As long as it meets the criteria of a supported target (see the reference documentation), the system will automatically discover and
execute the target in the appropriate phase.

Using the architecture diagram above as an example, a `package` target can be added to the `bar` project with one addition to the
`Earthfile`:

```earthly
package:
    FROM ubuntu:latest

    RUN ....
```

On the next invocation of the CI system, the newly added `package` target will be discovered and executed during the package phase.