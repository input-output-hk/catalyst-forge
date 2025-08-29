import "encoding/json"

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
				version:   "0.13.4"
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
							env: {
								AUTH_MAPPING_PATH: value: "/etc/auth/mappings.yaml"
								...
							}
							mounts: {
								mappings: {
									ref: config: name: "mappings"
									path:    "/etc/auth/mappings.yaml"
									subPath: "mappings.yaml"
								}
								...
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
							...
						}
						...
					}

					configs: mappings: data: {
						"mappings.yaml": json.Marshal({
							mappings: {
								consent: {
									requirements: {
										[
											"has(kratos.identity.traits.email)",
											"has(kratos.identity.traits.domain)",
										]
									}
									id_token: {
										email:  "kratos.identity.traits.email"
										domain: "lower(kratos.identity.traits.domain)"
									}
									access_token: ext: {
										email:  "kratos.identity.traits.email"
										domain: "lower(kratos.identity.traits.domain)"
									}
								}
								token_hooks: {
									"https://token.actions.githubusercontent.com": {
										requirements: [
											"has(jwt.repository)",
											"has(jwt.ref)",
											"has(jwt.sha)",
										]
										access_token: ext: {
											gh_repository:  "jwt.repository"
											gh_ref:         "jwt.ref"
											gh_sha:         "jwt.sha"
											gh_actor:       "jwt.actor"
											gh_environment: "jwt.environment"
										}
									}
								}
							}
							policy: {
								on_error:          "deny"
								on_unknown_issuer: "warn_passthrough"
								merge_strategy:    "deep"
							}
						})
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
											value: "/api/v1"
										}
									},
								]
								target: port: 8080
							},
						]
						...
					}

					service: {}
					...
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
