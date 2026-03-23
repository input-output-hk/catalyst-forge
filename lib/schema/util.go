package schema

import (
	"reflect"

	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global"
	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global/providers"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
)

// HasAWSProviderDefined returns true if the blueprint has an AWS provider defined.
func HasAWSProviderDefined(b blueprint.Blueprint) bool {
	return !reflect.DeepEqual(b.Global.Ci.Providers.Aws, providers.AWS{})
}

// HasEarthlyProviderDefined returns true if the blueprint has an earthly provider defined.
func HasEarthlyProviderDefined(b blueprint.Blueprint) bool {
	return !reflect.DeepEqual(b.Global.Ci.Providers.Earthly, providers.Earthly{})
}

// HasProjectDefined returns true if the blueprint has a project defined.
func HasProjectDefined(b blueprint.Blueprint) bool {
	return !reflect.DeepEqual(b.Project, sp.Project{})
}

// HasGlobalCIDefined returns true if the blueprint has a global and ci defined.
func HasGlobalCIDefined(b blueprint.Blueprint) bool {
	return !reflect.DeepEqual(b.Global, global.Global{}) && !reflect.DeepEqual(b.Global.Ci, global.CI{})
}

// HasProjectCiDefined returns true if the blueprint has a project and ci defined.
func HasProjectCiDefined(b blueprint.Blueprint) bool {
	return HasProjectDefined(b) && !reflect.DeepEqual(b.Project.Ci, sp.CI{})
}
