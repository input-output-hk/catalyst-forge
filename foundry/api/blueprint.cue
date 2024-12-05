version: "1.0"
project: {
	name: "foundry-api"
	ci: targets: {
		docker: {
			args: {
				version: string | *"dev" @forge(name="GIT_TAG")
			}
		}
		github: {
			args: {
				version: string | *"dev" @forge(name="GIT_TAG")
			}
		}
	}
	deployment: {
		on: {
			always: {}
		}
		environment: "dev"
		modules: main: {
			container: "foundry-api-deployment"
			version:   "0.1.1"
			values: {
				environment: name: "dev"
				server: image: {
					tag: _ @forge(name="GIT_HASH_OR_TAG")
				}
			}
		}
	}
	release: {
		docker: {
			on: {
				merge: {}
				tag: {}
			}
			config: {
				tag: _ @forge(name="GIT_HASH_OR_TAG")
			}
		}
	}
}
