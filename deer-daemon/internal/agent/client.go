// Package agent implements the gRPC client that connects the sandbox host
// to the control plane. It handles registration, heartbeat, and dispatching
// of commands received from the control plane to the sandbox provider.
package agent

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	deerv1 "github.com/aspectrr/deer.sh/proto/gen/go/deer/v1"

	"github.com/aspectrr/deer.sh/deer-daemon/internal/kafkastub"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/provider"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/snapshotpull"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/sshconfig"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/state"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Client connects to the control plane via gRPC bidirectional streaming.
type Client struct {
	hostID          string
	hostname        string
	version         string
	cpAddr          string
	token           string
	insec           bool
	certFile        string
	keyFile         string
	caFile          string
	sshIdentityFile string

	prov       provider.SandboxProvider
	localStore *state.Store
	puller     *snapshotpull.Puller
	kafkaMgr   *kafkastub.Manager
	logger     *slog.Logger

	// stream is the active bidirectional stream to the control plane.
	mu     sync.Mutex
	stream deerv1.HostService_ConnectClient
	conn   *grpc.ClientConn

	// sendMu serializes writes to the gRPC stream.
	sendMu sync.Mutex

	// handlerSem bounds the number of concurrent command handler goroutines.
	handlerSem chan struct{}
}

// Config holds configuration for the gRPC agent client.
type Config struct {
	HostID          string
	Hostname        string
	Version         string
	Address         string
	Token           string
	Insecure        bool
	CertFile        string
	KeyFile         string
	CAFile          string
	SSHIdentityFile string
}

// NewClient creates a new agent client.
func NewClient(
	cfg Config,
	prov provider.SandboxProvider,
	localStore *state.Store,
	puller *snapshotpull.Puller,
	logger *slog.Logger,
) *Client {
	hostname := cfg.Hostname
	if hostname == "" {
		hostname, _ = os.Hostname()
	}

	kafkaBaseDir := filepath.Join(os.TempDir(), "deer-kafka-stub", cfg.HostID)
	kafkaMgr, err := newKafkaManager(kafkaBaseDir, logger, localStore)
	if err != nil && logger != nil {
		logger.Warn("failed to initialize kafka stub manager", "error", err)
	}

	return &Client{
		hostID:          cfg.HostID,
		hostname:        hostname,
		version:         cfg.Version,
		cpAddr:          cfg.Address,
		token:           cfg.Token,
		insec:           cfg.Insecure,
		certFile:        cfg.CertFile,
		keyFile:         cfg.KeyFile,
		caFile:          cfg.CAFile,
		sshIdentityFile: cfg.SSHIdentityFile,
		prov:            prov,
		localStore:      localStore,
		puller:          puller,
		kafkaMgr:        kafkaMgr,
		logger:          logger.With("component", "agent"),
		handlerSem:      make(chan struct{}, 64),
	}
}

// tokenCreds implements grpc.PerRPCCredentials to attach a Bearer token
// to every gRPC call.
type tokenCreds struct {
	token    string
	insecure bool
}

func (t tokenCreds) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + t.token}, nil
}

func (t tokenCreds) RequireTransportSecurity() bool {
	return !t.insecure
}

// sendMessage serializes writes to the gRPC stream.
func (c *Client) sendMessage(stream deerv1.HostService_ConnectClient, msg *deerv1.HostMessage) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	return stream.Send(msg)
}

// Run connects to the control plane and runs the message loop. It reconnects
// automatically on failure using exponential backoff. Blocks until ctx is done.
func (c *Client) Run(ctx context.Context) error {
	return RunWithReconnect(ctx, c.logger, c.connectAndServe)
}

// connectAndServe establishes a single connection, registers, and runs the
// message loop. Returns an error when the connection drops.
func (c *Client) connectAndServe(ctx context.Context) error {
	opts := []grpc.DialOption{}
	if c.insec {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		tlsCreds, err := c.buildTLSCredentials()
		if err != nil {
			return fmt.Errorf("build TLS credentials: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(tlsCreds))
	}

	if c.token != "" {
		opts = append(opts, grpc.WithPerRPCCredentials(tokenCreds{
			token:    c.token,
			insecure: c.insec,
		}))
	}

	conn, err := grpc.NewClient(c.cpAddr, opts...)
	if err != nil {
		return fmt.Errorf("dial control plane %s: %w", c.cpAddr, err)
	}
	defer func() { _ = conn.Close() }()

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	client := deerv1.NewHostServiceClient(conn)
	stream, err := client.Connect(ctx)
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}

	c.mu.Lock()
	c.stream = stream
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.stream = nil
		c.conn = nil
		c.mu.Unlock()
	}()

	if err := c.register(stream); err != nil {
		return err
	}

	heartbeatCtx, heartbeatCancel := context.WithCancel(ctx)
	defer heartbeatCancel()
	go c.heartbeatLoop(heartbeatCtx, stream)

	return c.recvLoop(ctx, stream)
}

