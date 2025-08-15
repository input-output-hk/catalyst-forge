package validate

// KPM-like annotations on the manifest (allow extras).
#KPMAnnotations: {
	name:         #NonEmpty
	version:      #NonEmpty
	description?: string | *""
	// Hex sha256 of packaged tar (no "sha256:" prefix)
	sum:      #Hex64
	[string]: string
}

// The single tar layer shape.
#TarLayer: {
	mediaType: #MTTar
	digest:    #Digest
	size?:     int & >=0
	[string]:  _
}

// Minimal manifest projection required for compat checks.
// Note: artifactType may be absent if an image-manifest fallback was used.
// If present, enforce it matches #MTTar.
#CompatManifest: {
	artifactType?: string & #MTTar
	annotations:   #KPMAnnotations
	layers: [...#TarLayer] & {if len(layers) != 1 {_|_}}
	[string]: _
}

// ---------- TOP: Validation input for compat ----------
//
// Provide a small JSON doc with:
// {
//   "manifest":  <#CompatManifest>,
//   "tarHex":    "<64hex>",               // sha256 hex of actual tar bytes (no prefix)
//   "layerHex":  "<64hex>"                // OPTIONAL: digest hex of the first layer (no prefix)
// }
//
// The schema enforces:
// - manifest.annotations.sum == tarHex
// - if layerHex present: "sha256:"+layerHex equals layers[0].digest
//
#KCLCompatValidation: {
	manifest:  #CompatManifest
	tarHex:    #Hex64
	layerHex?: #Hex64

	// Cross-check checksum equality with annotation.
	manifest: {
		annotations: {
			sum: tarHex
		}
	}

	// If you supply layerHex, enforce layer digest matches it.
	if layerHex != _|_ {
		manifest: {
			layers: [{
				digest: "sha256:\(layerHex)"
			}]
		}
	}

	[string]: _
}
