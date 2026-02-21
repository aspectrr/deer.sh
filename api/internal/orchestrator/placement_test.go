package orchestrator

import (
	"testing"
	"time"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"

	"github.com/aspectrr/fluid.sh/api/internal/registry"
)

// mockStream implements registry.HostStream for testing.
type mockStream struct{}

func (m *mockStream) Send(_ *fluidv1.ControlMessage) error { return nil }

func newRegistryWithHost(t *testing.T, hostID, orgID string, reg *fluidv1.HostRegistration) *registry.Registry {
	t.Helper()
	r := registry.New()
	if err := r.Register(hostID, orgID, hostID+"-hostname", &mockStream{}); err != nil {
		t.Fatalf("Register(%s): %v", hostID, err)
	}
	if reg != nil {
		r.SetRegistration(hostID, reg)
	}
	return r
}

func TestSelectHost_NoHosts(t *testing.T) {
	r := registry.New()
	_, err := SelectHost(r, "ubuntu-22.04", "org-1", 90*time.Second, 2, 2048)
	if err == nil {
		t.Fatal("SelectHost: expected error when no hosts are connected")
	}
}

func TestSelectHost_MatchingImage(t *testing.T) {
	r := registry.New()
	_ = r.Register("host-1", "org-1", "h1", &mockStream{})
	r.SetRegistration("host-1", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04", "debian-12"},
		AvailableCpus:     4,
		AvailableMemoryMb: 8192,
	})

	h, err := SelectHost(r, "ubuntu-22.04", "org-1", 90*time.Second, 2, 2048)
	if err != nil {
		t.Fatalf("SelectHost: unexpected error: %v", err)
	}
	if h.HostID != "host-1" {
		t.Errorf("HostID = %q, want %q", h.HostID, "host-1")
	}
}

func TestSelectHost_NoMatchingImage(t *testing.T) {
	r := registry.New()
	_ = r.Register("host-1", "org-1", "h1", &mockStream{})
	r.SetRegistration("host-1", &fluidv1.HostRegistration{
		BaseImages:        []string{"centos-9"},
		AvailableCpus:     4,
		AvailableMemoryMb: 8192,
	})

	_, err := SelectHost(r, "ubuntu-22.04", "org-1", 90*time.Second, 2, 2048)
	if err == nil {
		t.Fatal("SelectHost: expected error when no host has the requested image")
	}
}

func TestSelectHost_InsufficientCPU(t *testing.T) {
	r := registry.New()
	_ = r.Register("host-1", "org-1", "h1", &mockStream{})
	r.SetRegistration("host-1", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04"},
		AvailableCpus:     0, // No CPUs available.
		AvailableMemoryMb: 8192,
	})

	_, err := SelectHost(r, "ubuntu-22.04", "org-1", 90*time.Second, 2, 2048)
	if err == nil {
		t.Fatal("SelectHost: expected error when host has insufficient CPUs")
	}
}

func TestSelectHost_InsufficientMemory(t *testing.T) {
	r := registry.New()
	_ = r.Register("host-1", "org-1", "h1", &mockStream{})
	r.SetRegistration("host-1", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04"},
		AvailableCpus:     4,
		AvailableMemoryMb: 256, // Below required 2048.
	})

	_, err := SelectHost(r, "ubuntu-22.04", "org-1", 90*time.Second, 2, 2048)
	if err == nil {
		t.Fatal("SelectHost: expected error when host has insufficient memory")
	}
}

func TestSelectHost_StaleHeartbeat(t *testing.T) {
	r := registry.New()
	_ = r.Register("host-1", "org-1", "h1", &mockStream{})
	r.SetRegistration("host-1", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04"},
		AvailableCpus:     4,
		AvailableMemoryMb: 8192,
	})

	// Use a heartbeat timeout of 1ms and sleep to ensure staleness.
	time.Sleep(5 * time.Millisecond)

	_, err := SelectHost(r, "ubuntu-22.04", "org-1", time.Millisecond, 2, 2048)
	if err == nil {
		t.Fatal("SelectHost: expected error when host has stale heartbeat")
	}
}

func TestSelectHost_NilRegistration(t *testing.T) {
	r := registry.New()
	_ = r.Register("host-1", "org-1", "h1", &mockStream{})
	// No SetRegistration - Registration is nil.

	_, err := SelectHost(r, "ubuntu-22.04", "org-1", 90*time.Second, 2, 2048)
	if err == nil {
		t.Fatal("SelectHost: expected error when host has nil registration")
	}
}

