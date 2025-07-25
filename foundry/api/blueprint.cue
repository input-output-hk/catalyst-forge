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

		test: privileged: true
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

						env: {
							"HTTP_PORT": {
								value: "8080"
							}
							"GIN_MODE": {
								value: "debug"
							}
							"LOG_LEVEL": {
								value: "debug"
							}
							"LOG_FORMAT": {
								value: "json"
							}
							"AUTH_PRIVATE_KEY": {
								value: "/certs/private.pem"
							}
							"AUTH_PUBLIC_KEY": {
								value: "/certs/public.pem"
							}
							"DB_INIT": {
								value: "true"
							}
							"DB_SSLMODE": {
								value: "require"
							}
							"DB_NAME": {
								value: "foundry"
							}
							"DB_ROOT_NAME": {
								value: "postgres"
							}
							"DB_HOST": {
								secret: {
									name: "db"
									key:  "host"
								}
							}
							"DB_PORT": {
								secret: {
									name: "db"
									key:  "port"
								}
							}
							"DB_USER": {
								secret: {
									name: "db"
									key:  "username"
								}
							}
							"DB_PASSWORD": {
								secret: {
									name: "db"
									key:  "password"
								}
							}
							"DB_SUPER_USER": {
								secret: {
									name: "db-root"
									key:  "username"
								}
							}
							"DB_SUPER_PASSWORD": {
								secret: {
									name: "db-root"
									key:  "password"
								}
							}
						}

						mounts: {
							certs: {
								ref: secret: name: "certs"
								path: "/certs"
							}
						}

						port: 8080

						probes: {
							liveness: path:  "/healthz"
							readiness: path: "/healthz"
						}
					}

					service: {
						targetPort: 8080
						port:       8080
					}

					secrets: {
						certs: {
							ref: "foundry-api/certs"
						}
						db: {
							ref: "db/foundry"
						}
						"db-root": {
							ref: "db/root_account"
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
