package certificates

import (
	"encoding/json"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/certificates"
)

// outputCertificateSigningResponseJSON outputs the certificate signing response as JSON
func outputCertificateSigningResponseJSON(response *certificates.CertificateSigningResponse) error {
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}
