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
							name: "ghcr.io/input-output-hk/catalyst-forge/foundry-api"
							tag:  _ @forge(name="GIT_HASH_OR_TAG")
						}
						port: 8080
						probes: {
							liveness: path:  "/"
							readiness: path: "/"
						}
					}
					service: {
						targetPort: 8080
						port:       8080
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
