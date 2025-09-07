registry_host: "registry.projectcatalyst.dev"

dns: {
	domain: "projectcatalyst.dev"
}

k3d: {
	cluster_name:   "forge"
	servers:        1
	agents:         0
	http_port:      80
	https_port:     443
	api_port:       0 // 0 means auto-assign
	kubeconfig_out: "playground/kubeconfig"
	output_json:    "playground/cluster.json"
	force_recreate: false
	assume_yes:     true
}

argocd: {
	namespace:      "argocd"
	domain:         "argo.projectcatalyst.dev"
	admin_password: "admin"
	repositories: [
		{
			name: "catalyst-forge"
			url:  "https://gitea.projectcatalyst.dev/catalyst-forge"
		},
	]
}

cert_manager: {
	namespace:           "cert-manager"
	cluster_issuer_name: "selfsigned-issuer"
}

envoy_gateway: {
	namespace:          "envoy-gateway-system"
	gateway_class_name: "envoy-gateway"
}

external_secrets: {
	namespace:         "external-secrets"
	secret_store_name: "localstack-store"
}

generate: {
	namespace:           "default"
	earthly_config_path: "playground/.earthly"
	client_cert_secret:  "earthly-client-cert"
	ca_cert_secret:      "mkcert-ca"
}

gitea: {
	namespace:    "gitea"
	domain:       "git.projectcatalyst.dev"
	ssh_port:     2222
	oauth_secret: "gitea-oauth-secret-change-in-production"
	security: {
		secret_key:     "gitea-secret-key-change-in-production"
		internal_token: "gitea-internal-token-change-in-production"
	}
	persistence: {
		enabled: true
		size:    "10Gi"
	}
	repositories: [
		{
			name: "catalyst-forge"
			org:  "forge"
		},
	]
}

keycloak_operator: {
	namespace:    "keycloak-system"
	channel:      "stable"
	install_mode: "AllNamespaces"
}

keycloak: {
	namespace:      "keycloak"
	domain:         "auth.projectcatalyst.dev"
	admin_username: "admin"
	admin_password: "admin"
	realm_name:     "forge"
	client_id:      "catalyst-services"
	client_secret:  "catalyst-secret-change-me"
	features: {
		enabled: [
			"preview",
			"account-api",
			"admin-api",
			"account3",
			"admin2",
		]
		disabled: ["impersonation"]
	}
	transaction: {"xaEnabled": false}
}

localstack: {
	namespace: "localstack"
	services:  "secretsmanager,sqs"
	persistence: {
		enabled:       true
		size:          "1Gi"
		storage_class: "local-path"
	}
	region:         "us-east-1"
	aws_access_key: "test"
	aws_secret_key: "test"
	debug:          false
}

mailpit: {
	namespace:           "mailpit"
	hostname:            "mailpit.projectcatalyst.dev"
	max_messages:        500
	disable_web_ui_auth: true
}

postgres: {
	namespace: "postgres"
	version:   "15.2.0"
	database:  "postgres"
	username:  "postgres"
	password:  "postgres"
	host:      "postgres-postgresql.postgres.svc.cluster.local"
	port:      5432
	persistence: {
		enabled: true
		size:    "10Gi"
	}
	metrics_enabled: false
}

registry: {
	namespace: "registry"
	username:  "admin"
	password:  "registry-password-123"
	persistence: {
		enabled:       true
		size:          "20Gi"
		storage_class: "local-path"
	}
}

temporal: {
	namespace: "temporal"
	domain:    "temporal.projectcatalyst.dev"
	namespaces: ["default"]
	web_enabled: true
	schema: {
		setup_enabled:  true
		update_enabled: true
	}
}

trust_manager: {
	namespace: "cert-manager"
	app: {
		trust: {
			namespace: "cert-manager"
			package:   "cert-manager-package"
		}
	}
	ca_bundle: {
		name:           "mkcert-ca-bundle"
		secret_name:    "mkcert-ca"
		secret_key:     "tls.crt"
		config_map_key: "ca.crt"
	}
}

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
						name: "\(registry_host)/\(deployments.api.image.name)"
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
						name: "\(registry_host)/\(deployments.frontend.image.name)"
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
						name: "\(registry_host)/\(deployments.oidc.image.name)"
						tag:  deployments.oidc.image.tag
					}
					mounts: {
						key: {
							ref: secret: name: "mock-oidc-github-signing"
							path:    "/keys/signing.pem"
							subPath: "signing_key.pem"
						}
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
						{
							id:                   "github"
							public:               true
							override:             false
							signing_key_pem_path: "/keys/signing.pem"
							issuer_override:      "https://token.actions.githubusercontent.com"
							default_audience: ["https://auth.projectcatalyst.dev/hydra/public/oauth2/token"]
							personas: {
								default: {
									sub: "repo:acme/repo:ref:refs/heads/main"
									claims: {
										repository:  "acme/repo"
										ref:         "refs/heads/main"
										sha:         "deadbeef"
										actor:       "runner"
										environment: "dev"
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
						name: "\(registry_host)/\(deployments.auth.image.name)"
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
			gha_ci: {
				client_id:   "gha-ci"
				client_name: "GitHub Actions (dev)"
				scope:       "openid"
				grant_types: ["urn:ietf:params:oauth:grant-type:jwt-bearer"]
				token_endpoint_auth_method: "none"
				redirect_uris: []
				audience: [
					"https://forge.projectcatalyst.dev/api",
				]
			}
		}
		trusted_jwt_grant_issuers: [
			{
				issuer:            "https://token.actions.githubusercontent.com"
				jwks_uri:          "https://oidc.projectcatalyst.dev/github/jwks"
				allow_any_subject: true
			},
		]
	}
}
