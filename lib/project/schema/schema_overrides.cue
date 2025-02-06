package schema

#Blueprint: {
	version: string & =~"^\\d+\\.\\d+"
}

#Deployment: {
	environment: _ | *"dev"
}

#GlobalDeployment: {
	environment: _ | *"dev"
}

#DeploymentModule: {
	namespace: _ | *"default"
	type:      _ | *"kcl"
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
