package secrets

type Provider string

const (
	ProviderAWS   Provider = "aws"
	ProviderEnv   Provider = "env"
	ProviderLocal Provider = "local"
)
