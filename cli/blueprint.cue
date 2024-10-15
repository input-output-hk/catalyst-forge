version: "1.0"
project: {
	name: "forge"
	ci: targets: {
		publish: {
			args: {
				version: string | *"dev" @forge(name="GIT_TAG")
			}
			platforms: [
				"linux/amd64",
				"linux/arm64",
			]
		}
		release: {
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
			config: {}
			on: ["merge", "tag"]
			target: "publish"
			type:   "docker"
		}
		github: {
			config: {
				token: {
					provider: "env"
					path:     "GITHUB_TOKEN"
				}
			}
			on: ["tag"]
			target: "release"
			type:   "github"
		}
	}
}
