package main

import (
	"os"

	forgeCmd "github.com/input-output-hk/catalyst-forge/cli/cmd/cobra"
)

func main() {
	forgeCmd.InitConfig()
	rootCmd := forgeCmd.NewRootCommand()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
