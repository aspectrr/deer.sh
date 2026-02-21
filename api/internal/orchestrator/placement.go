package orchestrator

import (
	"fmt"
	"time"

	"github.com/aspectrr/fluid.sh/api/internal/registry"
)

// SelectHost picks the best connected host for a sandbox that needs the given
// base image. Filters by image availability, resources, and health.
func SelectHost(reg *registry.Registry, baseImage, orgID string, heartbeatTimeout time.Duration, requiredCPUs int32, requiredMemoryMB int32) (registry.ConnectedHost, error) {
	hosts := reg.ListConnectedByOrg(orgID)
	if len(hosts) == 0 {
		return registry.ConnectedHost{}, fmt.Errorf("no connected hosts")
	}

	now := time.Now()
	var best *registry.ConnectedHost

	for i := range hosts {
		h := &hosts[i]
		if h.Registration == nil {
			continue
		}

		if !hostHasImage(*h, baseImage) {
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

		if best == nil || hostScore(*h) > hostScore(*best) {
			best = h
		}
	}

	if best == nil {
		return registry.ConnectedHost{}, fmt.Errorf("no healthy host with image %q and sufficient resources", baseImage)
	}

	return *best, nil
}

// SelectHostForSourceVM picks a connected host that has the given source VM.
// When requiredCPUs or requiredMemoryMB are non-zero, hosts without sufficient
// resources are skipped (used as CreateSandbox fallback).
func SelectHostForSourceVM(reg *registry.Registry, vmName, orgID string, heartbeatTimeout time.Duration, requiredCPUs int32, requiredMemoryMB int32) (registry.ConnectedHost, error) {
	hosts := reg.ListConnectedByOrg(orgID)
	if len(hosts) == 0 {
		return registry.ConnectedHost{}, fmt.Errorf("no connected hosts")
	}

	now := time.Now()
	var best *registry.ConnectedHost

	for i := range hosts {
		h := &hosts[i]
		if h.Registration == nil {
			continue
		}

		if now.Sub(h.LastHeartbeat) > heartbeatTimeout {
			continue
		}

		if requiredCPUs > 0 && h.Registration.GetAvailableCpus() < requiredCPUs {
			continue
		}
		if requiredMemoryMB > 0 && h.Registration.GetAvailableMemoryMb() < int64(requiredMemoryMB) {
			continue
		}

		hasVM := false
		for _, vm := range h.Registration.GetSourceVms() {
			if vm.GetName() == vmName {
				hasVM = true
				break
			}
		}
		if !hasVM {
			continue
		}

		if best == nil || hostScore(*h) > hostScore(*best) {
			best = h
		}
	}

	if best == nil {
		return registry.ConnectedHost{}, fmt.Errorf("no connected host has source VM %q", vmName)
	}
	return *best, nil
}

// hostScore computes a placement score considering both memory and CPU.
func hostScore(h registry.ConnectedHost) float64 {
	return float64(h.Registration.GetAvailableMemoryMb()) + float64(h.Registration.GetAvailableCpus())*1024
}

func hostHasImage(h registry.ConnectedHost, baseImage string) bool {
	for _, img := range h.Registration.GetBaseImages() {
		if img == baseImage {
			return true
		}
	}
	return false
}
