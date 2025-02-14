package providers

#CUE: {
	// Install contains whether to install CUE in the CI environment.
	install?: bool | *false

	// Registry contains the CUE registry to use for publishing CUE modules.
	registry?: string

	// RegistryPrefix contains the prefix to use for CUE registries.
	registryPrefix?: string

	// The version of CUE to use in CI.
	version?: string | *"latest"
}
