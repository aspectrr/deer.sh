package grpc

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"

	"github.com/aspectrr/fluid.sh/api/internal/auth"
	"github.com/aspectrr/fluid.sh/api/internal/registry"
	"github.com/aspectrr/fluid.sh/api/internal/store"

	"google.golang.org/grpc/metadata"
)

// ---------------------------------------------------------------------------
// mockStore - minimal implementation of store.Store for grpc tests
// ---------------------------------------------------------------------------

type mockStore struct{}

func (m *mockStore) Config() store.Config       { return store.Config{} }
func (m *mockStore) Ping(context.Context) error { return nil }
func (m *mockStore) Close() error               { return nil }
func (m *mockStore) WithTx(ctx context.Context, fn func(tx store.DataStore) error) error {
	return fn(m)
}

func (m *mockStore) CreateUser(context.Context, *store.User) error        { return nil }
func (m *mockStore) GetUser(context.Context, string) (*store.User, error) { return nil, nil }
func (m *mockStore) GetUserByEmail(context.Context, string) (*store.User, error) {
	return nil, nil
}
func (m *mockStore) UpdateUser(context.Context, *store.User) error { return nil }

func (m *mockStore) CreateOAuthAccount(context.Context, *store.OAuthAccount) error { return nil }
func (m *mockStore) GetOAuthAccount(context.Context, string, string) (*store.OAuthAccount, error) {
	return nil, nil
}
func (m *mockStore) GetOAuthAccountsByUser(context.Context, string) ([]*store.OAuthAccount, error) {
	return nil, nil
}

func (m *mockStore) CreateSession(context.Context, *store.Session) error        { return nil }
func (m *mockStore) GetSession(context.Context, string) (*store.Session, error) { return nil, nil }
func (m *mockStore) DeleteSession(context.Context, string) error                { return nil }
func (m *mockStore) DeleteExpiredSessions(context.Context) error                { return nil }

func (m *mockStore) CreateOrganization(context.Context, *store.Organization) error { return nil }
func (m *mockStore) GetOrganization(context.Context, string) (*store.Organization, error) {
	return nil, nil
}
func (m *mockStore) GetOrganizationBySlug(context.Context, string) (*store.Organization, error) {
	return nil, nil
}
func (m *mockStore) ListOrganizationsByUser(context.Context, string) ([]*store.Organization, error) {
	return nil, nil
}
func (m *mockStore) UpdateOrganization(context.Context, *store.Organization) error { return nil }
func (m *mockStore) DeleteOrganization(context.Context, string) error              { return nil }

func (m *mockStore) CreateOrgMember(context.Context, *store.OrgMember) error { return nil }
func (m *mockStore) GetOrgMember(context.Context, string, string) (*store.OrgMember, error) {
	return nil, nil
}
func (m *mockStore) GetOrgMemberByID(context.Context, string, string) (*store.OrgMember, error) {
	return nil, nil
}
func (m *mockStore) ListOrgMembers(context.Context, string) ([]*store.OrgMember, error) {
	return nil, nil
}
func (m *mockStore) DeleteOrgMember(context.Context, string, string) error { return nil }

func (m *mockStore) CreateSubscription(context.Context, *store.Subscription) error { return nil }
func (m *mockStore) GetSubscriptionByOrg(context.Context, string) (*store.Subscription, error) {
	return nil, nil
}
func (m *mockStore) UpdateSubscription(context.Context, *store.Subscription) error { return nil }

func (m *mockStore) CreateUsageRecord(context.Context, *store.UsageRecord) error { return nil }
func (m *mockStore) ListUsageRecords(context.Context, string, time.Time, time.Time) ([]*store.UsageRecord, error) {
	return nil, nil
}

