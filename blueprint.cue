version: "1.0"
ci: {
	providers: {
		aws: {
			region: "eu-central-1"
			role:   "arn:aws:iam::332405224602:role/ci"
		}
	}
}