// register sends the HostRegistration message and waits for RegistrationAck.
func (c *Client) register(stream deerv1.HostService_ConnectClient) error {
	reg := c.buildRegistration()

	reqID := uuid.New().String()
	msg := &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_Registration{
			Registration: reg,
		},
	}

	c.logger.Info("sending registration",
		"host_id", c.hostID,
		"hostname", c.hostname,
	)

	if err := c.sendMessage(stream, msg); err != nil {
		return fmt.Errorf("send registration: %w", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("recv registration ack: %w", err)
	}

	ack := resp.GetRegistrationAck()
	if ack == nil {
		return fmt.Errorf("expected RegistrationAck, got different message type")
	}

	if !ack.GetAccepted() {
		return fmt.Errorf("registration rejected: %s", ack.GetReason())
	}

	if assigned := ack.GetAssignedHostId(); assigned != "" && assigned != c.hostID {
		c.logger.Info("host ID reassigned by control plane", "old", c.hostID, "new", assigned)
		c.hostID = assigned
	}

	c.logger.Info("registered with control plane", "host_id", c.hostID)
	return nil
}

// buildRegistration constructs the HostRegistration message via the provider.
func (c *Client) buildRegistration() *deerv1.HostRegistration {
	reg := &deerv1.HostRegistration{
		HostId:   c.hostID,
		Hostname: c.hostname,
		Version:  c.version,
	}

	if c.prov != nil {
		caps, err := c.prov.Capabilities(context.Background())
		if err == nil {
			reg.TotalCpus = int32(caps.TotalCPUs)
			reg.AvailableCpus = int32(caps.AvailableCPUs)
			reg.BaseImages = caps.BaseImages
		}

		vms, err := c.prov.ListSourceVMs(context.Background())
		if err == nil {
			for _, vm := range vms {
				reg.SourceVms = append(reg.SourceVms, &deerv1.SourceVMInfo{
					Name:      vm.Name,
					State:     vm.State,
					IpAddress: vm.IPAddress,
					Prepared:  vm.Prepared,
				})
			}
		}
	}

	return reg
}

// heartbeatLoop sends periodic heartbeats to the control plane.
func (c *Client) heartbeatLoop(ctx context.Context, stream deerv1.HostService_ConnectClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hb := &deerv1.Heartbeat{}

			if c.prov != nil {
				caps, err := c.prov.Capabilities(ctx)
				if err == nil {
					hb.AvailableCpus = int32(caps.AvailableCPUs)
				}
				hb.ActiveSandboxes = int32(c.prov.ActiveSandboxCount())
				vms, err := c.prov.ListSourceVMs(ctx)
				if err == nil {
					hb.SourceVmCount = int32(len(vms))
				}
			}

			msg := &deerv1.HostMessage{
				Payload: &deerv1.HostMessage_Heartbeat{
					Heartbeat: hb,
				},
			}

			if err := c.sendMessage(stream, msg); err != nil {
				c.logger.Error("send heartbeat failed", "error", err)
				return
			}
		}
	}
}

// recvLoop receives and dispatches ControlMessages from the control plane.
func (c *Client) recvLoop(ctx context.Context, stream deerv1.HostService_ConnectClient) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				c.logger.Info("stream closed by control plane")
				return nil
			}
			return fmt.Errorf("recv: %w", err)
		}

		select {
		case c.handlerSem <- struct{}{}:
			go func() {
				defer func() { <-c.handlerSem }()
				c.handleCommand(ctx, stream, msg)
			}()
		default:
			c.logger.Warn("too many concurrent command handlers, dropping", "request_id", msg.GetRequestId())
		}
	}
}

