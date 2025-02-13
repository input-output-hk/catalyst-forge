package schema

#Blueprint: {
	version: string & =~"^\\d+\\.\\d+"
}

#GlobalDeployment: {
	root: _ | *"k8s"
}

#DeploymentModule: {
	environment: _ | *"dev"
	namespace:   _ | *"default"
	type:        _ | *"kcl"
}

#Project: {
	name:      _ & =~"^[a-z][a-z0-9_-]*$"
	container: _ | *name
}

#Tagging: {
	strategy: _ & "commit"
}

#TimoniProvider: {
	install: _ | *true
	version: _ | *"latest"
}
