package auth

import (
	"fmt"
	"time"

	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt/tokens"
)

type GenerateCmd struct {
	Admin       bool              `kong:"short='a',help='Generate admin token'"`
	Expiration  time.Duration     `kong:"short='e',help='Expiration time for the token',default='1h'"`
	Permissions []auth.Permission `kong:"short='p',help='Permissions to generate'"`
	PrivateKey  string            `kong:"short='k',help='Path to the private key to use for signing',type='existingfile'"`
	Subject     string            `kong:"short='s',help='Subject (email) to use in sub claim'"`
}

func (g *GenerateCmd) Run() error {
	// Use the new ES256Manager
	manager, err := jwt.NewES256Manager(g.PrivateKey, "")
	if err != nil {
		return err
	}

	// Determine user ID (subject) and permissions
	userID := g.Subject
	if userID == "" {
		userID = "user@foundry.dev"
	}
	permissions := g.Permissions
	if g.Admin {
		if g.Subject == "" {
			userID = "admin@foundry.dev"
		}
		permissions = auth.AllPermissions
	}

	// Generate token using the new tokens package (include default user_ver=1 to satisfy freshness check)
	token, err := tokens.GenerateAuthToken(
		manager,
		userID,
		permissions,
		g.Expiration,
		jwt.WithAdditionalClaims(map[string]any{
			"user_ver": 1,
		}),
	)
	if err != nil {
		return err
	}

	fmt.Println(token)
	return nil
}