// handleCommand dispatches a ControlMessage to the appropriate handler.
func (c *Client) handleCommand(ctx context.Context, stream deerv1.HostService_ConnectClient, msg *deerv1.ControlMessage) {
	reqID := msg.GetRequestId()

	var resp *deerv1.HostMessage

	switch cmd := msg.Payload.(type) {
	case *deerv1.ControlMessage_CreateSandbox:
		resp = c.handleCreateSandbox(ctx, reqID, cmd.CreateSandbox)
	case *deerv1.ControlMessage_DestroySandbox:
		resp = c.handleDestroySandbox(ctx, reqID, cmd.DestroySandbox)
	case *deerv1.ControlMessage_StartSandbox:
		resp = c.handleStartSandbox(ctx, reqID, cmd.StartSandbox)
	case *deerv1.ControlMessage_StopSandbox:
		resp = c.handleStopSandbox(ctx, reqID, cmd.StopSandbox)
	case *deerv1.ControlMessage_ListSandboxKafkaStubs:
		resp = c.handleListSandboxKafkaStubs(ctx, reqID, cmd.ListSandboxKafkaStubs)
	case *deerv1.ControlMessage_GetSandboxKafkaStub:
		resp = c.handleGetSandboxKafkaStub(ctx, reqID, cmd.GetSandboxKafkaStub)
	case *deerv1.ControlMessage_StartSandboxKafkaStub:
		resp = c.handleStartSandboxKafkaStub(ctx, reqID, cmd.StartSandboxKafkaStub)
	case *deerv1.ControlMessage_StopSandboxKafkaStub:
		resp = c.handleStopSandboxKafkaStub(ctx, reqID, cmd.StopSandboxKafkaStub)
	case *deerv1.ControlMessage_RestartSandboxKafkaStub:
		resp = c.handleRestartSandboxKafkaStub(ctx, reqID, cmd.RestartSandboxKafkaStub)
	case *deerv1.ControlMessage_GetKafkaCaptureStatus:
		resp = c.handleGetKafkaCaptureStatus(ctx, reqID, cmd.GetKafkaCaptureStatus)
	case *deerv1.ControlMessage_RunCommand:
		resp = c.handleRunCommand(ctx, reqID, cmd.RunCommand)
	case *deerv1.ControlMessage_CreateSnapshot:
		resp = c.handleCreateSnapshot(ctx, reqID, cmd.CreateSnapshot)
	case *deerv1.ControlMessage_PrepareSourceVm:
		resp = c.handlePrepareSourceVM(ctx, reqID, cmd.PrepareSourceVm)
	case *deerv1.ControlMessage_RunSourceCommand:
		resp = c.handleRunSourceCommand(ctx, reqID, cmd.RunSourceCommand)
	case *deerv1.ControlMessage_ReadSourceFile:
		resp = c.handleReadSourceFile(ctx, reqID, cmd.ReadSourceFile)
	case *deerv1.ControlMessage_ListSourceVms:
		resp = c.handleListSourceVMs(ctx, reqID)
	case *deerv1.ControlMessage_ValidateSourceVm:
		resp = c.handleValidateSourceVM(ctx, reqID, cmd.ValidateSourceVm)
	case *deerv1.ControlMessage_DiscoverHosts:
		resp = c.handleDiscoverHosts(ctx, reqID, cmd.DiscoverHosts)
	default:
		c.logger.Warn("unknown command type", "request_id", reqID)
		resp = errorResponse(reqID, "", "unknown command type")
	}

	if resp != nil {
		if err := c.sendMessage(stream, resp); err != nil {
			c.logger.Error("send response failed", "request_id", reqID, "error", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Sandbox command handlers
// ---------------------------------------------------------------------------

func (c *Client) handleCreateSandbox(ctx context.Context, reqID string, cmd *deerv1.CreateSandboxCommand) *deerv1.HostMessage {
	sandboxID := cmd.GetSandboxId()
	c.logger.Info("creating sandbox", "sandbox_id", sandboxID, "base_image", cmd.GetBaseImage())

	// Snapshot-pull if source host connection is provided
	baseImage := cmd.GetBaseImage()
	if conn := cmd.GetSourceHostConnection(); conn != nil && cmd.GetSourceVm() != "" && c.puller != nil {
		var backend snapshotpull.SnapshotBackend
		switch conn.GetType() {
		case "libvirt":
			backend = snapshotpull.NewLibvirtBackend(
				conn.GetSshHost(), int(conn.GetSshPort()),
				conn.GetSshUser(), c.sshIdentityFile, c.logger)
		case "proxmox":
			backend = snapshotpull.NewProxmoxBackend(
				conn.GetProxmoxHost(), conn.GetProxmoxTokenId(),
				conn.GetProxmoxSecret(), conn.GetProxmoxNode(),
				conn.GetProxmoxVerifySsl(), c.logger)
		}
		if backend != nil {
			mode := "cached"
			if cmd.GetSnapshotMode() == deerv1.SnapshotMode_SNAPSHOT_MODE_FRESH {
				mode = "fresh"
			}
			pullResult, err := c.puller.Pull(ctx, snapshotpull.PullRequest{
				SourceHost:   conn.GetSshHost(),
				VMName:       cmd.GetSourceVm(),
				SnapshotMode: mode,
			}, backend)
			if err != nil {
				return errorResponse(reqID, sandboxID, fmt.Sprintf("pull snapshot: %v", err))
			}
			baseImage = pullResult.ImageName
			c.logger.Info("snapshot pulled", "image", baseImage, "cached", pullResult.Cached)
		}
	}

	result, err := c.prov.CreateSandbox(ctx, provider.CreateRequest{
		SandboxID:    sandboxID,
		Name:         cmd.GetName(),
		BaseImage:    baseImage,
		SourceVM:     cmd.GetSourceVm(),
		Network:      cmd.GetNetwork(),
		VCPUs:        int(cmd.GetVcpus()),
		MemoryMB:     int(cmd.GetMemoryMb()),
		TTLSeconds:   int(cmd.GetTtlSeconds()),
		AgentID:      cmd.GetAgentId(),
		SSHPublicKey: cmd.GetSshPublicKey(),
		DataSources:  providerDataSourcesFromProto(cmd.GetDataSources(), cmd.GetKafkaCaptureConfigs()),
		KafkaBroker:  kafkaBrokerConfigForDataSources(cmd.GetDataSources(), cmd.GetKafkaCaptureConfigs()),
	})
	if err != nil {
		return errorResponse(reqID, sandboxID, fmt.Sprintf("create sandbox: %v", err))
	}

	// Persist to local state
	localSandbox := &state.Sandbox{
		ID:         sandboxID,
		Name:       result.Name,
		BaseImage:  baseImage,
		State:      result.State,
		IPAddress:  result.IPAddress,
		MACAddress: result.MACAddress,
		TAPDevice:  "",
		Bridge:     result.Bridge,
		VCPUs:      int(cmd.GetVcpus()),
		MemoryMB:   int(cmd.GetMemoryMb()),
		TTLSeconds: int(cmd.GetTtlSeconds()),
		AgentID:    cmd.GetAgentId(),
	}
	if err := c.localStore.CreateSandbox(ctx, localSandbox); err != nil {
		c.logger.Error("failed to persist sandbox locally", "sandbox_id", sandboxID, "error", err)
	}

	kafkaStubs := c.attachKafkaDataSources(ctx, sandboxID, result.IPAddress, cmd.GetDataSources(), cmd.GetKafkaCaptureConfigs())

	c.logger.Info("sandbox created",
		"sandbox_id", sandboxID,
		"ip", result.IPAddress,
		"bridge", result.Bridge,
	)

	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_SandboxCreated{
			SandboxCreated: &deerv1.SandboxCreated{
				SandboxId:  sandboxID,
				Name:       result.Name,
				State:      result.State,
				IpAddress:  result.IPAddress,
				MacAddress: result.MACAddress,
				Bridge:     result.Bridge,
				Pid:        int32(result.PID),
				KafkaStubs: kafkaStubs,
			},
		},
	}
}

func (c *Client) handleDestroySandbox(ctx context.Context, reqID string, cmd *deerv1.DestroySandboxCommand) *deerv1.HostMessage {
	sandboxID := cmd.GetSandboxId()
	c.logger.Info("destroying sandbox", "sandbox_id", sandboxID)

	if err := c.prov.DestroySandbox(ctx, sandboxID); err != nil {
		c.logger.Error("destroy sandbox failed", "sandbox_id", sandboxID, "error", err)
		return errorResponse(reqID, sandboxID, fmt.Sprintf("destroy failed: %s", err.Error()))
	}

	if err := c.localStore.DeleteSandbox(ctx, sandboxID); err != nil {
		c.logger.Error("delete local sandbox state failed", "sandbox_id", sandboxID, "error", err)
	}
	c.detachKafkaStubs(ctx, sandboxID)

	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_SandboxDestroyed{
			SandboxDestroyed: &deerv1.SandboxDestroyed{
				SandboxId: sandboxID,
			},
		},
	}
}

func (c *Client) handleStartSandbox(ctx context.Context, reqID string, cmd *deerv1.StartSandboxCommand) *deerv1.HostMessage {
	sandboxID := cmd.GetSandboxId()

	result, err := c.prov.StartSandbox(ctx, sandboxID)
	if err != nil {
		return errorResponse(reqID, sandboxID, fmt.Sprintf("start sandbox: %v", err))
	}

	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_SandboxStarted{
			SandboxStarted: &deerv1.SandboxStarted{
				SandboxId: sandboxID,
				State:     result.State,
				IpAddress: result.IPAddress,
			},
		},
	}
}

func (c *Client) handleStopSandbox(ctx context.Context, reqID string, cmd *deerv1.StopSandboxCommand) *deerv1.HostMessage {
	sandboxID := cmd.GetSandboxId()

	if err := c.prov.StopSandbox(ctx, sandboxID, cmd.GetForce()); err != nil {
		return errorResponse(reqID, sandboxID, fmt.Sprintf("stop: %v", err))
	}

	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_SandboxStopped{
			SandboxStopped: &deerv1.SandboxStopped{
				SandboxId: sandboxID,
				State:     "STOPPED",
			},
		},
	}
}

