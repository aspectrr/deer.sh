package image

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExtractKernel extracts a vmlinux kernel from a QCOW2 base image.
// The kernel is mounted via NBD, copied out, and saved alongside the image.
//
// Strategy:
//  1. Mount the QCOW2 via qemu-nbd
//  2. Mount the root partition
//  3. Copy /boot/vmlinuz-* or /vmlinuz
//  4. Decompress if needed (extract-vmlinux)
//  5. Save as <image-name>.vmlinux
func ExtractKernel(ctx context.Context, imagePath string) (string, error) {
	baseName := strings.TrimSuffix(filepath.Base(imagePath), ".qcow2")
	outputPath := filepath.Join(filepath.Dir(imagePath), baseName+".vmlinux")

	if fileExists(outputPath) {
		return outputPath, nil
	}

	// Check for virt-ls/virt-cat (libguestfs) - easier approach
	if _, err := exec.LookPath("virt-cat"); err == nil {
		return extractKernelGuestfs(ctx, imagePath, outputPath)
	}

	// Fallback to manual NBD mount
	return extractKernelNBD(ctx, imagePath, outputPath)
}

// extractKernelGuestfs uses libguestfs tools to extract the kernel.
func extractKernelGuestfs(ctx context.Context, imagePath, outputPath string) (string, error) {
	// List /boot to find kernel
	cmd := exec.CommandContext(ctx, "virt-ls", "-a", imagePath, "/boot/")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("virt-ls /boot/: %w", err)
	}

	// Find vmlinuz file
	var kernelFile string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "vmlinuz") {
			kernelFile = "/boot/" + line
			break
		}
	}

	if kernelFile == "" {
		return "", fmt.Errorf("no vmlinuz found in /boot/ of %s", imagePath)
	}

	// Extract kernel
	kernelCompressed := outputPath + ".compressed"
	cmd = exec.CommandContext(ctx, "virt-cat", "-a", imagePath, kernelFile)
	kernelData, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("virt-cat %s: %w", kernelFile, err)
	}

	if err := os.WriteFile(kernelCompressed, kernelData, 0o644); err != nil {
		return "", fmt.Errorf("write compressed kernel: %w", err)
	}
	defer os.Remove(kernelCompressed)

	// Try to decompress (extract-vmlinux script or direct use)
	if err := decompressKernel(ctx, kernelCompressed, outputPath); err != nil {
		// If decompression fails, try using the compressed kernel directly
		// (microvm may support compressed kernels)
		if err := os.Rename(kernelCompressed, outputPath); err != nil {
			return "", fmt.Errorf("rename kernel: %w", err)
		}
	}

	return outputPath, nil
}

// extractKernelNBD uses qemu-nbd to mount and extract the kernel.
func extractKernelNBD(ctx context.Context, imagePath, outputPath string) (string, error) {
	// This requires root and available NBD device
	nbdDev := "/dev/nbd0"

	// Load nbd module
	_ = exec.CommandContext(ctx, "modprobe", "nbd", "max_part=8").Run()

	// Connect image
	cmd := exec.CommandContext(ctx, "qemu-nbd", "--connect="+nbdDev, imagePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("qemu-nbd connect: %w: %s", err, string(output))
	}
	defer func() {
		_ = exec.CommandContext(ctx, "qemu-nbd", "--disconnect", nbdDev).Run()
	}()

	// Mount root partition
	mountDir, err := os.MkdirTemp("", "fluid-extract-")
	if err != nil {
		return "", fmt.Errorf("create mount dir: %w", err)
	}
	defer os.RemoveAll(mountDir)

	// Try partition 1 first, then the device itself
	mounted := false
	for _, dev := range []string{nbdDev + "p1", nbdDev} {
		cmd = exec.CommandContext(ctx, "mount", "-o", "ro", dev, mountDir)
		if err := cmd.Run(); err == nil {
			mounted = true
			defer func() {
				_ = exec.CommandContext(ctx, "umount", mountDir).Run()
			}()
			break
		}
	}

	if !mounted {
		return "", fmt.Errorf("could not mount any partition from %s", imagePath)
	}

	// Find kernel
	bootDir := filepath.Join(mountDir, "boot")
	entries, err := os.ReadDir(bootDir)
	if err != nil {
		return "", fmt.Errorf("read /boot: %w", err)
	}

	var kernelPath string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "vmlinuz") {
			kernelPath = filepath.Join(bootDir, entry.Name())
			break
		}
	}

	if kernelPath == "" {
		return "", fmt.Errorf("no vmlinuz found in /boot/ of %s", imagePath)
	}

	// Copy and decompress
	kernelData, err := os.ReadFile(kernelPath)
	if err != nil {
		return "", fmt.Errorf("read kernel: %w", err)
	}

	kernelCompressed := outputPath + ".compressed"
	if err := os.WriteFile(kernelCompressed, kernelData, 0o644); err != nil {
		return "", fmt.Errorf("write compressed kernel: %w", err)
	}
	defer os.Remove(kernelCompressed)

	if err := decompressKernel(ctx, kernelCompressed, outputPath); err != nil {
		if err := os.Rename(kernelCompressed, outputPath); err != nil {
			return "", fmt.Errorf("rename kernel: %w", err)
		}
	}

	return outputPath, nil
}

// decompressKernel attempts to decompress a compressed kernel using extract-vmlinux.
func decompressKernel(ctx context.Context, inputPath, outputPath string) error {
	// Try extract-vmlinux script (ships with linux kernel source)
	extractScript, err := exec.LookPath("extract-vmlinux")
	if err != nil {
		// Try common locations
		for _, path := range []string{
			"/usr/src/linux-headers-*/scripts/extract-vmlinux",
			"/usr/lib/linux-tools-*/extract-vmlinux",
		} {
			matches, _ := filepath.Glob(path)
			if len(matches) > 0 {
				extractScript = matches[0]
				break
			}
		}
	}

	if extractScript != "" {
		cmd := exec.CommandContext(ctx, extractScript, inputPath)
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			return os.WriteFile(outputPath, output, 0o644)
		}
	}

	// Manual decompression attempts
	// Try gzip
	cmd := exec.CommandContext(ctx, "zcat", inputPath)
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return os.WriteFile(outputPath, output, 0o644)
	}

	// Try xz
	cmd = exec.CommandContext(ctx, "xzcat", inputPath)
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		return os.WriteFile(outputPath, output, 0o644)
	}

	return fmt.Errorf("could not decompress kernel (tried extract-vmlinux, gzip, xz)")
}
