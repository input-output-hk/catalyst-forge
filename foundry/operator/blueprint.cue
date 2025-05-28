import "encoding/json"

project: {
	name: "foundry-operator"

	deployment: {
		on: {
			merge: {}
			tag: {}
		}

		bundle: {
			env: "shared-services"
			modules: {
				crd: {
					name:     "crd"
					registry: "https://github.com/input-output-hk/catalyst-forge"
					type:     "git"
					values: {
						paths: [
							"foundry/operator/config/crd/bases/foundry.projectcatalyst.io_releasedeployments.yaml",
						]
					}
					version: _ @forge(name="GIT_HASH_OR_TAG")
				}

				main: {
					name:    "app"
					version: "0.5.0"
					values: {
						deployment: containers: main: {
							image: {
								name: _ @forge(name="CONTAINER_IMAGE")
								tag:  _ @forge(name="GIT_HASH_OR_TAG")
							}

							env: {
								"CONFIG_PATH": {
									value: "/config/operator.json"
								}
								"NAMESPACE": {
									value: "default"
								}
							}

							mounts: {
								config: {
									ref: config: name: "config"
									path: "/config"
								}
							}

							// probes: {
							// 	liveness: path:  "/healthz"
							// 	readiness: path: "/healthz"
							// }
						}

						configs: config: data: "operator.json": json.Marshal({
							api_url: "http://foundry-api:8080"
							deployer: {
								git: {
									creds: {
										provider: "aws"
										path:     "global/ci/deploy"
									}
									ref: "master"
									url: "https://github.com/input-output-hk/catalyst-world"
								}
								root_dir: "k8s"
							}
							max_attempts: 3
						})

						serviceAccount: {
							create: false
							name:   "foundry-operator"
							roles: rbac: rules: [
								{
									apiGroups: ["foundry.projectcatalyst.io"]
									resources: ["releasedeployments"]
									verbs: ["create", "delete", "get", "list", "patch", "update", "watch"]
								},
								{
									apiGroups: ["foundry.projectcatalyst.io"]
									resources: ["releasedeployments/finalizers"]
									verbs: ["update"]
								},
								{
									apiGroups: ["foundry.projectcatalyst.io"]
									resources: ["releasedeployments/status"]
									verbs: ["get", "patch", "update"]
								},
							]
						}
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
