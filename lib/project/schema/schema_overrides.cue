package schema

#Blueprint: {
	version: string & =~"^\\d+\\.\\d+"
}

#Deployment: {
	environment: _ | *"dev"
}

#Module: {
	namespace: _ | *"default"
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
