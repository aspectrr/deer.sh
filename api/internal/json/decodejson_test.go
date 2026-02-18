package json

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSON_Success(t *testing.T) {
	body := `{"name":"alice","age":30}`
	r := httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")

	var dst struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	err := DecodeJSON(context.Background(), r, &dst)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if dst.Name != "alice" {
		t.Errorf("expected name alice, got %s", dst.Name)
	}
	if dst.Age != 30 {
		t.Errorf("expected age 30, got %d", dst.Age)
	}
}

func TestDecodeJSON_InvalidJSON(t *testing.T) {
	body := `{not valid json}`
	r := httptest.NewRequest("POST", "/", bytes.NewBufferString(body))

	var dst struct {
		Name string `json:"name"`
	}

	err := DecodeJSON(context.Background(), r, &dst)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "invalid json") {
		t.Errorf("expected error to contain 'invalid json', got %v", err)
	}
}

func TestDecodeJSON_UnknownFields(t *testing.T) {
	body := `{"name":"alice","unknown_field":"value"}`
	r := httptest.NewRequest("POST", "/", bytes.NewBufferString(body))

	var dst struct {
		Name string `json:"name"`
	}

	err := DecodeJSON(context.Background(), r, &dst)
	if err == nil {
		t.Fatal("expected error for unknown fields, got nil")
	}
	if !strings.Contains(err.Error(), "invalid json") {
		t.Errorf("expected error to contain 'invalid json', got %v", err)
	}
}

func TestDecodeJSON_BodyTooLarge(t *testing.T) {
	// 1 MiB = 1048576 bytes; create a body slightly larger
	large := strings.Repeat("x", 1<<20+100)
	body := `{"name":"` + large + `"}`
	r := httptest.NewRequest("POST", "/", bytes.NewBufferString(body))

	var dst struct {
		Name string `json:"name"`
	}

	err := DecodeJSON(context.Background(), r, &dst)
	if err == nil {
		t.Fatal("expected error for body too large, got nil")
	}
}

func TestDecodeJSON_TrailingData(t *testing.T) {
	body := `{"name":"alice"}{"name":"bob"}`
	r := httptest.NewRequest("POST", "/", bytes.NewBufferString(body))

	var dst struct {
		Name string `json:"name"`
	}

	err := DecodeJSON(context.Background(), r, &dst)
	if err == nil {
		t.Fatal("expected error for trailing data, got nil")
	}
	if !strings.Contains(err.Error(), "trailing data") {
		t.Errorf("expected error to contain 'trailing data', got %v", err)
	}
}

func TestDecodeJSON_EmptyBody(t *testing.T) {
	r := httptest.NewRequest("POST", "/", bytes.NewBufferString(""))

	var dst struct {
		Name string `json:"name"`
	}

	err := DecodeJSON(context.Background(), r, &dst)
	if err == nil {
		t.Fatal("expected error for empty body, got nil")
	}
}
