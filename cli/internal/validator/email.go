package validator

import (
	"fmt"
	"regexp"
)

// Email validates an email address.
func Email(s string) error {
	if s == "" {
		return fmt.Errorf("email is required")
	}
	// Basic email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(s) {
		return fmt.Errorf("please enter a valid email address")
	}
	return nil
}