func (c *Client) handleListSandboxKafkaStubs(ctx context.Context, reqID string, cmd *deerv1.ListSandboxKafkaStubsCommand) *deerv1.HostMessage {
	stubs, err := c.listKafkaStubs(ctx, cmd.GetSandboxId())
	if err != nil {
		return errorResponse(reqID, cmd.GetSandboxId(), fmt.Sprintf("list sandbox kafka stubs: %v", err))
	}
	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_ListSandboxKafkaStubsResponse{
			ListSandboxKafkaStubsResponse: &deerv1.ListSandboxKafkaStubsResponse{Stubs: stubs},
		},
	}
}

func (c *Client) handleGetSandboxKafkaStub(ctx context.Context, reqID string, cmd *deerv1.GetSandboxKafkaStubCommand) *deerv1.HostMessage {
	stub, err := c.getKafkaStub(ctx, cmd.GetSandboxId(), cmd.GetStubId())
	if err != nil {
		return errorResponse(reqID, cmd.GetSandboxId(), fmt.Sprintf("get sandbox kafka stub: %v", err))
	}
	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_SandboxKafkaStubInfo{
			SandboxKafkaStubInfo: stub,
		},
	}
}

func (c *Client) handleStartSandboxKafkaStub(ctx context.Context, reqID string, cmd *deerv1.StartSandboxKafkaStubCommand) *deerv1.HostMessage {
	stub, err := c.transitionKafkaStub(ctx, cmd.GetSandboxId(), cmd.GetStubId(), "start")
	if err != nil {
		return errorResponse(reqID, cmd.GetSandboxId(), fmt.Sprintf("start sandbox kafka stub: %v", err))
	}
	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_SandboxKafkaStubInfo{
			SandboxKafkaStubInfo: stub,
		},
	}
}

