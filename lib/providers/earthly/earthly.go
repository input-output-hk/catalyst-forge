package earthly

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/lib/providers/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
)

// EarthlyProviderCreds is the struct that holds the credentials for the Earthly provider.
type EarthlyProviderCreds struct {
	Host        string
	PrivateKey  string
	Certificate string
	Ca          string
}

// GetEarthlyProviderCreds loads the Earthly provider credentials from the given secret.
func GetEarthlyProviderCreds(s *common.Secret, store *secrets.SecretStore, logger *slog.Logger) (EarthlyProviderCreds, error) {
	m, err := secrets.GetSecretMap(s, store, logger)
	if err != nil {
		return EarthlyProviderCreds{}, fmt.Errorf("could not get secret: %w", err)
	}

	host, ok := m["host"]
	if !ok {
		return EarthlyProviderCreds{}, fmt.Errorf("host is missing in secret")
	}

	privateKey, ok := m["private_key"]
	if !ok {
		return EarthlyProviderCreds{}, fmt.Errorf("private key is missing in secret")
	}

	certificate, ok := m["certificate"]
	if !ok {
		return EarthlyProviderCreds{}, fmt.Errorf("certificate is missing in secret")
	}

	ca, ok := m["ca_certificate"]
	if !ok {
		return EarthlyProviderCreds{}, fmt.Errorf("ca is missing in secret")
	}

	return EarthlyProviderCreds{
		Host:        host,
		PrivateKey:  privateKey,
		Certificate: certificate,
		Ca:          ca,
	}, nil
}
