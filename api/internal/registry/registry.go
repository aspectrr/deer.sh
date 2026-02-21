package registry

import (
	"fmt"
	"sync"
	"time"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"
)

// HostStream is the interface for sending control messages to a connected host.
type HostStream interface {
	Send(msg *fluidv1.ControlMessage) error
}

// ConnectedHost represents a sandbox host that is actively connected via gRPC.
type ConnectedHost struct {
	HostID          string
	OrgID           string
	Hostname        string
	Stream          HostStream
	LastHeartbeat   time.Time
	Registration    *fluidv1.HostRegistration
	ActiveSandboxes int32
	SourceVMCount   int32
}

// Registry tracks all currently connected sandbox hosts in memory.
type Registry struct {
	mu    sync.RWMutex
	hosts map[string]*ConnectedHost
}

// New creates an empty host registry.
func New() *Registry {
	return &Registry{
		hosts: make(map[string]*ConnectedHost),
	}
}

// Register adds or replaces a connected host in the registry.
func (r *Registry) Register(hostID, orgID, hostname string, stream HostStream) error {
	if hostID == "" {
		return fmt.Errorf("host ID must not be empty")
	}
	if stream == nil {
		return fmt.Errorf("stream must not be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.hosts[hostID] = &ConnectedHost{
		HostID:        hostID,
		OrgID:         orgID,
		Hostname:      hostname,
		Stream:        stream,
		LastHeartbeat: time.Now(),
	}
	return nil
}

// Unregister removes a host from the registry.
func (r *Registry) Unregister(hostID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.hosts, hostID)
}

// GetHost returns a value copy of the connected host for the given ID, if present.
func (r *Registry) GetHost(hostID string) (ConnectedHost, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.hosts[hostID]
	if !ok {
		return ConnectedHost{}, false
	}
	return *h, ok
}

// ListConnected returns value copies of all currently connected hosts.
func (r *Registry) ListConnected() []ConnectedHost {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ConnectedHost, 0, len(r.hosts))
	for _, h := range r.hosts {
		result = append(result, *h)
	}
	return result
}

// ListConnectedByOrg returns value copies of connected hosts belonging to the given org.
func (r *Registry) ListConnectedByOrg(orgID string) []ConnectedHost {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []ConnectedHost
	for _, h := range r.hosts {
		if h.OrgID == orgID {
			result = append(result, *h)
		}
	}
	return result
}

// SetRegistration updates the registration info and heartbeat for a host.
func (r *Registry) SetRegistration(hostID string, reg *fluidv1.HostRegistration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if h, ok := r.hosts[hostID]; ok {
		h.Registration = reg
		h.LastHeartbeat = time.Now()
	}
}

// UpdateHeartbeat records the latest heartbeat time for a connected host.
func (r *Registry) UpdateHeartbeat(hostID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if h, ok := r.hosts[hostID]; ok {
		h.LastHeartbeat = time.Now()
	}
}

// UpdateResources updates the in-memory available CPU and memory for a host.
func (r *Registry) UpdateResources(hostID string, cpus int32, memMB int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.hosts[hostID]
	if !ok {
		return
	}
	if h.Registration != nil {
		h.Registration.AvailableCpus = cpus
		h.Registration.AvailableMemoryMb = memMB
	}
}

// UpdateHeartbeatCounts updates the per-host sandbox and source VM counts from a heartbeat.
func (r *Registry) UpdateHeartbeatCounts(hostID string, activeSandboxes, sourceVMCount int32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if h, ok := r.hosts[hostID]; ok {
		h.LastHeartbeat = time.Now()
		h.ActiveSandboxes = activeSandboxes
		h.SourceVMCount = sourceVMCount
	}
}

// OrgResourceCounts returns aggregated resource counts for an org across all connected hosts.
func (r *Registry) OrgResourceCounts(orgID string) (sandboxes, sourceVMs, daemons int) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, h := range r.hosts {
		if h.OrgID == orgID {
			sandboxes += int(h.ActiveSandboxes)
			sourceVMs += int(h.SourceVMCount)
			daemons++
		}
	}
	return
}
