package error

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRespondError_StatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	RespondError(w, http.StatusBadRequest, errors.New("bad input"))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestRespondError_BodyContainsErrorMessage(t *testing.T) {
	w := httptest.NewRecorder()
	RespondError(w, http.StatusNotFound, errors.New("item not found"))

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}

	if resp.Error != "item not found" {
		t.Errorf("expected error 'item not found', got %q", resp.Error)
	}
	if resp.Code != http.StatusNotFound {
		t.Errorf("expected code 404, got %d", resp.Code)
	}
}

func TestRespondError_ContentType(t *testing.T) {
	w := httptest.NewRecorder()
	RespondError(w, http.StatusInternalServerError, errors.New("internal"))

	ct := w.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type 'application/json; charset=utf-8', got %q", ct)
	}
}

func TestRespondError_DetailsOmittedWhenEmpty(t *testing.T) {
	w := httptest.NewRecorder()
	RespondError(w, http.StatusForbidden, errors.New("forbidden"))

	var raw map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}

	if _, ok := raw["details"]; ok {
		t.Error("expected 'details' field to be omitted when empty")
	}
}

func TestRespondError_MultipleStatuses(t *testing.T) {
	tests := []struct {
		status int
		msg    string
	}{
		{http.StatusBadRequest, "bad request"},
		{http.StatusUnauthorized, "unauthorized"},
		{http.StatusForbidden, "forbidden"},
		{http.StatusNotFound, "not found"},
		{http.StatusInternalServerError, "server error"},
	}

	for _, tt := range tests {
		w := httptest.NewRecorder()
		RespondError(w, tt.status, errors.New(tt.msg))

		if w.Code != tt.status {
			t.Errorf("status %d: expected %d, got %d", tt.status, tt.status, w.Code)
		}

		var resp ErrorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("status %d: response body is not valid JSON: %v", tt.status, err)
		}
		if resp.Error != tt.msg {
			t.Errorf("status %d: expected error %q, got %q", tt.status, tt.msg, resp.Error)
		}
		if resp.Code != tt.status {
			t.Errorf("status %d: expected code %d in body, got %d", tt.status, tt.status, resp.Code)
		}
	}
}
