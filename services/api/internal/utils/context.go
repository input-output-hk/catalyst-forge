package utils

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// GetString returns a non-empty, trimmed string value from gin.Context.
func GetString(c *gin.Context, key string) (string, bool) {
	v, ok := c.Get(key)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}

// GetCSV returns a non-empty slice parsed from a comma-separated string in gin.Context.
// Values are trimmed, empty entries discarded, order preserved, and duplicates removed.
func GetCSV(c *gin.Context, key string) ([]string, bool) {
	s, ok := GetString(c, key)
	if !ok {
		return nil, false
	}
	parts := strings.Split(s, ",")
	if len(parts) == 0 {
		return nil, false
	}
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, exists := seen[p]; exists {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

// GetDuration returns a positive duration value from gin.Context.
func GetDuration(c *gin.Context, key string) (time.Duration, bool) {
	v, ok := c.Get(key)
	if !ok {
		return 0, false
	}
	d, ok := v.(time.Duration)
	if !ok || d <= 0 {
		return 0, false
	}
	return d, true
}
