modules: main: values: {
	deployment: containers: main: {
		image: {
			name: "registry.projectcatalyst.dev/api"
			tag:  "latest"
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
			name:      "envoy-gateway"
			namespace: "envoy-gateway-system"
		}
	}
}
