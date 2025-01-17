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
			merge: {}
			tag: {}
		}
		environment: "dev"
		modules: {
			main: {
				module:  "app"
				version: "0.2.0"
				values: {
					deployment: containers: main: {
						image: {
							name: "nginx"
							tag:  "latest"
						}
						port: 80
					}
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
