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
	"strings"
	"sync"
	"time"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"

	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/provider"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/snapshotpull"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/sshconfig"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/state"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Client connects to the control plane via gRPC bidirectional streaming.
type Client struct {
	hostID   string
	hostname string
	version  string
	cpAddr   string
	insec    bool
	certFile string
	keyFile  string
	caFile   string

	prov       provider.SandboxProvider
	localStore *state.Store
	puller     *snapshotpull.Puller
	logger     *slog.Logger

	// stream is the active bidirectional stream to the control plane.
	mu     sync.Mutex
	stream fluidv1.HostService_ConnectClient
	conn   *grpc.ClientConn

	// sendMu serializes writes to the gRPC stream.
	sendMu sync.Mutex

	// handlerSem bounds the number of concurrent command handler goroutines.
	handlerSem chan struct{}
}

// Config holds configuration for the gRPC agent client.
type Config struct {
	HostID   string
	Hostname string
	Version  string
	Address  string
	Insecure bool
	CertFile string
	KeyFile  string
	CAFile   string
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

	return &Client{
		hostID:     cfg.HostID,
		hostname:   hostname,
		version:    cfg.Version,
		cpAddr:     cfg.Address,
		insec:      cfg.Insecure,
		certFile:   cfg.CertFile,
		keyFile:    cfg.KeyFile,
		caFile:     cfg.CAFile,
		prov:       prov,
		localStore: localStore,
		puller:     puller,
		logger:     logger.With("component", "agent"),
		handlerSem: make(chan struct{}, 64),
	}
}

