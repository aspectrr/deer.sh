package rest

// Playbook handler tests - commented out, not yet ready for integration.

/*
import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aspectrr/fluid.sh/api/internal/store"
)

func TestHandleCreatePlaybook(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)

	var created *store.Playbook
	ms.CreatePlaybookFn = func(_ context.Context, pb *store.Playbook) error {
		created = pb
		return nil
	}

	s := newTestServer(ms, nil)
	body := httptest.NewRequest("POST", "/v1/orgs/test-org/playbooks",
		bytes.NewBufferString(`{"name":"Deploy App","description":"Deploy the application"}`))
	body.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/playbooks", body)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if created == nil {
		t.Fatal("expected CreatePlaybook to be called")
	}
	if created.Name != "Deploy App" {
		t.Fatalf("expected name 'Deploy App', got %q", created.Name)
	}
	if created.OrgID != testOrg.ID {
		t.Fatalf("expected orgID %q, got %q", testOrg.ID, created.OrgID)
	}
}

func TestHandleListPlaybooks(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.ListPlaybooksByOrgFn = func(_ context.Context, orgID string) ([]*store.Playbook, error) {
		return []*store.Playbook{
			{ID: "pb-1", OrgID: testOrg.ID, Name: "Playbook 1"},
			{ID: "pb-2", OrgID: testOrg.ID, Name: "Playbook 2"},
		}, nil
	}

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/playbooks", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := parseJSONResponse(rr)
	if resp["count"] != float64(2) {
		t.Fatalf("expected count=2, got %v", resp["count"])
	}
}

func TestHandleGetPlaybook(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetPlaybookFn = func(_ context.Context, id string) (*store.Playbook, error) {
			if id == "pb-1" {
				return &store.Playbook{ID: "pb-1", OrgID: testOrg.ID, Name: "Deploy"}, nil
			}
			return nil, store.ErrNotFound
		}
		ms.ListPlaybookTasksFn = func(_ context.Context, pbID string) ([]*store.PlaybookTask, error) {
			return []*store.PlaybookTask{
				{ID: "task-1", PlaybookID: pbID, Name: "Install"},
			}, nil
		}

		s := newTestServer(ms, nil)
		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/playbooks/pb-1", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		resp := parseJSONResponse(rr)
		pb, ok := resp["playbook"].(map[string]any)
		if !ok {
			t.Fatalf("expected playbook object, got %v", resp["playbook"])
		}
		if pb["id"] != "pb-1" {
			t.Fatalf("expected playbook id=pb-1, got %v", pb["id"])
		}
		tasks, ok := resp["tasks"].([]any)
		if !ok || len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %v", resp["tasks"])
		}
	})

	t.Run("not_found", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetPlaybookFn = func(_ context.Context, id string) (*store.Playbook, error) {
			return nil, store.ErrNotFound
		}

		s := newTestServer(ms, nil)
		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/playbooks/nonexistent", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("wrong_org", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetPlaybookFn = func(_ context.Context, id string) (*store.Playbook, error) {
			return &store.Playbook{ID: "pb-1", OrgID: "ORG-other", Name: "Other"}, nil
		}

		s := newTestServer(ms, nil)
		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/playbooks/pb-1", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestHandleUpdatePlaybook(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.GetPlaybookFn = func(_ context.Context, id string) (*store.Playbook, error) {
		if id == "pb-1" {
			return &store.Playbook{ID: "pb-1", OrgID: testOrg.ID, Name: "Old Name", Description: "Old Desc"}, nil
		}
		return nil, store.ErrNotFound
	}
	var updated *store.Playbook
	ms.UpdatePlaybookFn = func(_ context.Context, pb *store.Playbook) error {
		updated = pb
		return nil
	}

	s := newTestServer(ms, nil)
	body := httptest.NewRequest("PATCH", "/v1/orgs/test-org/playbooks/pb-1",
		bytes.NewBufferString(`{"name":"New Name"}`))
	body.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "PATCH", "/v1/orgs/test-org/playbooks/pb-1", body)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if updated == nil {
		t.Fatal("expected UpdatePlaybook to be called")
	}
	if updated.Name != "New Name" {
		t.Fatalf("expected name 'New Name', got %q", updated.Name)
	}
	// Description should remain unchanged
	if updated.Description != "Old Desc" {
		t.Fatalf("expected description 'Old Desc', got %q", updated.Description)
	}
}

func TestHandleDeletePlaybook(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.GetPlaybookFn = func(_ context.Context, id string) (*store.Playbook, error) {
		if id == "pb-1" {
			return &store.Playbook{ID: "pb-1", OrgID: testOrg.ID}, nil
		}
		return nil, store.ErrNotFound
	}
	deleted := false
	ms.DeletePlaybookFn = func(_ context.Context, id string) error {
		if id == "pb-1" {
			deleted = true
		}
		return nil
	}

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "DELETE", "/v1/orgs/test-org/playbooks/pb-1", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !deleted {
		t.Fatal("expected DeletePlaybook to be called")
	}
	resp := parseJSONResponse(rr)
	if resp["deleted"] != true {
		t.Fatalf("expected deleted=true, got %v", resp["deleted"])
	}
}

func TestHandleCreatePlaybookTask(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.GetPlaybookFn = func(_ context.Context, id string) (*store.Playbook, error) {
		if id == "pb-1" {
			return &store.Playbook{ID: "pb-1", OrgID: testOrg.ID}, nil
		}
		return nil, store.ErrNotFound
	}
	ms.ListPlaybookTasksFn = func(_ context.Context, pbID string) ([]*store.PlaybookTask, error) {
		return nil, nil
	}
	var created *store.PlaybookTask
	ms.CreatePlaybookTaskFn = func(_ context.Context, task *store.PlaybookTask) error {
		created = task
		return nil
	}

	s := newTestServer(ms, nil)
	body := httptest.NewRequest("POST", "/v1/orgs/test-org/playbooks/pb-1/tasks",
		bytes.NewBufferString(`{"name":"Install nginx","module":"apt","params":{"name":"nginx","state":"present"}}`))
	body.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/playbooks/pb-1/tasks", body)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if created == nil {
		t.Fatal("expected CreatePlaybookTask to be called")
	}
	if created.Name != "Install nginx" {
		t.Fatalf("expected task name 'Install nginx', got %q", created.Name)
	}
	if created.Module != "apt" {
		t.Fatalf("expected module 'apt', got %q", created.Module)
	}
	if created.PlaybookID != "pb-1" {
		t.Fatalf("expected playbookID 'pb-1', got %q", created.PlaybookID)
	}
}

func TestHandleListPlaybookTasks(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.GetPlaybookFn = func(_ context.Context, id string) (*store.Playbook, error) {
		if id == "pb-1" {
			return &store.Playbook{ID: "pb-1", OrgID: testOrg.ID}, nil
		}
		return nil, store.ErrNotFound
	}
	ms.ListPlaybookTasksFn = func(_ context.Context, pbID string) ([]*store.PlaybookTask, error) {
		return []*store.PlaybookTask{
			{ID: "task-1", PlaybookID: pbID, Name: "Task 1", Module: "shell"},
			{ID: "task-2", PlaybookID: pbID, Name: "Task 2", Module: "copy"},
		}, nil
	}

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/playbooks/pb-1/tasks", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := parseJSONResponse(rr)
	if resp["count"] != float64(2) {
		t.Fatalf("expected count=2, got %v", resp["count"])
	}
}

func TestHandleUpdatePlaybookTask(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.GetPlaybookFn = func(_ context.Context, id string) (*store.Playbook, error) {
		if id == "pb-1" {
			return &store.Playbook{ID: "pb-1", OrgID: testOrg.ID}, nil
		}
		return nil, store.ErrNotFound
	}
	ms.GetPlaybookTaskFn = func(_ context.Context, id string) (*store.PlaybookTask, error) {
		if id == "task-1" {
			return &store.PlaybookTask{ID: "task-1", PlaybookID: "pb-1", Name: "Old Task", Module: "shell", Params: "{}"}, nil
		}
		return nil, store.ErrNotFound
	}
	var updated *store.PlaybookTask
	ms.UpdatePlaybookTaskFn = func(_ context.Context, task *store.PlaybookTask) error {
		updated = task
		return nil
	}

	s := newTestServer(ms, nil)
	body := httptest.NewRequest("PATCH", "/v1/orgs/test-org/playbooks/pb-1/tasks/task-1",
		bytes.NewBufferString(`{"name":"Updated Task"}`))
	body.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "PATCH", "/v1/orgs/test-org/playbooks/pb-1/tasks/task-1", body)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if updated == nil {
		t.Fatal("expected UpdatePlaybookTask to be called")
	}
	if updated.Name != "Updated Task" {
		t.Fatalf("expected name 'Updated Task', got %q", updated.Name)
	}
	// Module should remain unchanged
	if updated.Module != "shell" {
		t.Fatalf("expected module 'shell', got %q", updated.Module)
	}
}

func TestHandleDeletePlaybookTask(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.GetPlaybookFn = func(_ context.Context, id string) (*store.Playbook, error) {
		if id == "pb-1" {
			return &store.Playbook{ID: "pb-1", OrgID: testOrg.ID}, nil
		}
		return nil, store.ErrNotFound
	}
	ms.GetPlaybookTaskFn = func(_ context.Context, id string) (*store.PlaybookTask, error) {
		if id == "task-1" {
			return &store.PlaybookTask{ID: "task-1", PlaybookID: "pb-1"}, nil
		}
		return nil, store.ErrNotFound
	}
	deleted := false
	ms.DeletePlaybookTaskFn = func(_ context.Context, id string) error {
		if id == "task-1" {
			deleted = true
		}
		return nil
	}

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "DELETE", "/v1/orgs/test-org/playbooks/pb-1/tasks/task-1", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !deleted {
		t.Fatal("expected DeletePlaybookTask to be called")
	}
	resp := parseJSONResponse(rr)
	if resp["deleted"] != true {
		t.Fatalf("expected deleted=true, got %v", resp["deleted"])
	}
}
*/
