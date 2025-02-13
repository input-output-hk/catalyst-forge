# Catalyst Forge Documentation

Catalyst Forge (Forge) is the development platform that powers the software lifecycle of all applications and services for the
Project  Catalyst team.
It ships as a suite of configurations, services, and command-line utilities that, together, accelerate development of our projects.

## What is it?

Forge can best be described as a light-weight
[internal development platform](https://internaldeveloperplatform.org/what-is-an-internal-developer-platform/) (IDP).
It adheres to many of the [platform engineering](https://tag-app-delivery.cncf.io/whitepapers/platforms/) principles.
Namely, it provides a suite of tools and processes that allow developers to easily control the lifecycle of their services.
It does so in a self-service way that attempts to streamline the process and stay out of the way of developers.

The majority of Forge is shipped via a Go-based CLI tool that provides the primary interface for interaction.
The CI pipeline provided by Forge is shipped via a set of reusable [GitHub Actions] workflows.
Additionally, support for deployments is provided via [KCL] modules.
However, all operations performed by Forge are ultimately done via the CLI, even in the CI pipeline.

Forge stands on the shoulders of giants.
Building out a platform from scratch is not an easy task, nor is it necessary in today's modern software ecosystem.
Forge is built as a hybrid system with custom components intermixed with other open-source tools that excel at specific tasks.
Where possible, difficult and complex problems are handed off to these systems and Forge plays the role of orchestrating work
between them.

Forge is also an evergreen project.
It is never in a "finished" state and is always growing and/or changing to meet the needs of Project Catalyst developers.
The roadmap is constantly evolving and can be most easily tracked by examining the
[GitHub Issues](https://github.com/input-output-hk/catalyst-forge/issues).

## Why?

Innovation is the heart of the Project Catalyst community, but it's also the heart of our development team.
To foster innovation, developers need a standardized system for solving common delivery problems that "just works."
Additionally, this system needs to be accessible to ensure development time is focused on building and not troubleshooting.
However, most modern "turnkey" platforms are overly restrictive, generally do not value open source, and are often prohibitively
expensive.
For this reason, Catlayst Forge was born, with the mission of unburdening our development processes of toil and doing so in an open
and streamlined way.
It is designed by and for the Project Catalyst team.

## Goals

- Provide a centralized (per-project) configuration file for describing the software delivery cycle.
- Provide a standardized CI pipeline for building, testing and deploying projects.
- Ensure all operations are consistent both locally and in automated workflows.
- Optimize for mono-repo environments.
- Build on existing open-source tools where possible.

## Non-Goals

- Provide a comprehensive platform that addresses problem spaces outside the scope of Project Catalyst.
- Include support for tools and utilities not broadly used within Project Catalyst's existing toolset.
- Ship with a full custom graphical interface for interacting with all platform components.

## Built in the Open

Like many of our projects, Forge is being built in the open under a very permissible license.
We encourage contributors to our projects to explore Forge to learn more about our development processes.
We also openly accept contributions and/or bug fixes via pull requests.
To get started, please review our
[code of conduct](https://github.com/input-output-hk/catalyst-forge/blob/master/CODE_OF_CONDUCT.md) and
[contribution guidelines](https://github.com/input-output-hk/catalyst-forge/blob/master/CONTRIBUTING.md).


!!! note

    If you're an external contributor to Project Catalyst, we greatly value your work!
    Being familiar with Catalyst Forge is a great way to ensure your contributions are merged as quickly as possible.
    However, note that some of the features provided by Catalyst Forge will only work for core developers within the Project
    Catalyst team.
    This documentation will denote these features ahead of time.

## Getting Started

If you're new to Catalyst Forge, the quickest way to get familiar with it is through the
[getting started tutorial](./tutorials/getting_started.md).

[GitHub Actions]: https://docs.github.com/en/actions
[KCL]: https://www.kcl-lang.io/