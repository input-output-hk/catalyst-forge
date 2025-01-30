{
	main: {
		instance:  "foundry-api"
		name:      "app"
		namespace: "default"
		registry:  "332405224602.dkr.ecr.eu-central-1.amazonaws.com/catalyst-deployments"
		values: {
			service: {
				targetPort: 8080
				port:       8080
			}
			deployment: {
				containers: {
					main: {
						probes: {
							readiness: {
								path: "/"
							}
							liveness: {
								path: "/"
							}
						}
						port: 8080
						image: {
							tag:  "dd724218713a01e0d96d95d60b9dcd044980399b"
							name: "ghcr.io/input-output-hk/catalyst-forge/foundry-api"
						}
					}
				}
			}
		}
		version: "0.2.0"
	}
}