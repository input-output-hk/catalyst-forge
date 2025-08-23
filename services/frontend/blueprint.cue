project: {
	name: "frontend"
	deployment: {
		on: {
			merge: {}
			tag: {}
		}

		bundle: {
			env: string | *"dev"
			modules: main: {
				name:    "app"
				version: "0.13.3"
				values: {
					deployment: {
						replicas: number | *1
						containers: main: {
							image: {
								name: _ @forge(name="CONTAINER_IMAGE", concrete=false)
								tag:  _ @forge(name="GIT_HASH_OR_TAG", concrete=false)
							}
							mounts: {
								config: {
									ref: {
										config: {
											name: "caddy"
										}
									}
									path:    "/etc/caddy/Caddyfile"
									subPath: "Caddyfile"
								}
							}
							ports: {
								http: port:    8080
								metrics: port: 8081
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

					configs: caddy: data: "Caddyfile": """
						{
						  admin :8081
						  metrics
						}
						http://:8080 {
						        root * /app

						        handle /healthz {
						          respond `{"status":"ok"}` 200
						        }

						        handle {
						          try_files {path} /index.html
						          file_server
						        }

						        header {
						          Cross-Origin-Opener-Policy "same-origin"
						          Cross-Origin-Embedder-Policy "require-corp"

						          / Cache-Control "public, max-age=3600, must-revalidate"
						        }

						        handle_errors {
						          rewrite * /50x.html
						          file_server
						        }

						        log
						}
						"""

					dns: {
						subdomain: "forge"
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
