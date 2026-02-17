package lxc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/provider"
)

// mockProxmox provides a configurable mock Proxmox API server for testing the LXC provider.
type mockProxmox struct {
	mu         sync.Mutex
	cts        []CTListEntry
	statuses   map[int]CTStatus
	configs    map[int]CTConfig
	ifaces     map[int][]CTInterface
	taskQueue  map[string]TaskStatus
	nodeStatus *NodeStatus
	cloneCount int
}

func newMockProxmox() *mockProxmox {
	return &mockProxmox{
		statuses:  make(map[int]CTStatus),
		configs:   make(map[int]CTConfig),
		ifaces:    make(map[int][]CTInterface),
		taskQueue: make(map[string]TaskStatus),
		nodeStatus: &NodeStatus{
			MaxCPU: 8,
			Memory: MemoryStatus{Total: 16 * 1024 * 1024 * 1024, Free: 12 * 1024 * 1024 * 1024},
			RootFS: DiskStatus{Total: 100 * 1024 * 1024 * 1024, Available: 70 * 1024 * 1024 * 1024},
		},
	}
}

func (m *mockProxmox) respond(w http.ResponseWriter, data any) {
	d, _ := json.Marshal(data)
	resp := struct {
		Data json.RawMessage `json:"data"`
	}{Data: d}
	b, _ := json.Marshal(resp)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func (m *mockProxmox) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()

		path := r.URL.Path
		// Strip api2/json prefix
		idx := strings.Index(path, "/nodes/")
		if idx >= 0 {
			path = path[idx:]
		}

		switch {
		// Task status (must come before node status to avoid path collision)
		case r.Method == http.MethodGet && strings.Contains(path, "/tasks/"):
			parts := strings.Split(path, "/tasks/")
			if len(parts) > 1 {
				upidPart := strings.TrimSuffix(parts[1], "/status")
				if ts, ok := m.taskQueue[upidPart]; ok {
					m.respond(w, ts)
					return
				}
			}
			// Default: task is done
			m.respond(w, TaskStatus{Status: "stopped", ExitStatus: "OK"})

		// List CTs
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/lxc"):
			m.respond(w, m.cts)

		// Node status
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/status") && !strings.Contains(path, "/lxc/"):
			m.respond(w, m.nodeStatus)

		// CT status
		case r.Method == http.MethodGet && strings.Contains(path, "/status/current"):
			vmid := extractVMID(path)
			if s, ok := m.statuses[vmid]; ok {
				m.respond(w, s)
			} else {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"data":null}`))
			}

		// CT config GET
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/config"):
			vmid := extractVMID(path)
			if c, ok := m.configs[vmid]; ok {
				m.respond(w, c)
			} else {
				m.respond(w, CTConfig{})
			}

		// CT config PUT
		case r.Method == http.MethodPut && strings.HasSuffix(path, "/config"):
			m.respond(w, nil)

		// Clone
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/clone"):
			m.cloneCount++
			upid := fmt.Sprintf("UPID:pve:clone:%d", m.cloneCount)
			// Mark task as immediately done
			m.taskQueue[upid] = TaskStatus{Status: "stopped", ExitStatus: "OK"}
			m.respond(w, upid)

		// Start
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/status/start"):
			vmid := extractVMID(path)
			upid := fmt.Sprintf("UPID:pve:start:%d", vmid)
			m.taskQueue[upid] = TaskStatus{Status: "stopped", ExitStatus: "OK"}
			// Update status to running
			if s, ok := m.statuses[vmid]; ok {
				s.Status = "running"
				m.statuses[vmid] = s
			}
			m.respond(w, upid)

		// Stop
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/status/stop"):
			vmid := extractVMID(path)
			upid := fmt.Sprintf("UPID:pve:stop:%d", vmid)
			m.taskQueue[upid] = TaskStatus{Status: "stopped", ExitStatus: "OK"}
			m.respond(w, upid)

		// Shutdown
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/status/shutdown"):
			vmid := extractVMID(path)
			upid := fmt.Sprintf("UPID:pve:shutdown:%d", vmid)
			m.taskQueue[upid] = TaskStatus{Status: "stopped", ExitStatus: "OK"}
			m.respond(w, upid)

		// Delete
		case r.Method == http.MethodDelete && strings.Contains(path, "/lxc/"):
			vmid := extractVMID(path)
			upid := fmt.Sprintf("UPID:pve:delete:%d", vmid)
			m.taskQueue[upid] = TaskStatus{Status: "stopped", ExitStatus: "OK"}
			m.respond(w, upid)

		// Interfaces
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/interfaces"):
			vmid := extractVMID(path)
			if iface, ok := m.ifaces[vmid]; ok {
				m.respond(w, iface)
			} else {
				m.respond(w, []CTInterface{})
			}

		// Snapshot
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/snapshot"):
			upid := "UPID:pve:snapshot:1"
			m.taskQueue[upid] = TaskStatus{Status: "stopped", ExitStatus: "OK"}
			m.respond(w, upid)

		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"data":null}`))
		}
	})
}

