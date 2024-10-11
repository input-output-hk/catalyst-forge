version: "1.0"
project: {
	name: "foundry-api"
	ci: targets: {
		publish: {
			args: {
				version: string | *"dev" @forge(name="GIT_TAG")
			}
		}
		release: {
			args: {
				version: string | *"dev" @forge(name="GIT_TAG")
			}
		}
	}
	deployment: {
		environment: "dev"
		modules: main: {
			container: "foundry-api-deployment"
			version:   "0.1.0"
			values: {
				environment: name: "dev"
				server: image: {
					tag: _ @forge(name="GIT_IMAGE_TAG")
				}
			}
		}
	}
	release: {
		docker: {
			config: {}
			on: ["merge", "tag"]
			target: "publish"
			type:   "docker"
		}
		github: {
			config: {}
			on: ["merge", "tag"]
			target: "release"
			type:   "github"
		}
	}
}