// sendMessage serializes writes to the gRPC stream.
func (c *Client) sendMessage(stream fluidv1.HostService_ConnectClient, msg *fluidv1.HostMessage) error {
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

	conn, err := grpc.NewClient(c.cpAddr, opts...)
	if err != nil {
		return fmt.Errorf("dial control plane %s: %w", c.cpAddr, err)
	}
	defer func() { _ = conn.Close() }()

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	client := fluidv1.NewHostServiceClient(conn)
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
func (c *Client) register(stream fluidv1.HostService_ConnectClient) error {
	reg := c.buildRegistration()

	reqID := uuid.New().String()
	msg := &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_Registration{
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
func (c *Client) buildRegistration() *fluidv1.HostRegistration {
	reg := &fluidv1.HostRegistration{
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
				reg.SourceVms = append(reg.SourceVms, &fluidv1.SourceVMInfo{
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
func (c *Client) heartbeatLoop(ctx context.Context, stream fluidv1.HostService_ConnectClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hb := &fluidv1.Heartbeat{}

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

			msg := &fluidv1.HostMessage{
				Payload: &fluidv1.HostMessage_Heartbeat{
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
func (c *Client) recvLoop(ctx context.Context, stream fluidv1.HostService_ConnectClient) error {
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
func (c *Client) handleCommand(ctx context.Context, stream fluidv1.HostService_ConnectClient, msg *fluidv1.ControlMessage) {
	reqID := msg.GetRequestId()

	var resp *fluidv1.HostMessage

	switch cmd := msg.Payload.(type) {
	case *fluidv1.ControlMessage_CreateSandbox:
		resp = c.handleCreateSandbox(ctx, reqID, cmd.CreateSandbox)
	case *fluidv1.ControlMessage_DestroySandbox:
		resp = c.handleDestroySandbox(ctx, reqID, cmd.DestroySandbox)
	case *fluidv1.ControlMessage_StartSandbox:
		resp = c.handleStartSandbox(ctx, reqID, cmd.StartSandbox)
	case *fluidv1.ControlMessage_StopSandbox:
		resp = c.handleStopSandbox(ctx, reqID, cmd.StopSandbox)
	case *fluidv1.ControlMessage_RunCommand:
		resp = c.handleRunCommand(ctx, reqID, cmd.RunCommand)
	case *fluidv1.ControlMessage_CreateSnapshot:
		resp = c.handleCreateSnapshot(ctx, reqID, cmd.CreateSnapshot)
	case *fluidv1.ControlMessage_PrepareSourceVm:
		resp = c.handlePrepareSourceVM(ctx, reqID, cmd.PrepareSourceVm)
	case *fluidv1.ControlMessage_RunSourceCommand:
		resp = c.handleRunSourceCommand(ctx, reqID, cmd.RunSourceCommand)
	case *fluidv1.ControlMessage_ReadSourceFile:
		resp = c.handleReadSourceFile(ctx, reqID, cmd.ReadSourceFile)
	case *fluidv1.ControlMessage_ListSourceVms:
		resp = c.handleListSourceVMs(ctx, reqID)
	case *fluidv1.ControlMessage_ValidateSourceVm:
		resp = c.handleValidateSourceVM(ctx, reqID, cmd.ValidateSourceVm)
	case *fluidv1.ControlMessage_DiscoverHosts:
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

func (c *Client) handleCreateSandbox(ctx context.Context, reqID string, cmd *fluidv1.CreateSandboxCommand) *fluidv1.HostMessage {
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
				conn.GetSshUser(), conn.GetSshIdentityFile(), c.logger)
		case "proxmox":
			backend = snapshotpull.NewProxmoxBackend(
				conn.GetProxmoxHost(), conn.GetProxmoxTokenId(),
				conn.GetProxmoxSecret(), conn.GetProxmoxNode(),
				conn.GetProxmoxVerifySsl(), c.logger)
		}
		if backend != nil {
			mode := "cached"
			if cmd.GetSnapshotMode() == fluidv1.SnapshotMode_SNAPSHOT_MODE_FRESH {
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

	c.logger.Info("sandbox created",
		"sandbox_id", sandboxID,
		"ip", result.IPAddress,
		"bridge", result.Bridge,
	)

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SandboxCreated{
			SandboxCreated: &fluidv1.SandboxCreated{
				SandboxId:  sandboxID,
				Name:       result.Name,
				State:      result.State,
				IpAddress:  result.IPAddress,
				MacAddress: result.MACAddress,
				Bridge:     result.Bridge,
				Pid:        int32(result.PID),
			},
		},
	}
}

func (c *Client) handleDestroySandbox(ctx context.Context, reqID string, cmd *fluidv1.DestroySandboxCommand) *fluidv1.HostMessage {
	sandboxID := cmd.GetSandboxId()
	c.logger.Info("destroying sandbox", "sandbox_id", sandboxID)

	if err := c.prov.DestroySandbox(ctx, sandboxID); err != nil {
		c.logger.Error("destroy sandbox failed", "sandbox_id", sandboxID, "error", err)
		return errorResponse(reqID, sandboxID, fmt.Sprintf("destroy failed: %s", err.Error()))
	}

	if err := c.localStore.DeleteSandbox(ctx, sandboxID); err != nil {
		c.logger.Error("delete local sandbox state failed", "sandbox_id", sandboxID, "error", err)
	}

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SandboxDestroyed{
			SandboxDestroyed: &fluidv1.SandboxDestroyed{
				SandboxId: sandboxID,
			},
		},
	}
}

func (c *Client) handleStartSandbox(ctx context.Context, reqID string, cmd *fluidv1.StartSandboxCommand) *fluidv1.HostMessage {
	sandboxID := cmd.GetSandboxId()

	result, err := c.prov.StartSandbox(ctx, sandboxID)
	if err != nil {
		return errorResponse(reqID, sandboxID, fmt.Sprintf("start sandbox: %v", err))
	}

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SandboxStarted{
			SandboxStarted: &fluidv1.SandboxStarted{
				SandboxId: sandboxID,
				State:     result.State,
				IpAddress: result.IPAddress,
			},
		},
	}
}

func (c *Client) handleStopSandbox(ctx context.Context, reqID string, cmd *fluidv1.StopSandboxCommand) *fluidv1.HostMessage {
	sandboxID := cmd.GetSandboxId()

	if err := c.prov.StopSandbox(ctx, sandboxID, cmd.GetForce()); err != nil {
		return errorResponse(reqID, sandboxID, fmt.Sprintf("stop: %v", err))
	}

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SandboxStopped{
			SandboxStopped: &fluidv1.SandboxStopped{
				SandboxId: sandboxID,
				State:     "STOPPED",
			},
		},
	}
}

func (c *Client) handleRunCommand(ctx context.Context, reqID string, cmd *fluidv1.RunCommandCommand) *fluidv1.HostMessage {
	sandboxID := cmd.GetSandboxId()
	command := cmd.GetCommand()

	c.logger.Info("running command", "sandbox_id", sandboxID, "command", command)

	timeout := time.Duration(cmd.GetTimeoutSeconds()) * time.Second

	result, err := c.prov.RunCommand(ctx, sandboxID, command, timeout)
	if err != nil {
		return errorResponse(reqID, sandboxID, fmt.Sprintf("run command: %v", err))
	}

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_CommandResult{
			CommandResult: &fluidv1.CommandResult{
				SandboxId:  sandboxID,
				Stdout:     result.Stdout,
				Stderr:     result.Stderr,
				ExitCode:   int32(result.ExitCode),
				DurationMs: result.DurationMS,
			},
		},
	}
}

func (c *Client) handleCreateSnapshot(ctx context.Context, reqID string, cmd *fluidv1.SnapshotCommand) *fluidv1.HostMessage {
	sandboxID := cmd.GetSandboxId()
	name := cmd.GetSnapshotName()

	result, err := c.prov.CreateSnapshot(ctx, sandboxID, name)
	if err != nil {
		return errorResponse(reqID, sandboxID, fmt.Sprintf("create snapshot: %v", err))
	}

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SnapshotCreated{
			SnapshotCreated: &fluidv1.SnapshotCreated{
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

func (c *Client) handlePrepareSourceVM(ctx context.Context, reqID string, cmd *fluidv1.PrepareSourceVMCommand) *fluidv1.HostMessage {
	vmName := cmd.GetSourceVm()

	result, err := c.prov.PrepareSourceVM(ctx, vmName, cmd.GetSshUser(), cmd.GetSshKeyPath())
	if err != nil {
		return errorResponse(reqID, "", fmt.Sprintf("prepare source VM %s: %v", vmName, err))
	}

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SourceVmPrepared{
			SourceVmPrepared: &fluidv1.SourceVMPrepared{
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

func (c *Client) handleRunSourceCommand(ctx context.Context, reqID string, cmd *fluidv1.RunSourceCommandCommand) *fluidv1.HostMessage {
	vmName := cmd.GetSourceVm()
	command := cmd.GetCommand()

	timeout := time.Duration(cmd.GetTimeoutSeconds()) * time.Second

	result, err := c.prov.RunSourceCommand(ctx, vmName, command, timeout)
	if err != nil {
		return errorResponse(reqID, "", fmt.Sprintf("run source command: %v", err))
	}

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SourceCommandResult{
			SourceCommandResult: &fluidv1.SourceCommandResult{
				SourceVm: vmName,
				ExitCode: int32(result.ExitCode),
				Stdout:   result.Stdout,
				Stderr:   result.Stderr,
			},
		},
	}
}

func (c *Client) handleReadSourceFile(ctx context.Context, reqID string, cmd *fluidv1.ReadSourceFileCommand) *fluidv1.HostMessage {
	vmName := cmd.GetSourceVm()

	content, err := c.prov.ReadSourceFile(ctx, vmName, cmd.GetPath())
	if err != nil {
		return errorResponse(reqID, "", fmt.Sprintf("read source file: %v", err))
	}

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SourceFileResult{
			SourceFileResult: &fluidv1.SourceFileResult{
				SourceVm: vmName,
				Path:     cmd.GetPath(),
				Content:  content,
			},
		},
	}
}

func (c *Client) handleListSourceVMs(ctx context.Context, reqID string) *fluidv1.HostMessage {
	vms, err := c.prov.ListSourceVMs(ctx)
	if err != nil {
		return errorResponse(reqID, "", fmt.Sprintf("list VMs: %v", err))
	}

	entries := make([]*fluidv1.SourceVMListEntry, len(vms))
	for i, vm := range vms {
		entries[i] = &fluidv1.SourceVMListEntry{
			Name:      vm.Name,
			State:     vm.State,
			IpAddress: vm.IPAddress,
			Prepared:  vm.Prepared,
		}
	}

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SourceVmsList{
			SourceVmsList: &fluidv1.SourceVMsList{
				Vms: entries,
			},
		},
	}
}

func (c *Client) handleValidateSourceVM(ctx context.Context, reqID string, cmd *fluidv1.ValidateSourceVMCommand) *fluidv1.HostMessage {
	vmName := cmd.GetSourceVm()

	result, err := c.prov.ValidateSourceVM(ctx, vmName)
	if err != nil {
		return errorResponse(reqID, "", fmt.Sprintf("validate source VM: %v", err))
	}

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_SourceVmValidation{
			SourceVmValidation: &fluidv1.SourceVMValidation{
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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (c *Client) handleDiscoverHosts(ctx context.Context, reqID string, cmd *fluidv1.DiscoverHostsCommand) *fluidv1.HostMessage {
	c.logger.Info("discovering hosts from SSH config")

	hosts, err := sshconfig.Parse(strings.NewReader(cmd.GetSshConfigContent()))
	if err != nil {
		return errorResponse(reqID, "", fmt.Sprintf("parse ssh config: %v", err))
	}

	probeResults := sshconfig.ProbeAll(ctx, hosts)

	discovered := make([]*fluidv1.DiscoveredHost, 0, len(probeResults))
	for _, pr := range probeResults {
		discovered = append(discovered, &fluidv1.DiscoveredHost{
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

	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_DiscoverHostsResult{
			DiscoverHostsResult: &fluidv1.DiscoverHostsResult{
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
func errorResponse(reqID, sandboxID, errMsg string) *fluidv1.HostMessage {
	return &fluidv1.HostMessage{
		RequestId: reqID,
		Payload: &fluidv1.HostMessage_ErrorReport{
			ErrorReport: &fluidv1.ErrorReport{
				Error:     errMsg,
				SandboxId: sandboxID,
			},
		},
	}
}
