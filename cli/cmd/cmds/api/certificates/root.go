package certificates

import (
	"context"
	"fmt"
	"os"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client"
)

type GetRootCmd struct {
	Output string `short:"o" help:"Output file for the root certificate (default: stdout)."`
}

func (c *GetRootCmd) Run(ctx run.RunContext, cl client.Client) error {
	// Get the root certificate
	rootCert, err := cl.Certificates().GetRootCertificate(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get root certificate: %w", err)
	}

	// Output the result
	if c.Output != "" {
		// Write to file
		file, err := os.Create(c.Output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()

		_, err = file.Write(rootCert)
		if err != nil {
			return fmt.Errorf("failed to write root certificate to file: %w", err)
		}

		fmt.Printf("Root certificate written to %s\n", c.Output)
	} else {
		// Write to stdout
		fmt.Print(string(rootCert))
	}

	return nil
}
