package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aspectrr/fluid.sh/api/internal/store"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"
)

func TestHandleListSandboxes(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.ListSandboxesByOrgFn = func(_ context.Context, orgID string) ([]store.Sandbox, error) {
		return []store.Sandbox{
			{
				ID:    "SBX-1234",
				OrgID: testOrg.ID,
				Name:  "test-sandbox",
				State: store.SandboxStateRunning,
			},
		}, nil
	}
	s := newTestServer(ms, nil)

	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/sandboxes", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := parseJSONResponse(rr)
	sandboxes, ok := body["sandboxes"].([]any)
	if !ok {
		t.Fatal("expected sandboxes array in response")
	}
	if len(sandboxes) != 1 {
		t.Fatalf("expected 1 sandbox, got %d", len(sandboxes))
	}
}

func TestHandleGetSandbox(t *testing.T) {
	testSandbox := &store.Sandbox{
		ID:        "SBX-get1234",
		OrgID:     testOrg.ID,
		HostID:    "HOST-1",
		Name:      "get-sandbox",
		State:     store.SandboxStateRunning,
		IPAddress: "10.0.0.5",
	}

	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			if sandboxID == testSandbox.ID {
				return testSandbox, nil
			}
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/sandboxes/"+testSandbox.ID, nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["id"] != testSandbox.ID {
			t.Fatalf("expected id %s, got %v", testSandbox.ID, body["id"])
		}
	})

	t.Run("not found", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/sandboxes/SBX-nonexistent", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		wrongOrgSandbox := &store.Sandbox{
			ID:    "SBX-wrongorg",
			OrgID: "ORG-different",
			Name:  "wrong-org-sandbox",
			State: store.SandboxStateRunning,
		}
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			if sandboxID == wrongOrgSandbox.ID {
				return wrongOrgSandbox, nil
			}
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/sandboxes/"+wrongOrgSandbox.ID, nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestHandleListCommands(t *testing.T) {
	testSandbox := &store.Sandbox{
		ID:    "SBX-cmds1234",
		OrgID: testOrg.ID,
		Name:  "cmds-sandbox",
		State: store.SandboxStateRunning,
	}

	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
		if sandboxID == testSandbox.ID {
			return testSandbox, nil
		}
		return nil, store.ErrNotFound
	}
	ms.ListSandboxCommandsFn = func(_ context.Context, sandboxID string) ([]store.Command, error) {
		return []store.Command{
			{
				ID:        "CMD-1",
				SandboxID: testSandbox.ID,
				Command:   "ls -la",
				ExitCode:  0,
			},
		}, nil
	}
	s := newTestServer(ms, nil)

	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/sandboxes/"+testSandbox.ID+"/commands", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := parseJSONResponse(rr)
	commands, ok := body["commands"].([]any)
	if !ok {
		t.Fatal("expected commands array in response")
	}
	if len(commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(commands))
	}
}

