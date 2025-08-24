modules: main: values: {
	deployment: containers: main: {
		image: {
			name: "registry.projectcatalyst.dev/frontend"
			tag:  "latest"
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
