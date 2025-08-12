package main

import (
	"fmt"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/cmd/api/auth"
	libauth "github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	"github.com/spf13/cobra"
)

// authCmd represents the auth command.
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication management commands",
	Long:  `Manage authentication tokens and configuration for the Foundry API.`,
}

// authGenerateCmd represents the auth generate command.
var authGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate authentication tokens",
	Long:  `Generate JWT tokens for authentication purposes.`,
	RunE:  runAuthGenerate,
}

// authInitCmd represents the auth init command.
var authInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize authentication configuration",
	Long:  `Initialize authentication configuration by generating new key pairs.`,
	RunE:  runAuthInit,
}

// authValidateCmd represents the auth validate command.
var authValidateCmd = &cobra.Command{
	Use:   "validate [token]",
	Short: "Validate authentication tokens",
	Long:  `Validate JWT tokens and check their claims.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runAuthValidate,
}

func init() {
	// Add auth command to root
	rootCmd.AddCommand(authCmd)

	// Add subcommands to auth
	authCmd.AddCommand(authGenerateCmd)
	authCmd.AddCommand(authInitCmd)
	authCmd.AddCommand(authValidateCmd)

	// Generate command flags
	authGenerateCmd.Flags().StringP("private-key", "k", "", "Path to private key file")
	authGenerateCmd.Flags().StringP("subject", "s", "", "Subject (email) to use in sub claim")
	authGenerateCmd.Flags().DurationP("expiration", "e", time.Hour, "Expiration time for the token")
	authGenerateCmd.Flags().BoolP("admin", "a", false, "Generate admin token")
	authGenerateCmd.Flags().StringSliceP("permissions", "p", []string{}, "Permissions to generate")
    if err := authGenerateCmd.MarkFlagRequired("private-key"); err != nil {
        panic(err)
    }

	// Init command flags
	authInitCmd.Flags().String("output-dir", "./auth-keys", "Output directory for generated keys")

	// Validate command flags
	authValidateCmd.Flags().StringP("public-key", "k", "", "Path to public key file")
    if err := authValidateCmd.MarkFlagRequired("public-key"); err != nil {
        panic(err)
    }
}

func runAuthGenerate(cmd *cobra.Command, args []string) error {
	// Get flag values
	privateKey, _ := cmd.Flags().GetString("private-key")
	subject, _ := cmd.Flags().GetString("subject")
	expiration, _ := cmd.Flags().GetDuration("expiration")
	admin, _ := cmd.Flags().GetBool("admin")
	permissions, _ := cmd.Flags().GetStringSlice("permissions")

	// Convert string permissions to auth.Permission type
	var perms []libauth.Permission
	for _, p := range permissions {
		perms = append(perms, libauth.Permission(p))
	}

	// Create the generate command
	generateCmd := &auth.GenerateCmd{
		PrivateKey:  privateKey,
		Subject:     subject,
		Expiration:  expiration,
		Admin:       admin,
		Permissions: perms,
	}

	return generateCmd.Run()
}

func runAuthInit(cmd *cobra.Command, args []string) error {
	// Get flag values
	outputDir, _ := cmd.Flags().GetString("output-dir")

	// Create the init command
	initCmd := &auth.InitCmd{
		OutputDir: outputDir,
	}

	return initCmd.Run()
}

func runAuthValidate(cmd *cobra.Command, args []string) error {
	// Get flag values
	publicKey, _ := cmd.Flags().GetString("public-key")

	// Token is passed as an argument
	if len(args) == 0 {
		return fmt.Errorf("token argument is required")
	}
	token := args[0]

	// Create the validate command
	validateCmd := &auth.ValidateCmd{
		PublicKey: publicKey,
		Token:     token,
	}

	return validateCmd.Run()
}
