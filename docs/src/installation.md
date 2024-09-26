# Installation

## Pre-requisites

Catalyst Forge heavily relies on [Earthly](https://earthly.dev/) underneath the hood.
All CI execution steps rely on calling Earthly targets defined in `Earthfile`'s.
It's recommended you install Earthly on your local system using their [installation instructions](https://earthly.dev/get-earthly).

## Forge CLI

The `forge` CLI is the primary interface in which you, as a developer, will interact with Catalyst Forge.
The CLI is written in Go and is distributed as a single binary that is available for multiple platforms.

To get started, head over to the [releases](https://github.com/input-output-hk/catalyst-forge/releases) and download the archive
that is compatible with your local system.
Once downloaded, extract the archive to find a single `forge` binary.
It's recommended you place the binary in a location on your local system that is accessible by `$PATH`.

## Validate

To validate that forge is installed correctly, you can run:

```shell
$ forge version
forge version 0.1.0 linux/amd64
config schema version 1.0.0
```