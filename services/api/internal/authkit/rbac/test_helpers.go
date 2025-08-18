package rbac

// test-only helper: BuildAncestryRef constructs a ResourceRef chain from the
// leaf upwards and returns the leaf: leaf (resType/resID) -> environment? -> project? -> org?
func BuildAncestryRef(orgID, projectID, envID, resType, resID string) ResourceRef {
	leaf := ResourceRef{Type: resType, ID: resID}
	current := &leaf
	if envID != "" {
		env := ResourceRef{Type: "environment", ID: envID}
		current.Parent = &env
		current = current.Parent
	}
	if projectID != "" {
		proj := ResourceRef{Type: "project", ID: projectID}
		current.Parent = &proj
		current = current.Parent
	}
	if orgID != "" {
		org := ResourceRef{Type: "org", ID: orgID}
		current.Parent = &org
	}
	return leaf
}
