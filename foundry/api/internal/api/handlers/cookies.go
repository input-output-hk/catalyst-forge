package handlers

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func setAccessCookie(c *gin.Context, jwt string, ttl time.Duration) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("cforge_at", jwt, int(ttl.Seconds()), "/", cookieDomain(c), true, true)
}

func setRefreshCookie(c *gin.Context, opaque string, ttl time.Duration) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("cforge_rt", opaque, int(ttl.Seconds()), "/", cookieDomain(c), true, true)
}

func clearAccessCookie(c *gin.Context) {
	c.SetCookie("cforge_at", "", -1, "/", cookieDomain(c), true, true)
}

func clearRefreshCookie(c *gin.Context) {
	c.SetCookie("cforge_rt", "", -1, "/", cookieDomain(c), true, true)
}

func cookieDomain(c *gin.Context) string {
	baseURL := os.Getenv("PUBLIC_BASE_URL")
	if v, ok := c.Get("public_base_url"); ok {
		if s, ok2 := v.(string); ok2 && s != "" {
			baseURL = s
		}
	}
	if baseURL == "" {
		return ""
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	host := u.Host
	if host == "" {
		host = u.Path
	}
	if host == "" {
		return ""
	}
	// Strip port if present
	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		return parts[0]
	}
	return host
}
