package network

import (
	"testing"
)

func TestTAPName(t *testing.T) {
	tests := []struct {
		sandboxID string
		want      string
	}{
		{"SBX-abc123def", "fl-abc123def"},
		{"SBX-xyz", "fl-xyz"},
		{"abc123def456", "fl-abc123def"},
		{"short", "fl-short"},
	}

	for _, tt := range tests {
		got := TAPName(tt.sandboxID)
		if got != tt.want {
			t.Errorf("TAPName(%q) = %q, want %q", tt.sandboxID, got, tt.want)
		}
	}
}
