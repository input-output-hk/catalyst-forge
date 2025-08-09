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
							PUBLIC_BASE_URL: value: "https://foundry.projectcatalyst.io"
							HTTP_PORT: value:       "5050"
							GIN_MODE: value:        "release"
							LOG_LEVEL: value:       "info"
							LOG_FORMAT: value:      "json"

							// Auth keys mounted from Secrets Manager via CSI
							AUTH_PRIVATE_KEY: value: "/auth/jwt_private_key.pem"
							AUTH_PUBLIC_KEY: value:  "/auth/jwt_public_key.pem"

							// Invite/Refresh HMAC secrets from shared-services/foundry/auth
							INVITE_HASH_SECRET: secret: {name: "auth", key: "invite_hmac_secret"}
							REFRESH_HASH_SECRET: secret: {name: "auth", key: "refresh_hmac_secret"}

							// Database
							DB_INIT: value:      "true"
							DB_SSLMODE: value:   "require"
							DB_NAME: value:      "foundry"
							DB_ROOT_NAME: value: "postgres"
							DB_HOST: secret: {name: "db", key: "host"}
							DB_PORT: secret: {name: "db", key: "port"}
							DB_USER: secret: {name: "db", key: "username"}
							DB_PASSWORD: secret: {name: "db", key: "password"}
							DB_SUPER_USER: secret: {name: "db-root", key: "username"}
							DB_SUPER_PASSWORD: secret: {name: "db-root", key: "password"}

							// PCA configuration (non-secret)
							PCA_CLIENT_CA_ARN: value:       "arn:aws:acm-pca:REGION:ACCT:certificate-authority/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
							PCA_SERVER_CA_ARN: value:       "arn:aws:acm-pca:REGION:ACCT:certificate-authority/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
							PCA_CLIENT_TEMPLATE_ARN: value: "arn:aws:acm-pca:::template/EndEntityClientAuth/V1"
							PCA_SERVER_TEMPLATE_ARN: value: "arn:aws:acm-pca:::template/EndEntityServerAuth/V1"
							PCA_SIGNING_ALGO_CLIENT: value: "SHA256WITHECDSA"
							PCA_SIGNING_ALGO_SERVER: value: "SHA256WITHECDSA"
							PCA_TIMEOUT: value:             "10s"

							// Policy
							CLIENT_CERT_TTL_DEV: value:    "90m"
							CLIENT_CERT_TTL_CI_MAX: value: "120m"
							SERVER_CERT_TTL: value:        "144h"
							ISSUANCE_RATE_HOURLY: value:   "6"
							SESSION_MAX_ACTIVE: value:     "10"
							REQUIRE_PERMS_AND: value:      "true"

							// Email (optional)
							EMAIL_ENABLED: value:  "false"
							EMAIL_PROVIDER: value: "ses"
							EMAIL_SENDER: value:   "no-reply@example.com"
							SES_REGION: value:     "us-east-1"
						}

						mounts: {
							auth: {
								ref: secret: name: "auth"
								path: "/auth"
							}
						}

						port: 5050

						probes: {
							liveness: path:  "/healthz"
							readiness: path: "/healthz"
						}
					}

					service: {
						targetPort: 5050
						port:       5050
					}

					secrets: {
						auth: {
							ref: "shared-services/foundry/auth"
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