func (m *mockStore) CreateHost(context.Context, *store.Host) error                { return nil }
func (m *mockStore) GetHost(context.Context, string) (*store.Host, error)         { return nil, nil }
func (m *mockStore) ListHosts(context.Context) ([]store.Host, error)              { return nil, nil }
func (m *mockStore) ListHostsByOrg(context.Context, string) ([]store.Host, error) { return nil, nil }
func (m *mockStore) UpdateHost(context.Context, *store.Host) error                { return nil }
func (m *mockStore) UpdateHostHeartbeat(context.Context, string, int32, int64, int64) error {
	return nil
}

func (m *mockStore) CreateSandbox(context.Context, *store.Sandbox) error        { return nil }
func (m *mockStore) GetSandbox(context.Context, string) (*store.Sandbox, error) { return nil, nil }
func (m *mockStore) GetSandboxByOrg(context.Context, string, string) (*store.Sandbox, error) {
	return nil, nil
}
func (m *mockStore) ListSandboxes(context.Context) ([]store.Sandbox, error) { return nil, nil }
func (m *mockStore) ListSandboxesByOrg(context.Context, string) ([]store.Sandbox, error) {
	return nil, nil
}
func (m *mockStore) UpdateSandbox(context.Context, *store.Sandbox) error { return nil }
func (m *mockStore) DeleteSandbox(context.Context, string) error         { return nil }
func (m *mockStore) GetSandboxesByHostID(context.Context, string) ([]store.Sandbox, error) {
	return nil, nil
}
func (m *mockStore) CountSandboxesByHostIDs(context.Context, []string) (map[string]int, error) {
	return map[string]int{}, nil
}
func (m *mockStore) ListExpiredSandboxes(context.Context, time.Duration) ([]store.Sandbox, error) {
	return nil, nil
}

func (m *mockStore) CreateCommand(context.Context, *store.Command) error { return nil }
func (m *mockStore) ListSandboxCommands(context.Context, string) ([]store.Command, error) {
	return nil, nil
}

func (m *mockStore) CreateSourceHost(context.Context, *store.SourceHost) error { return nil }
func (m *mockStore) GetSourceHost(context.Context, string) (*store.SourceHost, error) {
	return nil, nil
}
func (m *mockStore) ListSourceHostsByOrg(context.Context, string) ([]*store.SourceHost, error) {
	return nil, nil
}
func (m *mockStore) DeleteSourceHost(context.Context, string) error { return nil }

func (m *mockStore) CreateHostToken(context.Context, *store.HostToken) error { return nil }
func (m *mockStore) GetHostTokenByHash(context.Context, string) (*store.HostToken, error) {
	return nil, nil
}
func (m *mockStore) ListHostTokensByOrg(context.Context, string) ([]store.HostToken, error) {
	return nil, nil
}
func (m *mockStore) DeleteHostToken(context.Context, string, string) error { return nil }

// Agent/playbook mock methods removed - interface methods commented out in store.go

func (m *mockStore) GetOrganizationByStripeCustomerID(context.Context, string) (*store.Organization, error) {
	return nil, nil
}
func (m *mockStore) GetModelMeter(context.Context, string) (*store.ModelMeter, error) {
	return nil, store.ErrNotFound
}
func (m *mockStore) CreateModelMeter(context.Context, *store.ModelMeter) error { return nil }
func (m *mockStore) GetOrgModelSubscription(context.Context, string, string) (*store.OrgModelSubscription, error) {
	return nil, store.ErrNotFound
}
func (m *mockStore) CreateOrgModelSubscription(context.Context, *store.OrgModelSubscription) error {
	return nil
}
func (m *mockStore) SumTokenUsage(context.Context, string, time.Time, time.Time) (float64, error) {
	return 0, nil
}
func (m *mockStore) ListActiveSubscriptions(context.Context) ([]*store.Subscription, error) {
	return nil, nil
}
func (m *mockStore) GetSubscriptionByStripeID(context.Context, string) (*store.Subscription, error) {
	return nil, nil
}
func (m *mockStore) AcquireAdvisoryLock(context.Context, int64) error { return nil }
func (m *mockStore) ReleaseAdvisoryLock(context.Context, int64) error { return nil }

