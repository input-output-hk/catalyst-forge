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
		environment: "dev"
		modules: main: {
			namespace: string | *"default" @env(name="NAMESPACE",type="string")
			container: "foundry-api-deployment"
			version:   "0.1.1"
			values: {
				environment: name: "dev"
				server: image: {
					tag: _ @forge(name="GIT_COMMIT_HASH")
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
				tag: _ @forge(name="GIT_COMMIT_HASH")
			}
		}
	}
}