func extractVMID(path string) int {
	// Extract VMID from paths like /nodes/pve/lxc/100/...
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if p == "lxc" && i+1 < len(parts) {
			var vmid int
			_, _ = fmt.Sscanf(parts[i+1], "%d", &vmid)
			return vmid
		}
	}
	return 0
}

func testProvider(t *testing.T, mock *mockProxmox) (*Provider, *httptest.Server) {
	t.Helper()
	srv := httptest.NewTLSServer(mock.handler())
	t.Cleanup(srv.Close)

	cfg := Config{
		Host:      srv.URL,
		TokenID:   "test@pam!tok",
		Secret:    "secret",
		Node:      "pve",
		Bridge:    "vmbr0",
		VMIDStart: 9000,
		VMIDEnd:   9999,
		VerifySSL: false,
		Timeout:   10 * time.Second,
	}

	prov, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Override the client's httpClient to use the test server's TLS client
	prov.client.httpClient = srv.Client()
	prov.client.httpClient.Timeout = 10 * time.Second

	return prov, srv
}

func TestProvider_ListTemplates(t *testing.T) {
	mock := newMockProxmox()
	mock.cts = []CTListEntry{
		{VMID: 100, Name: "ubuntu-template", Template: 1, Status: "stopped"},
		{VMID: 101, Name: "debian-template", Template: 1, Status: "stopped"},
		{VMID: 200, Name: "web-server", Template: 0, Status: "running"},
	}

	prov, _ := testProvider(t, mock)

	templates, err := prov.ListTemplates(context.Background())
	if err != nil {
		t.Fatalf("ListTemplates() error: %v", err)
	}

	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}

	names := make(map[string]bool)
	for _, n := range templates {
		names[n] = true
	}
	if !names["ubuntu-template"] || !names["debian-template"] {
		t.Errorf("unexpected templates: %v", templates)
	}
}

func TestProvider_ListSourceVMs(t *testing.T) {
	mock := newMockProxmox()
	mock.cts = []CTListEntry{
		{VMID: 100, Name: "ubuntu-template", Template: 1, Status: "stopped"},
		{VMID: 200, Name: "web-server", Template: 0, Status: "running"},
		{VMID: 201, Name: "db-server", Template: 0, Status: "stopped"},
		{VMID: 9001, Name: "sbx-abc12345", Template: 0, Status: "running"},
	}
	mock.ifaces[200] = []CTInterface{
		{Name: "eth0", Inet: "10.0.0.5/24"},
	}

	prov, _ := testProvider(t, mock)

	vms, err := prov.ListSourceVMs(context.Background())
	if err != nil {
		t.Fatalf("ListSourceVMs() error: %v", err)
	}

	// Should exclude templates and sbx- prefixed containers
	if len(vms) != 2 {
		t.Fatalf("expected 2 source VMs, got %d: %+v", len(vms), vms)
	}

	names := make(map[string]bool)
	for _, vm := range vms {
		names[vm.Name] = true
	}
	if !names["web-server"] || !names["db-server"] {
		t.Errorf("unexpected VMs: %+v", vms)
	}
}

func TestProvider_Capabilities(t *testing.T) {
	mock := newMockProxmox()
	mock.cts = []CTListEntry{
		{VMID: 100, Name: "template-1", Template: 1},
	}

	prov, _ := testProvider(t, mock)

	caps, err := prov.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities() error: %v", err)
	}

	if caps.TotalCPUs != 8 {
		t.Errorf("TotalCPUs = %d, want 8", caps.TotalCPUs)
	}
	if caps.TotalMemoryMB != 16*1024 {
		t.Errorf("TotalMemoryMB = %d, want %d", caps.TotalMemoryMB, 16*1024)
	}
	if caps.AvailableMemMB != 12*1024 {
		t.Errorf("AvailableMemMB = %d, want %d", caps.AvailableMemMB, 12*1024)
	}
}