// ---------------------------------------------------------------------------
// mockConnectServer implements fluidv1.HostService_ConnectServer
// (which is grpc.BidiStreamingServer[HostMessage, ControlMessage])
// ---------------------------------------------------------------------------

type mockConnectServer struct {
	sentMessages []*fluidv1.ControlMessage
	sendErr      error
	ctx          context.Context
}

func (m *mockConnectServer) Send(msg *fluidv1.ControlMessage) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sentMessages = append(m.sentMessages, msg)
	return nil
}

func (m *mockConnectServer) Recv() (*fluidv1.HostMessage, error) {
	return nil, fmt.Errorf("not implemented in mock")
}

func (m *mockConnectServer) SetHeader(metadata.MD) error  { return nil }
func (m *mockConnectServer) SendHeader(metadata.MD) error { return nil }
func (m *mockConnectServer) SetTrailer(metadata.MD)       {}
func (m *mockConnectServer) Context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}
func (m *mockConnectServer) SendMsg(interface{}) error { return nil }
func (m *mockConnectServer) RecvMsg(interface{}) error { return nil }

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestSendAndWait_HostNotConnected(t *testing.T) {
	reg := registry.New()
	handler := NewStreamHandler(reg, &mockStore{}, nil, 90*time.Second)

	msg := &fluidv1.ControlMessage{
		RequestId: "req-1",
	}

	_, err := handler.SendAndWait(context.Background(), "nonexistent-host", msg, 5*time.Second)
	if err == nil {
		t.Fatal("SendAndWait: expected error for disconnected host")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "not connected")
	}
}

func TestSendAndWait_MissingRequestID(t *testing.T) {
	reg := registry.New()
	handler := NewStreamHandler(reg, &mockStore{}, nil, 90*time.Second)

	// Store a mock stream so the host is "connected".
	mock := &mockConnectServer{}
	handler.streams.Store("host-1", fluidv1.HostService_ConnectServer(mock))

	msg := &fluidv1.ControlMessage{
		RequestId: "", // Empty request ID.
	}

	_, err := handler.SendAndWait(context.Background(), "host-1", msg, 5*time.Second)
	if err == nil {
		t.Fatal("SendAndWait: expected error for empty request_id")
	}
	if !strings.Contains(err.Error(), "request_id") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "request_id")
	}
}

func TestSendAndWait_Success(t *testing.T) {
	reg := registry.New()
	handler := NewStreamHandler(reg, &mockStore{}, nil, 90*time.Second)

	mock := &mockConnectServer{}
	handler.streams.Store("host-1", fluidv1.HostService_ConnectServer(mock))

	msg := &fluidv1.ControlMessage{
		RequestId: "req-123",
		Payload: &fluidv1.ControlMessage_DestroySandbox{
			DestroySandbox: &fluidv1.DestroySandboxCommand{
				SandboxId: "sbx-1",
			},
		},
	}

	// Simulate the host responding asynchronously.
	go func() {
		// Wait briefly for SendAndWait to register the pending request.
		time.Sleep(50 * time.Millisecond)

		response := &fluidv1.HostMessage{
			RequestId: "req-123",
			Payload: &fluidv1.HostMessage_SandboxDestroyed{
				SandboxDestroyed: &fluidv1.SandboxDestroyed{
					SandboxId: "sbx-1",
				},
			},
		}

		// Deliver response via the pendingRequests map.
		if ch, ok := handler.pendingRequests.Load("req-123"); ok {
			respCh := ch.(chan *fluidv1.HostMessage)
			respCh <- response
		}
	}()

	resp, err := handler.SendAndWait(context.Background(), "host-1", msg, 5*time.Second)
	if err != nil {
		t.Fatalf("SendAndWait: unexpected error: %v", err)
	}

	destroyed := resp.GetSandboxDestroyed()
	if destroyed == nil {
		t.Fatal("response: expected SandboxDestroyed payload")
	}
	if destroyed.GetSandboxId() != "sbx-1" {
		t.Errorf("SandboxId = %q, want %q", destroyed.GetSandboxId(), "sbx-1")
	}

	// Verify the message was actually sent to the mock stream.
	if len(mock.sentMessages) != 1 {
		t.Fatalf("sentMessages: got %d, want 1", len(mock.sentMessages))
	}
	if mock.sentMessages[0].GetRequestId() != "req-123" {
		t.Errorf("sent RequestId = %q, want %q", mock.sentMessages[0].GetRequestId(), "req-123")
	}
}

