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

	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/shellutil"
)

const fluidSnapPrefix = "fluid-tmp-snap"

// LibvirtBackend snapshots and pulls a VM disk from a remote libvirt host via SSH.
type LibvirtBackend struct {
	sshHost         string
	sshPort         int
	sshUser         string
	sshIdentityFile string
	virshURI        string
	logger          *slog.Logger
}

// NewLibvirtBackend creates a backend that uses SSH + virsh to snapshot and rsync to pull.
func NewLibvirtBackend(host string, port int, user, identityFile, virshURI string, logger *slog.Logger) *LibvirtBackend {
	if port == 0 {
		port = 22
	}
	if user == "" {
		user = "root"
	}
	if virshURI == "" {
		virshURI = "qemu:///system"
	}
	if logger == nil {
		logger = slog.Default()
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

// virshCmd builds a virsh command string with the connection URI.
func (b *LibvirtBackend) virshCmd(args string) string {
	return fmt.Sprintf("virsh -c %s %s", shellutil.Quote(b.virshURI), args)
}

// SnapshotAndPull creates a temporary external snapshot, rsyncs the original
// (now read-only) disk to destPath, then blockcommits back and removes the snapshot.
func (b *LibvirtBackend) SnapshotAndPull(ctx context.Context, vmName string, destPath string) error {
	b.logger.Info("starting snapshot-and-pull", "vm", vmName, "dest", destPath)

	// 1. Clean up any stale snapshot from a previous failed blockcommit,
	//    then find the (clean) disk path.
	diskPath, err := b.findCleanDiskPath(ctx, vmName)
	if err != nil {
		return fmt.Errorf("find disk path: %w", err)
	}
	b.logger.Info("found disk path", "vm", vmName, "disk", diskPath)

	// 2. Create external snapshot with unique name (makes original disk read-only)
	snapName := fmt.Sprintf("%s-%d", fluidSnapPrefix, time.Now().Unix())
	if err := b.createSnapshot(ctx, vmName, snapName); err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}

	// 3. Always clean up: blockcommit + delete all fluid snapshot metadata
	defer func() {
		if err := b.blockcommitWithRetry(ctx, vmName); err != nil {
			b.logger.Error("blockcommit failed after retries", "vm", vmName, "error", err)
		}
		b.cleanupAllFluidSnapshots(ctx, vmName)
	}()

	// 4. Rsync the now read-only original disk to local destPath
	if err := b.rsyncDisk(ctx, diskPath, destPath); err != nil {
		return fmt.Errorf("rsync disk: %w", err)
	}

	b.logger.Info("snapshot-and-pull complete", "vm", vmName, "dest", destPath)
	return nil
}

// findDiskPath uses virsh domblklist to find the primary disk path.
func (b *LibvirtBackend) findDiskPath(ctx context.Context, vmName string) (string, error) {
	out, err := b.sshCommand(ctx, b.virshCmd(fmt.Sprintf("domblklist %s --details", shellutil.Quote(vmName))))
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

// createSnapshot creates an external disk-only snapshot.
func (b *LibvirtBackend) createSnapshot(ctx context.Context, vmName, snapName string) error {
	_, err := b.sshCommand(ctx, b.virshCmd(fmt.Sprintf(
		"snapshot-create-as %s %s --disk-only --atomic",
		shellutil.Quote(vmName), shellutil.Quote(snapName),
	)))
	return err
}

// waitForBlockJob polls virsh blockjob until no job is active on the VM's vda disk.
// Handles RUNNING (waits), READY (pivots then re-checks), and no-job (returns).
func (b *LibvirtBackend) waitForBlockJob(ctx context.Context, vmName string) error {
	const pollInterval = 2 * time.Second
	for {
		out, err := b.sshCommand(ctx, b.virshCmd(fmt.Sprintf("blockjob %s vda --info", shellutil.Quote(vmName))))
		if err != nil || strings.TrimSpace(out) == "" {
			// No active block job (command errors or empty = no job)
			return nil
		}
		out = strings.TrimSpace(out)

		// "No current block job for vda" means no active job
		if strings.Contains(strings.ToLower(out), "no current block job") {
			return nil
		}

		if strings.Contains(strings.ToLower(out), "ready") {
			// Job completed merge phase, needs pivot
			b.logger.Info("block job ready, pivoting", "vm", vmName)
			_, _ = b.sshCommand(ctx, b.virshCmd(fmt.Sprintf("blockjob %s vda --pivot", shellutil.Quote(vmName))))
			continue // re-check after pivot
		}

		// Job still running, wait and re-poll
		b.logger.Info("waiting for block job to complete", "vm", vmName, "status", out)
		select {
		case <-time.After(pollInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// blockcommit merges the snapshot back into the original disk and pivots.
func (b *LibvirtBackend) blockcommit(ctx context.Context, vmName string) error {
	// Wait for any in-flight block job to finish before starting a new one
	if err := b.waitForBlockJob(ctx, vmName); err != nil {
		return fmt.Errorf("wait for block job: %w", err)
	}
	_, err := b.sshCommand(ctx, b.virshCmd(fmt.Sprintf(
		"blockcommit %s vda --active --pivot --delete",
		shellutil.Quote(vmName),
	)))
	return err
}

// blockcommitWithRetry retries blockcommit with exponential backoff.
// QEMU's internal write lock is transient and typically clears within 1-3s.
func (b *LibvirtBackend) blockcommitWithRetry(ctx context.Context, vmName string) error {
	const maxAttempts = 5
	for i := 0; i < maxAttempts; i++ {
		err := b.blockcommit(ctx, vmName)
		if err == nil {
			return nil
		}
		if i < maxAttempts-1 {
			delay := time.Duration(1<<uint(i)) * time.Second
			b.logger.Warn("blockcommit attempt failed, retrying", "vm", vmName, "attempt", i+1, "delay", delay, "error", err)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		} else {
			return err
		}
	}
	return nil // unreachable
}

// findCleanDiskPath returns the real disk path for a VM, cleaning up any stale
// snapshot state first. Handles four scenarios:
//  1. No stale state - returns disk path directly.
//  2. VM pointing at overlay with backing file (failed blockcommit) - try blockcommit,
//     fall back to blockpull if write lock persists, clean metadata, then re-query.
//  3. VM pointing at overlay-named disk WITHOUT backing file (post-blockpull) - the
//     overlay IS the disk now, just clean metadata and return it.
//  4. VM pointing at real disk but orphaned overlay files exist - remove them.
func (b *LibvirtBackend) findCleanDiskPath(ctx context.Context, vmName string) (string, error) {
	diskPath, err := b.findDiskPath(ctx, vmName)
	if err != nil {
		return "", err
	}

	// Check if the current disk path looks like a fluid overlay
	if isFluidOverlay(diskPath) {
		hasBacking, _ := b.hasBackingFile(ctx, diskPath)
		if !hasBacking {
			// Post-blockpull: overlay is standalone, just clean metadata
			b.logger.Info("disk has overlay name but no backing file, using as-is", "vm", vmName, "disk", diskPath)
			b.cleanupAllFluidSnapshots(ctx, vmName)
			return diskPath, nil
		}

		b.logger.Warn("VM disk is stale overlay, recovering", "vm", vmName, "disk", diskPath)

		// Abort any stuck block jobs before attempting blockcommit
		_, _ = b.sshCommand(ctx, b.virshCmd(fmt.Sprintf("blockjob %s vda --abort", shellutil.Quote(vmName))))
		time.Sleep(time.Second)

		// Try blockcommit first
		if err := b.blockcommitWithRetry(ctx, vmName); err != nil {
			b.logger.Warn("blockcommit failed, falling back to blockpull", "vm", vmName, "error", err)
			if pullErr := b.blockpull(ctx, vmName); pullErr != nil {
				return "", fmt.Errorf("recover stale overlay: blockcommit: %w, blockpull: %w", err, pullErr)
			}
			// After blockpull, overlay is standalone. Clean metadata and return overlay path.
			b.cleanupAllFluidSnapshots(ctx, vmName)
			diskPath, err = b.findDiskPath(ctx, vmName)
			if err != nil {
				return "", fmt.Errorf("find disk after blockpull: %w", err)
			}
			return diskPath, nil
		}

		b.cleanupAllFluidSnapshots(ctx, vmName)

		// Remove the orphaned overlay file
		b.logger.Warn("removing stale overlay file", "vm", vmName, "path", diskPath)
		_, _ = b.sshCommand(ctx, fmt.Sprintf("rm -f %s", shellutil.Quote(diskPath)))

		// Re-query to get the real disk path after pivot
		diskPath, err = b.findDiskPath(ctx, vmName)
		if err != nil {
			return "", fmt.Errorf("find disk path after recovery: %w", err)
		}

		// Sanity check - should no longer be an overlay
		if isFluidOverlay(diskPath) {
			return "", fmt.Errorf("disk still points at overlay after blockcommit: %s", diskPath)
		}
	}

	// Clean up any lingering snapshot metadata
	b.cleanupAllFluidSnapshots(ctx, vmName)

	// Clean up orphaned overlay files on disk (metadata gone but files remain)
	b.cleanupOrphanOverlayFiles(ctx, vmName, diskPath)

	return diskPath, nil
}

// isFluidOverlay checks if a disk path looks like a fluid snapshot overlay.
func isFluidOverlay(diskPath string) bool {
	return strings.Contains(filepath.Base(diskPath), fluidSnapPrefix)
}

// hasBackingFile checks if a disk image has a backing file (is part of a chain).
func (b *LibvirtBackend) hasBackingFile(ctx context.Context, diskPath string) (bool, error) {
	out, err := b.sshCommand(ctx, fmt.Sprintf("qemu-img info %s", shellutil.Quote(diskPath)))
	if err != nil {
		return false, err
	}
	return strings.Contains(out, "backing file:"), nil
}

// blockpull pulls backing data into the active overlay, making it standalone.
func (b *LibvirtBackend) blockpull(ctx context.Context, vmName string) error {
	_, err := b.sshCommand(ctx, b.virshCmd(fmt.Sprintf("blockpull %s vda --wait", shellutil.Quote(vmName))))
	return err
}

// cleanupAllFluidSnapshots removes all fluid snapshot metadata from a VM.
func (b *LibvirtBackend) cleanupAllFluidSnapshots(ctx context.Context, vmName string) {
	out, err := b.sshCommand(ctx, b.virshCmd(fmt.Sprintf("snapshot-list %s --name", shellutil.Quote(vmName))))
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		name := strings.TrimSpace(line)
		if strings.HasPrefix(name, fluidSnapPrefix) {
			b.logger.Warn("cleaning up stale snapshot metadata", "vm", vmName, "snapshot", name)
			_ = b.deleteSnapshotMetadata(ctx, vmName, name)
		}
	}
}

// cleanupOrphanOverlayFiles removes any fluid overlay files that are no longer in use.
func (b *LibvirtBackend) cleanupOrphanOverlayFiles(ctx context.Context, vmName, currentDiskPath string) {
	dir := currentDiskPath[:strings.LastIndex(currentDiskPath, "/")+1]
	base := currentDiskPath[strings.LastIndex(currentDiskPath, "/")+1:]
	if dot := strings.LastIndex(base, "."); dot >= 0 {
		base = base[:dot]
	}
	pattern := dir + base + "." + fluidSnapPrefix + "*"
	// Pattern is derived from virsh domblklist output (trusted), needs glob expansion so not quoted.
	out, err := b.sshCommand(ctx, fmt.Sprintf("ls %s 2>/dev/null", pattern))
	if err != nil || strings.TrimSpace(out) == "" {
		return
	}
	for _, f := range strings.Split(strings.TrimSpace(out), "\n") {
		f = strings.TrimSpace(f)
		if f != "" && f != currentDiskPath {
			b.logger.Warn("removing orphaned overlay file", "vm", vmName, "path", f)
			_, _ = b.sshCommand(ctx, fmt.Sprintf("rm -f %s", shellutil.Quote(f)))
		}
	}
}

// deleteSnapshotMetadata removes snapshot metadata from libvirt.
func (b *LibvirtBackend) deleteSnapshotMetadata(ctx context.Context, vmName, snapName string) error {
	_, err := b.sshCommand(ctx, b.virshCmd(fmt.Sprintf(
		"snapshot-delete %s %s --metadata",
		shellutil.Quote(vmName), shellutil.Quote(snapName),
	)))
	return err
}

// rsyncDisk pulls the remote disk to a local path.
func (b *LibvirtBackend) rsyncDisk(ctx context.Context, remotePath, localPath string) error {
	sshOpts := b.sshOpts()
	src := fmt.Sprintf("%s@%s:%s", b.sshUser, b.sshHost, remotePath)

	args := []string{
		"-avz", "--progress",
		"-e", fmt.Sprintf("ssh %s", sshOpts),
		src, localPath,
	}

	cmd := exec.CommandContext(ctx, "rsync", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rsync: %w: %s", err, stderr.String())
	}
	return nil
}

// sshCommand runs a command on the remote host via SSH.
func (b *LibvirtBackend) sshCommand(ctx context.Context, command string) (string, error) {
	args := []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "BatchMode=yes",
		"-p", fmt.Sprintf("%d", b.sshPort),
	}
	if b.sshIdentityFile != "" {
		args = append(args, "-i", b.sshIdentityFile)
	}
	args = append(args, fmt.Sprintf("%s@%s", b.sshUser, b.sshHost), command)

	cmd := exec.CommandContext(ctx, "ssh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh command %q: %w: %s", command, err, stderr.String())
	}
	return stdout.String(), nil
}

// sshOpts returns the SSH option string for use with rsync -e.
func (b *LibvirtBackend) sshOpts() string {
	opts := fmt.Sprintf("-o StrictHostKeyChecking=accept-new -o BatchMode=yes -p %d", b.sshPort)
	if b.sshIdentityFile != "" {
		opts += fmt.Sprintf(" -i %s", shellutil.Quote(b.sshIdentityFile))
	}
	return opts
}
