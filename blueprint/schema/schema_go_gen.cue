// Code generated by cue get go. DO NOT EDIT.

//cue:generate cue get go github.com/input-output-hk/catalyst-forge/blueprint/schema

package schema

// Blueprint contains the schema for blueprint files.
#Blueprint: {
	version: string @go(Version)
	ci:      #CI    @go(CI)
}

#CI: {
	global: #Global @go(Global)
	secrets: {[string]: #Secret} @go(Secrets,map[string]Secret)
	targets: {[string]: #Target} @go(Targets,map[string]Target)
}

// Global contains the global configuration.
#Global: {
	registry:  string @go(Registry)
	satellite: string @go(Satellite)
}

// Secret contains the secret provider and a list of mappings
#Secret: {
	path:     string @go(Path)
	provider: string @go(Provider)
	maps: {[string]: string} @go(Maps,map[string]string)
}

// Target contains the configuration for a single target.
#Target: {
	args: {[string]: string} @go(Args,map[string]string)
	privileged: bool @go(Privileged)
	retries:    int  @go(Retries)
	secrets: [...#Secret] @go(Secrets,[]Secret)
}
