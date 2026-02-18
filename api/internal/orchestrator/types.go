package orchestrator

import "time"

// CreateSandboxRequest is the request for creating a sandbox.
type CreateSandboxRequest struct {
	OrgID        string `json:"org_id"`
	AgentID      string `json:"agent_id"`
	SourceVM     string `json:"source_vm"`
	Name         string `json:"name"`
	VCPUs        int    `json:"vcpus,omitempty"`
	MemoryMB     int    `json:"memory_mb,omitempty"`
	TTLSeconds   int    `json:"ttl_seconds,omitempty"`
	Network      string `json:"network,omitempty"`
	SourceHostID string `json:"source_host_id,omitempty"`
	Live         bool   `json:"live,omitempty"`
}

// DiscoveredHost is a host discovered from SSH config parsing + probing.
type DiscoveredHost struct {
	Name         string   `json:"name"`
	Hostname     string   `json:"hostname"`
	User         string   `json:"user"`
	Port         int      `json:"port"`
	IdentityFile string   `json:"identity_file"`
	Reachable    bool     `json:"reachable"`
	HasLibvirt   bool     `json:"has_libvirt"`
	HasProxmox   bool     `json:"has_proxmox"`
	VMs          []string `json:"vms,omitempty"`
	Error        string   `json:"error,omitempty"`
}

// RunCommandRequest is the request for running a command in a sandbox.
type RunCommandRequest struct {
	Command    string            `json:"command"`
	TimeoutSec int               `json:"timeout_seconds,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
}

// SnapshotRequest is the request for creating a snapshot.
type SnapshotRequest struct {
	Name string `json:"name"`
}

// SnapshotResponse is returned after creating a snapshot.
type SnapshotResponse struct {
	SnapshotID   string    `json:"snapshot_id"`
	SandboxID    string    `json:"sandbox_id"`
	SnapshotName string    `json:"snapshot_name"`
	CreatedAt    time.Time `json:"created_at"`
}

// HostInfo is the REST representation of a connected host.
type HostInfo struct {
	HostID          string   `json:"host_id"`
	Hostname        string   `json:"hostname"`
	Status          string   `json:"status"`
	ActiveSandboxes int      `json:"active_sandboxes"`
	AvailableCPUs   int32    `json:"available_cpus"`
	AvailableMemMB  int64    `json:"available_memory_mb"`
	AvailableDiskMB int64    `json:"available_disk_mb"`
	BaseImages      []string `json:"base_images"`
	LastHeartbeat   string   `json:"last_heartbeat"`
}

// VMInfo is the REST representation of a source VM.
type VMInfo struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	IPAddress string `json:"ip_address,omitempty"`
	Prepared  bool   `json:"prepared"`
	HostID    string `json:"host_id,omitempty"`
}

// PrepareRequest is the request for preparing a source VM.
type PrepareRequest struct {
	SSHUser    string `json:"ssh_user"`
	SSHKeyPath string `json:"ssh_key_path"`
}

// RunSourceRequest is the request for running a command on a source VM.
type RunSourceRequest struct {
	Command    string `json:"command"`
	TimeoutSec int    `json:"timeout_seconds,omitempty"`
}

// ReadSourceRequest is the request for reading a file from a source VM.
type ReadSourceRequest struct {
	Path string `json:"path"`
}

// SourceCommandResult is the response for a source VM command.
type SourceCommandResult struct {
	SourceVM string `json:"source_vm"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// SourceFileResult is the response for reading a source VM file.
type SourceFileResult struct {
	SourceVM string `json:"source_vm"`
	Path     string `json:"path"`
	Content  string `json:"content"`
}
