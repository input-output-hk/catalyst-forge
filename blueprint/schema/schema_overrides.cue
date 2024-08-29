package schema

#Blueprint: {
	version: string & =~"^\\d+\\.\\d+"
}

#CI: {
	secrets: _ | *{}
	targets: _ | *{}
}

#Global: {
	registry:  _ | *""
	satellite: _ | *""
}

#Target: {
	args: _ | *{}
	privileged: _ | *false
	retries:    _ | *0
	secrets: _ | *[]
}
