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

	// Manually set the heartbeat to 2 minutes ago by directly accessing the host.
	h, ok := r.GetHost("host-1")
	if !ok {
		t.Fatal("GetHost: host not found")
	}
	h.LastHeartbeat = time.Now().Add(-2 * time.Minute)

	_, err := SelectHost(r, "ubuntu-22.04", "org-1", 90*time.Second, 2, 2048)
	if err == nil {
		t.Fatal("SelectHost: expected error when host has stale heartbeat (>90s)")
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

	h, err := SelectHostForSourceVM(r, "web-server", "org-1", 90*time.Second)
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

	_, err := SelectHostForSourceVM(r, "nonexistent-vm", "org-1", 90*time.Second)
	if err == nil {
		t.Fatal("SelectHostForSourceVM: expected error when no host has the source VM")
	}
}

func TestSelectHostForSourceVM_NoHosts(t *testing.T) {
	r := registry.New()
	_, err := SelectHostForSourceVM(r, "web-server", "org-1", 90*time.Second)
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

	// Set heartbeat to stale (>90s).
	h, _ := r.GetHost("host-1")
	h.LastHeartbeat = time.Now().Add(-2 * time.Minute)

	_, err := SelectHostForSourceVM(r, "web-server", "org-1", 90*time.Second)
	if err == nil {
		t.Fatal("SelectHostForSourceVM: expected error when host heartbeat is stale")
	}
}

func TestSelectHostForSourceVM_NilRegistration(t *testing.T) {
	r := registry.New()
	_ = r.Register("host-1", "org-1", "h1", &mockStream{})
	// No SetRegistration.

	_, err := SelectHostForSourceVM(r, "web-server", "org-1", 90*time.Second)
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

	_, err := SelectHostForSourceVM(r, "web-server", "org-other", 90*time.Second)
	if err == nil {
		t.Fatal("SelectHostForSourceVM: expected error when no hosts belong to the org")
	}
}
