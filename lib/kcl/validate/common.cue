package validate

// --------- Common patterns ---------

#NonEmpty: string & !=""

// "sha256:<64 hex>"
#Digest: string & =~"^sha256:[0-9a-f]{64}$"

// 64 hex chars (no "sha256:" prefix)
#Hex64: string & =~"^[0-9a-f]{64}$"

// RFC3339 timestamp (Z or offset)
#Timestamp: string & =~"^\\d{4}-\\d\\d-\\d\\dT\\d\\d:\\d\\d:\\d\\d(\\.\\d+)?(Z|[+\\-]\\d\\d:\\d\\d)$"

// Semver (loose)
#Semver: string & =~"^(0|[1-9]\\d*)\\.(0|[1-9]\\d*)\\.(0|[1-9]\\d*)(?:-[0-9A-Za-z-.]+)?(?:\\+[0-9A-Za-z-.]+)?$"

// Simple URL
#URL: string & =~"^https?://"

// --------- Media types / artifact types ---------

// KPM/compat (tar-only)
#MTTar: "application/vnd.oci.image.layer.v1.tar"

// Forge strict
#MTStrictArtifact: "application/vnd.projectcatalyst.kcl.module.v1+tar"
#MTStrictTar:      "application/vnd.projectcatalyst.kcl.module.v1+tar"
#MTStrictMeta:     "application/vnd.projectcatalyst.kcl.module.meta.v1+json"
