package snapshotpull

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const deerSnapPrefix = "deer-tmp-snap"

// LibvirtBackend snapshots and pulls a VM disk from a remote libvirt host via virsh over SSH transport.
type LibvirtBackend struct {
	sshHost         string
	sshPort         int
	sshUser         string
	sshIdentityFile string
	virshURI        string
	logger          *slog.Logger
}

// NewLibvirtBackend creates a backend that uses local virsh with qemu+ssh transport to manage the remote libvirt.
// No SSH prefix needed: virsh handles the connection via its built-in SSH transport.
func NewLibvirtBackend(host string, port int, user, identityFile string, logger *slog.Logger) *LibvirtBackend {
	if port == 0 {
		port = 22
	}
	if user == "" {
		user = "root"
	}
	if logger == nil {
		logger = slog.Default()
	}

	// Build qemu+ssh URI so local virsh connects to remote libvirtd without needing a sudoers entry.
	// no_tty=1 prevents TTY prompts in daemon context.
	var hostPart string
	if port == 22 {
		hostPart = fmt.Sprintf("%s@%s", user, host)
	} else {
		hostPart = fmt.Sprintf("%s@%s:%d", user, host, port)
	}
	virshURI := fmt.Sprintf("qemu+ssh://%s/system?no_tty=1", hostPart)
	if identityFile != "" {
		virshURI += "&keyfile=" + identityFile
	}

	return &LibvirtBackend{
		sshHost:         host,
		sshPort:         port,
		sshUser:         user,
		sshIdentityFile: identityFile,
		virshURI:        virshURI,
		logger:          logger.With("component", "libvirt-backend"),
	}
}

// runVirshCmd executes a virsh subcommand locally against the remote libvirt via qemu+ssh transport.
func (b *LibvirtBackend) runVirshCmd(ctx context.Context, args ...string) (string, error) {
	cmdArgs := append([]string{"-c", b.virshURI}, args...)
	cmd := exec.CommandContext(ctx, "virsh", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("virsh %v: %w: %s", args, err, stderr.String())
	}
	return stdout.String(), nil
}

// SnapshotAndPull downloads vmName's disk directly via virsh vol-download.
// No snapshot is created: the live disk is downloaded as-is. This avoids the
// blockcommit write-lock failures that plagued the snapshot-based flow.
func (b *LibvirtBackend) SnapshotAndPull(ctx context.Context, vmName string, destPath string) error {
	b.logger.Info("starting disk pull", "vm", vmName, "dest", destPath)

	diskPath, err := b.findCleanDiskPath(ctx, vmName)
	if err != nil {
		return fmt.Errorf("find disk path: %w", err)
	}
	b.logger.Info("found disk path", "vm", vmName, "disk", diskPath)

	// Refresh storage pools so libvirt knows about files created outside pool
	// management (e.g. overlay files from previous snapshot operations).
	b.refreshStoragePools(ctx)

	b.logger.Info("downloading disk via vol-download", "vm", vmName, "src", diskPath, "dest", destPath)
	if _, err := b.runVirshCmd(ctx, "vol-download", diskPath, destPath); err != nil {
		return fmt.Errorf("download disk: %w", err)
	}

	b.logger.Info("disk pull complete", "vm", vmName, "dest", destPath)
	return nil
}

// findDiskPath uses virsh domblklist to find the primary disk path.
func (b *LibvirtBackend) findDiskPath(ctx context.Context, vmName string) (string, error) {
	out, err := b.runVirshCmd(ctx, "domblklist", vmName, "--details")
	if err != nil {
		return "", err
	}

	// Parse output - look for "disk" type entries
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 4 && fields[0] == "file" && fields[1] == "disk" {
			return fields[3], nil
		}
	}
	return "", fmt.Errorf("no disk found for VM %s", vmName)
}

