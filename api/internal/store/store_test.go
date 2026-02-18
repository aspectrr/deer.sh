package store

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// StringSlice
// ---------------------------------------------------------------------------

func TestStringSlice_Value_Nil(t *testing.T) {
	var s StringSlice
	v, err := s.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "[]" {
		t.Errorf("expected '[]', got %v", v)
	}
}

func TestStringSlice_Value_NonNil(t *testing.T) {
	s := StringSlice{"a", "b", "c"}
	v, err := s.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	str, ok := v.(string)
	if !ok {
		t.Fatalf("expected string, got %T", v)
	}

	var result []string
	if err := json.Unmarshal([]byte(str), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}
	if result[0] != "a" || result[1] != "b" || result[2] != "c" {
		t.Errorf("unexpected values: %v", result)
	}
}

func TestStringSlice_Value_Empty(t *testing.T) {
	s := StringSlice{}
	v, err := s.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	str, ok := v.(string)
	if !ok {
		t.Fatalf("expected string, got %T", v)
	}
	if str != "[]" {
		t.Errorf("expected '[]', got %q", str)
	}
}

func TestStringSlice_Scan_String(t *testing.T) {
	var s StringSlice
	err := s.Scan(`["x","y"]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s) != 2 {
		t.Fatalf("expected 2 items, got %d", len(s))
	}
	if s[0] != "x" || s[1] != "y" {
		t.Errorf("unexpected values: %v", s)
	}
}

func TestStringSlice_Scan_Bytes(t *testing.T) {
	var s StringSlice
	err := s.Scan([]byte(`["p","q"]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s) != 2 {
		t.Fatalf("expected 2 items, got %d", len(s))
	}
	if s[0] != "p" || s[1] != "q" {
		t.Errorf("unexpected values: %v", s)
	}
}

func TestStringSlice_Scan_Nil(t *testing.T) {
	var s StringSlice
	err := s.Scan(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(s) != 0 {
		t.Errorf("expected empty slice, got %v", s)
	}
}

func TestStringSlice_Scan_InvalidType(t *testing.T) {
	var s StringSlice
	err := s.Scan(12345)
	if err == nil {
		t.Fatal("expected error for invalid type, got nil")
	}
}

func TestStringSlice_Scan_InvalidJSON(t *testing.T) {
	var s StringSlice
	err := s.Scan(`{not valid}`)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// ---------------------------------------------------------------------------
// SourceVMSlice
// ---------------------------------------------------------------------------

func TestSourceVMSlice_Value_Nil(t *testing.T) {
	var s SourceVMSlice
	v, err := s.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "[]" {
		t.Errorf("expected '[]', got %v", v)
	}
}

func TestSourceVMSlice_Value_NonNil(t *testing.T) {
	s := SourceVMSlice{
		{Name: "vm1", State: "running", IPAddress: "10.0.0.1", Prepared: true},
		{Name: "vm2", State: "stopped", IPAddress: "", Prepared: false},
	}
	v, err := s.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	str, ok := v.(string)
	if !ok {
		t.Fatalf("expected string, got %T", v)
	}

	var result []SourceVMJSON
	if err := json.Unmarshal([]byte(str), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
	if result[0].Name != "vm1" || !result[0].Prepared {
		t.Errorf("unexpected first item: %+v", result[0])
	}
}

func TestSourceVMSlice_Scan_String(t *testing.T) {
	var s SourceVMSlice
	err := s.Scan(`[{"name":"vm1","state":"running","ip_address":"10.0.0.1","prepared":true}]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s) != 1 {
		t.Fatalf("expected 1 item, got %d", len(s))
	}
	if s[0].Name != "vm1" {
		t.Errorf("expected name 'vm1', got %q", s[0].Name)
	}
	if !s[0].Prepared {
		t.Error("expected prepared to be true")
	}
}

func TestSourceVMSlice_Scan_Bytes(t *testing.T) {
	var s SourceVMSlice
	err := s.Scan([]byte(`[{"name":"vm2","state":"stopped","ip_address":"","prepared":false}]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s) != 1 {
		t.Fatalf("expected 1 item, got %d", len(s))
	}
	if s[0].Name != "vm2" {
		t.Errorf("expected name 'vm2', got %q", s[0].Name)
	}
}

func TestSourceVMSlice_Scan_Nil(t *testing.T) {
	var s SourceVMSlice
	err := s.Scan(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(s) != 0 {
		t.Errorf("expected empty slice, got %v", s)
	}
}

func TestSourceVMSlice_Scan_InvalidType(t *testing.T) {
	var s SourceVMSlice
	err := s.Scan(12345)
	if err == nil {
		t.Fatal("expected error for invalid type, got nil")
	}
}

// ---------------------------------------------------------------------------
// BridgeSlice
// ---------------------------------------------------------------------------

func TestBridgeSlice_Value_Nil(t *testing.T) {
	var s BridgeSlice
	v, err := s.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "[]" {
		t.Errorf("expected '[]', got %v", v)
	}
}

func TestBridgeSlice_Value_NonNil(t *testing.T) {
	s := BridgeSlice{
		{Name: "br0", Subnet: "10.0.0.0/24"},
		{Name: "br1", Subnet: "192.168.1.0/24"},
	}
	v, err := s.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	str, ok := v.(string)
	if !ok {
		t.Fatalf("expected string, got %T", v)
	}

	var result []BridgeJSON
	if err := json.Unmarshal([]byte(str), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
	if result[0].Name != "br0" || result[0].Subnet != "10.0.0.0/24" {
		t.Errorf("unexpected first item: %+v", result[0])
	}
}

func TestBridgeSlice_Scan_String(t *testing.T) {
	var s BridgeSlice
	err := s.Scan(`[{"name":"br0","subnet":"10.0.0.0/24"}]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s) != 1 {
		t.Fatalf("expected 1 item, got %d", len(s))
	}
	if s[0].Name != "br0" {
		t.Errorf("expected name 'br0', got %q", s[0].Name)
	}
	if s[0].Subnet != "10.0.0.0/24" {
		t.Errorf("expected subnet '10.0.0.0/24', got %q", s[0].Subnet)
	}
}

func TestBridgeSlice_Scan_Bytes(t *testing.T) {
	var s BridgeSlice
	err := s.Scan([]byte(`[{"name":"br1","subnet":"192.168.0.0/16"}]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s) != 1 {
		t.Fatalf("expected 1 item, got %d", len(s))
	}
	if s[0].Name != "br1" {
		t.Errorf("expected name 'br1', got %q", s[0].Name)
	}
}

func TestBridgeSlice_Scan_Nil(t *testing.T) {
	var s BridgeSlice
	err := s.Scan(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(s) != 0 {
		t.Errorf("expected empty slice, got %v", s)
	}
}

func TestBridgeSlice_Scan_InvalidType(t *testing.T) {
	var s BridgeSlice
	err := s.Scan(12345)
	if err == nil {
		t.Fatal("expected error for invalid type, got nil")
	}
}
