package secrets

type Provider string

const (
	ProviderLocal Provider = "local"
	ProviderAWS   Provider = "aws"
)
