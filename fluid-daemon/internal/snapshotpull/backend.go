// Package snapshotpull provides snapshot-and-pull functionality for creating
// sandboxes from remote production VMs.
package snapshotpull

import "context"

// SnapshotBackend abstracts the mechanism for snapshotting a VM disk
// on a remote host and pulling it locally.
type SnapshotBackend interface {
	// SnapshotAndPull creates a temporary snapshot of vmName's disk,
	// transfers the backing image to destPath, then cleans up the snapshot.
	SnapshotAndPull(ctx context.Context, vmName string, destPath string) error
}
