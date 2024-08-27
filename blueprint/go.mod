module github.com/input-output-hk/catalyst-forge/blueprint

require (
	cuelang.org/go v0.10.0
	github.com/Masterminds/semver/v3 v3.2.1
	github.com/input-output-hk/catalyst-forge/cuetools v0.0.0
	github.com/spf13/afero v1.11.0
)

require (
	github.com/cockroachdb/apd/v3 v3.2.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/mod v0.20.0 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/input-output-hk/catalyst-forge/cuetools => ../cuetools

go 1.22.3
