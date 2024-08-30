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
				maps: {
					usernames: "username"
					passwords: "password"
				}
			}
		}
		earthly: {
			credentials: {
				provider: "aws"
				path:     "global/ci/earthly"
			}
		}
	}
}
