package utils

// IntPtr returns a pointer to an integer.
func IntPtr(i int) *int {
	return &i
}

// StringPtr returns a pointer to a string.
func StringPtr(s string) *string {
	return &s
}

// BoolPtr returns a pointer to a boolean.
func BoolPtr(b bool) *bool {
	return &b
}
