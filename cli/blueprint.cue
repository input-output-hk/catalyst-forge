version: "1.0"
project: {
	name: "forge"
	ci: targets: {
		docker: {
			args: {
				version: string | *"dev" @forge(name="GIT_TAG")
			}
			platforms: [
				"linux/amd64",
				"linux/arm64",
			]
		}
		github: {
			args: {
				version: string | *"dev" @forge(name="GIT_TAG")
			}
			platforms: [
				"linux/amd64",
				"linux/arm64",
				"darwin/amd64",
				"darwin/arm64",
			]
		}
		test: retries: 3
	}
	release: {
		docker: {
			on: ["merge", "tag"]
			config: {}
		}
		github: {
			on: ["tag"]
			config: {
				token: {
					provider: "env"
					path:     "GITHUB_TOKEN"
				}
			}
		}
	}
}
