package model

// CloneResult represents the outcome of cloning a VM to a container.
type CloneResult struct {
	// VM is the name of the source VM that was cloned.
	VM string `json:"vm"`

	// ContainerID is the stable identifier of the created container.
	ContainerID string `json:"container_id"`

	// Image is the full image tag (e.g., "vmclone/node-c:20251215T183000Z").
	Image string `json:"image"`

	// Mode indicates the extraction method used: "snapshot" for running VMs,
	// "offline" for stopped VMs.
	Mode string `json:"mode"`

	// Status indicates the final state: "ready" on success.
	Status string `json:"status"`
}

// Extraction modes
const (
	ModeSnapshot = "snapshot"
	ModeOffline  = "offline"
)

// Result statuses
const (
	StatusReady = "ready"
)
