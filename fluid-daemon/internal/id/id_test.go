package id

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestGenerate_PrefixAndLength(t *testing.T) {
	prefix := "TST-"
	got := Generate(prefix)
	if !strings.HasPrefix(got, prefix) {
		t.Errorf("expected prefix %q, got %q", prefix, got)
	}
	if len(got) != len(prefix)+16 {
		t.Errorf("expected length %d, got %d (%q)", len(prefix)+16, len(got), got)
	}
}

func TestGenerate_ValidHex(t *testing.T) {
	prefix := "X-"
	got := Generate(prefix)
	hexPart := got[len(prefix):]
	if _, err := hex.DecodeString(hexPart); err != nil {
		t.Errorf("hex part %q is not valid hex: %v", hexPart, err)
	}
}

func TestGenerateRaw_Length(t *testing.T) {
	got := GenerateRaw()
	if len(got) != 16 {
		t.Errorf("expected length 16, got %d (%q)", len(got), got)
	}
}

func TestGenerateRaw_ValidHex(t *testing.T) {
	got := GenerateRaw()
	if _, err := hex.DecodeString(got); err != nil {
		t.Errorf("%q is not valid hex: %v", got, err)
	}
}

func TestGenerate_NoCollisions(t *testing.T) {
	seen := make(map[string]struct{}, 10000)
	for i := 0; i < 10000; i++ {
		id := Generate("T-")
		if _, ok := seen[id]; ok {
			t.Fatalf("collision at iteration %d: %s", i, id)
		}
		seen[id] = struct{}{}
	}
}
