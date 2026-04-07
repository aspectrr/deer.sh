package netutil

import "testing"

func TestIsLocalHost(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		{"localhost", true},
		{"", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"[::1]", true},
		{"192.168.1.1", false},
		{"example.com", false},
		{"10.0.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := IsLocalHost(tt.host); got != tt.expected {
				t.Errorf("IsLocalHost(%q) = %v, want %v", tt.host, got, tt.expected)
			}
		})
	}
}
