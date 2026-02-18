package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleListVMs(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)

	// With an empty registry (no connected hosts), ListVMs returns empty.
	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/vms", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := parseJSONResponse(rr)
	count, ok := resp["count"].(float64)
	if !ok {
		t.Fatalf("expected count field, got %v", resp)
	}
	if count != 0 {
		t.Fatalf("expected count=0 (no connected hosts), got %v", count)
	}
	vms, ok := resp["vms"]
	if !ok {
		t.Fatal("expected vms field in response")
	}
	// vms should be nil or empty slice serialized as null or []
	if vms != nil {
		vmList, ok := vms.([]any)
		if ok && len(vmList) != 0 {
			t.Fatalf("expected empty vms list, got %v", vms)
		}
	}
}