func TestHandleDestroySandbox(t *testing.T) {
	testSandbox := &store.Sandbox{
		ID:     "SBX-destroy1",
		OrgID:  testOrg.ID,
		HostID: "HOST-1",
		Name:   "destroy-sandbox",
		State:  store.SandboxStateRunning,
	}

	t.Run("not_found", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "DELETE", "/v1/orgs/test-org/sandboxes/SBX-nonexistent", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("wrong_org", func(t *testing.T) {
		wrongOrgSandbox := &store.Sandbox{
			ID:     "SBX-wrongorg",
			OrgID:  "ORG-different",
			HostID: "HOST-1",
			Name:   "wrong-org-sandbox",
			State:  store.SandboxStateRunning,
		}
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			if sandboxID == wrongOrgSandbox.ID {
				return wrongOrgSandbox, nil
			}
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "DELETE", "/v1/orgs/test-org/sandboxes/"+wrongOrgSandbox.ID, nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			if sandboxID == testSandbox.ID {
				return testSandbox, nil
			}
			return nil, store.ErrNotFound
		}
		ms.DeleteSandboxFn = func(_ context.Context, sandboxID string) error {
			return nil
		}
		sender := &mockHostSender{
			SendAndWaitFn: func(_ context.Context, hostID string, msg *fluidv1.ControlMessage, timeout time.Duration) (*fluidv1.HostMessage, error) {
				return &fluidv1.HostMessage{
					RequestId: msg.RequestId,
					Payload: &fluidv1.HostMessage_SandboxDestroyed{
						SandboxDestroyed: &fluidv1.SandboxDestroyed{
							SandboxId: testSandbox.ID,
						},
					},
				}, nil
			},
		}
		s := newTestServerWithSender(ms, sender, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "DELETE", "/v1/orgs/test-org/sandboxes/"+testSandbox.ID, nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["destroyed"] != true {
			t.Fatalf("expected destroyed=true, got %v", body["destroyed"])
		}
	})
}

func TestHandleRunCommand(t *testing.T) {
	testSandbox := &store.Sandbox{
		ID:     "SBX-runcmd1",
		OrgID:  testOrg.ID,
		HostID: "HOST-1",
		Name:   "runcmd-sandbox",
		State:  store.SandboxStateRunning,
	}

	t.Run("not_found", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		body := strings.NewReader(`{"command":"ls -la"}`)
		req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/sandboxes/SBX-nonexistent/run",
			httptest.NewRequest("POST", "/v1/orgs/test-org/sandboxes/SBX-nonexistent/run", body))
		req.Header.Set("Content-Type", "application/json")
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("wrong_org", func(t *testing.T) {
		wrongOrgSandbox := &store.Sandbox{
			ID:     "SBX-wrongorg",
			OrgID:  "ORG-different",
			HostID: "HOST-1",
			Name:   "wrong-org-sandbox",
			State:  store.SandboxStateRunning,
		}
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			if sandboxID == wrongOrgSandbox.ID {
				return wrongOrgSandbox, nil
			}
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		body := strings.NewReader(`{"command":"ls -la"}`)
		req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/sandboxes/"+wrongOrgSandbox.ID+"/run",
			httptest.NewRequest("POST", "/v1/orgs/test-org/sandboxes/"+wrongOrgSandbox.ID+"/run", body))
		req.Header.Set("Content-Type", "application/json")
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("empty_command", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			if sandboxID == testSandbox.ID {
				return testSandbox, nil
			}
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		body := strings.NewReader(`{"command":""}`)
		path := "/v1/orgs/test-org/sandboxes/" + testSandbox.ID + "/run"
		req := authenticatedRequest(ms, "POST", path,
			httptest.NewRequest("POST", path, body))
		req.Header.Set("Content-Type", "application/json")
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			if sandboxID == testSandbox.ID {
				return testSandbox, nil
			}
			return nil, store.ErrNotFound
		}
		ms.CreateCommandFn = func(_ context.Context, cmd *store.Command) error {
			return nil
		}
		sender := &mockHostSender{
			SendAndWaitFn: func(_ context.Context, hostID string, msg *fluidv1.ControlMessage, timeout time.Duration) (*fluidv1.HostMessage, error) {
				return &fluidv1.HostMessage{
					RequestId: msg.RequestId,
					Payload: &fluidv1.HostMessage_CommandResult{
						CommandResult: &fluidv1.CommandResult{
							SandboxId:  testSandbox.ID,
							Stdout:     "file1\nfile2\n",
							Stderr:     "",
							ExitCode:   0,
							DurationMs: 50,
						},
					},
				}, nil
			},
		}
		s := newTestServerWithSender(ms, sender, nil)

		rr := httptest.NewRecorder()
		body := strings.NewReader(`{"command":"ls -la"}`)
		path := "/v1/orgs/test-org/sandboxes/" + testSandbox.ID + "/run"
		req := authenticatedRequest(ms, "POST", path,
			httptest.NewRequest("POST", path, body))
		req.Header.Set("Content-Type", "application/json")
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		respBody := parseJSONResponse(rr)
		if respBody["command"] != "ls -la" {
			t.Fatalf("expected command 'ls -la', got %v", respBody["command"])
		}
	})
}

