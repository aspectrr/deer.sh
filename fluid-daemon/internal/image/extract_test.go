package image

import (
	"context"
	"testing"
)

func TestExtractKernel_Disabled(t *testing.T) {
	_, err := ExtractKernel(context.Background(), "/some/path.qcow2")
	if err == nil {
		t.Fatal("expected error from disabled ExtractKernel, got nil")
	}
}
