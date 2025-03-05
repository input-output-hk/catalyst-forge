package schema

import "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"

// HasAWSProviderDefined returns true if the blueprint has an AWS provider defined.
func HasAWSProviderDefined(b blueprint.Blueprint) bool {
	return b.Global != nil && b.Global.Ci != nil &&
		b.Global.Ci.Providers != nil &&
		b.Global.Ci.Providers.Aws != nil
}

// HasEarthlyProviderDefined returns true if the blueprint has an earthly provider defined.
func HasEarthlyProviderDefined(b blueprint.Blueprint) bool {
	return b.Global != nil && b.Global.Ci != nil &&
		b.Global.Ci.Providers != nil &&
		b.Global.Ci.Providers.Earthly != nil
}

// HasProjectDefined returns true if the blueprint has a project defined.
func HasProjectDefined(b blueprint.Blueprint) bool {
	return b.Project != nil
}

// HasGlobalCIDefined returns true if the blueprint has a global and ci defined.
func HasGlobalCIDefined(b blueprint.Blueprint) bool {
	return b.Global != nil && b.Global.Ci != nil
}

// HasProjectCiDefined returns true if the blueprint has a project and ci defined.
func HasProjectCiDefined(b blueprint.Blueprint) bool {
	return HasProjectDefined(b) && b.Project.Ci != nil
}
