package schema

#Blueprint: {
	version: string & =~"^\\d+\\.\\d+"
}

#Project: {
	name:      _ & =~"^[a-z][a-z0-9_-]*$"
	container: _ | *name
}

#Tagging: {
	strategy: _ & "commit"
}
