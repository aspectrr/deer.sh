package docsprogress

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterSession(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/docs-progress/register" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	RegisterSession(srv.URL, "ABC123")

	if gotBody["session_code"] != "ABC123" {
		t.Errorf("session_code = %q, want %q", gotBody["session_code"], "ABC123")
	}
}

func TestReportCompletion(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/docs-progress/complete" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ReportCompletion(srv.URL, "XYZ789", 3)

	if gotBody["session_code"] != "XYZ789" {
		t.Errorf("session_code = %v, want %q", gotBody["session_code"], "XYZ789")
	}
	if gotBody["step_index"] != float64(3) {
		t.Errorf("step_index = %v, want 3", gotBody["step_index"])
	}
}

func TestRegisterSession_InvalidURL(t *testing.T) {
	// Should not panic on invalid URL
	RegisterSession("://bad-url", "CODE")
}

func TestReportCompletion_InvalidURL(t *testing.T) {
	// Should not panic on invalid URL
	ReportCompletion("://bad-url", "CODE", 1)
}
