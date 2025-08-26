registry: "registry.projectcatalyst.dev"

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
							id:     "google"
							public: false
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
}
