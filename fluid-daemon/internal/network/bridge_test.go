package network

import (
	"context"
	"log/slog"
	"testing"
)

func TestNetworkManager_ResolveBridge_Default(t *testing.T) {
	nm := NewNetworkManager("br0", nil, "dnsmasq", slog.Default())

	bridge, err := nm.ResolveBridge(context.Background(), "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if bridge != "br0" {
		t.Fatalf("expected %q, got %q", "br0", bridge)
	}
}

func TestNetworkManager_ResolveBridge_FromMap(t *testing.T) {
	bridgeMap := map[string]string{
		"default": "virbr0",
		"mgmt":    "br-mgmt",
	}
	nm := NewNetworkManager("br0", bridgeMap, "dnsmasq", slog.Default())

	tests := []struct {
		name             string
		requestedNetwork string
		wantBridge       string
	}{
		{"default network", "default", "virbr0"},
		{"mgmt network", "mgmt", "br-mgmt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bridge, err := nm.ResolveBridge(context.Background(), "", tt.requestedNetwork)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if bridge != tt.wantBridge {
				t.Fatalf("expected %q, got %q", tt.wantBridge, bridge)
			}
		})
	}
}

func TestNetworkManager_ResolveBridge_Explicit(t *testing.T) {
	nm := NewNetworkManager("br0", nil, "dnsmasq", slog.Default())

	tests := []struct {
		name             string
		requestedNetwork string
	}{
		{"virbr prefix", "virbr5"},
		{"br prefix", "br-custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bridge, err := nm.ResolveBridge(context.Background(), "", tt.requestedNetwork)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if bridge != tt.requestedNetwork {
				t.Fatalf("expected %q, got %q", tt.requestedNetwork, bridge)
			}
		})
	}
}

func TestNetworkManager_ResolveBridge_UnknownNetwork(t *testing.T) {
	bridgeMap := map[string]string{
		"default": "virbr0",
	}
	nm := NewNetworkManager("br0", bridgeMap, "dnsmasq", slog.Default())

	_, err := nm.ResolveBridge(context.Background(), "", "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown network, got nil")
	}

	want := `unknown network "nonexistent": not found in bridge_map`
	if err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}

func TestNetworkManager_DHCPMode(t *testing.T) {
	tests := []struct {
		mode string
	}{
		{"dnsmasq"},
		{"static"},
		{""},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			nm := NewNetworkManager("br0", nil, tt.mode, slog.Default())
			if got := nm.DHCPMode(); got != tt.mode {
				t.Fatalf("expected %q, got %q", tt.mode, got)
			}
		})
	}
}