// findCleanDiskPath returns the disk path for a VM, recovering from any stale
// overlay state left by a previous failed blockcommit. Handles three scenarios:
//  1. No stale state - returns disk path directly.
//  2. VM pointing at overlay-named disk WITHOUT backing file (post-blockpull) -
//     the overlay IS the disk now, just clean metadata and return it.
//  3. VM pointing at overlay with backing file (stale) - blockpull to make
//     standalone, clean metadata, then re-query.
func (b *LibvirtBackend) findCleanDiskPath(ctx context.Context, vmName string) (string, error) {
	diskPath, err := b.findDiskPath(ctx, vmName)
	if err != nil {
		return "", err
	}

	if isDeerOverlay(diskPath) {
		hasBacking, _ := b.hasBackingFile(ctx, diskPath)
		if !hasBacking {
			// Post-blockpull: overlay is standalone, just clean metadata
			b.logger.Info("disk has overlay name but no backing file, using as-is", "vm", vmName, "disk", diskPath)
			b.cleanupAllDeerSnapshots(ctx, vmName)
			return diskPath, nil
		}

		// Stale overlay WITH backing file: blockpull to make standalone
		b.logger.Warn("VM disk is stale overlay, recovering via blockpull", "vm", vmName, "disk", diskPath)
		_, _ = b.runVirshCmd(ctx, "blockjob", vmName, "vda", "--abort")
		time.Sleep(time.Second)

		if err := b.blockpull(ctx, vmName); err != nil {
			return "", fmt.Errorf("recover stale overlay via blockpull: %w", err)
		}

		b.cleanupAllDeerSnapshots(ctx, vmName)
		diskPath, err = b.findDiskPath(ctx, vmName)
		if err != nil {
			return "", fmt.Errorf("find disk after blockpull: %w", err)
		}
		return diskPath, nil
	}

	b.cleanupAllDeerSnapshots(ctx, vmName)
	return diskPath, nil
}

// isDeerOverlay checks if a disk path looks like a deer snapshot overlay.
func isDeerOverlay(diskPath string) bool {
	return strings.Contains(filepath.Base(diskPath), deerSnapPrefix)
}

// hasBackingFile checks if a disk image has a backing file (is part of a chain).
func (b *LibvirtBackend) hasBackingFile(ctx context.Context, diskPath string) (bool, error) {
	out, err := b.runVirshCmd(ctx, "vol-dumpxml", diskPath)
	if err != nil {
		return false, err
	}
	return strings.Contains(out, "</backingStore>"), nil
}

// blockpull pulls backing data into the active overlay, making it standalone.
func (b *LibvirtBackend) blockpull(ctx context.Context, vmName string) error {
	_, err := b.runVirshCmd(ctx, "blockpull", vmName, "vda", "--wait")
	return err
}

// cleanupAllDeerSnapshots removes all deer snapshot metadata from a VM.
func (b *LibvirtBackend) cleanupAllDeerSnapshots(ctx context.Context, vmName string) {
	out, err := b.runVirshCmd(ctx, "snapshot-list", vmName, "--name")
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		name := strings.TrimSpace(line)
		if strings.HasPrefix(name, deerSnapPrefix) {
			b.logger.Warn("cleaning up stale snapshot metadata", "vm", vmName, "snapshot", name)
			_ = b.deleteSnapshotMetadata(ctx, vmName, name)
		}
	}
}

// refreshStoragePools refreshes all active storage pools so libvirt discovers
// files created outside pool management (e.g. overlay files from snapshots).
func (b *LibvirtBackend) refreshStoragePools(ctx context.Context) {
	out, err := b.runVirshCmd(ctx, "pool-list", "--name")
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		pool := strings.TrimSpace(line)
		if pool != "" {
			_, _ = b.runVirshCmd(ctx, "pool-refresh", pool)
		}
	}
}

// deleteSnapshotMetadata removes snapshot metadata from libvirt.
func (b *LibvirtBackend) deleteSnapshotMetadata(ctx context.Context, vmName, snapName string) error {
	_, err := b.runVirshCmd(ctx, "snapshot-delete", vmName, snapName, "--metadata")
	return err
}