func (c *Client) handleStopSandboxKafkaStub(ctx context.Context, reqID string, cmd *deerv1.StopSandboxKafkaStubCommand) *deerv1.HostMessage {
	stub, err := c.transitionKafkaStub(ctx, cmd.GetSandboxId(), cmd.GetStubId(), "stop")
	if err != nil {
		return errorResponse(reqID, cmd.GetSandboxId(), fmt.Sprintf("stop sandbox kafka stub: %v", err))
	}
	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_SandboxKafkaStubInfo{
			SandboxKafkaStubInfo: stub,
		},
	}
}

func (c *Client) handleRestartSandboxKafkaStub(ctx context.Context, reqID string, cmd *deerv1.RestartSandboxKafkaStubCommand) *deerv1.HostMessage {
	stub, err := c.transitionKafkaStub(ctx, cmd.GetSandboxId(), cmd.GetStubId(), "restart")
	if err != nil {
		return errorResponse(reqID, cmd.GetSandboxId(), fmt.Sprintf("restart sandbox kafka stub: %v", err))
	}
	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_SandboxKafkaStubInfo{
			SandboxKafkaStubInfo: stub,
		},
	}
}

func (c *Client) handleGetKafkaCaptureStatus(ctx context.Context, reqID string, cmd *deerv1.KafkaCaptureStatusRequest) *deerv1.HostMessage {
	var statuses []*deerv1.KafkaCaptureStatus
	if c.kafkaMgr != nil {
		items, err := c.kafkaMgr.ListCaptureStatuses(ctx, cmd.GetCaptureConfigIds())
		if err != nil {
			return errorResponse(reqID, "", fmt.Sprintf("get kafka capture status: %v", err))
		}
		statuses = make([]*deerv1.KafkaCaptureStatus, 0, len(items))
		for _, item := range items {
			_ = mergeCaptureStatus(ctx, c.localStore, item)
			statuses = append(statuses, &deerv1.KafkaCaptureStatus{
				CaptureConfigId:      item.CaptureConfigID,
				SourceVm:             item.SourceVM,
				State:                item.State,
				BufferedBytes:        item.BufferedBytes,
				SegmentCount:         int32(item.SegmentCount),
				UpdatedAtUnix:        item.UpdatedAt.Unix(),
				AttachedSandboxCount: int32(item.AttachedSandboxCount),
				LastError:            item.LastError,
				LastResumeCursor:     item.LastResumeCursor,
			})
		}
	}
	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_KafkaCaptureStatusResponse{
			KafkaCaptureStatusResponse: &deerv1.KafkaCaptureStatusResponse{Statuses: statuses},
		},
	}
}