func TestSendAndWait_Timeout(t *testing.T) {
	reg := registry.New()
	handler := NewStreamHandler(reg, &mockStore{}, nil, 90*time.Second)

	mock := &mockConnectServer{}
	handler.streams.Store("host-1", fluidv1.HostService_ConnectServer(mock))

	msg := &fluidv1.ControlMessage{
		RequestId: "req-timeout",
	}

	// Use a very short timeout; no response will come.
	_, err := handler.SendAndWait(context.Background(), "host-1", msg, 100*time.Millisecond)
	if err == nil {
		t.Fatal("SendAndWait: expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "timeout")
	}
}

func TestSendAndWait_SendError(t *testing.T) {
	reg := registry.New()
	handler := NewStreamHandler(reg, &mockStore{}, nil, 90*time.Second)

	mock := &mockConnectServer{
		sendErr: fmt.Errorf("stream broken"),
	}
	handler.streams.Store("host-1", fluidv1.HostService_ConnectServer(mock))

	msg := &fluidv1.ControlMessage{
		RequestId: "req-fail",
	}

	_, err := handler.SendAndWait(context.Background(), "host-1", msg, 5*time.Second)
	if err == nil {
		t.Fatal("SendAndWait: expected error when stream Send fails")
	}
	if !strings.Contains(err.Error(), "send to host") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "send to host")
	}
}

func TestSendAndWait_CleansPendingOnTimeout(t *testing.T) {
	reg := registry.New()
	handler := NewStreamHandler(reg, &mockStore{}, nil, 90*time.Second)

	mock := &mockConnectServer{}
	handler.streams.Store("host-1", fluidv1.HostService_ConnectServer(mock))

	msg := &fluidv1.ControlMessage{
		RequestId: "req-cleanup",
	}

	_, _ = handler.SendAndWait(context.Background(), "host-1", msg, 50*time.Millisecond)

	// After timeout, the pending request should be cleaned up via defer.
	_, exists := handler.pendingRequests.Load("req-cleanup")
	if exists {
		t.Error("pending request should be cleaned up after timeout")
	}
}

// ---------------------------------------------------------------------------
// connectTestStore - mockStore override for Connect tests.
// Embeds mockStore for all methods, overrides GetHost/CreateHost/UpdateHost/
// UpdateHostHeartbeat so persistHostRegistration does not nil-deref.
// ---------------------------------------------------------------------------

type connectTestStore struct {
	mockStore

	hostCreated       bool
	hostUpdated       bool
	heartbeatCalled   bool
	heartbeatHostID   string
	heartbeatCPUs     int32
	heartbeatMemoryMB int64
	heartbeatDiskMB   int64
	getHostReturn     *store.Host
	getHostErr        error
}

func (s *connectTestStore) GetHost(_ context.Context, _ string) (*store.Host, error) {
	if s.getHostErr != nil {
		return nil, s.getHostErr
	}
	if s.getHostReturn != nil {
		return s.getHostReturn, nil
	}
	// Default: return a valid host so the update path works.
	return &store.Host{}, nil
}

func (s *connectTestStore) CreateHost(_ context.Context, _ *store.Host) error {
	s.hostCreated = true
	return nil
}

func (s *connectTestStore) UpdateHost(_ context.Context, _ *store.Host) error {
	s.hostUpdated = true
	return nil
}

