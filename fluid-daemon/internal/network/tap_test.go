package network

import (
	"testing"
)

func TestTAPName(t *testing.T) {
	tests := []struct {
		sandboxID string
		want      string
	}{
		{"SBX-abc123def", "fluid-abc123"},
		{"SBX-xyz", "fluid-xyz"},
		{"abc123def456", "fluid-abc123"},
		{"short", "fluid-short"},
	}

	for _, tt := range tests {
		got := TAPName(tt.sandboxID)
		if got != tt.want {
			t.Errorf("TAPName(%q) = %q, want %q", tt.sandboxID, got, tt.want)
		}
	}
}
