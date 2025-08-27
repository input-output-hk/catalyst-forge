package claims

// MapKratosTraitsToTokens maps Kratos identity traits to id_token and access_token.ext fields.
// TODO: Add proper mapping for all traits
func MapKratosTraitsToTokens(traits map[string]any) (idToken map[string]any, accessExt map[string]any) {
	id := map[string]any{}
	ext := map[string]any{}

	if traits == nil {
		return id, ext
	}

	// Common trait keys
	if v, ok := traits["email"]; ok {
		id["email"] = v
	}
	if v, ok := traits["name"]; ok {
		switch t := v.(type) {
		case string:
			id["name"] = t
		case map[string]any:
			if given, ok := t["given"]; ok {
				id["given_name"] = given
			}
			if family, ok := t["family"]; ok {
				id["family_name"] = family
			}
		}
	}

	// Example ext mappings (tenancy/roles if present)
	if v, ok := traits["org_id"]; ok {
		ext["org_id"] = v
	}
	if v, ok := traits["roles"]; ok {
		ext["roles"] = v
	}
	if v, ok := traits["tenant"]; ok {
		ext["tenant"] = v
	}

	return id, ext
}
