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
				version: "0.13.3"
				values: {
					deployment: containers: main: {
						image: {
							name: _ @forge(name="CONTAINER_IMAGE", concrete=false)
							tag:  _ @forge(name="GIT_HASH_OR_TAG", concrete=false)
						}

						env: {
							SERVER_PUBLICBASEURL: value: string | *"https://foundry.projectcatalyst.io"
							SERVER_HTTPPORT: value:      string | *"5050"
							GIN_MODE: value:             string | *"release"
							LOG_LEVEL: value:            string | *"info"
							LOG_FORMAT: value:           string | *"json"

							// Database
							DATABASE_INIT: value:      string | *"true"
							DATABASE_SSLMODE: value:   string | *"require"
							DATABASE_NAME: value:      string | *"foundry"
							DATABASE_ROOT_NAME: value: string | *"postgres"
							DATABASE_HOST: secret: {name: "db", key: "host"}
							DATABASE_PORT: secret: {name: "db", key: "port"}
							DATABASE_USER: secret: {name: "db", key: "username"}
							DATABASE_PASSWORD: secret: {name: "db", key: "password"}
							DATABASE_ROOT_USER: secret: {name: "db-root", key: "username"}
							DATABASE_ROOT_PASSWORD: secret: {name: "db-root", key: "password"}

							// PCA configuration (non-secret)
							// PCA_CLIENT_CA_ARN: value:       "arn:aws:acm-pca:REGION:ACCT:certificate-authority/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
							// PCA_SERVER_CA_ARN: value:       "arn:aws:acm-pca:REGION:ACCT:certificate-authority/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
							// PCA_CLIENT_TEMPLATE_ARN: value: "arn:aws:acm-pca:::template/EndEntityClientAuth/V1"
							// PCA_SERVER_TEMPLATE_ARN: value: "arn:aws:acm-pca:::template/EndEntityServerAuth/V1"
							// PCA_SIGNING_ALGO_CLIENT: value: "SHA256WITHECDSA"
							// PCA_SIGNING_ALGO_SERVER: value: "SHA256WITHECDSA"
							// PCA_TIMEOUT: value:             "10s"

							// Policy
							// CLIENT_CERT_TTL_DEV: value:    "90m"
							// CLIENT_CERT_TTL_CI_MAX: value: "120m"
							// SERVER_CERT_TTL: value:        "144h"
							// ISSUANCE_RATE_HOURLY: value:   "6"
							// SESSION_MAX_ACTIVE: value:     "10"
							// REQUIRE_PERMS_AND: value:      "true"

							// Email (optional)
							// EMAIL_ENABLED: value:  "false"
							// EMAIL_PROVIDER: value: "ses"
							// EMAIL_SENDER: value:   "no-reply@example.com"
							// SES_REGION: value:     "us-east-1"
						}

						ports: {
							http: port: 5050
						}
						probes: {
							liveness: {
								path: "/healthz"
								port: 5050
							}
							readiness: {
								path: "/healthz"
								port: 5050
							}
						}
					}

					dns: {
						subdomain: "forge"
						...
					}
					route: {
						excludeMaintenancePage: true
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
								target: port: 5050
							},
						]
						...
					}

					service: {}

					secrets: {
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
