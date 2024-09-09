module github.com/input-output-hk/catalyst-forge/blueprint

require (
	cuelang.org/go v0.10.0
	github.com/Masterminds/semver/v3 v3.2.1
	github.com/input-output-hk/catalyst-forge/tools v0.0.0
	github.com/stretchr/testify v1.9.0
)

require (
	github.com/cockroachdb/apd/v3 v3.2.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	golang.org/x/mod v0.20.0 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/input-output-hk/catalyst-forge/tools => ../tools

go 1.22.3
