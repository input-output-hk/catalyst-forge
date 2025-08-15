package validate

// ---------- meta.json for strict profile ----------
#KCLModuleMetaV1: {
	module: {
		name: #NonEmpty
		// Prefer SemVer; allow non-empty if you need pre-SemVer modules.
		version:    #Semver | #NonEmpty
		entry:      #NonEmpty // path to entry KCL (relative to tar root)
		kclVersion: #NonEmpty | *""
		[string]:   _
	}

	source?: {
		repo:     #URL
		commit:   string & =~"^[0-9a-f]{40}$"
		path?:    string
		[string]: _
	}

	pack: {
		// Deterministic tar checksum (64 hex, no "sha256:" prefix)
		checksum: #Hex64
		include?: [...string]
		exclude?: [...string]
		[string]: _
	}

	// Optional schema for values (JSON/JSONSchema/KCL schema as JSON)
	schema?: {
		values?:  _
		[string]: _
	}

	[string]: _
}

// ---------- Manifest shape for strict profile ----------

// Exactly two layers: one tar + one meta JSON (in any order).
#StrictManifest: {
	artifactType?: string & #MTStrictArtifact
	annotations?: {[string]: string} // allow standard OCI annotations
	// Two-element list with one tar and one meta
	layers: ([#StrictTar, #StrictMeta] | [#StrictMeta, #StrictTar]) & {if len(layers) != 2 {_|_}}
	[string]: _
}

#StrictTar: {
	mediaType: #MTStrictTar
	digest:    #Digest
	size?:     int & >=0
	[string]:  _
}

#StrictMeta: {
	mediaType: #MTStrictMeta
	digest:    #Digest
	size?:     int & >=0
	[string]:  _
}

// ---------- TOP: Validation input for strict ----------
//
// Provide a JSON doc like:
//
// {
//   "manifest": <#StrictManifest>,
//   "meta":     <#KCLModuleMetaV1>,
//   "tarHex":   "<64hex>" // sha256 hex of the tar layer (no prefix)
// }
//
// The schema enforces:
// - meta.pack.checksum == tarHex
// - manifest has exactly one tar layer and one meta layer
// - (optional) artifactType matches strict type if present
//
#KCLStrictValidation: {
	manifest: #StrictManifest
	meta:     #KCLModuleMetaV1
	tarHex:   #Hex64

	// Cross-check checksum equality.
	meta: {
		pack: {checksum: tarHex}
	}

	[string]: _
}
