// Package image manages base QCOW2 images.
//
// Kernel extraction has been replaced by a configurable kernel path
// (microvm.kernel_path in daemon config). The functions below are
// retained but disabled so the build stays clean.
package image

import (
	"context"
	"fmt"
)

// ExtractKernel previously extracted a vmlinux kernel from a QCOW2 image.
// Kernel extraction is no longer used - the daemon now references a
// pre-downloaded kernel via the microvm.kernel_path config field.
func ExtractKernel(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("kernel extraction is disabled: use microvm.kernel_path config instead")
}