func TestSelectHost_PicksHighestMemory(t *testing.T) {
	r := registry.New()
	_ = r.Register("host-1", "org-1", "h1", &mockStream{})
	r.SetRegistration("host-1", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04"},
		AvailableCpus:     4,
		AvailableMemoryMb: 4096,
	})
	_ = r.Register("host-2", "org-1", "h2", &mockStream{})
	r.SetRegistration("host-2", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04"},
		AvailableCpus:     4,
		AvailableMemoryMb: 16384,
	})

	h, err := SelectHost(r, "ubuntu-22.04", "org-1", 90*time.Second, 2, 2048)
	if err != nil {
		t.Fatalf("SelectHost: unexpected error: %v", err)
	}
	if h.HostID != "host-2" {
		t.Errorf("HostID = %q, want %q (host with more memory)", h.HostID, "host-2")
	}
}

func TestSelectHost_FiltersByOrg(t *testing.T) {
	r := registry.New()
	_ = r.Register("host-1", "org-1", "h1", &mockStream{})
	r.SetRegistration("host-1", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04"},
		AvailableCpus:     4,
		AvailableMemoryMb: 8192,
	})

	_, err := SelectHost(r, "ubuntu-22.04", "org-other", 90*time.Second, 2, 2048)
	if err == nil {
		t.Fatal("SelectHost: expected error when no hosts belong to the org")
	}
}

func TestSelectHostForSourceVM_Success(t *testing.T) {
	r := registry.New()
	_ = r.Register("host-1", "org-1", "h1", &mockStream{})
	r.SetRegistration("host-1", &fluidv1.HostRegistration{
		SourceVms: []*fluidv1.SourceVMInfo{
			{Name: "web-server", State: "running"},
			{Name: "db-server", State: "stopped"},
		},
		AvailableCpus:     4,
		AvailableMemoryMb: 8192,
	})

	h, err := SelectHostForSourceVM(r, "web-server", "org-1", 90*time.Second, 0, 0)
	if err != nil {
		t.Fatalf("SelectHostForSourceVM: unexpected error: %v", err)
	}
	if h.HostID != "host-1" {
		t.Errorf("HostID = %q, want %q", h.HostID, "host-1")
	}
}

func TestSelectHostForSourceVM_NoMatch(t *testing.T) {
	r := registry.New()
	_ = r.Register("host-1", "org-1", "h1", &mockStream{})
	r.SetRegistration("host-1", &fluidv1.HostRegistration{
		SourceVms: []*fluidv1.SourceVMInfo{
			{Name: "web-server"},
		},
	})

	_, err := SelectHostForSourceVM(r, "nonexistent-vm", "org-1", 90*time.Second, 0, 0)
	if err == nil {
		t.Fatal("SelectHostForSourceVM: expected error when no host has the source VM")
	}
}

func TestSelectHostForSourceVM_NoHosts(t *testing.T) {
	r := registry.New()
	_, err := SelectHostForSourceVM(r, "web-server", "org-1", 90*time.Second, 0, 0)
	if err == nil {
		t.Fatal("SelectHostForSourceVM: expected error when no hosts are connected")
	}
}

func TestSelectHostForSourceVM_StaleHeartbeat(t *testing.T) {
	r := registry.New()
	_ = r.Register("host-1", "org-1", "h1", &mockStream{})
	r.SetRegistration("host-1", &fluidv1.HostRegistration{
		SourceVms: []*fluidv1.SourceVMInfo{
			{Name: "web-server"},
		},
	})

	// Use a heartbeat timeout of 1ms and sleep to ensure staleness.
	time.Sleep(5 * time.Millisecond)

	_, err := SelectHostForSourceVM(r, "web-server", "org-1", time.Millisecond, 0, 0)
	if err == nil {
		t.Fatal("SelectHostForSourceVM: expected error when host heartbeat is stale")
	}
}

func TestSelectHostForSourceVM_NilRegistration(t *testing.T) {
	r := registry.New()
	_ = r.Register("host-1", "org-1", "h1", &mockStream{})
	// No SetRegistration.

	_, err := SelectHostForSourceVM(r, "web-server", "org-1", 90*time.Second, 0, 0)
	if err == nil {
		t.Fatal("SelectHostForSourceVM: expected error when host has nil registration")
	}
}

func TestSelectHostForSourceVM_FiltersByOrg(t *testing.T) {
	r := registry.New()
	_ = r.Register("host-1", "org-1", "h1", &mockStream{})
	r.SetRegistration("host-1", &fluidv1.HostRegistration{
		SourceVms: []*fluidv1.SourceVMInfo{
			{Name: "web-server"},
		},
	})

	_, err := SelectHostForSourceVM(r, "web-server", "org-other", 90*time.Second, 0, 0)
	if err == nil {
		t.Fatal("SelectHostForSourceVM: expected error when no hosts belong to the org")
	}
}

