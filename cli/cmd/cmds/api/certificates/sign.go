package certificates

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/certificates"
)

type SignCmd struct {
	CSRFile    string   `short:"f" help:"Path to the PEM-encoded Certificate Signing Request file." required:"true"`
	SANs       []string `short:"s" help:"Additional Subject Alternative Names to include (can be specified multiple times)."`
	CommonName string   `short:"c" help:"Override the Common Name in the CSR."`
	TTL        string   `short:"t" help:"Requested certificate lifetime (e.g. 24h, 30d, 1y)." default:"24h"`
	Output     string   `short:"o" help:"Output file for the signed certificate (default: stdout)."`
	JSON       bool     `short:"j" help:"Output as prettified JSON instead of PEM format."`
}

func (c *SignCmd) Run(ctx run.RunContext, cl client.Client) error {
	// Read the CSR file
	csrContent, err := c.readCSRFile()
	if err != nil {
		return err
	}

	// Create the signing request
	req := &certificates.CertificateSigningRequest{
		CSR:        csrContent,
		SANs:       c.SANs,
		CommonName: c.CommonName,
		TTL:        c.TTL,
	}

	// Sign the certificate
	response, err := cl.Certificates().SignCertificate(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to sign certificate: %w", err)
	}

	// Output the result
	if c.JSON {
		return outputCertificateSigningResponseJSON(response)
	}

	return c.outputCertificate(response.Certificate)
}

func (c *SignCmd) readCSRFile() (string, error) {
	file, err := os.Open(c.CSRFile)
	if err != nil {
		return "", fmt.Errorf("failed to open CSR file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read CSR file: %w", err)
	}

	return string(content), nil
}

func (c *SignCmd) outputCertificate(certificate string) error {
	if c.Output != "" {
		// Write to file
		file, err := os.Create(c.Output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()

		_, err = file.WriteString(certificate)
		if err != nil {
			return fmt.Errorf("failed to write certificate to file: %w", err)
		}

		fmt.Printf("Certificate written to %s\n", c.Output)
	} else {
		// Write to stdout
		fmt.Print(certificate)
	}

	return nil
}
