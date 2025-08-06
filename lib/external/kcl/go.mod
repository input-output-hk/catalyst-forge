module github.com/input-output-hk/catalyst-forge/lib/external/kcl

go 1.24.2

require (
	cuelang.org/go v0.12.0
	github.com/input-output-hk/catalyst-forge/lib/tools v0.0.0-00010101000000-000000000000
)

require (
	github.com/cockroachdb/apd/v3 v3.2.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	golang.org/x/mod v0.22.0 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/input-output-hk/catalyst-forge/lib/tools => ../../tools
