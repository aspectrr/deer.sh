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

func TestRespondError_BodyContainsGenericMessage(t *testing.T) {
	w := httptest.NewRecorder()
	RespondError(w, http.StatusNotFound, errors.New("item not found"))

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}

	// Should return generic HTTP status text, not the internal error
	if resp.Error != "Not Found" {
		t.Errorf("expected error 'Not Found', got %q", resp.Error)
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
		status      int
		internalMsg string
		expectedMsg string
	}{
		{http.StatusBadRequest, "bad request", "Bad Request"},
		{http.StatusUnauthorized, "unauthorized", "Unauthorized"},
		{http.StatusForbidden, "forbidden", "Forbidden"},
		{http.StatusNotFound, "not found", "Not Found"},
		{http.StatusInternalServerError, "server error", "Internal Server Error"},
	}

	for _, tt := range tests {
		w := httptest.NewRecorder()
		RespondError(w, tt.status, errors.New(tt.internalMsg))

		if w.Code != tt.status {
			t.Errorf("status %d: expected %d, got %d", tt.status, tt.status, w.Code)
		}

		var resp ErrorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("status %d: response body is not valid JSON: %v", tt.status, err)
		}
		// Should return generic HTTP status text, not the internal error
		if resp.Error != tt.expectedMsg {
			t.Errorf("status %d: expected error %q, got %q", tt.status, tt.expectedMsg, resp.Error)
		}
		if resp.Code != tt.status {
			t.Errorf("status %d: expected code %d in body, got %d", tt.status, tt.status, resp.Code)
		}
	}
}

func TestRespondError_NilError(t *testing.T) {
	w := httptest.NewRecorder()
	RespondError(w, http.StatusBadRequest, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if resp.Error != "Bad Request" {
		t.Errorf("expected error 'Bad Request', got %q", resp.Error)
	}
}

func TestRespondErrorMsg(t *testing.T) {
	w := httptest.NewRecorder()
	RespondErrorMsg(w, http.StatusBadRequest, "email is required", errors.New("validation: email field empty"))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	// RespondErrorMsg returns the user-facing message, not the internal error
	if resp.Error != "email is required" {
		t.Errorf("expected error 'email is required', got %q", resp.Error)
	}
}
