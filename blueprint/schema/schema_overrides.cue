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

#Providers: {
	aws: _ | *{}
	docker: _ | *{}
	earthly: _ | *{}
}

#ProviderAWS: {
	role:   _ | *""
	region: _ | *""
}

#ProviderDocker: {
	credentials: _ | *#Secret
}

#ProviderEarthly: {
	credentials: _ | *#Secret
}

#Secret: {
	path:     _ | *""
	provider: _ | *""
	maps: _ | *{}
}

#Target: {
	args: _ | *{}
	privileged: _ | *false
	retries:    _ | *0
	secrets: _ | *[]
}
