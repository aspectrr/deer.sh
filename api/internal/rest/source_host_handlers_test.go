package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aspectrr/fluid.sh/api/internal/store"
)

func TestHandleListSourceHosts(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.ListSourceHostsByOrgFn = func(_ context.Context, orgID string) ([]*store.SourceHost, error) {
		if orgID != testOrg.ID {
			t.Fatalf("unexpected orgID: %s", orgID)
		}
		return []*store.SourceHost{
			{ID: "sh-1", OrgID: testOrg.ID, Name: "host-1", Hostname: "192.168.1.10", Type: "libvirt"},
			{ID: "sh-2", OrgID: testOrg.ID, Name: "host-2", Hostname: "192.168.1.11", Type: "proxmox"},
		}, nil
	}

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/source-hosts", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := parseJSONResponse(rr)
	if resp["count"] != float64(2) {
		t.Fatalf("expected count=2, got %v", resp["count"])
	}
	hosts, ok := resp["source_hosts"].([]any)
	if !ok || len(hosts) != 2 {
		t.Fatalf("expected 2 source hosts, got %v", resp["source_hosts"])
	}
}

func TestHandleDeleteSourceHost(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	deleted := false
	ms.GetSourceHostFn = func(_ context.Context, id string) (*store.SourceHost, error) {
		return &store.SourceHost{ID: id, OrgID: testOrg.ID, Name: "host-1", Hostname: "192.168.1.10", Type: "libvirt"}, nil
	}
	ms.DeleteSourceHostFn = func(_ context.Context, id string) error {
		if id == "sh-1" {
			deleted = true
		}
		return nil
	}

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "DELETE", "/v1/orgs/test-org/source-hosts/sh-1", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !deleted {
		t.Fatal("expected DeleteSourceHost to be called with id sh-1")
	}
	resp := parseJSONResponse(rr)
	if resp["deleted"] != true {
		t.Fatalf("expected deleted=true, got %v", resp["deleted"])
	}
}