func TestHandleStartSandbox(t *testing.T) {
	testSandbox := &store.Sandbox{
		ID:     "SBX-start1",
		OrgID:  testOrg.ID,
		HostID: "HOST-1",
		Name:   "start-sandbox",
		State:  store.SandboxStateStopped,
	}

	t.Run("not_found", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/sandboxes/SBX-nonexistent/start", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("wrong_org", func(t *testing.T) {
		wrongOrgSandbox := &store.Sandbox{
			ID:     "SBX-wrongorg",
			OrgID:  "ORG-different",
			HostID: "HOST-1",
			Name:   "wrong-org-sandbox",
			State:  store.SandboxStateStopped,
		}
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			if sandboxID == wrongOrgSandbox.ID {
				return wrongOrgSandbox, nil
			}
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/sandboxes/"+wrongOrgSandbox.ID+"/start", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			if sandboxID == testSandbox.ID {
				return testSandbox, nil
			}
			return nil, store.ErrNotFound
		}
		ms.UpdateSandboxFn = func(_ context.Context, sandbox *store.Sandbox) error {
			return nil
		}
		sender := &mockHostSender{
			SendAndWaitFn: func(_ context.Context, hostID string, msg *fluidv1.ControlMessage, timeout time.Duration) (*fluidv1.HostMessage, error) {
				return &fluidv1.HostMessage{
					RequestId: msg.RequestId,
					Payload: &fluidv1.HostMessage_SandboxStarted{
						SandboxStarted: &fluidv1.SandboxStarted{
							SandboxId: testSandbox.ID,
							State:     "running",
							IpAddress: "10.0.0.10",
						},
					},
				}, nil
			},
		}
		s := newTestServerWithSender(ms, sender, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/sandboxes/"+testSandbox.ID+"/start", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["started"] != true {
			t.Fatalf("expected started=true, got %v", body["started"])
		}
	})
}

func TestHandleStopSandbox(t *testing.T) {
	testSandbox := &store.Sandbox{
		ID:     "SBX-stop1",
		OrgID:  testOrg.ID,
		HostID: "HOST-1",
		Name:   "stop-sandbox",
		State:  store.SandboxStateRunning,
	}

	t.Run("not_found", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/sandboxes/SBX-nonexistent/stop", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("wrong_org", func(t *testing.T) {
		wrongOrgSandbox := &store.Sandbox{
			ID:     "SBX-wrongorg",
			OrgID:  "ORG-different",
			HostID: "HOST-1",
			Name:   "wrong-org-sandbox",
			State:  store.SandboxStateRunning,
		}
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			if sandboxID == wrongOrgSandbox.ID {
				return wrongOrgSandbox, nil
			}
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/sandboxes/"+wrongOrgSandbox.ID+"/stop", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			if sandboxID == testSandbox.ID {
				return testSandbox, nil
			}
			return nil, store.ErrNotFound
		}
		ms.UpdateSandboxFn = func(_ context.Context, sandbox *store.Sandbox) error {
			return nil
		}
		sender := &mockHostSender{
			SendAndWaitFn: func(_ context.Context, hostID string, msg *fluidv1.ControlMessage, timeout time.Duration) (*fluidv1.HostMessage, error) {
				return &fluidv1.HostMessage{
					RequestId: msg.RequestId,
					Payload: &fluidv1.HostMessage_SandboxStopped{
						SandboxStopped: &fluidv1.SandboxStopped{
							SandboxId: testSandbox.ID,
							State:     "stopped",
						},
					},
				}, nil
			},
		}
		s := newTestServerWithSender(ms, sender, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/sandboxes/"+testSandbox.ID+"/stop", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["stopped"] != true {
			t.Fatalf("expected stopped=true, got %v", body["stopped"])
		}
	})
}

