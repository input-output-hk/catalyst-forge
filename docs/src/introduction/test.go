package main

import "testing"

func Test_greet(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Test 1",
			input:    "Alice",
			expected: "Hello, Alice!",
		},
		{
			name:     "Test 2",
			input:    "Bob",
			expected: "Hello, Bob!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := greet(tt.input)
			if actual != tt.expected {
				t.Errorf("greet(%s): expected %s, actual %s", tt.input, tt.expected, actual)
			}
		})
	}
}
