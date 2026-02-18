package registry

import (
	"sync"
	"testing"
	"time"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"
)

// mockStream implements HostStream for testing.
type mockStream struct{}

func (m *mockStream) Send(_ *fluidv1.ControlMessage) error { return nil }

func TestRegister(t *testing.T) {
	reg := New()
	s := &mockStream{}

	if err := reg.Register("host-1", "org-1", "myhost", s); err != nil {
		t.Fatalf("Register: unexpected error: %v", err)
	}

	h, ok := reg.GetHost("host-1")
	if !ok {
		t.Fatal("GetHost: expected host to be found")
	}
	if h.HostID != "host-1" {
		t.Errorf("HostID = %q, want %q", h.HostID, "host-1")
	}
	if h.OrgID != "org-1" {
		t.Errorf("OrgID = %q, want %q", h.OrgID, "org-1")
	}
	if h.Hostname != "myhost" {
		t.Errorf("Hostname = %q, want %q", h.Hostname, "myhost")
	}
	if h.Stream == nil {
		t.Error("Stream: expected non-nil")
	}
}

func TestRegister_EmptyHostID(t *testing.T) {
	reg := New()
	err := reg.Register("", "org-1", "myhost", &mockStream{})
	if err == nil {
		t.Fatal("Register: expected error for empty host ID")
	}
}

func TestRegister_NilStream(t *testing.T) {
	reg := New()
	err := reg.Register("host-1", "org-1", "myhost", nil)
	if err == nil {
		t.Fatal("Register: expected error for nil stream")
	}
}

func TestUnregister(t *testing.T) {
	reg := New()
	_ = reg.Register("host-1", "org-1", "myhost", &mockStream{})

	reg.Unregister("host-1")

	_, ok := reg.GetHost("host-1")
	if ok {
		t.Fatal("GetHost: expected host to be removed after Unregister")
	}
}

func TestUnregister_Nonexistent(t *testing.T) {
	reg := New()
	// Should not panic.
	reg.Unregister("nonexistent")
}

func TestListConnected(t *testing.T) {
	reg := New()
	_ = reg.Register("host-1", "org-1", "h1", &mockStream{})
	_ = reg.Register("host-2", "org-1", "h2", &mockStream{})
	_ = reg.Register("host-3", "org-2", "h3", &mockStream{})

	hosts := reg.ListConnected()
	if len(hosts) != 3 {
		t.Fatalf("ListConnected: got %d hosts, want 3", len(hosts))
	}
}

func TestListConnectedByOrg(t *testing.T) {
	reg := New()
	_ = reg.Register("host-1", "org-1", "h1", &mockStream{})
	_ = reg.Register("host-2", "org-1", "h2", &mockStream{})
	_ = reg.Register("host-3", "org-2", "h3", &mockStream{})

	hosts := reg.ListConnectedByOrg("org-1")
	if len(hosts) != 2 {
		t.Fatalf("ListConnectedByOrg(org-1): got %d hosts, want 2", len(hosts))
	}

	hosts = reg.ListConnectedByOrg("org-2")
	if len(hosts) != 1 {
		t.Fatalf("ListConnectedByOrg(org-2): got %d hosts, want 1", len(hosts))
	}

	hosts = reg.ListConnectedByOrg("org-none")
	if len(hosts) != 0 {
		t.Fatalf("ListConnectedByOrg(org-none): got %d hosts, want 0", len(hosts))
	}
}