func TestProvider_RecoverState(t *testing.T) {
	mock := newMockProxmox()
	mock.cts = []CTListEntry{
		{VMID: 100, Name: "web-server", Template: 0, Status: "running"},
		{VMID: 9001, Name: "sbx-sandbox1", Template: 0, Status: "running"},
		{VMID: 9002, Name: "sbx-sandbox2", Template: 0, Status: "stopped"},
		{VMID: 200, Name: "ubuntu-template", Template: 1, Status: "stopped"},
	}

	prov, _ := testProvider(t, mock)

	err := prov.RecoverState(context.Background())
	if err != nil {
		t.Fatalf("RecoverState() error: %v", err)
	}

	// Should recover only sbx- prefixed non-template containers
	if prov.ActiveSandboxCount() != 2 {
		t.Errorf("ActiveSandboxCount = %d, want 2", prov.ActiveSandboxCount())
	}
}

func TestProvider_ActiveSandboxCount(t *testing.T) {
	mock := newMockProxmox()
	prov, _ := testProvider(t, mock)

	if prov.ActiveSandboxCount() != 0 {
		t.Errorf("initial ActiveSandboxCount = %d, want 0", prov.ActiveSandboxCount())
	}

	// Manually track sandboxes
	prov.mu.Lock()
	prov.sandboxes["sbx-1"] = 9001
	prov.sandboxes["sbx-2"] = 9002
	prov.mu.Unlock()

	if prov.ActiveSandboxCount() != 2 {
		t.Errorf("ActiveSandboxCount = %d, want 2", prov.ActiveSandboxCount())
	}
}

func TestProvider_DestroySandbox_NotTracked(t *testing.T) {
	mock := newMockProxmox()
	prov, _ := testProvider(t, mock)

	err := prov.DestroySandbox(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for untracked sandbox")
	}
	if !strings.Contains(err.Error(), "not tracked") {
		t.Errorf("error = %q, want containing 'not tracked'", err.Error())
	}
}

func TestProvider_DestroySandbox(t *testing.T) {
	mock := newMockProxmox()
	mock.cts = []CTListEntry{
		{VMID: 9001, Name: "sbx-test", Status: "running"},
	}
	mock.statuses[9001] = CTStatus{VMID: 9001, Status: "running"}

	prov, _ := testProvider(t, mock)

	// Track the sandbox
	prov.mu.Lock()
	prov.sandboxes["test-sandbox"] = 9001
	prov.mu.Unlock()

	err := prov.DestroySandbox(context.Background(), "test-sandbox")
	if err != nil {
		t.Fatalf("DestroySandbox() error: %v", err)
	}

	// Should be removed from tracking
	if prov.ActiveSandboxCount() != 0 {
		t.Errorf("ActiveSandboxCount = %d, want 0", prov.ActiveSandboxCount())
	}
}

func TestProvider_ValidateSourceVM_Found(t *testing.T) {
	mock := newMockProxmox()
	mock.cts = []CTListEntry{
		{VMID: 100, Name: "web-server", Status: "running"},
	}
	mock.statuses[100] = CTStatus{VMID: 100, Name: "web-server", Status: "running"}
	mock.configs[100] = CTConfig{Net0: "name=eth0,bridge=vmbr0"}
	mock.ifaces[100] = []CTInterface{
		{Name: "eth0", Inet: "10.0.0.5/24"},
	}

	prov, _ := testProvider(t, mock)

	result, err := prov.ValidateSourceVM(context.Background(), "web-server")
	if err != nil {
		t.Fatalf("ValidateSourceVM() error: %v", err)
	}

	if !result.Valid {
		t.Error("expected Valid=true")
	}
	if result.State != "running" {
		t.Errorf("State = %q, want running", result.State)
	}
	if !result.HasNetwork {
		t.Error("expected HasNetwork=true")
	}
	if result.IPAddress != "10.0.0.5" {
		t.Errorf("IPAddress = %q, want 10.0.0.5", result.IPAddress)
	}
}

func TestProvider_ValidateSourceVM_NotFound(t *testing.T) {
	mock := newMockProxmox()
	mock.cts = []CTListEntry{}

	prov, _ := testProvider(t, mock)

	result, err := prov.ValidateSourceVM(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("ValidateSourceVM() error: %v", err)
	}

	if result.Valid {
		t.Error("expected Valid=false for nonexistent CT")
	}
	if len(result.Errors) == 0 {
		t.Error("expected at least one error")
	}
}

func TestProvider_ValidateSourceVM_NoNetwork(t *testing.T) {
	mock := newMockProxmox()
	mock.cts = []CTListEntry{
		{VMID: 100, Name: "isolated-ct", Status: "stopped"},
	}
	mock.statuses[100] = CTStatus{VMID: 100, Name: "isolated-ct", Status: "stopped"}
	mock.configs[100] = CTConfig{Net0: ""}

	prov, _ := testProvider(t, mock)

	result, err := prov.ValidateSourceVM(context.Background(), "isolated-ct")
	if err != nil {
		t.Fatalf("ValidateSourceVM() error: %v", err)
	}

	if !result.HasNetwork {
		// Correctly detected no network
	} else {
		t.Error("expected HasNetwork=false for CT with empty Net0")
	}
}

