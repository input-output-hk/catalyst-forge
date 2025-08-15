package attestation

// SLSAProvenance represents SLSA provenance predicate
type SLSAProvenance struct {
	Builder   Builder    `json:"builder"`
	BuildType string     `json:"buildType"`
	Invocation Invocation `json:"invocation,omitempty"`
	Materials []Material `json:"materials,omitempty"`
}

// Builder represents the builder in SLSA provenance
type Builder struct {
	ID string `json:"id"`
}

// Invocation represents build invocation details
type Invocation struct {
	ConfigSource ConfigSource `json:"configSource,omitempty"`
	Parameters   interface{}  `json:"parameters,omitempty"`
	Environment  interface{}  `json:"environment,omitempty"`
}

// ConfigSource represents the source of the build config
type ConfigSource struct {
	URI        string            `json:"uri,omitempty"`
	Digest     map[string]string `json:"digest,omitempty"`
	EntryPoint string            `json:"entryPoint,omitempty"`
}

// Material represents a material used in the build
type Material struct {
	URI    string            `json:"uri"`
	Digest map[string]string `json:"digest,omitempty"`
}

// CreateSLSAProvenance creates a SLSA provenance predicate
func CreateSLSAProvenance(builderID, buildType string, materials []Material) *SLSAProvenance {
	return &SLSAProvenance{
		Builder: Builder{
			ID: builderID,
		},
		BuildType: buildType,
		Materials: materials,
	}
}