package auth

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/auth"
)

type ValidateCmd struct {
	Token     string `kong:"arg='',help='Token to validate'"`
	PublicKey string `kong:"short='k',help='Path to the public key to use for validation',type='existingfile'"`
}

func (g *ValidateCmd) Run() error {
	am, err := auth.NewAuthManager("", g.PublicKey)
	if err != nil {
		return err
	}

	claims, err := am.ValidateToken(g.Token)
	if err != nil {
		return err
	}
	fmt.Println(claims)

	return nil
}
