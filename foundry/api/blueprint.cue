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
		bundle: {
			env: "shared-services"
			modules: main: {
				name:    "app"
				version: "0.4.3"
				values: {
					deployment: containers: main: {
						image: {
							name: _ @forge(name="CONTAINER_IMAGE")
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
