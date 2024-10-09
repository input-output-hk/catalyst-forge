version: "1.0"
project: {
	name: "forge"
	ci: targets: {
		publish: {
			args: {
				version: string | *"dev" @env(name="GIT_TAG",type="string")
			}
			platforms: [
				"linux/amd64",
				"linux/arm64",
			]
		}
		release: {
			args: {
				version: string | *"dev" @env(name="GIT_TAG",type="string")
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
			target: "publish"
			type:   "docker"
		}
	}
}
