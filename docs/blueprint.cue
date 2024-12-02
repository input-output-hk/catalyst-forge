version: "1.0.0"
project: {
	name: "forge-docs"
	release: {
		docs: {
			on: merge: {}
			config: {
				token: {
					provider: "env"
					path:     "GITHUB_TOKEN"
				}
			}
		}
	}
}
