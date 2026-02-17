package snapshotpull

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

// LibvirtBackend snapshots and pulls a VM disk from a remote libvirt host via SSH.
type LibvirtBackend struct {
	sshHost         string
	sshPort         int
	sshUser         string
	sshIdentityFile string
	logger          *slog.Logger
}

// NewLibvirtBackend creates a backend that uses SSH + virsh to snapshot and rsync to pull.
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
	return &LibvirtBackend{
		sshHost:         host,
		sshPort:         port,
		sshUser:         user,
		sshIdentityFile: identityFile,
		logger:          logger.With("component", "libvirt-backend"),
	}
}

// SnapshotAndPull creates a temporary external snapshot, rsyncs the original
// (now read-only) disk to destPath, then blockcommits back and removes the snapshot.
func (b *LibvirtBackend) SnapshotAndPull(ctx context.Context, vmName string, destPath string) error {
	b.logger.Info("starting snapshot-and-pull", "vm", vmName, "dest", destPath)

	// 1. Find the disk path
	diskPath, err := b.findDiskPath(ctx, vmName)
	if err != nil {
		return fmt.Errorf("find disk path: %w", err)
	}
	b.logger.Info("found disk path", "vm", vmName, "disk", diskPath)

	// 2. Create external snapshot (makes original disk read-only)
	snapName := "fluid-tmp-snap"
	if err := b.createSnapshot(ctx, vmName, snapName); err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}

	// 3. Always clean up: blockcommit + delete snapshot metadata
	defer func() {
		if err := b.blockcommit(ctx, vmName); err != nil {
			b.logger.Error("blockcommit failed", "vm", vmName, "error", err)
		}
		if err := b.deleteSnapshotMetadata(ctx, vmName, snapName); err != nil {
			b.logger.Warn("delete snapshot metadata failed", "vm", vmName, "error", err)
		}
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
	out, err := b.sshCommand(ctx, fmt.Sprintf("virsh domblklist %s --details", vmName))
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
	_, err := b.sshCommand(ctx, fmt.Sprintf(
		"virsh snapshot-create-as %s %s --disk-only --atomic",
		vmName, snapName,
	))
	return err
}

// blockcommit merges the snapshot back into the original disk and pivots.
func (b *LibvirtBackend) blockcommit(ctx context.Context, vmName string) error {
	_, err := b.sshCommand(ctx, fmt.Sprintf(
		"virsh blockcommit %s vda --active --pivot --delete",
		vmName,
	))
	return err
}

// deleteSnapshotMetadata removes snapshot metadata from libvirt.
func (b *LibvirtBackend) deleteSnapshotMetadata(ctx context.Context, vmName, snapName string) error {
	_, err := b.sshCommand(ctx, fmt.Sprintf(
		"virsh snapshot-delete %s %s --metadata",
		vmName, snapName,
	))
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
		"-o", "StrictHostKeyChecking=no",
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
	opts := fmt.Sprintf("-o StrictHostKeyChecking=no -o BatchMode=yes -p %d", b.sshPort)
	if b.sshIdentityFile != "" {
		opts += fmt.Sprintf(" -i %s", b.sshIdentityFile)
	}
	return opts
}
