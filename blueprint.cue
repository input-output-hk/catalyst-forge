global: {
	ci: {
		local: [
			"^check(-.*)?$",
			"^build(-.*)?$",
			"^package(-.*)?$",
			"^test(-.*)?$",
			"^nightly(-.*)?$",
		]
		registries: [
			"ghcr.io/input-output-hk/catalyst-forge",
		]
		providers: {
			aws: {
				ecr: {
					autoCreate: true
					registry:   "332405224602.dkr.ecr.eu-central-1.amazonaws.com"
				}
				region: "eu-central-1"
				role:   "arn:aws:iam::332405224602:role/ci"
			}

			cue: {
				install:        true
				registry:       aws.ecr.registry
				registryPrefix: "cue"
				version:        "0.11.0"
			}

			docker: credentials: {
				provider: "aws"
				path:     "global/ci/docker"
			}

			earthly: {
				credentials: {
					provider: "aws"
					path:     "global/ci/earthly"
				}
				org:       "Catalyst"
				satellite: "ci"
				version:   "0.8.15"
			}

			git: credentials: {
				provider: "aws"
				path:     "global/ci/deploy"
			}

			github: {
				credentials: {
					provider: "aws"
					path:     "global/ci/github"
				}
				registry: "ghcr.io"
			}

			kcl: {
				install: true
				registries: [
					"ghcr.io/input-output-hk/catalyst-forge",
				]
				version: "v0.11.0"
			}
		}
		secrets: [
			{
				name:     "GITHUB_TOKEN"
				optional: true
				provider: "env"
				path:     "GITHUB_TOKEN"
			},
		]
	}
	deployment: {
		registries: {
			containers: "ghcr.io/input-output-hk/catalyst-forge"
			modules:    ci.providers.aws.ecr.registry + "/catalyst-deployments"
		}
		repo: {
			url: "https://github.com/input-output-hk/catalyst-world"
			ref: "master"
		}
		root: "k8s"
	}
	repo: {
		defaultBranch: "master"
		name:          "input-output-hk/catalyst-forge"
	}
}
