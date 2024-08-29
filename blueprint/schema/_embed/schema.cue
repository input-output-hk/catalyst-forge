package schema

// Blueprint contains the schema for blueprint files.
#Blueprint: {
	version: =~"^\\d+\\.\\d+" @go(Version)
	ci:      #CI              @go(CI)
}
#CI: {
	global: #Global @go(Global)
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

// Secret contains the secret provider and a list of mappings
#Secret: {
	path:     string @go(Path)
	provider: string @go(Provider)
	maps: {
		[string]: string
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
