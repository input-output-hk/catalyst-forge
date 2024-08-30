version: "1.0"
ci: {
	registries: [
		ci.providers.aws.registry,
	]
	providers: {
		aws: {
			region:   "eu-central-1"
			registry: "332405224602.dkr.ecr.eu-central-1.amazonaws.com"
			role:     "arn:aws:iam::332405224602:role/ci"
		}
		docker: {
			credentials: {
				provider: "aws"
				path:     "global/ci/docker"
			}
		}
		earthly: {
			credentials: {
				provider: "aws"
				path:     "global/ci/earthly"
			}
			org:       "Catalyst"
			satellite: "ci"
		}
		github: {
			registry: "ghcr.io"
		}
	}
}
