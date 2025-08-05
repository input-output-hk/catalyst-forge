# Catalyst Forge

<p align="center">
  <img src="logo.png" />
</p>

> A comprehensive developer platform that powers intelligent deployments and project management for Project Catalyst.

Catalyst Forge is a custom developer platform that provides intelligent deployment capabilities, project management, and CI/CD automation. It's designed as a multi-project monorepo containing several discrete components that work together to provide a comprehensive development and deployment platform.

## 🎯 Purpose

Catalyst Forge enables developers to:
- **Deploy applications** with intelligent, GitOps-style automation
- **Manage projects** through a unified CLI and API
- **Automate CI/CD** with custom GitHub Actions and workflows
- **Configure infrastructure** using CUE-based blueprints and KCL modules
- **Orchestrate containers** with Kubernetes operators and custom resources

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Docker
- Kubernetes cluster (for deployment features)
- GitHub account (for CI/CD features)

### Installation

```bash
# Clone the repository
git clone https://github.com/your-org/catalyst-forge.git
cd catalyst-forge

# Build the CLI
cd cli
go build -o forge ./cmd

# Run a basic command
./forge --help
```

### Basic Usage

```bash
# Scan a project for blueprints
./forge scan blueprint /path/to/project

# Run an Earthfile target
./forge run +my-target
```

## 📦 What's Included

### Core Components
- **CLI** (`cli/`) - Command-line interface for project management and deployments
- **Foundry** (`foundry/`) - API server and Kubernetes operator for intelligent deployments
- **Libraries** (`lib/`) - Shared Go libraries for project management and external integrations
- **Actions** (`actions/`) - Custom GitHub Actions for CI/CD automation
- **Modules** (`modules/`) - KCL-based deployment modules for Kubernetes manifests

### Documentation & Resources
- **Documentation** (`docs/`) - User guides, tutorials, and API reference
- **Examples** (`modules/examples/`) - Sample deployments and usage patterns
- **Schemas** (`lib/schema/`) - CUE-based configuration schemas

## 📋 Requirements

### Development Environment
- **Go 1.21+** - Primary development language
- **Docker** - Container runtime and builds
- **Earthly** - Build system and execution engine
- **Kubernetes** - Container orchestration (for deployment features)

### External Services
- **GitHub** - Source control and CI/CD integration
- **AWS** - Cloud infrastructure (ECR, S3, Secrets Manager)
- **PostgreSQL** - Database for API server (optional)

## 🔧 Technology Stack

### Core Technologies
- **Go** - CLI, API server, and Kubernetes operator
- **CUE** - Configuration and schema language
- **KCL** - Kubernetes Configuration Language for modules
- **Earthly** - Build system and execution engine
- **Kubernetes** - Container orchestration platform

### External Integrations
- **GitHub** - Source control and CI/CD
- **AWS** - Cloud infrastructure services
- **PostgreSQL** - Database for API persistence

## 📚 Documentation

### Getting Started
- **[Installation Guide](docs/installation.md)** - Complete setup instructions
- **[Getting Started Tutorial](docs/tutorials/getting_started.md)** - Step-by-step walkthrough
- **[Concepts](docs/concepts/)** - Core concepts and architecture

### Reference
- **[CLI Reference](cli/)** - Command-line interface documentation
- **[API Reference](foundry/api/)** - REST API documentation
- **[Blueprint Reference](docs/reference/blueprint.md)** - Configuration schema reference
- **[Deployment Reference](docs/reference/deployments.md)** - Deployment configuration guide

### Development
- **[Contributing Guidelines](CONTRIBUTING.md)** - How to contribute to the project
- **[Code of Conduct](CODE_OF_CONDUCT.md)** - Community guidelines
- **[Architecture Overview](.ai/OVERVIEW.md)** - Detailed technical architecture

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details on:
- Setting up your development environment
- Code style and standards
- Testing requirements
- Pull request process

## 📄 License

This project is dual licensed under:
- [Apache License 2.0](LICENSE-APACHE)
- [MIT License](LICENSE-MIT)

## 🔗 Related Projects

- **[Project Catalyst](https://projectcatalyst.io/)** - The platform powered by Catalyst Forge
- **[Earthly](https://earthly.dev/)** - Build system used by Catalyst Forge
- **[CUE](https://cuelang.org/)** - Configuration language for blueprints
- **[KCL](https://kcl-lang.io/)** - Kubernetes Configuration Language for modules