func (c *Client) handleRunCommand(ctx context.Context, reqID string, cmd *deerv1.RunCommandCommand) *deerv1.HostMessage {
	sandboxID := cmd.GetSandboxId()
	command := cmd.GetCommand()

	c.logger.Info("running command", "sandbox_id", sandboxID, "command", command)

	timeout := time.Duration(cmd.GetTimeoutSeconds()) * time.Second

	result, err := c.prov.RunCommand(ctx, sandboxID, command, timeout)
	if err != nil {
		return errorResponse(reqID, sandboxID, fmt.Sprintf("run command: %v", err))
	}

	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_CommandResult{
			CommandResult: &deerv1.CommandResult{
				SandboxId:  sandboxID,
				Stdout:     result.Stdout,
				Stderr:     result.Stderr,
				ExitCode:   int32(result.ExitCode),
				DurationMs: result.DurationMS,
			},
		},
	}
}

func (c *Client) handleCreateSnapshot(ctx context.Context, reqID string, cmd *deerv1.SnapshotCommand) *deerv1.HostMessage {
	sandboxID := cmd.GetSandboxId()
	name := cmd.GetSnapshotName()

	result, err := c.prov.CreateSnapshot(ctx, sandboxID, name)
	if err != nil {
		return errorResponse(reqID, sandboxID, fmt.Sprintf("create snapshot: %v", err))
	}

	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_SnapshotCreated{
			SnapshotCreated: &deerv1.SnapshotCreated{
				SandboxId:    sandboxID,
				SnapshotId:   result.SnapshotID,
				SnapshotName: result.SnapshotName,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Source VM command handlers
// ---------------------------------------------------------------------------

func (c *Client) handlePrepareSourceVM(ctx context.Context, reqID string, cmd *deerv1.PrepareSourceVMCommand) *deerv1.HostMessage {
	vmName := cmd.GetSourceVm()

	result, err := c.prov.PrepareSourceVM(ctx, vmName, cmd.GetSshUser(), cmd.GetSshKeyPath())
	if err != nil {
		return errorResponse(reqID, "", fmt.Sprintf("prepare source VM %s: %v", vmName, err))
	}

	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_SourceVmPrepared{
			SourceVmPrepared: &deerv1.SourceVMPrepared{
				SourceVm:          result.SourceVM,
				IpAddress:         result.IPAddress,
				Prepared:          result.Prepared,
				UserCreated:       result.UserCreated,
				ShellInstalled:    result.ShellInstalled,
				CaKeyInstalled:    result.CAKeyInstalled,
				SshdConfigured:    result.SSHDConfigured,
				PrincipalsCreated: result.PrincipalsCreated,
				SshdRestarted:     result.SSHDRestarted,
			},
		},
	}
}

func (c *Client) handleRunSourceCommand(ctx context.Context, reqID string, cmd *deerv1.RunSourceCommandCommand) *deerv1.HostMessage {
	vmName := cmd.GetSourceVm()
	command := cmd.GetCommand()

	timeout := time.Duration(cmd.GetTimeoutSeconds()) * time.Second

	result, err := c.prov.RunSourceCommand(ctx, vmName, command, timeout)
	if err != nil {
		return errorResponse(reqID, "", fmt.Sprintf("run source command: %v", err))
	}

	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_SourceCommandResult{
			SourceCommandResult: &deerv1.SourceCommandResult{
				SourceVm: vmName,
				ExitCode: int32(result.ExitCode),
				Stdout:   result.Stdout,
				Stderr:   result.Stderr,
			},
		},
	}
}

func (c *Client) handleReadSourceFile(ctx context.Context, reqID string, cmd *deerv1.ReadSourceFileCommand) *deerv1.HostMessage {
	vmName := cmd.GetSourceVm()

	content, err := c.prov.ReadSourceFile(ctx, vmName, cmd.GetPath())
	if err != nil {
		return errorResponse(reqID, "", fmt.Sprintf("read source file: %v", err))
	}

	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_SourceFileResult{
			SourceFileResult: &deerv1.SourceFileResult{
				SourceVm: vmName,
				Path:     cmd.GetPath(),
				Content:  content,
			},
		},
	}
}