func TestProvider_GetSandboxIP_NotTracked(t *testing.T) {
	mock := newMockProxmox()
	prov, _ := testProvider(t, mock)

	_, err := prov.GetSandboxIP(context.Background(), "unknown")
	if err == nil {
		t.Fatal("expected error for untracked sandbox")
	}
}

func TestProvider_StopSandbox_NotTracked(t *testing.T) {
	mock := newMockProxmox()
	prov, _ := testProvider(t, mock)

	err := prov.StopSandbox(context.Background(), "unknown", false)
	if err == nil {
		t.Fatal("expected error for untracked sandbox")
	}
}

func TestProvider_CreateSnapshot_NotTracked(t *testing.T) {
	mock := newMockProxmox()
	prov, _ := testProvider(t, mock)

	_, err := prov.CreateSnapshot(context.Background(), "unknown", "snap")
	if err == nil {
		t.Fatal("expected error for untracked sandbox")
	}
}

func TestProvider_StartSandbox(t *testing.T) {
	mock := newMockProxmox()
	mock.cts = []CTListEntry{
		{VMID: 9001, Name: "sbx-test", Status: "stopped"},
	}
	mock.statuses[9001] = CTStatus{VMID: 9001, Status: "stopped"}
	mock.ifaces[9001] = []CTInterface{
		{Name: "eth0", Inet: "10.0.0.10/24"},
	}

	prov, _ := testProvider(t, mock)
	prov.mu.Lock()
	prov.sandboxes["test-sbx"] = 9001
	prov.mu.Unlock()

	result, err := prov.StartSandbox(context.Background(), "test-sbx")
	if err != nil {
		t.Fatalf("StartSandbox() error: %v", err)
	}

	if result.State != "RUNNING" {
		t.Errorf("State = %q, want RUNNING", result.State)
	}
	if result.IPAddress != "10.0.0.10" {
		t.Errorf("IPAddress = %q, want 10.0.0.10", result.IPAddress)
	}
}

func TestProvider_StopSandbox_Force(t *testing.T) {
	mock := newMockProxmox()
	prov, _ := testProvider(t, mock)
	prov.mu.Lock()
	prov.sandboxes["test-sbx"] = 9001
	prov.mu.Unlock()

	err := prov.StopSandbox(context.Background(), "test-sbx", true)
	if err != nil {
		t.Fatalf("StopSandbox(force) error: %v", err)
	}
}

func TestProvider_StopSandbox_Graceful(t *testing.T) {
	mock := newMockProxmox()
	prov, _ := testProvider(t, mock)
	prov.mu.Lock()
	prov.sandboxes["test-sbx"] = 9001
	prov.mu.Unlock()

	err := prov.StopSandbox(context.Background(), "test-sbx", false)
	if err != nil {
		t.Fatalf("StopSandbox(graceful) error: %v", err)
	}
}

func TestProvider_CreateSnapshot(t *testing.T) {
	mock := newMockProxmox()
	prov, _ := testProvider(t, mock)
	prov.mu.Lock()
	prov.sandboxes["test-sbx"] = 9001
	prov.mu.Unlock()

	result, err := prov.CreateSnapshot(context.Background(), "test-sbx", "my-snapshot")
	if err != nil {
		t.Fatalf("CreateSnapshot() error: %v", err)
	}

	if result.SnapshotName != "my-snapshot" {
		t.Errorf("SnapshotName = %q, want my-snapshot", result.SnapshotName)
	}
	if !strings.HasPrefix(result.SnapshotID, "SNP-") {
		t.Errorf("SnapshotID = %q, want SNP- prefix", result.SnapshotID)
	}
}

