package orchestrator

import (
	"fmt"
	"time"

	"github.com/aspectrr/fluid.sh/api/internal/registry"
)

// SelectHost picks the best connected host for a sandbox that needs the given
// base image. Filters by image availability, resources, and health.
func SelectHost(reg *registry.Registry, baseImage, orgID string, heartbeatTimeout time.Duration, requiredCPUs int32, requiredMemoryMB int32) (*registry.ConnectedHost, error) {
	hosts := reg.ListConnectedByOrg(orgID)
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no connected hosts")
	}

	now := time.Now()
	var best *registry.ConnectedHost

	for _, h := range hosts {
		if h.Registration == nil {
			continue
		}

		if !hostHasImage(h, baseImage) {
			continue
		}

		if h.Registration.GetAvailableCpus() < int32(requiredCPUs) {
			continue
		}
		if h.Registration.GetAvailableMemoryMb() < int64(requiredMemoryMB) {
			continue
		}

		if now.Sub(h.LastHeartbeat) > heartbeatTimeout {
			continue
		}

		if best == nil || h.Registration.GetAvailableMemoryMb() > best.Registration.GetAvailableMemoryMb() {
			best = h
		}
	}

	if best == nil {
		return nil, fmt.Errorf("no healthy host with image %q and sufficient resources", baseImage)
	}

	return best, nil
}

// SelectHostForSourceVM picks a connected host that has the given source VM.
func SelectHostForSourceVM(reg *registry.Registry, vmName, orgID string, heartbeatTimeout time.Duration) (*registry.ConnectedHost, error) {
	hosts := reg.ListConnectedByOrg(orgID)
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no connected hosts")
	}

	now := time.Now()

	for _, h := range hosts {
		if h.Registration == nil {
			continue
		}

		if now.Sub(h.LastHeartbeat) > heartbeatTimeout {
			continue
		}

		for _, vm := range h.Registration.GetSourceVms() {
			if vm.GetName() == vmName {
				return h, nil
			}
		}
	}

	return nil, fmt.Errorf("no connected host has source VM %q", vmName)
}

func hostHasImage(h *registry.ConnectedHost, baseImage string) bool {
	for _, img := range h.Registration.GetBaseImages() {
		if img == baseImage {
			return true
		}
	}
	return false
}
