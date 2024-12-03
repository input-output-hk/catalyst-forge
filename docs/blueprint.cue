version: "1.0.0"
project: {
	name: "forge-docs"
	release: {
		docs: {
			on: always: {}
			config: {
				branch: "gh-pages"
				branches: {
					enabled: true
					path:    "branch"
				}
				targetPath: "."
				token: {
					provider: "env"
					path:     "GITHUB_TOKEN"
				}
			}
		}
	}
}
