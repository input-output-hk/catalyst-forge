{
	main: {
		instance:  "foundry-api"
		namespace: "default"
		path:      "."
		values: {
			image:    "nginx"
			replicas: 2
			port:     80
		}
	}
}
