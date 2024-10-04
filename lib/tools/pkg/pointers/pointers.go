package pointers

// Bool returns a pointer to a bool.
func Bool(b bool) *bool {
	return &b
}

// Int returns a pointer to an int.
func Int(i int) *int {
	return &i
}

// Int64 returns a pointer to an int64.
func Int64(i int64) *int64 {
	return &i
}

// String returns a pointer to a string.
func String(s string) *string {
	return &s
}
