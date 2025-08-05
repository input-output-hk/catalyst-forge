module github.com/input-output-hk/catalyst-forge/lib/foundry/client

go 1.24.2

require github.com/input-output-hk/catalyst-forge/lib/foundry/auth v0.0.0-00010101000000-000000000000

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cyphar/filepath-securejoin v0.3.6 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-git/go-billy/v5 v5.5.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.3 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/input-output-hk/catalyst-forge/lib/tools v0.0.0-00010101000000-000000000000 // indirect
	github.com/redis/go-redis/v9 v9.11.0 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	golang.org/x/sys v0.29.0 // indirect
)

replace github.com/input-output-hk/catalyst-forge/foundry/api => ../../../foundry/api

replace github.com/input-output-hk/catalyst-forge/lib/tools => ../../tools

replace github.com/input-output-hk/catalyst-forge/lib/foundry/auth => ../auth
