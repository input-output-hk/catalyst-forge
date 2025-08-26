import "encoding/json"

project: {
	name: "mock-oidc"

	deployment: {
		on: {
			merge: {}
			tag: {}
		}

		bundle: {
			env: string | *"dev"
			modules: main: {
				name:      "app"
				namespace: "oidc"
				version:   "0.13.3"
				values: {
					deployment: {
						replicas: number | *1
						containers: main: {
							image: {
								name: _ @forge(name="CONTAINER_IMAGE", concrete=false)
								tag:  _ @forge(name="GIT_HASH_OR_TAG", concrete=false)
							}
							env: {
								CONFIG_PATH: value: "/config/config.yaml"
							}
							mounts: {
								config: {
									ref: {
										config: {
											name: "server"
										}
									}
									path: "/config"
								}
							}
							ports: {
								http: port: 8080
							}
							probes: {
								liveness: {
									path: "/healthz"
									port: 8080
								}
								readiness: {
									path: "/healthz"
									port: 8080
								}
							}
							resources: requests: {
								cpu:    string | *"256m"
								memory: string | *"256Mi"
							}
						}
					}

					#config: {
						listen_addr:      string | *":8080"
						base_url:         string | *"http://localhost:8080"
						global_secret:    string | *"0123456789abcdef0123456789abcdef"
						allow_pkce_plain: bool | *true
						providers: _ | *[]
						...
					}
					configs: server: data: {
						"config.yaml": json.Marshal(#config)
					}

					namespaces: ["oidc"]

					dns: {
						subdomain: "oidc"
						...
					}
					route: {
						rules: [
							{
								matches: [
									{
										path: {
											type:  "PathPrefix"
											value: "/"
										}
									},
								]
								target: port: 8080
							},
						]
						...
					}

					service: {}
				}
			}
		}
	}

	publishers: {
		docker: {
			on: {
				merge: {}
				tag: {}
			}

			target: "docker"
			type:   "docker"

			config: {
				tag: _ @forge(name="GIT_HASH_OR_TAG")
			}
		}
	}
}