func (s *connectTestStore) UpdateHostHeartbeat(_ context.Context, hostID string, cpus int32, memMB int64, diskMB int64) error {
	s.heartbeatCalled = true
	s.heartbeatHostID = hostID
	s.heartbeatCPUs = cpus
	s.heartbeatMemoryMB = memMB
	s.heartbeatDiskMB = diskMB
	return nil
}

// ---------------------------------------------------------------------------
// mockConnectServerQueued - mock stream that returns queued messages from Recv.
// After all messages are consumed, Recv returns io.EOF.
// ---------------------------------------------------------------------------

type mockConnectServerQueued struct {
	msgs    []*fluidv1.HostMessage
	idx     int
	sent    []*fluidv1.ControlMessage
	sendErr error
	ctx     context.Context
}

func (m *mockConnectServerQueued) Recv() (*fluidv1.HostMessage, error) {
	if m.idx >= len(m.msgs) {
		return nil, io.EOF
	}
	msg := m.msgs[m.idx]
	m.idx++
	return msg, nil
}

func (m *mockConnectServerQueued) Send(msg *fluidv1.ControlMessage) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sent = append(m.sent, msg)
	return nil
}

func (m *mockConnectServerQueued) SetHeader(metadata.MD) error  { return nil }
func (m *mockConnectServerQueued) SendHeader(metadata.MD) error { return nil }
func (m *mockConnectServerQueued) SetTrailer(metadata.MD)       {}
func (m *mockConnectServerQueued) Context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}
func (m *mockConnectServerQueued) SendMsg(interface{}) error { return nil }
func (m *mockConnectServerQueued) RecvMsg(interface{}) error { return nil }

// ---------------------------------------------------------------------------
// Connect() tests
// ---------------------------------------------------------------------------

func TestConnect_RecvError(t *testing.T) {
	reg := registry.New()
	st := &connectTestStore{}
	handler := NewStreamHandler(reg, st, nil, 90*time.Second)

	mock := &mockConnectServerQueued{
		msgs: nil, // empty - Recv immediately returns EOF
	}

	err := handler.Connect(mock)
	if err == nil {
		t.Fatal("Connect: expected error when first Recv fails")
	}
	if !strings.Contains(err.Error(), "recv registration") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "recv registration")
	}
}

func TestConnect_FirstMessageNotRegistration(t *testing.T) {
	reg := registry.New()
	st := &connectTestStore{}
	handler := NewStreamHandler(reg, st, nil, 90*time.Second)

	mock := &mockConnectServerQueued{
		msgs: []*fluidv1.HostMessage{
			{Payload: &fluidv1.HostMessage_Heartbeat{
				Heartbeat: &fluidv1.Heartbeat{AvailableCpus: 4},
			}},
		},
	}

	err := handler.Connect(mock)
	if err == nil {
		t.Fatal("Connect: expected error when first message is not registration")
	}
	if !strings.Contains(err.Error(), "first message must be HostRegistration") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "first message must be HostRegistration")
	}
}

func TestConnect_SendAckError(t *testing.T) {
	reg := registry.New()
	st := &connectTestStore{}
	handler := NewStreamHandler(reg, st, nil, 90*time.Second)

	mock := &mockConnectServerQueued{
		msgs: []*fluidv1.HostMessage{
			{Payload: &fluidv1.HostMessage_Registration{
				Registration: &fluidv1.HostRegistration{
					HostId:   "host-1",
					Hostname: "test-host",
				},
			}},
		},
		sendErr: fmt.Errorf("broken pipe"),
		ctx:     auth.WithTokenID(auth.WithOrgID(context.Background(), "org-1"), "host-1"),
	}

	err := handler.Connect(mock)
	if err == nil {
		t.Fatal("Connect: expected error when Send(ack) fails")
	}
	if !strings.Contains(err.Error(), "send registration ack") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "send registration ack")
	}
}

