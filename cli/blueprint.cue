project: {
	name: "forge-cli"
	ci: targets: {
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
		github: {
			on: tag: {}
			config: {
				name:   string | *"dev" @forge(name="GIT_TAG")
				prefix: project.name
			}
		}
	}
}
