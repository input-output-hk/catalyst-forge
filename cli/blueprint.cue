project: {
	name: "forge-cli"
	ci: targets: {
		check: tags: ["nightly"]
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
		test: retries: attempts: 3
	}
	release: {
		github: {
			target: "github"
			on: tag: {}
			config: {
				name:   string | *"dev" @forge(name="GIT_TAG")
				prefix: project.name
				brew: {
					template: "go-v1"
					description: "Catalyst Forge CLI - A tool for building and deploying Catalyst projects"
					binary_name: "forge"
					templates: {
						repository: "https://github.com/input-output-hk/catalyst-forge.git"
						branch: "brew-release"
						path: "templates/brew"
					}
					tap: {
						repository: "https://github.com/input-output-hk/catalyst-brew.git"
						branch: "testing"
					}
				}
			}
		}
	}
}