func TestConnect_SuccessfulRegistrationAndEOF(t *testing.T) {
	reg := registry.New()
	st := &connectTestStore{}
	handler := NewStreamHandler(reg, st, nil, 90*time.Second)

	mock := &mockConnectServerQueued{
		msgs: []*fluidv1.HostMessage{
			{Payload: &fluidv1.HostMessage_Registration{
				Registration: &fluidv1.HostRegistration{
					HostId:   "host-1",
					Hostname: "test-host",
					Version:  "1.0.0",
				},
			}},
			// No more messages - next Recv returns EOF.
		},
		ctx: auth.WithTokenID(auth.WithOrgID(context.Background(), "org-1"), "host-1"),
	}

	err := handler.Connect(mock)
	if err != nil {
		t.Fatalf("Connect: unexpected error: %v", err)
	}

	// Verify RegistrationAck was sent.
	if len(mock.sent) != 1 {
		t.Fatalf("sent messages: got %d, want 1", len(mock.sent))
	}
	ack := mock.sent[0].GetRegistrationAck()
	if ack == nil {
		t.Fatal("expected RegistrationAck in first sent message")
	}
	if !ack.GetAccepted() {
		t.Error("RegistrationAck.Accepted = false, want true")
	}
	if ack.GetAssignedHostId() != "host-1" {
		t.Errorf("AssignedHostId = %q, want %q", ack.GetAssignedHostId(), "host-1")
	}

	// Verify host was persisted (update path since GetHost returns &store.Host{}).
	if !st.hostUpdated {
		t.Error("expected store.UpdateHost to be called")
	}

	// After EOF, the defer should have unregistered the host from the registry.
	if _, ok := reg.GetHost("host-1"); ok {
		t.Error("host should be unregistered from registry after disconnect")
	}

	// Stream should also be cleaned up.
	if _, ok := handler.streams.Load("host-1"); ok {
		t.Error("stream should be deleted after disconnect")
	}
}

func TestConnect_PersistHostCreatesWhenNotFound(t *testing.T) {
	reg := registry.New()
	st := &connectTestStore{
		getHostErr: store.ErrNotFound, // GetHost returns error -> create path
	}
	handler := NewStreamHandler(reg, st, nil, 90*time.Second)

	mock := &mockConnectServerQueued{
		msgs: []*fluidv1.HostMessage{
			{Payload: &fluidv1.HostMessage_Registration{
				Registration: &fluidv1.HostRegistration{
					HostId:   "host-new",
					Hostname: "new-host",
				},
			}},
		},
		ctx: auth.WithTokenID(auth.WithOrgID(context.Background(), "org-1"), "host-new"),
	}

	err := handler.Connect(mock)
	if err != nil {
		t.Fatalf("Connect: unexpected error: %v", err)
	}

	if !st.hostCreated {
		t.Error("expected store.CreateHost to be called when GetHost returns error")
	}
	if st.hostUpdated {
		t.Error("store.UpdateHost should not be called when GetHost returns error")
	}
}

func TestConnect_HeartbeatDispatch(t *testing.T) {
	reg := registry.New()
	st := &connectTestStore{}
	handler := NewStreamHandler(reg, st, nil, 90*time.Second)

	mock := &mockConnectServerQueued{
		msgs: []*fluidv1.HostMessage{
			{Payload: &fluidv1.HostMessage_Registration{
				Registration: &fluidv1.HostRegistration{
					HostId:   "host-hb",
					Hostname: "heartbeat-host",
				},
			}},
			{Payload: &fluidv1.HostMessage_Heartbeat{
				Heartbeat: &fluidv1.Heartbeat{
					AvailableCpus:     8,
					AvailableMemoryMb: 16384,
					AvailableDiskMb:   256000,
				},
			}},
			// EOF after heartbeat.
		},
		ctx: auth.WithTokenID(auth.WithOrgID(context.Background(), "org-1"), "host-hb"),
	}

	err := handler.Connect(mock)
	if err != nil {
		t.Fatalf("Connect: unexpected error: %v", err)
	}

	if !st.heartbeatCalled {
		t.Fatal("expected UpdateHostHeartbeat to be called")
	}
	if st.heartbeatHostID != "host-hb" {
		t.Errorf("heartbeat hostID = %q, want %q", st.heartbeatHostID, "host-hb")
	}
	if st.heartbeatCPUs != 8 {
		t.Errorf("heartbeat CPUs = %d, want 8", st.heartbeatCPUs)
	}
	if st.heartbeatMemoryMB != 16384 {
		t.Errorf("heartbeat MemoryMB = %d, want 16384", st.heartbeatMemoryMB)
	}
	if st.heartbeatDiskMB != 256000 {
		t.Errorf("heartbeat DiskMB = %d, want 256000", st.heartbeatDiskMB)
	}
}

