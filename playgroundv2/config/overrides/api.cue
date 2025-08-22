modules: main: values: {
	deployment: containers: main: {
		image: {
			name: "registry.local.io/api"
			tag:  "latest"
		}
		env: {
			DATABASE_SSLMODE: value: "disable"
		}
	}
	dns: {
		createEndpoint: false
		excludeEnv:     true
		rootDomain:     "local.io"
	}
	route: parent: {
		name:      "envoy-gateway"
		namespace: "envoy-gateway-system"
	}
}
