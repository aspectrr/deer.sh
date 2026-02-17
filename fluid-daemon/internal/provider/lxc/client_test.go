package lxc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// proxmoxResponse wraps data in the Proxmox API envelope.
func proxmoxResponse(data any) []byte {
	d, _ := json.Marshal(data)
	resp := struct {
		Data json.RawMessage `json:"data"`
	}{Data: d}
	b, _ := json.Marshal(resp)
	return b
}

func testClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	t.Cleanup(srv.Close)

	cfg := Config{
		Host:      srv.URL,
		TokenID:   "test@pam!testtoken",
		Secret:    "test-secret",
		Node:      "pve",
		VerifySSL: false,
		Timeout:   10 * time.Second,
	}
	client := NewClient(cfg, nil)
	// Override httpClient to use the test server's TLS client
	client.httpClient = srv.Client()
	client.httpClient.Timeout = 10 * time.Second
	return client, srv
}

func TestListCTs(t *testing.T) {
	cts := []CTListEntry{
		{VMID: 100, Name: "web-server", Status: "running"},
		{VMID: 101, Name: "db-server", Status: "stopped", Template: 1},
	}

	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/nodes/pve/lxc") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Check auth header
		auth := r.Header.Get("Authorization")
		if !strings.Contains(auth, "PVEAPIToken=") {
			t.Errorf("missing API token in Authorization header: %s", auth)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(proxmoxResponse(cts))
	}))

	result, err := client.ListCTs(context.Background())
	if err != nil {
		t.Fatalf("ListCTs() error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 CTs, got %d", len(result))
	}
	if result[0].Name != "web-server" {
		t.Errorf("result[0].Name = %q, want %q", result[0].Name, "web-server")
	}
	if result[1].Template != 1 {
		t.Errorf("result[1].Template = %d, want 1", result[1].Template)
	}
}

func TestGetCTStatus(t *testing.T) {
	status := CTStatus{
		VMID:   100,
		Name:   "web-server",
		Status: "running",
		CPU:    0.15,
		MaxMem: 2147483648,
		Mem:    536870912,
	}

	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/nodes/pve/lxc/100/status/current") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(proxmoxResponse(status))
	}))

	result, err := client.GetCTStatus(context.Background(), 100)
	if err != nil {
		t.Fatalf("GetCTStatus() error: %v", err)
	}
	if result.Status != "running" {
		t.Errorf("Status = %q, want %q", result.Status, "running")
	}
	if result.Name != "web-server" {
		t.Errorf("Name = %q, want %q", result.Name, "web-server")
	}
}

func TestGetCTConfig(t *testing.T) {
	cfg := CTConfig{
		Hostname: "test-ct",
		Memory:   2048,
		Cores:    4,
		Net0:     "name=eth0,bridge=vmbr0,ip=dhcp",
	}

	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/nodes/pve/lxc/100/config") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(proxmoxResponse(cfg))
	}))

	result, err := client.GetCTConfig(context.Background(), 100)
	if err != nil {
		t.Fatalf("GetCTConfig() error: %v", err)
	}
	if result.Cores != 4 {
		t.Errorf("Cores = %d, want 4", result.Cores)
	}
	if result.Net0 != cfg.Net0 {
		t.Errorf("Net0 = %q, want %q", result.Net0, cfg.Net0)
	}
}

func TestCloneCT(t *testing.T) {
	expectedUPID := "UPID:pve:000F1234:00B3C4D5:12345678:vzclone:100:user@pam:"

	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/nodes/pve/lxc/100/clone") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Check body params
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.FormValue("newid") != "9001" {
			t.Errorf("newid = %q, want 9001", r.FormValue("newid"))
		}
		if r.FormValue("hostname") != "sbx-test" {
			t.Errorf("hostname = %q, want sbx-test", r.FormValue("hostname"))
		}
		if r.FormValue("full") != "1" {
			t.Errorf("full = %q, want 1", r.FormValue("full"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(proxmoxResponse(expectedUPID))
	}))

	upid, err := client.CloneCT(context.Background(), 100, 9001, "sbx-test", true)
	if err != nil {
		t.Fatalf("CloneCT() error: %v", err)
	}
	if upid != expectedUPID {
		t.Errorf("UPID = %q, want %q", upid, expectedUPID)
	}
}

