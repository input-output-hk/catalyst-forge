version: "1.0.0"
project: {
	name: "forge-docs"
	release: {
		docs: {
			on: always: {}
			config: {
				branch: "gh-pages"
				token: {
					provider: "env"
					path:     "GITHUB_TOKEN"
				}
			}
		}
	}
}
