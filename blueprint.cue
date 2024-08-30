version: "1.0"
ci: {
	providers: {
		aws: {
			region: "eu-central-1"
			role:   "arn:aws:iam::332405224602:role/ci"
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
			satellite: "ci"
		}
	}
}