func (c *Client) handleListSourceVMs(ctx context.Context, reqID string) *deerv1.HostMessage {
	vms, err := c.prov.ListSourceVMs(ctx)
	if err != nil {
		return errorResponse(reqID, "", fmt.Sprintf("list VMs: %v", err))
	}

	entries := make([]*deerv1.SourceVMListEntry, len(vms))
	for i, vm := range vms {
		entries[i] = &deerv1.SourceVMListEntry{
			Name:      vm.Name,
			State:     vm.State,
			IpAddress: vm.IPAddress,
			Prepared:  vm.Prepared,
		}
	}

	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_SourceVmsList{
			SourceVmsList: &deerv1.SourceVMsList{
				Vms: entries,
			},
		},
	}
}

func (c *Client) handleValidateSourceVM(ctx context.Context, reqID string, cmd *deerv1.ValidateSourceVMCommand) *deerv1.HostMessage {
	vmName := cmd.GetSourceVm()

	result, err := c.prov.ValidateSourceVM(ctx, vmName)
	if err != nil {
		return errorResponse(reqID, "", fmt.Sprintf("validate source VM: %v", err))
	}

	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_SourceVmValidation{
			SourceVmValidation: &deerv1.SourceVMValidation{
				SourceVm:   result.VMName,
				Valid:      result.Valid,
				State:      result.State,
				MacAddress: result.MACAddress,
				IpAddress:  result.IPAddress,
				HasNetwork: result.HasNetwork,
				Warnings:   result.Warnings,
				Errors:     result.Errors,
			},
		},
	}
}

func (c *Client) attachKafkaDataSources(ctx context.Context, sandboxID, sandboxIP string, dataSources []*deerv1.DataSourceAttachment, fallback []*deerv1.KafkaCaptureConfigBinding) []*deerv1.SandboxKafkaStubInfo {
	attachments := kafkaSandboxAttachmentsFromProto(dataSources, fallback)
	if c.kafkaMgr == nil || len(attachments) == 0 {
		return nil
	}

	for _, attachment := range attachments {
		cfg := attachment.CaptureConfig
		_ = c.localStore.UpsertKafkaCaptureConfig(ctx, kafkaCaptureConfigToLocal(cfg))
	}

	stubs, err := c.kafkaMgr.AttachSandbox(ctx, sandboxID, sandboxBrokerEndpoint(sandboxIP), attachments)
	if err != nil {
		c.logger.Error("attach kafka stubs failed", "sandbox_id", sandboxID, "error", err)
		return nil
	}

	out := make([]*deerv1.SandboxKafkaStubInfo, 0, len(stubs))
	for _, stub := range stubs {
		_ = c.localStore.UpsertSandboxKafkaStub(ctx, sandboxKafkaStubToLocal(stub))
		out = append(out, sandboxKafkaStubToProto(stub))
	}
	return out
}

func (c *Client) detachKafkaStubs(ctx context.Context, sandboxID string) {
	if c.kafkaMgr != nil {
		_ = c.kafkaMgr.DetachSandbox(ctx, sandboxID)
	}
	_ = c.localStore.DeleteSandboxKafkaStubs(ctx, sandboxID)
}