func TestStartCT(t *testing.T) {
	expectedUPID := "UPID:pve:0001:0002:12345678:vzstart:100:user@pam:"

	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/nodes/pve/lxc/100/status/start") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(proxmoxResponse(expectedUPID))
	}))

	upid, err := client.StartCT(context.Background(), 100)
	if err != nil {
		t.Fatalf("StartCT() error: %v", err)
	}
	if upid != expectedUPID {
		t.Errorf("UPID = %q, want %q", upid, expectedUPID)
	}
}

func TestStopCT(t *testing.T) {
	expectedUPID := "UPID:pve:0001:0002:12345678:vzstop:100:user@pam:"

	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/nodes/pve/lxc/100/status/stop") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(proxmoxResponse(expectedUPID))
	}))

	upid, err := client.StopCT(context.Background(), 100)
	if err != nil {
		t.Fatalf("StopCT() error: %v", err)
	}
	if upid != expectedUPID {
		t.Errorf("UPID = %q, want %q", upid, expectedUPID)
	}
}

func TestShutdownCT(t *testing.T) {
	expectedUPID := "UPID:pve:0001:0002:12345678:vzshutdown:100:user@pam:"

	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/nodes/pve/lxc/100/status/shutdown") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(proxmoxResponse(expectedUPID))
	}))

	upid, err := client.ShutdownCT(context.Background(), 100)
	if err != nil {
		t.Fatalf("ShutdownCT() error: %v", err)
	}
	if upid != expectedUPID {
		t.Errorf("UPID = %q, want %q", upid, expectedUPID)
	}
}

func TestDeleteCT(t *testing.T) {
	expectedUPID := "UPID:pve:0001:0002:12345678:vzdel:100:user@pam:"

	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/nodes/pve/lxc/100") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Check purge and force query params
		if r.URL.Query().Get("purge") != "1" {
			t.Errorf("expected purge=1 in query")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(proxmoxResponse(expectedUPID))
	}))

	upid, err := client.DeleteCT(context.Background(), 100)
	if err != nil {
		t.Fatalf("DeleteCT() error: %v", err)
	}
	if upid != expectedUPID {
		t.Errorf("UPID = %q, want %q", upid, expectedUPID)
	}
}

func TestGetCTInterfaces(t *testing.T) {
	ifaces := []CTInterface{
		{Name: "lo", HWAddr: "00:00:00:00:00:00", Inet: "127.0.0.1/8"},
		{Name: "eth0", HWAddr: "AA:BB:CC:DD:EE:FF", Inet: "10.0.0.5/24"},
	}

	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/nodes/pve/lxc/100/interfaces") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(proxmoxResponse(ifaces))
	}))

	result, err := client.GetCTInterfaces(context.Background(), 100)
	if err != nil {
		t.Fatalf("GetCTInterfaces() error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 interfaces, got %d", len(result))
	}
	if result[1].Inet != "10.0.0.5/24" {
		t.Errorf("result[1].Inet = %q, want %q", result[1].Inet, "10.0.0.5/24")
	}
}

func TestCreateSnapshot(t *testing.T) {
	expectedUPID := "UPID:pve:0001:0002:12345678:vzsnapshot:100:user@pam:"

	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/nodes/pve/lxc/100/snapshot") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.FormValue("snapname") != "snap-1" {
			t.Errorf("snapname = %q, want snap-1", r.FormValue("snapname"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(proxmoxResponse(expectedUPID))
	}))

	upid, err := client.CreateSnapshot(context.Background(), 100, "snap-1")
	if err != nil {
		t.Fatalf("CreateSnapshot() error: %v", err)
	}
	if upid != expectedUPID {
		t.Errorf("UPID = %q, want %q", upid, expectedUPID)
	}
}

func TestGetNodeStatus(t *testing.T) {
	status := NodeStatus{
		CPU:    0.25,
		MaxCPU: 8,
		Memory: MemoryStatus{Total: 16 * 1024 * 1024 * 1024, Used: 4 * 1024 * 1024 * 1024, Free: 12 * 1024 * 1024 * 1024},
		RootFS: DiskStatus{Total: 100 * 1024 * 1024 * 1024, Used: 30 * 1024 * 1024 * 1024, Available: 70 * 1024 * 1024 * 1024},
	}

	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/nodes/pve/status") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(proxmoxResponse(status))
	}))

	result, err := client.GetNodeStatus(context.Background())
	if err != nil {
		t.Fatalf("GetNodeStatus() error: %v", err)
	}
	if result.MaxCPU != 8 {
		t.Errorf("MaxCPU = %d, want 8", result.MaxCPU)
	}
	if result.Memory.Free != 12*1024*1024*1024 {
		t.Errorf("Memory.Free = %d, unexpected", result.Memory.Free)
	}
}

