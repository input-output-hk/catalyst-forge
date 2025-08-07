package auth

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt/tokens"
)

type ValidateCmd struct {
	Token     string `kong:"arg='',help='Token to validate'"`
	PublicKey string `kong:"short='k',help='Path to the public key to use for validation',type='existingfile'"`
}

func (g *ValidateCmd) Run() error {
	am, err := jwt.NewES256Manager("", g.PublicKey)
	if err != nil {
		return err
	}

	claims, err := tokens.VerifyAuthToken(am, g.Token)
	if err != nil {
		return err
	}
	fmt.Printf("Token valid! User: %s, Permissions: %v\n", claims.UserID, claims.Permissions)

	return nil
}
