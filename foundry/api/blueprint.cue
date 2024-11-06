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
			namespace: string | *"default" @env(name="ARGOCD_APP_NAMESPACE",type="string")
			container: "foundry-api-new-deployment"
			version:   "0.1.11"
			values: {
				app: {
					environment: "dev"
					image: {
						tag: _ @forge(name="GIT_COMMIT_HASH")
					}
					presync: {
						repoName:      "catalyst-forge"
						repoOwner:     "input-output-hk"
						commitHash:    _ @forge(name="GIT_COMMIT_HASH")
						checkInterval: 5
						timeout:       300
					}
				}
			}
		}
	}
	release: {
		docker: {
			on: {
				//merge: {}
				//tag: {}
				always: {}
			}
			config: {
				tag: _ @forge(name="GIT_COMMIT_HASH")
			}
		}
	}
}
