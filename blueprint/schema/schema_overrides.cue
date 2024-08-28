package schema

#Blueprint: {
	version:  string & =~"^\\d+\\.\\d+"
	registry: _ | *""
}

#Global: {
	satellite: _ | *""
}

#Target: {
	args: _ | *{}
	privileged: _ | *false
	retries:    _ | *0
}
