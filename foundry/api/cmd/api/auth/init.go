package auth

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt/keys"
)

type InitCmd struct {
	OutputDir string `kong:"help='Output directory for generated keys',default='./auth-keys'"`
}

// Run executes the auth init subcommand
func (i *InitCmd) Run() error {
	if err := os.MkdirAll(i.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	keyPair, err := keys.GenerateES256Keys()
	if err != nil {
		return fmt.Errorf("failed to generate ES256 keys: %w", err)
	}

	privateKeyPath := filepath.Join(i.OutputDir, "private.pem")
	if err := os.WriteFile(privateKeyPath, keyPair.PrivateKeyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	publicKeyPath := filepath.Join(i.OutputDir, "public.pem")
	if err := os.WriteFile(publicKeyPath, keyPair.PublicKeyPEM, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	fmt.Printf("✅ Successfully generated ES256 key pair\n")
	fmt.Printf("📁 Private key: %s\n", privateKeyPath)
	fmt.Printf("📁 Public key: %s\n", publicKeyPath)
	fmt.Printf("🔐 Key type: ES256 (ECDSA with P-256 curve and SHA-256)\n")
	fmt.Printf("⚠️ Keep your private key secure and never share it!\n")

	return nil
}