func TestHandleGetSandboxIP(t *testing.T) {
	testSandbox := &store.Sandbox{
		ID:        "SBX-ip1",
		OrgID:     testOrg.ID,
		HostID:    "HOST-1",
		Name:      "ip-sandbox",
		State:     store.SandboxStateRunning,
		IPAddress: "10.0.0.42",
	}

	t.Run("not_found", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/sandboxes/SBX-nonexistent/ip", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("wrong_org", func(t *testing.T) {
		wrongOrgSandbox := &store.Sandbox{
			ID:        "SBX-wrongorg",
			OrgID:     "ORG-different",
			HostID:    "HOST-1",
			Name:      "wrong-org-sandbox",
			State:     store.SandboxStateRunning,
			IPAddress: "10.0.0.99",
		}
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			if sandboxID == wrongOrgSandbox.ID {
				return wrongOrgSandbox, nil
			}
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/sandboxes/"+wrongOrgSandbox.ID+"/ip", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			if sandboxID == testSandbox.ID {
				return testSandbox, nil
			}
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/sandboxes/"+testSandbox.ID+"/ip", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["ip_address"] != "10.0.0.42" {
			t.Fatalf("expected ip_address=10.0.0.42, got %v", body["ip_address"])
		}
		if body["sandbox_id"] != testSandbox.ID {
			t.Fatalf("expected sandbox_id=%s, got %v", testSandbox.ID, body["sandbox_id"])
		}
	})
}

func TestHandleCreateSnapshot(t *testing.T) {
	testSandbox := &store.Sandbox{
		ID:     "SBX-snap1",
		OrgID:  testOrg.ID,
		HostID: "HOST-1",
		Name:   "snap-sandbox",
		State:  store.SandboxStateRunning,
	}

	t.Run("not_found", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		body := strings.NewReader(`{"name":"my-snapshot"}`)
		path := "/v1/orgs/test-org/sandboxes/SBX-nonexistent/snapshot"
		req := authenticatedRequest(ms, "POST", path,
			httptest.NewRequest("POST", path, body))
		req.Header.Set("Content-Type", "application/json")
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("wrong_org", func(t *testing.T) {
		wrongOrgSandbox := &store.Sandbox{
			ID:     "SBX-wrongorg",
			OrgID:  "ORG-different",
			HostID: "HOST-1",
			Name:   "wrong-org-sandbox",
			State:  store.SandboxStateRunning,
		}
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			if sandboxID == wrongOrgSandbox.ID {
				return wrongOrgSandbox, nil
			}
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		body := strings.NewReader(`{"name":"my-snapshot"}`)
		path := "/v1/orgs/test-org/sandboxes/" + wrongOrgSandbox.ID + "/snapshot"
		req := authenticatedRequest(ms, "POST", path,
			httptest.NewRequest("POST", path, body))
		req.Header.Set("Content-Type", "application/json")
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
			if sandboxID == testSandbox.ID {
				return testSandbox, nil
			}
			return nil, store.ErrNotFound
		}
		sender := &mockHostSender{
			SendAndWaitFn: func(_ context.Context, hostID string, msg *fluidv1.ControlMessage, timeout time.Duration) (*fluidv1.HostMessage, error) {
				return &fluidv1.HostMessage{
					RequestId: msg.RequestId,
					Payload: &fluidv1.HostMessage_SnapshotCreated{
						SnapshotCreated: &fluidv1.SnapshotCreated{
							SandboxId:    testSandbox.ID,
							SnapshotId:   "SNAP-abc123",
							SnapshotName: "my-snapshot",
						},
					},
				}, nil
			},
		}
		s := newTestServerWithSender(ms, sender, nil)

		rr := httptest.NewRecorder()
		body := strings.NewReader(`{"name":"my-snapshot"}`)
		path := "/v1/orgs/test-org/sandboxes/" + testSandbox.ID + "/snapshot"
		req := authenticatedRequest(ms, "POST", path,
			httptest.NewRequest("POST", path, body))
		req.Header.Set("Content-Type", "application/json")
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}

		respBody := parseJSONResponse(rr)
		if respBody["snapshot_name"] != "my-snapshot" {
			t.Fatalf("expected snapshot_name=my-snapshot, got %v", respBody["snapshot_name"])
		}
		if respBody["sandbox_id"] != testSandbox.ID {
			t.Fatalf("expected sandbox_id=%s, got %v", testSandbox.ID, respBody["sandbox_id"])
		}
	})
}
