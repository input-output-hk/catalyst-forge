package ca

import (
	"testing"
	"time"
)

func TestClampTTL(t *testing.T) {
	tests := []struct {
		name string
		req  time.Duration
		cap  time.Duration
		want time.Duration
	}{
		{"no_cap", 5 * time.Hour, 0, 5 * time.Hour},
		{"below_cap", 30 * time.Minute, 1 * time.Hour, 30 * time.Minute},
		{"equal_cap", 1 * time.Hour, 1 * time.Hour, 1 * time.Hour},
		{"above_cap", 3 * time.Hour, 1 * time.Hour, 1 * time.Hour},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ClampTTL(tc.req, tc.cap)
			if got != tc.want {
				t.Fatalf("ClampTTL(%v,%v)=%v want %v", tc.req, tc.cap, got, tc.want)
			}
		})
	}
}