// TestSelectHost_FallbackToSourceVM verifies the orchestrator-level fallback:
// SelectHost fails (no matching base image) but SelectHostForSourceVM succeeds
// (host has the source VM).
func TestSelectHost_FallbackToSourceVM(t *testing.T) {
	r := registry.New()
	_ = r.Register("host-1", "org-1", "h1", &mockStream{})
	r.SetRegistration("host-1", &fluidv1.HostRegistration{
		BaseImages:        []string{"centos-9"}, // Does NOT match "web-server"
		AvailableCpus:     4,
		AvailableMemoryMb: 8192,
		SourceVms: []*fluidv1.SourceVMInfo{
			{Name: "web-server", State: "running"},
		},
	})

	// SelectHost should fail because base image "web-server" is not in BaseImages
	_, err := SelectHost(r, "web-server", "org-1", 90*time.Second, 2, 2048)
	if err == nil {
		t.Fatal("SelectHost: expected error when base image doesn't match")
	}

	// But SelectHostForSourceVM should succeed because host has the source VM
	h, err := SelectHostForSourceVM(r, "web-server", "org-1", 90*time.Second, 0, 0)
	if err != nil {
		t.Fatalf("SelectHostForSourceVM: unexpected error: %v", err)
	}
	if h.HostID != "host-1" {
		t.Errorf("HostID = %q, want %q", h.HostID, "host-1")
	}
}

func TestSelectHostForSourceVM_PicksBestScore(t *testing.T) {
	r := registry.New()
	// Host 1: lower resources
	_ = r.Register("host-1", "org-1", "h1", &mockStream{})
	r.SetRegistration("host-1", &fluidv1.HostRegistration{
		SourceVms: []*fluidv1.SourceVMInfo{
			{Name: "web-server", State: "running"},
		},
		AvailableCpus:     2,
		AvailableMemoryMb: 4096,
	})
	// Host 2: higher resources, same source VM
	_ = r.Register("host-2", "org-1", "h2", &mockStream{})
	r.SetRegistration("host-2", &fluidv1.HostRegistration{
		SourceVms: []*fluidv1.SourceVMInfo{
			{Name: "web-server", State: "running"},
		},
		AvailableCpus:     8,
		AvailableMemoryMb: 16384,
	})

	h, err := SelectHostForSourceVM(r, "web-server", "org-1", 90*time.Second, 0, 0)
	if err != nil {
		t.Fatalf("SelectHostForSourceVM: unexpected error: %v", err)
	}
	// host-1 score: 4096 + 2*1024 = 6144
	// host-2 score: 16384 + 8*1024 = 24576
	if h.HostID != "host-2" {
		t.Errorf("HostID = %q, want %q (host with higher score)", h.HostID, "host-2")
	}
}

func TestSelectHost_ScorePrefersCPUAndMemory(t *testing.T) {
	r := registry.New()
	// Host 1: more memory but fewer CPUs
	_ = r.Register("host-1", "org-1", "h1", &mockStream{})
	r.SetRegistration("host-1", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04"},
		AvailableCpus:     2,
		AvailableMemoryMb: 16384,
	})
	// Host 2: fewer memory but more CPUs
	_ = r.Register("host-2", "org-1", "h2", &mockStream{})
	r.SetRegistration("host-2", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04"},
		AvailableCpus:     16,
		AvailableMemoryMb: 4096,
	})

	// host-1 score: 16384 + 2*1024 = 18432
	// host-2 score: 4096 + 16*1024 = 20480
	h, err := SelectHost(r, "ubuntu-22.04", "org-1", 90*time.Second, 2, 2048)
	if err != nil {
		t.Fatalf("SelectHost: unexpected error: %v", err)
	}
	if h.HostID != "host-2" {
		t.Errorf("HostID = %q, want %q (host with higher combined score)", h.HostID, "host-2")
	}
}

func TestSelectHost_EqualScorePicksFirst(t *testing.T) {
	r := registry.New()
	_ = r.Register("host-a", "org-1", "ha", &mockStream{})
	r.SetRegistration("host-a", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04"},
		AvailableCpus:     4,
		AvailableMemoryMb: 8192,
	})
	_ = r.Register("host-b", "org-1", "hb", &mockStream{})
	r.SetRegistration("host-b", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04"},
		AvailableCpus:     4,
		AvailableMemoryMb: 8192,
	})

	h, err := SelectHost(r, "ubuntu-22.04", "org-1", 90*time.Second, 2, 2048)
	if err != nil {
		t.Fatalf("SelectHost: unexpected error: %v", err)
	}
	// With equal scores, either host is acceptable - just verify no error.
	if h.HostID != "host-a" && h.HostID != "host-b" {
		t.Errorf("HostID = %q, want one of host-a or host-b", h.HostID)
	}
}
