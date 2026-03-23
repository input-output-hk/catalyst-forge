package ca

import (
	"time"
)

// ClampTTL clamps requested to be no more than cap when cap>0
func ClampTTL(requested time.Duration, cap time.Duration) time.Duration {
	if cap > 0 && requested > cap {
		return cap
	}
	return requested
}
