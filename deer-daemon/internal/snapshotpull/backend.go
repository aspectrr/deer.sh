// Package snapshotpull provides snapshot-and-pull functionality for creating
// sandboxes from remote production VMs.
package snapshotpull

import "context"

// SnapshotBackend abstracts the mechanism for pulling a VM disk from a remote host.
type SnapshotBackend interface {
	// SnapshotAndPull transfers vmName's disk to destPath.
	SnapshotAndPull(ctx context.Context, vmName string, destPath string) error
}
