project: {
	name: "auth"

	deployment: {
		on: {
			merge: {}
			tag: {}
		}

		bundle: {
			env: string | *"dev"
			modules: main: {
				name:      "app"
				namespace: "auth"
				version:   "0.13.3"
				values: {
					deployment: {
						replicas: number | *1
						containers: main: {
							image: {
								name: _ @forge(name="CONTAINER_IMAGE", concrete=false)
								tag:  _ @forge(name="GIT_HASH_OR_TAG", concrete=false)
							}
							ports: {
								http: port: 8080
							}
							env: {...}
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

					dns: {
						subdomain: "auth"
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
