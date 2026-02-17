package lxc

import (
	"context"
	"fmt"
	"sync"
)

// CTResolver resolves container names to VMIDs and vice versa.
type CTResolver struct {
	client *Client
	mu     sync.RWMutex
	byName map[string]int
	byID   map[int]string
}

// NewCTResolver creates a new CTResolver backed by the given client.
func NewCTResolver(client *Client) *CTResolver {
	return &CTResolver{
		client: client,
		byName: make(map[string]int),
		byID:   make(map[int]string),
	}
}

// Refresh reloads the container list from Proxmox and rebuilds the cache.
func (r *CTResolver) Refresh(ctx context.Context) error {
	cts, err := r.client.ListCTs(ctx)
	if err != nil {
		return fmt.Errorf("refresh CT list: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.byName = make(map[string]int, len(cts))
	r.byID = make(map[int]string, len(cts))
	for _, ct := range cts {
		r.byName[ct.Name] = ct.VMID
		r.byID[ct.VMID] = ct.Name
	}
	return nil
}

// ResolveVMID returns the VMID for a given container name.
// If the name is not in the cache, it refreshes first.
func (r *CTResolver) ResolveVMID(ctx context.Context, name string) (int, error) {
	r.mu.RLock()
	vmid, ok := r.byName[name]
	r.mu.RUnlock()
	if ok {
		return vmid, nil
	}

	if err := r.Refresh(ctx); err != nil {
		return 0, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	vmid, ok = r.byName[name]
	if !ok {
		return 0, fmt.Errorf("container %q not found", name)
	}
	return vmid, nil
}

// ResolveName returns the name for a given VMID.
func (r *CTResolver) ResolveName(ctx context.Context, vmid int) (string, error) {
	r.mu.RLock()
	name, ok := r.byID[vmid]
	r.mu.RUnlock()
	if ok {
		return name, nil
	}

	if err := r.Refresh(ctx); err != nil {
		return "", err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	name, ok = r.byID[vmid]
	if !ok {
		return "", fmt.Errorf("VMID %d not found", vmid)
	}
	return name, nil
}
