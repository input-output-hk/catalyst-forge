package conditions

import rbac "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rbac"

// Register wires domain-specific conditions into the RBAC engine.
func Register() {
	rbac.RegisterCondition(conditionDnsSANsSuffixIn{})
	rbac.RegisterCondition(conditionURISANsPrefixIn{})
	rbac.RegisterCondition(conditionIPSANsInCIDRs{})
}
