package image

import (
	"compress/gzip"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExtractKernel_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "test.qcow2")
	vmlinuxPath := filepath.Join(dir, "test.vmlinux")

	// Create dummy image and pre-existing kernel
	if err := os.WriteFile(imagePath, []byte("fake-qcow2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(vmlinuxPath, []byte("fake-kernel"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ExtractKernel(context.Background(), imagePath)
	if err != nil {
		t.Fatalf("ExtractKernel returned error: %v", err)
	}
	if got != vmlinuxPath {
		t.Fatalf("expected %s, got %s", vmlinuxPath, got)
	}

	// Verify content unchanged (no tool was invoked)
	data, err := os.ReadFile(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fake-kernel" {
		t.Fatalf("kernel content changed unexpectedly: %s", string(data))
	}
}

func TestExtractKernel_Guestfs(t *testing.T) {
	if _, err := exec.LookPath("virt-cat"); err != nil {
		t.Skip("virt-cat not in PATH, skipping guestfs test")
	}
	if _, err := exec.LookPath("virt-customize"); err != nil {
		t.Skip("virt-customize not in PATH, skipping guestfs test")
	}
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img not in PATH, skipping guestfs test")
	}

	dir := t.TempDir()
	imagePath := filepath.Join(dir, "test.qcow2")

	// Create a minimal qcow2 image with a filesystem
	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", imagePath, "256M")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("qemu-img create failed: %v: %s", err, out)
	}

	// Use virt-customize to inject a dummy kernel
	dummyKernel := filepath.Join(dir, "vmlinuz-test")
	if err := os.WriteFile(dummyKernel, []byte("dummy-kernel-content"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd = exec.Command("virt-customize",
		"-a", imagePath,
		"--mkdir", "/boot",
		"--upload", dummyKernel+":/boot/vmlinuz-test",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("virt-customize failed: %v: %s", err, out)
	}

	got, err := ExtractKernel(context.Background(), imagePath)
	if err != nil {
		t.Fatalf("ExtractKernel returned error: %v", err)
	}

	info, err := os.Stat(got)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output kernel file is empty")
	}
}

func TestExtractKernel_NoKernelInImage(t *testing.T) {
	if _, err := exec.LookPath("virt-cat"); err != nil {
		t.Skip("virt-cat not in PATH, skipping guestfs test")
	}
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img not in PATH, skipping test")
	}

	dir := t.TempDir()
	imagePath := filepath.Join(dir, "empty.qcow2")

	// Create an empty qcow2 (no filesystem, so virt-ls will fail)
	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", imagePath, "64M")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("qemu-img create failed: %v: %s", err, out)
	}

	_, err := ExtractKernel(context.Background(), imagePath)
	if err == nil {
		t.Fatal("expected error for image with no kernel, got nil")
	}
}

func TestDecompressKernel_Gzip(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("decompressKernel requires Linux zcat, skipping on " + runtime.GOOS)
	}
	if _, err := exec.LookPath("zcat"); err != nil {
		t.Skip("zcat not in PATH, skipping decompression test")
	}

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "kernel.gz")
	outputPath := filepath.Join(dir, "kernel.vmlinux")

	// Create a gzip-compressed file
	original := []byte("this-is-a-fake-decompressed-kernel-payload")
	f, err := os.Create(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	if _, err := gw.Write(original); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	err = decompressKernel(context.Background(), inputPath, outputPath)
	if err != nil {
		t.Fatalf("decompressKernel returned error: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("could not read output: %v", err)
	}
	if string(data) != string(original) {
		t.Fatalf("decompressed content mismatch: got %q, want %q", string(data), string(original))
	}
}
