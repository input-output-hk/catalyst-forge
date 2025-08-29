registry: "registry.projectcatalyst.dev"

deps: {
	db: {
		host:     "postgres-postgresql.postgres.svc.cluster.local"
		port:     5432
		user:     "postgres"
		password: "postgres"
		admin_db: "postgres"
	}
}

deployments: {
	api: {
		project: "services/api"
		target:  "docker"
		image: {
			name: "api"
			tag:  "latest"
		}
		overrides: {
			modules: main: values: {
				deployment: containers: main: {
					image: {
						name: "\(registry)/\(deployments.api.image.name)"
						tag:  deployments.api.image.tag
					}
					env: {
						DATABASE_SSLMODE: value: "disable"
					}
				}
				dns: {
					excludeEnv: true
					rootDomain: "projectcatalyst.dev"
				}
				route: {
					excludeMaintenancePage: true
					parent: {
						name:      "default"
						namespace: "envoy-gateway-system"
					}
				}
			}
		}
	}
	frontend: {
		project: "services/frontend"
		target:  "docker"
		image: {
			name: "frontend"
			tag:  "latest"
		}
		overrides: {
			modules: main: values: {
				deployment: containers: main: {
					image: {
						name: "\(registry)/\(deployments.frontend.image.name)"
						tag:  deployments.frontend.image.tag
					}
				}
				dns: {
					createEndpoint: false
					excludeEnv:     true
					rootDomain:     "projectcatalyst.dev"
				}
				route: {
					excludeMaintenancePage: true
					parent: {
						name:      "default"
						namespace: "envoy-gateway-system"
					}
				}
			}
		}
	}
	oidc: {
		project: "playground/services/mock-oidc"
		target:  "docker"
		image: {
			name: "mock-oidc"
			tag:  "latest"
		}
		overrides: {
			modules: main: values: {
				deployment: containers: main: {
					image: {
						name: "\(registry)/\(deployments.oidc.image.name)"
						tag:  deployments.oidc.image.tag
					}
				}
				#config: {
					base_url: "https://oidc.projectcatalyst.dev"
					providers: [
						{
							id:       "google"
							public:   false
							override: true
							client: {
								id:     "kratos-mock-client"
								secret: "kratos-mock-secret"
								redirect_uris: [
									"https://auth.projectcatalyst.dev/kratos/public/self-service/methods/oidc/callback/google",
								]
							}
							personas: {
								test: {
									sub: "00000000-0000-0000-0000-000000000001"
									claims: {
										email:          "admin@iohk.io"
										email_verified: true
										name:           "Admin User"
										hd:             "iohk.io"
									}
								}
							}
						},
					]
				}
				dns: {
					createEndpoint: false
					excludeEnv:     true
					rootDomain:     "projectcatalyst.dev"
				}
				route: {
					excludeMaintenancePage: true
					parent: {
						name:      "default"
						namespace: "envoy-gateway-system"
					}
				}
			}
		}
	}
	auth: {
		project: "services/ory/auth"
		target:  "docker"
		image: {
			name: "auth"
			tag:  "latest"
		}
		overrides: {
			modules: main: values: {
				deployment: containers: main: {
					image: {
						name: "\(registry)/\(deployments.auth.image.name)"
						tag:  deployments.auth.image.tag
					}
					env: {
						AUTH_SERVER_ADDR: value:       ":8080"
						AUTH_HYDRA_ADMIN_URL: value:   "http://hydra-admin:4445"
						AUTH_KRATOS_PUBLIC_URL: value: "https://auth.projectcatalyst.dev/kratos/public"
						AUTH_TLS_CA_FILE: value:       "/etc/ssl/certs/internal.crt"
					}
					mounts: {
						ca: {
							ref: config: name: "mkcert-root-bundle"
							path:    "/etc/ssl/certs/internal.crt"
							subPath: "ca.crt"
						}
					}
				}
				dns: {
					createEndpoint: false
					excludeEnv:     true
					rootDomain:     "projectcatalyst.dev"
				}
				route: {
					excludeMaintenancePage: true
					parent: {
						name:      "default"
						namespace: "envoy-gateway-system"
					}
				}
			}
		}
	}
}

tasks: {
	hydra: {
		clients: {
			forge_cli: {
				client_id:   "forge-cli"
				client_name: "Forge CLI (dev)"
				scope:       "openid offline"
				grant_types: ["authorization_code", "refresh_token"]
				response_types: ["code"]
				token_endpoint_auth_method: "none"
				redirect_uris: [
					"http://127.0.0.1:49152/callback",
					"http://127.0.0.1:34567/callback",
				]
				post_logout_redirect_uris: [
					"http://127.0.0.1:49152/logout",
				]
			}
		}
	}
}
