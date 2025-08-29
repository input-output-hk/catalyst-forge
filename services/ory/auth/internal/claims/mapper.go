package claims

// MapKratosTraitsToTokens maps Kratos identity traits to id_token and access_token.ext fields.
// This mapping reflects the traits produced by google.mapper.jsonnet which sets:
//
//	traits.email  (verified Google email)
//	traits.domain (lowercased Google Workspace domain)
//
// We mirror these into the ID Token for client use and into the access token ext
// so downstream APIs (via Oathkeeper) can read them without the ID Token.
func MapKratosTraitsToTokens(traits map[string]any) (idToken map[string]any, accessExt map[string]any) {
	id := map[string]any{}
	ext := map[string]any{}

	if traits == nil {
		return id, ext
	}

	// Email is used by clients and APIs; include in both ID token and access token ext
	if v, ok := traits["email"]; ok {
		id["email"] = v
		ext["email"] = v
	}
	// Domain (Google Workspace hosted domain) is useful for multi-tenant/rbac routing
	if v, ok := traits["domain"]; ok {
		id["domain"] = v
		ext["domain"] = v
	}

	return id, ext
}
