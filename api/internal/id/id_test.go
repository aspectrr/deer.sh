package id

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestGenerate_PrefixAndLength(t *testing.T) {
	prefix := "TST-"
	got, err := Generate(prefix)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(got, prefix) {
		t.Errorf("expected prefix %q, got %q", prefix, got)
	}
	// prefix + 16 hex chars
	if len(got) != len(prefix)+16 {
		t.Errorf("expected length %d, got %d (%q)", len(prefix)+16, len(got), got)
	}
}

func TestGenerate_ValidHex(t *testing.T) {
	prefix := "X-"
	got, err := Generate(prefix)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hexPart := got[len(prefix):]
	if _, err := hex.DecodeString(hexPart); err != nil {
		t.Errorf("hex part %q is not valid hex: %v", hexPart, err)
	}
}

func TestGenerate_NoCollisions(t *testing.T) {
	seen := make(map[string]struct{}, 10000)
	for i := 0; i < 10000; i++ {
		id, err := Generate("T-")
		if err != nil {
			t.Fatalf("unexpected error at iteration %d: %v", i, err)
		}
		if _, ok := seen[id]; ok {
			t.Fatalf("collision at iteration %d: %s", i, id)
		}
		seen[id] = struct{}{}
	}
}