func (c *Client) listKafkaStubs(ctx context.Context, sandboxID string) ([]*deerv1.SandboxKafkaStubInfo, error) {
	if c.kafkaMgr == nil {
		return nil, nil
	}
	stubs, err := c.kafkaMgr.ListSandboxStubs(ctx, sandboxID)
	if err != nil {
		return nil, err
	}
	out := make([]*deerv1.SandboxKafkaStubInfo, 0, len(stubs))
	for _, stub := range stubs {
		_ = c.localStore.UpsertSandboxKafkaStub(ctx, sandboxKafkaStubToLocal(stub))
		out = append(out, sandboxKafkaStubToProto(stub))
	}
	return out, nil
}

func (c *Client) getKafkaStub(ctx context.Context, sandboxID, stubID string) (*deerv1.SandboxKafkaStubInfo, error) {
	if c.kafkaMgr == nil {
		return nil, kafkastub.ErrNotFound
	}
	stub, err := c.kafkaMgr.GetSandboxStub(ctx, sandboxID, stubID)
	if err != nil {
		return nil, err
	}
	_ = c.localStore.UpsertSandboxKafkaStub(ctx, sandboxKafkaStubToLocal(stub))
	return sandboxKafkaStubToProto(stub), nil
}

func (c *Client) transitionKafkaStub(ctx context.Context, sandboxID, stubID, action string) (*deerv1.SandboxKafkaStubInfo, error) {
	if c.kafkaMgr == nil {
		return nil, kafkastub.ErrNotFound
	}
	var (
		stub *kafkastub.SandboxStub
		err  error
	)
	switch action {
	case "start":
		stub, err = c.kafkaMgr.StartSandboxStub(ctx, sandboxID, stubID)
	case "stop":
		stub, err = c.kafkaMgr.StopSandboxStub(ctx, sandboxID, stubID)
	case "restart":
		stub, err = c.kafkaMgr.RestartSandboxStub(ctx, sandboxID, stubID)
	default:
		return nil, fmt.Errorf("unsupported action %q", action)
	}
	if err != nil {
		return nil, err
	}
	_ = c.localStore.UpsertSandboxKafkaStub(ctx, sandboxKafkaStubToLocal(stub))
	return sandboxKafkaStubToProto(stub), nil
}

func (c *Client) handleDiscoverHosts(ctx context.Context, reqID string, cmd *deerv1.DiscoverHostsCommand) *deerv1.HostMessage {
	c.logger.Info("discovering hosts from SSH config")

	hosts, err := sshconfig.Parse(strings.NewReader(cmd.GetSshConfigContent()))
	if err != nil {
		return errorResponse(reqID, "", fmt.Sprintf("parse ssh config: %v", err))
	}

	probeResults := sshconfig.ProbeAll(ctx, hosts)

	discovered := make([]*deerv1.DiscoveredHost, 0, len(probeResults))
	for _, pr := range probeResults {
		discovered = append(discovered, &deerv1.DiscoveredHost{
			Name:         pr.Host.Name,
			Hostname:     pr.Host.HostName,
			User:         pr.Host.User,
			Port:         int32(pr.Host.Port),
			IdentityFile: pr.Host.IdentityFile,
			Reachable:    pr.Reachable,
			HasLibvirt:   pr.HasLibvirt,
			HasProxmox:   pr.HasProxmox,
			Vms:          pr.VMs,
			Error:        pr.Error,
		})
	}

	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_DiscoverHostsResult{
			DiscoverHostsResult: &deerv1.DiscoverHostsResult{
				Hosts: discovered,
			},
		},
	}
}

// buildTLSCredentials constructs gRPC transport credentials from the client's
// TLS configuration. If cert/key are provided, loads a client certificate.
// If a CA file is provided, uses it for server verification. Otherwise falls
// back to the system certificate pool.
func (c *Client) buildTLSCredentials() (credentials.TransportCredentials, error) {
	tlsCfg := &tls.Config{}

	if c.certFile != "" && c.keyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
		if err != nil {
			return nil, fmt.Errorf("load client cert/key: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	if c.caFile != "" {
		caPEM, err := os.ReadFile(c.caFile)
		if err != nil {
			return nil, fmt.Errorf("read CA file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsCfg.RootCAs = pool
	}

	return credentials.NewTLS(tlsCfg), nil
}

// errorResponse builds an ErrorReport HostMessage.
func errorResponse(reqID, sandboxID, errMsg string) *deerv1.HostMessage {
	return &deerv1.HostMessage{
		RequestId: reqID,
		Payload: &deerv1.HostMessage_ErrorReport{
			ErrorReport: &deerv1.ErrorReport{
				Error:     errMsg,
				SandboxId: sandboxID,
			},
		},
	}
}