func TestConnect_ResponseDispatch(t *testing.T) {
	reg := registry.New()
	st := &connectTestStore{}
	handler := NewStreamHandler(reg, st, nil, 90*time.Second)

	// Pre-populate a pending request channel.
	respCh := make(chan *fluidv1.HostMessage, 1)
	handler.pendingRequests.Store("req-test", respCh)

	mock := &mockConnectServerQueued{
		msgs: []*fluidv1.HostMessage{
			{Payload: &fluidv1.HostMessage_Registration{
				Registration: &fluidv1.HostRegistration{
					HostId:   "host-resp",
					Hostname: "resp-host",
				},
			}},
			{
				RequestId: "req-test",
				Payload: &fluidv1.HostMessage_SandboxDestroyed{
					SandboxDestroyed: &fluidv1.SandboxDestroyed{
						SandboxId: "sbx-1",
					},
				},
			},
			// EOF after response.
		},
		ctx: auth.WithTokenID(auth.WithOrgID(context.Background(), "org-1"), "host-resp"),
	}

	err := handler.Connect(mock)
	if err != nil {
		t.Fatalf("Connect: unexpected error: %v", err)
	}

	// Verify the response was delivered to the channel.
	select {
	case resp := <-respCh:
		destroyed := resp.GetSandboxDestroyed()
		if destroyed == nil {
			t.Fatal("expected SandboxDestroyed payload in response")
		}
		if destroyed.GetSandboxId() != "sbx-1" {
			t.Errorf("SandboxId = %q, want %q", destroyed.GetSandboxId(), "sbx-1")
		}
	default:
		t.Fatal("expected response to be delivered to pending request channel")
	}
}

func TestConnect_ErrorReportDoesNotPanic(t *testing.T) {
	reg := registry.New()
	st := &connectTestStore{}
	handler := NewStreamHandler(reg, st, nil, 90*time.Second)

	mock := &mockConnectServerQueued{
		msgs: []*fluidv1.HostMessage{
			{Payload: &fluidv1.HostMessage_Registration{
				Registration: &fluidv1.HostRegistration{
					HostId:   "host-err",
					Hostname: "err-host",
				},
			}},
			{Payload: &fluidv1.HostMessage_ErrorReport{
				ErrorReport: &fluidv1.ErrorReport{
					Error:     "disk full",
					SandboxId: "sbx-42",
					Context:   "creating overlay",
				},
			}},
		},
		ctx: auth.WithTokenID(auth.WithOrgID(context.Background(), "org-1"), "host-err"),
	}

	// Should not panic and should return nil on EOF.
	err := handler.Connect(mock)
	if err != nil {
		t.Fatalf("Connect: unexpected error: %v", err)
	}
}

