package middleware

import (
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authctx "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit"
)

// TestAuthBypass injects an AuthContext from headers when TEST_AUTH_BYPASS=1.
// Headers:
//
//	X-Test-User-ID (UUID), X-Test-Email, X-Test-Roles (csv), X-Test-Permissions (csv)
func TestAuthBypass() gin.HandlerFunc {
	return func(c *gin.Context) {
		if os.Getenv("TEST_AUTH_BYPASS") != "1" {
			c.Next()
			return
		}
		uidStr := c.GetHeader("X-Test-User-ID")
		if uidStr == "" {
			c.Next()
			return
		}
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			c.Next()
			return
		}
		email := c.GetHeader("X-Test-Email")
		roles := splitCSV(c.GetHeader("X-Test-Roles"))
		perms := splitCSV(c.GetHeader("X-Test-Permissions"))
		ac := authctx.AuthContext{
			UserID:           uid,
			Email:            email,
			Roles:            roles,
			Permissions:      perms,
			SessionVersion:   1,
			StepUpValidUntil: time.Now().Add(5 * time.Minute).UTC(),
		}
		ac.Set(c)
		c.Next()
	}
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