func TestProvider_CreateSandbox(t *testing.T) {
	mock := newMockProxmox()
	mock.cts = []CTListEntry{
		{VMID: 100, Name: "source-template", Template: 1, Status: "stopped"},
	}
	// After clone, the new CT shows up with interfaces
	mock.ifaces[9000] = []CTInterface{
		{Name: "lo", Inet: "127.0.0.1/8"},
		{Name: "eth0", Inet: "10.0.0.50/24"},
	}
	mock.statuses[9000] = CTStatus{VMID: 9000, Status: "stopped"}

	prov, _ := testProvider(t, mock)

	req := provider.CreateRequest{
		SandboxID: "sbx-12345678-abcd",
		Name:      "sbx-test-sandbox",
		SourceVM:  "source-template",
		VCPUs:     2,
		MemoryMB:  1024,
	}

	result, err := prov.CreateSandbox(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateSandbox() error: %v", err)
	}

	if result.State != "RUNNING" {
		t.Errorf("State = %q, want RUNNING", result.State)
	}
	if result.IPAddress != "10.0.0.50" {
		t.Errorf("IPAddress = %q, want 10.0.0.50", result.IPAddress)
	}
	if result.Bridge != "vmbr0" {
		t.Errorf("Bridge = %q, want vmbr0", result.Bridge)
	}

	// Should be tracked
	if prov.ActiveSandboxCount() != 1 {
		t.Errorf("ActiveSandboxCount = %d, want 1", prov.ActiveSandboxCount())
	}
}

func TestProvider_CreateSandbox_CustomBridge(t *testing.T) {
	mock := newMockProxmox()
	mock.cts = []CTListEntry{
		{VMID: 100, Name: "src", Template: 1, Status: "stopped"},
	}
	mock.ifaces[9000] = []CTInterface{
		{Name: "eth0", Inet: "192.168.1.10/24"},
	}
	mock.statuses[9000] = CTStatus{VMID: 9000, Status: "stopped"}

	prov, _ := testProvider(t, mock)

	req := provider.CreateRequest{
		SandboxID: "sbx-custom-bridge",
		SourceVM:  "src",
		Network:   "vmbr1",
	}

	result, err := prov.CreateSandbox(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateSandbox() error: %v", err)
	}

	if result.Bridge != "vmbr1" {
		t.Errorf("Bridge = %q, want vmbr1", result.Bridge)
	}
}

func TestProvider_DiscoverIP_FiltersLoopback(t *testing.T) {
	mock := newMockProxmox()
	mock.ifaces[100] = []CTInterface{
		{Name: "lo", Inet: "127.0.0.1/8"},
		{Name: "eth0", Inet: "10.0.0.5/24"},
	}

	prov, _ := testProvider(t, mock)

	ip, err := prov.discoverIP(context.Background(), 100, 5*time.Second)
	if err != nil {
		t.Fatalf("discoverIP() error: %v", err)
	}
	if ip != "10.0.0.5" {
		t.Errorf("IP = %q, want 10.0.0.5", ip)
	}
}

func TestProvider_DiscoverIP_FiltersLinkLocal(t *testing.T) {
	mock := newMockProxmox()
	mock.ifaces[100] = []CTInterface{
		{Name: "eth0", Inet: "169.254.1.1/16"},
		{Name: "eth1", Inet: "192.168.0.5/24"},
	}

	prov, _ := testProvider(t, mock)

	ip, err := prov.discoverIP(context.Background(), 100, 5*time.Second)
	if err != nil {
		t.Fatalf("discoverIP() error: %v", err)
	}
	if ip != "192.168.0.5" {
		t.Errorf("IP = %q, want 192.168.0.5", ip)
	}
}

func TestProvider_DiscoverIP_Timeout(t *testing.T) {
	mock := newMockProxmox()
	// No interfaces configured - will never find an IP

	prov, _ := testProvider(t, mock)

	_, err := prov.discoverIP(context.Background(), 100, 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("error = %q, want containing 'timeout'", err.Error())
	}
}

func TestProvider_New_InvalidConfig(t *testing.T) {
	cfg := Config{} // Missing required fields
	_, err := New(cfg, nil)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

func TestProvider_RunCommand_NotTracked(t *testing.T) {
	mock := newMockProxmox()
	prov, _ := testProvider(t, mock)

	_, err := prov.RunCommand(context.Background(), "unknown", "ls", 10*time.Second)
	if err == nil {
		t.Fatal("expected error for untracked sandbox")
	}
}

func TestProvider_ReadSourceFile_NotFound(t *testing.T) {
	mock := newMockProxmox()
	mock.cts = []CTListEntry{}

	prov, _ := testProvider(t, mock)

	_, err := prov.ReadSourceFile(context.Background(), "nonexistent", "/etc/hostname")
	if err == nil {
		t.Fatal("expected error for nonexistent source")
	}
}

func TestProvider_RunSourceCommand_NotFound(t *testing.T) {
	mock := newMockProxmox()
	mock.cts = []CTListEntry{}

	prov, _ := testProvider(t, mock)

	_, err := prov.RunSourceCommand(context.Background(), "nonexistent", "ls", 10*time.Second)
	if err == nil {
		t.Fatal("expected error for nonexistent source")
	}
}
