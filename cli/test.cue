{
	main: {
		instance:  "test"
		namespace: "default"
		path:      "../examples/module"
		values: {
			image:    "nginx"
			port:     80
			replicas: 2
		}
	}
}
