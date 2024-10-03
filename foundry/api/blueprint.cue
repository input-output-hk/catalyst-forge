version: "1.0"
project: {
	name: "foundry-api"
	ci: targets: {
		publish: {
			args: {
				version: string | *"dev" @env(name="GIT_TAG",type="string")
			}
		}
		release: {
			args: {
				version: string | *"dev" @env(name="GIT_TAG",type="string")
			}
		}
	}
	deployment: {
		environment: "dev"
		modules: main: {
			container: "foundry-api-deployment"
			version:   "0.1.1"
			values: {
				environment: name: "dev"
				server: image: {
					tag: _ @env(name="GIT_IMAGE_TAG",type="string")
				}
			}
		}
	}
}
