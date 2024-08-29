package schema

// Blueprint contains the schema for blueprint files.
#Blueprint: {
	version: =~"^\\d+\\.\\d+" @go(Version)
	ci:      #CI              @go(CI)
}
#CI: {
	global:    #Global    @go(Global)
	providers: #Providers @go(Providers)
	secrets: (_ | *{}) & {
		{
			[string]: #Secret
		}
	} @go(Secrets,map[string]Secret)
	targets: (_ | *{}) & {
		{
			[string]: #Target
		}
	} @go(Targets,map[string]Target)
}

// Global contains the global configuration.
#Global: {
	registry: (_ | *"") & {
		string
	} @go(Registry)
	satellite: (_ | *"") & {
		string
	} @go(Satellite)
}
#Providers: {
	aws: #ProviderAWS & (_ | *{}) @go(AWS)
	docker: #ProviderDocker & (_ | *{}) @go(Docker)
	earthly: #ProviderEarthly & (_ | *{}) @go(Earthly)
}
#ProviderAWS: {
	role: (_ | *"") & {
		string
	} @go(Role)
	region: (_ | *"") & {
		string
	} @go(Region)
}
#ProviderDocker: {
	credentials: #Secret & (_ | *#Secret) @go(Credentials)
}
#ProviderEarthly: {
	credentials: #Secret & (_ | *#Secret) @go(Credentials)
}

// Secret contains the secret provider and a list of mappings
#Secret: {
	path: (_ | *"") & {
		string
	} @go(Path)
	provider: (_ | *"") & {
		string
	} @go(Provider)
	maps: (_ | *{}) & {
		{
			[string]: string
		}
	} @go(Maps,map[string]string)
}
version: "1.0"

// Target contains the configuration for a single target.
#Target: {
	args: (_ | *{}) & {
		{
			[string]: string
		}
	} @go(Args,map[string]string)
	privileged: (_ | *false) & {
		bool
	} @go(Privileged)
	retries: (_ | *0) & {
		int
	} @go(Retries)
	secrets: [...#Secret] & (_ | *[]) @go(Secrets,[]Secret)
}
