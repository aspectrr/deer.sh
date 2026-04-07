package lxc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testResolverClient(t *testing.T, cts []CTListEntry) (*CTResolver, *httptest.Server) {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d, _ := json.Marshal(cts)
		resp := struct {
			Data json.RawMessage `json:"data"`
		}{Data: d}
		b, _ := json.Marshal(resp)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	}))
	t.Cleanup(srv.Close)

	cfg := Config{
		Host:      srv.URL,
		TokenID:   "test@pam!tok",
		Secret:    "secret",
		Node:      "pve",
		VerifySSL: false,
		Timeout:   5 * time.Second,
	}
	client := NewClient(cfg, nil)
	client.httpClient = srv.Client()
	client.httpClient.Timeout = 5 * time.Second

	return NewCTResolver(client), srv
}

func TestCTResolver_ResolveVMID(t *testing.T) {
	cts := []CTListEntry{
		{VMID: 100, Name: "web-server", Status: "running"},
		{VMID: 200, Name: "db-server", Status: "stopped"},
	}

	resolver, _ := testResolverClient(t, cts)

	vmid, err := resolver.ResolveVMID(context.Background(), "web-server")
	if err != nil {
		t.Fatalf("ResolveVMID() error: %v", err)
	}
	if vmid != 100 {
		t.Errorf("VMID = %d, want 100", vmid)
	}
}

func TestCTResolver_ResolveName(t *testing.T) {
	cts := []CTListEntry{
		{VMID: 100, Name: "web-server", Status: "running"},
	}

	resolver, _ := testResolverClient(t, cts)

	name, err := resolver.ResolveName(context.Background(), 100)
	if err != nil {
		t.Fatalf("ResolveName() error: %v", err)
	}
	if name != "web-server" {
		t.Errorf("Name = %q, want web-server", name)
	}
}

func TestCTResolver_NotFound(t *testing.T) {
	cts := []CTListEntry{
		{VMID: 100, Name: "web-server", Status: "running"},
	}

	resolver, _ := testResolverClient(t, cts)

	_, err := resolver.ResolveVMID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent CT name")
	}
}

func TestCTResolver_CacheHit(t *testing.T) {
	callCount := 0
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		cts := []CTListEntry{{VMID: 100, Name: "cached-ct"}}
		d, _ := json.Marshal(cts)
		resp := struct {
			Data json.RawMessage `json:"data"`
		}{Data: d}
		b, _ := json.Marshal(resp)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	cfg := Config{
		Host: srv.URL, TokenID: "t@p!t", Secret: "s", Node: "pve",
		VerifySSL: false, Timeout: 5 * time.Second,
	}
	client := NewClient(cfg, nil)
	client.httpClient = srv.Client()
	client.httpClient.Timeout = 5 * time.Second

	resolver := NewCTResolver(client)

	ctx := context.Background()

	// First call triggers refresh
	_, _ = resolver.ResolveVMID(ctx, "cached-ct")
	firstCount := callCount

	// Second call should use cache (no additional HTTP call)
	_, _ = resolver.ResolveVMID(ctx, "cached-ct")
	if callCount != firstCount {
		t.Errorf("expected cache hit, got %d calls (first had %d)", callCount, firstCount)
	}
}

func TestCTResolver_Refresh(t *testing.T) {
	cts := []CTListEntry{
		{VMID: 100, Name: "alpha"},
		{VMID: 200, Name: "beta"},
	}

	resolver, _ := testResolverClient(t, cts)

	err := resolver.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh() error: %v", err)
	}

	// Both should be cached now
	vmid, err := resolver.ResolveVMID(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("ResolveVMID(alpha) error: %v", err)
	}
	if vmid != 100 {
		t.Errorf("alpha VMID = %d, want 100", vmid)
	}

	name, err := resolver.ResolveName(context.Background(), 200)
	if err != nil {
		t.Fatalf("ResolveName(200) error: %v", err)
	}
	if name != "beta" {
		t.Errorf("200 name = %q, want beta", name)
	}
}