func TestConnect_ResourceReportUpdatesHeartbeat(t *testing.T) {
	reg := registry.New()
	st := &connectTestStore{}
	handler := NewStreamHandler(reg, st, nil, 90*time.Second)

	mock := &mockConnectServerQueued{
		msgs: []*fluidv1.HostMessage{
			{Payload: &fluidv1.HostMessage_Registration{
				Registration: &fluidv1.HostRegistration{
					HostId:   "host-rr",
					Hostname: "resource-host",
				},
			}},
			{Payload: &fluidv1.HostMessage_ResourceReport{
				ResourceReport: &fluidv1.ResourceReport{
					AvailableCpus:     16,
					AvailableMemoryMb: 32768,
					AvailableDiskMb:   512000,
				},
			}},
		},
		ctx: auth.WithTokenID(auth.WithOrgID(context.Background(), "org-1"), "host-rr"),
	}

	err := handler.Connect(mock)
	if err != nil {
		t.Fatalf("Connect: unexpected error: %v", err)
	}

	// ResourceReport should update the heartbeat in the registry, but since
	// the host is unregistered on disconnect (defer), we cannot check the
	// registry here. The test verifies no panic or error occurs.
}

func TestConnect_MessageWithoutRequestIDDropped(t *testing.T) {
	reg := registry.New()
	st := &connectTestStore{}
	handler := NewStreamHandler(reg, st, nil, 90*time.Second)

	mock := &mockConnectServerQueued{
		msgs: []*fluidv1.HostMessage{
			{Payload: &fluidv1.HostMessage_Registration{
				Registration: &fluidv1.HostRegistration{
					HostId:   "host-drop",
					Hostname: "drop-host",
				},
			}},
			// A response-type message with no request_id should be dropped.
			{
				RequestId: "",
				Payload: &fluidv1.HostMessage_SandboxCreated{
					SandboxCreated: &fluidv1.SandboxCreated{
						SandboxId: "sbx-orphan",
					},
				},
			},
		},
		ctx: auth.WithTokenID(auth.WithOrgID(context.Background(), "org-1"), "host-drop"),
	}

	// Should not panic; the message is silently dropped.
	err := handler.Connect(mock)
	if err != nil {
		t.Fatalf("Connect: unexpected error: %v", err)
	}
}

func TestConnect_UnmatchedRequestIDDropped(t *testing.T) {
	reg := registry.New()
	st := &connectTestStore{}
	handler := NewStreamHandler(reg, st, nil, 90*time.Second)

	// No pending requests pre-populated.

	mock := &mockConnectServerQueued{
		msgs: []*fluidv1.HostMessage{
			{Payload: &fluidv1.HostMessage_Registration{
				Registration: &fluidv1.HostRegistration{
					HostId:   "host-unmatched",
					Hostname: "unmatched-host",
				},
			}},
			// A response with a request_id that has no pending listener.
			{
				RequestId: "req-nobody",
				Payload: &fluidv1.HostMessage_SandboxDestroyed{
					SandboxDestroyed: &fluidv1.SandboxDestroyed{
						SandboxId: "sbx-gone",
					},
				},
			},
		},
		ctx: auth.WithTokenID(auth.WithOrgID(context.Background(), "org-1"), "host-unmatched"),
	}

	// Should not panic; the response is logged as orphan and dropped.
	err := handler.Connect(mock)
	if err != nil {
		t.Fatalf("Connect: unexpected error: %v", err)
	}
}

func TestConnect_EmptyTokenID_Rejected(t *testing.T) {
	reg := registry.New()
	st := &connectTestStore{}
	handler := NewStreamHandler(reg, st, nil, 90*time.Second)

	mock := &mockConnectServerQueued{
		msgs: []*fluidv1.HostMessage{
			{Payload: &fluidv1.HostMessage_Registration{
				Registration: &fluidv1.HostRegistration{
					HostId:   "host-1",
					Hostname: "test-host",
				},
			}},
		},
		// No WithTokenID - context has empty token ID.
		ctx: context.Background(),
	}

	err := handler.Connect(mock)
	if err == nil {
		t.Fatal("Connect: expected error when tokenID is empty")
	}
	if !strings.Contains(err.Error(), "missing token identity") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "missing token identity")
	}
}