func TestGetTaskStatus(t *testing.T) {
	taskStatus := TaskStatus{
		Status:     "stopped",
		ExitStatus: "OK",
		Type:       "vzstart",
		Node:       "pve",
	}

	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/nodes/pve/tasks/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(proxmoxResponse(taskStatus))
	}))

	result, err := client.GetTaskStatus(context.Background(), "UPID:pve:test")
	if err != nil {
		t.Fatalf("GetTaskStatus() error: %v", err)
	}
	if result.Status != "stopped" {
		t.Errorf("Status = %q, want stopped", result.Status)
	}
	if result.ExitStatus != "OK" {
		t.Errorf("ExitStatus = %q, want OK", result.ExitStatus)
	}
}

func TestNextVMID(t *testing.T) {
	cts := []CTListEntry{
		{VMID: 9000, Name: "ct-a"},
		{VMID: 9001, Name: "ct-b"},
		{VMID: 9003, Name: "ct-c"},
	}

	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(proxmoxResponse(cts))
	}))

	// Should skip 9000, 9001 and return 9002
	vmid, err := client.NextVMID(context.Background(), 9000, 9999)
	if err != nil {
		t.Fatalf("NextVMID() error: %v", err)
	}
	if vmid != 9002 {
		t.Errorf("VMID = %d, want 9002", vmid)
	}
}

func TestNextVMID_RangeExhausted(t *testing.T) {
	cts := []CTListEntry{
		{VMID: 100, Name: "ct-a"},
		{VMID: 101, Name: "ct-b"},
		{VMID: 102, Name: "ct-c"},
	}

	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(proxmoxResponse(cts))
	}))

	_, err := client.NextVMID(context.Background(), 100, 102)
	if err == nil {
		t.Fatal("expected error for exhausted range")
	}
	if !strings.Contains(err.Error(), "no available VMID") {
		t.Errorf("error = %q, want containing 'no available VMID'", err.Error())
	}
}

func TestClient_HTTPError4xx(t *testing.T) {
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"data":null,"errors":{"vmid":"not found"}}`))
	}))

	_, err := client.GetCTStatus(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error = %q, want containing '404'", err.Error())
	}
}

func TestClient_RetryOn500(t *testing.T) {
	attempts := 0
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"data":null}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(proxmoxResponse(CTStatus{VMID: 100, Status: "running"}))
	}))
	// Override maxRetries to ensure we retry
	client.maxRetries = 3

	result, err := client.GetCTStatus(context.Background(), 100)
	if err != nil {
		t.Fatalf("expected retry to succeed, got error: %v", err)
	}
	if result.Status != "running" {
		t.Errorf("Status = %q, want running", result.Status)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestClient_RetryExhausted(t *testing.T) {
	attempts := 0
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, `{"data":null,"errors":"server error attempt %d"}`, attempts)
	}))
	client.maxRetries = 2

	_, err := client.GetCTStatus(context.Background(), 100)
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.ListCTs(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestWaitForTask_EmptyUPID(t *testing.T) {
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not make any requests for empty UPID")
	}))

	err := client.WaitForTask(context.Background(), "")
	if err != nil {
		t.Fatalf("WaitForTask('') error: %v", err)
	}
}

func TestWaitForTask_FailedTask(t *testing.T) {
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(proxmoxResponse(TaskStatus{
			Status:     "stopped",
			ExitStatus: "clone failed: disk error",
		}))
	}))

	err := client.WaitForTask(context.Background(), "UPID:pve:test")
	if err == nil {
		t.Fatal("expected error for failed task")
	}
	if !strings.Contains(err.Error(), "task failed") {
		t.Errorf("error = %q, want containing 'task failed'", err.Error())
	}
}

func TestSetCTConfig(t *testing.T) {
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/nodes/pve/lxc/100/config") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.FormValue("cores") != "4" {
			t.Errorf("cores = %q, want 4", r.FormValue("cores"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(proxmoxResponse(nil))
	}))

	params := map[string][]string{
		"cores": {"4"},
	}
	err := client.SetCTConfig(context.Background(), 100, params)
	if err != nil {
		t.Fatalf("SetCTConfig() error: %v", err)
	}
}

func TestClient_AuthorizationHeader(t *testing.T) {
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		expected := "PVEAPIToken=test@pam!testtoken=test-secret"
		if auth != expected {
			t.Errorf("Authorization = %q, want %q", auth, expected)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(proxmoxResponse([]CTListEntry{}))
	}))

	_, _ = client.ListCTs(context.Background())
}
