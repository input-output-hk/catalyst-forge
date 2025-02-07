{
	helm: {
		instance:  "tailscale"
		name:      "nginx"
		namespace: "default"
		registry:  "https://charts.bitnami.com/bitnami"
		type:      "helm"
		values: {
			foo: "bar"
		}
		version: "18.3.6"
	}
}
