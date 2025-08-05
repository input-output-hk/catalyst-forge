package auth

import (
	"fmt"
	"time"

	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
)

type GenerateCmd struct {
	Admin       bool              `kong:"short='a',help='Generate admin token'"`
	Expiration  time.Duration     `kong:"short='e',help='Expiration time for the token',default='1h'"`
	Permissions []auth.Permission `kong:"short='p',help='Permissions to generate'"`
	PrivateKey  string            `kong:"short='k',help='Path to the private key to use for signing',type='existingfile'"`
}

func (g *GenerateCmd) Run() error {
	am, err := jwt.NewJWTManager(g.PrivateKey, "")
	if err != nil {
		return err
	}

	if g.Admin {
		token, err := am.GenerateToken("admin", auth.AllPermissions, g.Expiration)
		if err != nil {
			return err
		}
		fmt.Println(token)
		return nil
	}

	token, err := am.GenerateToken("user", g.Permissions, g.Expiration)
	if err != nil {
		return err
	}
	fmt.Println(token)

	return nil
}
