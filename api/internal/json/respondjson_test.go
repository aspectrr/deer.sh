package json

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRespondJSON_ContentType(t *testing.T) {
	w := httptest.NewRecorder()
	err := RespondJSON(w, http.StatusOK, map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type 'application/json; charset=utf-8', got %q", ct)
	}
}

func TestRespondJSON_StatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	err := RespondJSON(w, http.StatusCreated, map[string]string{"id": "123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}
}

func TestRespondJSON_ValidBody(t *testing.T) {
	data := map[string]any{
		"name":  "test",
		"count": float64(42),
	}

	w := httptest.NewRecorder()
	err := RespondJSON(w, http.StatusOK, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if result["name"] != "test" {
		t.Errorf("expected name 'test', got %v", result["name"])
	}
	if result["count"] != float64(42) {
		t.Errorf("expected count 42, got %v", result["count"])
	}
}

func TestRespondJSON_HTMLCharsNotEscaped(t *testing.T) {
	data := map[string]string{
		"html": "<b>bold</b> & \"quoted\"",
	}

	w := httptest.NewRecorder()
	err := RespondJSON(w, http.StatusOK, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := w.Body.String()
	// With SetEscapeHTML(false), the raw chars should appear unescaped
	if strings.Contains(body, `\u003c`) {
		t.Errorf("expected HTML chars not to be escaped, but found \\u003c in body: %s", body)
	}
	if strings.Contains(body, `\u0026`) {
		t.Errorf("expected HTML chars not to be escaped, but found \\u0026 in body: %s", body)
	}
	if !strings.Contains(body, "<b>") {
		t.Errorf("expected literal <b> in body, got: %s", body)
	}
	if !strings.Contains(body, "&") {
		t.Errorf("expected literal & in body, got: %s", body)
	}
}

func TestRespondJSON_XContentTypeOptions(t *testing.T) {
	w := httptest.NewRecorder()
	err := RespondJSON(w, http.StatusOK, "ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	xcto := w.Header().Get("X-Content-Type-Options")
	if xcto != "nosniff" {
		t.Errorf("expected X-Content-Type-Options 'nosniff', got %q", xcto)
	}
}
