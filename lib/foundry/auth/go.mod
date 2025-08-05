module github.com/input-output-hk/catalyst-forge/lib/foundry/auth

go 1.24.2

require (
	github.com/golang-jwt/jwt/v5 v5.2.3
	github.com/google/uuid v1.6.0
	github.com/input-output-hk/catalyst-forge/lib/tools v0.0.0-00010101000000-000000000000
	github.com/redis/go-redis/v9 v9.11.0
	github.com/stretchr/testify v1.10.0
	gopkg.in/square/go-jose.v2 v2.6.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cyphar/filepath-securejoin v0.3.6 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-git/go-billy/v5 v5.5.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/crypto v0.32.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/input-output-hk/catalyst-forge/lib/tools => ../../tools