func TestSetRegistration(t *testing.T) {
	reg := New()
	_ = reg.Register("host-1", "org-1", "h1", &mockStream{})

	regData := &fluidv1.HostRegistration{
		Hostname:          "updated-host",
		AvailableCpus:     8,
		AvailableMemoryMb: 16384,
		AvailableDiskMb:   102400,
		BaseImages:        []string{"ubuntu-22.04", "debian-12"},
		SourceVms: []*fluidv1.SourceVMInfo{
			{Name: "web-server", State: "running"},
		},
		Bridges: []*fluidv1.BridgeInfo{
			{Name: "br0", Subnet: "10.0.0.0/24"},
		},
	}

	beforeSet := time.Now()
	reg.SetRegistration("host-1", regData)

	h, ok := reg.GetHost("host-1")
	if !ok {
		t.Fatal("GetHost: host not found after SetRegistration")
	}
	if h.Registration == nil {
		t.Fatal("Registration: expected non-nil after SetRegistration")
	}
	if h.Registration.GetAvailableCpus() != 8 {
		t.Errorf("AvailableCpus = %d, want 8", h.Registration.GetAvailableCpus())
	}
	if h.Registration.GetAvailableMemoryMb() != 16384 {
		t.Errorf("AvailableMemoryMb = %d, want 16384", h.Registration.GetAvailableMemoryMb())
	}
	if len(h.Registration.GetBaseImages()) != 2 {
		t.Errorf("BaseImages: got %d, want 2", len(h.Registration.GetBaseImages()))
	}
	if len(h.Registration.GetSourceVms()) != 1 {
		t.Errorf("SourceVms: got %d, want 1", len(h.Registration.GetSourceVms()))
	}
	if len(h.Registration.GetBridges()) != 1 {
		t.Errorf("Bridges: got %d, want 1", len(h.Registration.GetBridges()))
	}
	// SetRegistration also updates heartbeat.
	if h.LastHeartbeat.Before(beforeSet) {
		t.Error("LastHeartbeat should have been updated by SetRegistration")
	}
}

func TestSetRegistration_NonexistentHost(t *testing.T) {
	reg := New()
	// Should not panic.
	reg.SetRegistration("nonexistent", &fluidv1.HostRegistration{})
}

func TestUpdateHeartbeat(t *testing.T) {
	reg := New()
	_ = reg.Register("host-1", "org-1", "h1", &mockStream{})

	h, _ := reg.GetHost("host-1")
	originalHB := h.LastHeartbeat

	// Small sleep to ensure time difference.
	time.Sleep(10 * time.Millisecond)
	reg.UpdateHeartbeat("host-1")

	h, _ = reg.GetHost("host-1")
	if !h.LastHeartbeat.After(originalHB) {
		t.Error("LastHeartbeat should be newer after UpdateHeartbeat")
	}
}

func TestUpdateHeartbeat_NonexistentHost(t *testing.T) {
	reg := New()
	// Should not panic.
	reg.UpdateHeartbeat("nonexistent")
}

func TestConcurrentAccess(t *testing.T) {
	reg := New()
	var wg sync.WaitGroup
	const n = 100

	// Register concurrently.
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			hostID := "host-" + string(rune('A'+id%26)) + string(rune('0'+id%10))
			_ = reg.Register(hostID, "org-1", "h", &mockStream{})
		}(i)
	}
	wg.Wait()

	// Read concurrently.
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reg.ListConnected()
		}()
	}
	wg.Wait()

	// Unregister concurrently.
	hosts := reg.ListConnected()
	for _, h := range hosts {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			reg.Unregister(id)
		}(h.HostID)
	}
	wg.Wait()

	remaining := reg.ListConnected()
	if len(remaining) != 0 {
		t.Errorf("after unregister all: got %d hosts, want 0", len(remaining))
	}
}

func TestRegister_ReplacesExisting(t *testing.T) {
	reg := New()
	s1 := &mockStream{}
	s2 := &mockStream{}

	_ = reg.Register("host-1", "org-1", "h1-old", s1)
	_ = reg.Register("host-1", "org-1", "h1-new", s2)

	h, ok := reg.GetHost("host-1")
	if !ok {
		t.Fatal("GetHost: expected host to exist")
	}
	if h.Hostname != "h1-new" {
		t.Errorf("Hostname = %q, want %q (should be replaced)", h.Hostname, "h1-new")
	}
}
