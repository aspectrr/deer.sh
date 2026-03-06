// Package daemon implements the DaemonService gRPC server for direct CLI access.
package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"

	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/audit"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/provider"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/redact"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/snapshotpull"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/sshconfig"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/sshkeys"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/state"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/telemetry"

	genid "github.com/aspectrr/fluid.sh/fluid-daemon/internal/id"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements the DaemonServiceServer interface.
type Server struct {
	fluidv1.UnimplementedDaemonServiceServer

	prov            provider.SandboxProvider
	store           *state.Store
	puller          *snapshotpull.Puller
	keyMgr          sshkeys.KeyProvider
	telemetry       telemetry.Service
	redactor        *redact.Redactor
	auditLog        *audit.Logger
	hostID          string
	version         string
	sshIdentityFile string
	caPubKey        string
	logger          *slog.Logger
}

// NewServer creates a new DaemonService server.
func NewServer(prov provider.SandboxProvider, store *state.Store, puller *snapshotpull.Puller, keyMgr sshkeys.KeyProvider, tele telemetry.Service, redactor *redact.Redactor, auditLog *audit.Logger, hostID, version, sshIdentityFile, caPubKey string, logger *slog.Logger) *Server {
	return &Server{
		prov:            prov,
		store:           store,
		puller:          puller,
		keyMgr:          keyMgr,
		telemetry:       tele,
		redactor:        redactor,
		auditLog:        auditLog,
		hostID:          hostID,
		version:         version,
		sshIdentityFile: sshIdentityFile,
		caPubKey:        caPubKey,
		logger:          logger.With("component", "daemon-service"),
	}
}

func (s *Server) CreateSandbox(ctx context.Context, req *fluidv1.CreateSandboxCommand) (*fluidv1.SandboxCreated, error) {
	start := time.Now()
	s.telemetry.Track("daemon_sandbox_created", nil)
	s.logger.Info("CreateSandbox", "base_image", req.GetBaseImage(), "source_vm", req.GetSourceVm(), "name", req.GetName())

	sandboxID := req.GetSandboxId()
	if sandboxID == "" {
		var err error
		sandboxID, err = genid.Generate("sbx-")
		if err != nil {
			return nil, status.Errorf(codes.Internal, "generate sandbox ID: %v", err)
		}
	}

	vcpus := int(req.GetVcpus())
	if vcpus == 0 {
		vcpus = 2
	}
	memMB := int(req.GetMemoryMb())
	if memMB == 0 {
		memMB = 2048
	}

	// If a source host connection is provided, snapshot+pull the image first
	baseImage := req.GetBaseImage()
	if conn := req.GetSourceHostConnection(); conn != nil && req.GetSourceVm() != "" && s.puller != nil {
		var backend snapshotpull.SnapshotBackend
		switch conn.GetType() {
		case "libvirt":
			backend = snapshotpull.NewLibvirtBackend(
				conn.GetSshHost(), int(conn.GetSshPort()),
				conn.GetSshUser(), s.sshIdentityFile, "qemu:///system", s.logger)
		case "proxmox":
			backend = snapshotpull.NewProxmoxBackend(
				conn.GetProxmoxHost(), conn.GetProxmoxTokenId(),
				conn.GetProxmoxSecret(), conn.GetProxmoxNode(),
				conn.GetProxmoxVerifySsl(), s.logger)
		}
		if backend != nil {
			mode := "cached"
			if req.GetSnapshotMode() == fluidv1.SnapshotMode_SNAPSHOT_MODE_FRESH {
				mode = "fresh"
			}
			pullResult, err := s.puller.Pull(ctx, snapshotpull.PullRequest{
				SourceHost:   conn.GetSshHost(),
				VMName:       req.GetSourceVm(),
				SnapshotMode: mode,
			}, backend)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "pull snapshot: %v", err)
			}
			baseImage = pullResult.ImageName
			s.logger.Info("snapshot pulled", "image", baseImage, "cached", pullResult.Cached)
		}
	}

	result, err := s.prov.CreateSandbox(ctx, provider.CreateRequest{
		SandboxID:    sandboxID,
		Name:         req.GetName(),
		BaseImage:    baseImage,
		SourceVM:     req.GetSourceVm(),
		Network:      req.GetNetwork(),
		VCPUs:        vcpus,
		MemoryMB:     memMB,
		TTLSeconds:   int(req.GetTtlSeconds()),
		AgentID:      req.GetAgentId(),
		SSHPublicKey: req.GetSshPublicKey(),
	})
	if err != nil {
		s.logger.Error("CreateSandbox failed", "error", err)
		return nil, status.Errorf(codes.Internal, "create sandbox: %v", err)
	}

	// Persist to state store
	now := time.Now().UTC()
	sb := &state.Sandbox{
		ID:         result.SandboxID,
		Name:       result.Name,
		AgentID:    req.GetAgentId(),
		BaseImage:  baseImage,
		Bridge:     result.Bridge,
		MACAddress: result.MACAddress,
		IPAddress:  result.IPAddress,
		State:      result.State,
		PID:        result.PID,
		VCPUs:      vcpus,
		MemoryMB:   memMB,
		TTLSeconds: int(req.GetTtlSeconds()),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.store.CreateSandbox(ctx, sb); err != nil {
		s.logger.Warn("failed to persist sandbox state", "sandbox_id", result.SandboxID, "error", err)
	}

	s.logAudit(audit.TypeSandboxCreated, map[string]any{
		"sandbox_id": result.SandboxID,
		"source_vm":  req.GetSourceVm(),
		"vcpus":      vcpus,
		"memory_mb":  memMB,
	}, nil, time.Since(start).Milliseconds())

	return &fluidv1.SandboxCreated{
		SandboxId:  result.SandboxID,
		Name:       result.Name,
		State:      result.State,
		IpAddress:  result.IPAddress,
		MacAddress: result.MACAddress,
		Bridge:     result.Bridge,
		Pid:        int32(result.PID),
	}, nil
}

func (s *Server) GetSandbox(ctx context.Context, req *fluidv1.GetSandboxRequest) (*fluidv1.SandboxInfo, error) {
	id := req.GetSandboxId()
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "sandbox_id is required")
	}

	sb, err := s.store.GetSandbox(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "sandbox not found: %v", err)
	}

	return sandboxToInfo(sb), nil
}

func (s *Server) ListSandboxes(ctx context.Context, _ *fluidv1.ListSandboxesRequest) (*fluidv1.ListSandboxesResponse, error) {
	sandboxes, err := s.store.ListSandboxes(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list sandboxes: %v", err)
	}

	infos := make([]*fluidv1.SandboxInfo, 0, len(sandboxes))
	for _, sb := range sandboxes {
		infos = append(infos, sandboxToInfo(sb))
	}

	return &fluidv1.ListSandboxesResponse{
		Sandboxes: infos,
		Count:     int32(len(infos)),
	}, nil
}

func (s *Server) DestroySandbox(ctx context.Context, req *fluidv1.DestroySandboxCommand) (*fluidv1.SandboxDestroyed, error) {
	start := time.Now()
	s.telemetry.Track("daemon_sandbox_destroyed", nil)

	id := req.GetSandboxId()
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "sandbox_id is required")
	}

	if err := s.prov.DestroySandbox(ctx, id); err != nil {
		s.logger.Error("DestroySandbox failed", "sandbox_id", id, "error", err)
		return nil, status.Errorf(codes.Internal, "destroy sandbox: %v", err)
	}

	if err := s.store.DeleteSandbox(ctx, id); err != nil {
		s.logger.Warn("failed to delete sandbox from store", "sandbox_id", id, "error", err)
	}

	s.logAudit(audit.TypeSandboxDestroyed, map[string]any{
		"sandbox_id": id,
	}, nil, time.Since(start).Milliseconds())

	return &fluidv1.SandboxDestroyed{SandboxId: id}, nil
}

func (s *Server) StartSandbox(ctx context.Context, req *fluidv1.StartSandboxCommand) (*fluidv1.SandboxStarted, error) {
	start := time.Now()
	s.telemetry.Track("daemon_sandbox_started", nil)

	id := req.GetSandboxId()
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "sandbox_id is required")
	}

	result, err := s.prov.StartSandbox(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "start sandbox: %v", err)
	}

	// Update state
	if sb, err := s.store.GetSandbox(ctx, id); err == nil {
		sb.State = result.State
		sb.IPAddress = result.IPAddress
		sb.UpdatedAt = time.Now().UTC()
		if err := s.store.UpdateSandbox(ctx, sb); err != nil {
			s.logger.Warn("failed to update sandbox state", "sandbox_id", id, "error", err)
		}
	}

	s.logAudit(audit.TypeSandboxStarted, map[string]any{
		"sandbox_id": id,
	}, nil, time.Since(start).Milliseconds())

	return &fluidv1.SandboxStarted{
		SandboxId: id,
		State:     result.State,
		IpAddress: result.IPAddress,
	}, nil
}

func (s *Server) StopSandbox(ctx context.Context, req *fluidv1.StopSandboxCommand) (*fluidv1.SandboxStopped, error) {
	start := time.Now()
	s.telemetry.Track("daemon_sandbox_stopped", nil)

	id := req.GetSandboxId()
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "sandbox_id is required")
	}

	if err := s.prov.StopSandbox(ctx, id, req.GetForce()); err != nil {
		return nil, status.Errorf(codes.Internal, "stop sandbox: %v", err)
	}

	// Update state
	if sb, err := s.store.GetSandbox(ctx, id); err == nil {
		sb.State = "STOPPED"
		sb.UpdatedAt = time.Now().UTC()
		if err := s.store.UpdateSandbox(ctx, sb); err != nil {
			s.logger.Warn("failed to update sandbox state", "sandbox_id", id, "error", err)
		}
	}

	s.logAudit(audit.TypeSandboxStopped, map[string]any{
		"sandbox_id": id,
	}, nil, time.Since(start).Milliseconds())

	return &fluidv1.SandboxStopped{
		SandboxId: id,
		State:     "STOPPED",
	}, nil
}

func (s *Server) RunCommand(ctx context.Context, req *fluidv1.RunCommandCommand) (*fluidv1.CommandResult, error) {
	start := time.Now()
	s.telemetry.Track("daemon_command_executed", nil)

	id := req.GetSandboxId()
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "sandbox_id is required")
	}
	if req.GetCommand() == "" {
		return nil, status.Error(codes.InvalidArgument, "command is required")
	}

	timeout := time.Duration(req.GetTimeoutSeconds()) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	result, err := s.prov.RunCommand(ctx, id, req.GetCommand(), timeout)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "run command: %v", err)
	}

	// Record command in state
	cmdID, _ := genid.GenerateRaw()
	cmdRecord := &state.Command{
		ID:         cmdID,
		SandboxID:  id,
		Command:    req.GetCommand(),
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
		ExitCode:   result.ExitCode,
		DurationMS: result.DurationMS,
		StartedAt:  time.Now().UTC().Add(-time.Duration(result.DurationMS) * time.Millisecond),
		EndedAt:    time.Now().UTC(),
	}
	_ = s.store.CreateCommand(ctx, cmdRecord)

	s.logAudit(audit.TypeCommandExecuted, map[string]any{
		"sandbox_id": id,
		"command":    req.GetCommand(),
		"exit_code":  result.ExitCode,
	}, nil, time.Since(start).Milliseconds())

	return &fluidv1.CommandResult{
		SandboxId:  id,
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
		ExitCode:   int32(result.ExitCode),
		DurationMs: result.DurationMS,
	}, nil
}

func (s *Server) CreateSnapshot(ctx context.Context, req *fluidv1.SnapshotCommand) (*fluidv1.SnapshotCreated, error) {
	start := time.Now()
	s.telemetry.Track("daemon_snapshot_created", nil)

	id := req.GetSandboxId()
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "sandbox_id is required")
	}

	name := req.GetSnapshotName()
	if name == "" {
		name = fmt.Sprintf("snap-%d", time.Now().Unix())
	}

	result, err := s.prov.CreateSnapshot(ctx, id, name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create snapshot: %v", err)
	}

	s.logAudit(audit.TypeSnapshotCreated, map[string]any{
		"sandbox_id":    id,
		"snapshot_name": result.SnapshotName,
	}, nil, time.Since(start).Milliseconds())

	return &fluidv1.SnapshotCreated{
		SandboxId:    id,
		SnapshotId:   result.SnapshotID,
		SnapshotName: result.SnapshotName,
	}, nil
}

func (s *Server) ListSourceVMs(ctx context.Context, req *fluidv1.ListSourceVMsCommand) (*fluidv1.SourceVMsList, error) {
	if conn := req.GetSourceHostConnection(); conn != nil {
		adhoc, err := s.adhocSourceVMManager(conn)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "create provider for host: %v", err)
		}
		vms, err := adhoc.ListVMs(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "list source VMs: %v", err)
		}
		entries := make([]*fluidv1.SourceVMListEntry, 0, len(vms))
		for _, vm := range vms {
			entries = append(entries, &fluidv1.SourceVMListEntry{
				Name:      vm.Name,
				State:     vm.State,
				IpAddress: vm.IPAddress,
				Prepared:  vm.Prepared,
			})
		}
		return &fluidv1.SourceVMsList{Vms: entries}, nil
	}

	vms, err := s.prov.ListSourceVMs(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list source VMs: %v", err)
	}

	entries := make([]*fluidv1.SourceVMListEntry, 0, len(vms))
	for _, vm := range vms {
		entries = append(entries, &fluidv1.SourceVMListEntry{
			Name:      vm.Name,
			State:     vm.State,
			IpAddress: vm.IPAddress,
			Prepared:  vm.Prepared,
		})
	}

	return &fluidv1.SourceVMsList{Vms: entries}, nil
}

func (s *Server) ValidateSourceVM(ctx context.Context, req *fluidv1.ValidateSourceVMCommand) (*fluidv1.SourceVMValidation, error) {
	if req.GetSourceVm() == "" {
		return nil, status.Error(codes.InvalidArgument, "source_vm is required")
	}

	if conn := req.GetSourceHostConnection(); conn != nil {
		adhoc, err := s.adhocSourceVMManager(conn)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "create provider for host: %v", err)
		}
		result, err := adhoc.ValidateSourceVM(ctx, req.GetSourceVm())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "validate source VM: %v", err)
		}
		return &fluidv1.SourceVMValidation{
			SourceVm:   result.VMName,
			Valid:      result.Valid,
			State:      result.State,
			MacAddress: result.MACAddress,
			IpAddress:  result.IPAddress,
			HasNetwork: result.HasNetwork,
			Warnings:   result.Warnings,
			Errors:     result.Errors,
		}, nil
	}

	result, err := s.prov.ValidateSourceVM(ctx, req.GetSourceVm())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "validate source VM: %v", err)
	}

	return &fluidv1.SourceVMValidation{
		SourceVm:   result.VMName,
		Valid:      result.Valid,
		State:      result.State,
		MacAddress: result.MACAddress,
		IpAddress:  result.IPAddress,
		HasNetwork: result.HasNetwork,
		Warnings:   result.Warnings,
		Errors:     result.Errors,
	}, nil
}

func (s *Server) PrepareSourceVM(ctx context.Context, req *fluidv1.PrepareSourceVMCommand) (*fluidv1.SourceVMPrepared, error) {
	if req.GetSourceVm() == "" {
		return nil, status.Error(codes.InvalidArgument, "source_vm is required")
	}

	if conn := req.GetSourceHostConnection(); conn != nil {
		adhoc, err := s.adhocSourceVMManager(conn)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "create provider for host: %v", err)
		}
		result, err := adhoc.PrepareSourceVM(ctx, req.GetSourceVm(), req.GetSshUser(), req.GetSshKeyPath())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "prepare source VM: %v", err)
		}
		return &fluidv1.SourceVMPrepared{
			SourceVm:          result.SourceVM,
			IpAddress:         result.IPAddress,
			Prepared:          result.Prepared,
			UserCreated:       result.UserCreated,
			ShellInstalled:    result.ShellInstalled,
			CaKeyInstalled:    result.CAKeyInstalled,
			SshdConfigured:    result.SSHDConfigured,
			PrincipalsCreated: result.PrincipalsCreated,
			SshdRestarted:     result.SSHDRestarted,
		}, nil
	}

	result, err := s.prov.PrepareSourceVM(ctx, req.GetSourceVm(), req.GetSshUser(), req.GetSshKeyPath())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "prepare source VM: %v", err)
	}

	return &fluidv1.SourceVMPrepared{
		SourceVm:          result.SourceVM,
		IpAddress:         result.IPAddress,
		Prepared:          result.Prepared,
		UserCreated:       result.UserCreated,
		ShellInstalled:    result.ShellInstalled,
		CaKeyInstalled:    result.CAKeyInstalled,
		SshdConfigured:    result.SSHDConfigured,
		PrincipalsCreated: result.PrincipalsCreated,
		SshdRestarted:     result.SSHDRestarted,
	}, nil
}

func (s *Server) RunSourceCommand(ctx context.Context, req *fluidv1.RunSourceCommandCommand) (*fluidv1.SourceCommandResult, error) {
	start := time.Now()
	if req.GetSourceVm() == "" {
		return nil, status.Error(codes.InvalidArgument, "source_vm is required")
	}
	if req.GetCommand() == "" {
		return nil, status.Error(codes.InvalidArgument, "command is required")
	}

	timeout := time.Duration(req.GetTimeoutSeconds()) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	if conn := req.GetSourceHostConnection(); conn != nil {
		adhoc, err := s.adhocSourceVMManager(conn)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "create provider for host: %v", err)
		}
		stdout, stderr, exitCode, err := adhoc.RunSourceCommand(ctx, req.GetSourceVm(), req.GetCommand(), timeout)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "run source command: %v", err)
		}
		s.logAudit(audit.TypeSourceCommand, map[string]any{
			"source_vm": req.GetSourceVm(),
			"command":   req.GetCommand(),
		}, nil, time.Since(start).Milliseconds())
		return &fluidv1.SourceCommandResult{
			SourceVm: req.GetSourceVm(),
			ExitCode: int32(exitCode),
			Stdout:   stdout,
			Stderr:   stderr,
		}, nil
	}

	result, err := s.prov.RunSourceCommand(ctx, req.GetSourceVm(), req.GetCommand(), timeout)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "run source command: %v", err)
	}

	s.logAudit(audit.TypeSourceCommand, map[string]any{
		"source_vm": req.GetSourceVm(),
		"command":   req.GetCommand(),
	}, nil, time.Since(start).Milliseconds())

	return &fluidv1.SourceCommandResult{
		SourceVm: req.GetSourceVm(),
		ExitCode: int32(result.ExitCode),
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
	}, nil
}

func (s *Server) ReadSourceFile(ctx context.Context, req *fluidv1.ReadSourceFileCommand) (*fluidv1.SourceFileResult, error) {
	start := time.Now()
	if req.GetSourceVm() == "" {
		return nil, status.Error(codes.InvalidArgument, "source_vm is required")
	}
	if req.GetPath() == "" {
		return nil, status.Error(codes.InvalidArgument, "path is required")
	}

	if conn := req.GetSourceHostConnection(); conn != nil {
		adhoc, err := s.adhocSourceVMManager(conn)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "create provider for host: %v", err)
		}
		content, err := adhoc.ReadSourceFile(ctx, req.GetSourceVm(), req.GetPath())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "read source file: %v", err)
		}
		s.logAudit(audit.TypeFileRead, map[string]any{
			"source_vm": req.GetSourceVm(),
			"path":      req.GetPath(),
		}, nil, time.Since(start).Milliseconds())
		return &fluidv1.SourceFileResult{
			SourceVm: req.GetSourceVm(),
			Path:     req.GetPath(),
			Content:  content,
		}, nil
	}

	content, err := s.prov.ReadSourceFile(ctx, req.GetSourceVm(), req.GetPath())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "read source file: %v", err)
	}

	s.logAudit(audit.TypeFileRead, map[string]any{
		"source_vm": req.GetSourceVm(),
		"path":      req.GetPath(),
	}, nil, time.Since(start).Milliseconds())

	return &fluidv1.SourceFileResult{
		SourceVm: req.GetSourceVm(),
		Path:     req.GetPath(),
		Content:  content,
	}, nil
}

func (s *Server) GetHostInfo(ctx context.Context, _ *fluidv1.GetHostInfoRequest) (*fluidv1.HostInfoResponse, error) {
	hostname, _ := os.Hostname()

	caps, err := s.prov.Capabilities(ctx)
	if err != nil {
		s.logger.Warn("failed to get capabilities", "error", err)
	}

	resp := &fluidv1.HostInfoResponse{
		HostId:          s.hostID,
		Hostname:        hostname,
		Version:         s.version,
		ActiveSandboxes: int32(s.prov.ActiveSandboxCount()),
		SshCaPubKey:     s.caPubKey,
	}

	if caps != nil {
		resp.TotalCpus = int32(caps.TotalCPUs)
		resp.TotalMemoryMb = int64(caps.TotalMemoryMB)
		resp.BaseImages = caps.BaseImages
	}

	return resp, nil
}

func (s *Server) Health(_ context.Context, _ *fluidv1.HealthRequest) (*fluidv1.HealthResponse, error) {
	return &fluidv1.HealthResponse{Status: "ok"}, nil
}

func (s *Server) DiscoverHosts(ctx context.Context, req *fluidv1.DiscoverHostsCommand) (*fluidv1.DiscoverHostsResult, error) {
	s.logger.Info("DiscoverHosts", "config_length", len(req.GetSshConfigContent()))

	hosts, err := sshconfig.Parse(strings.NewReader(req.GetSshConfigContent()))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "parse ssh config: %v", err)
	}

	if len(hosts) == 0 {
		return &fluidv1.DiscoverHostsResult{}, nil
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

	return &fluidv1.DiscoverHostsResult{Hosts: discovered}, nil
}

// logAudit records an operation to the audit log with redaction.
func (s *Server) logAudit(opType string, meta map[string]any, err error, durationMs int64) {
	if s.auditLog == nil {
		return
	}
	if s.redactor != nil {
		meta = s.redactor.RedactMap(meta)
	}
	s.auditLog.LogOperation(opType, meta, err, durationMs)
}

// sandboxToInfo converts a state.Sandbox to a proto SandboxInfo.
func sandboxToInfo(sb *state.Sandbox) *fluidv1.SandboxInfo {
	return &fluidv1.SandboxInfo{
		SandboxId: sb.ID,
		Name:      sb.Name,
		State:     sb.State,
		IpAddress: sb.IPAddress,
		BaseImage: sb.BaseImage,
		AgentId:   sb.AgentID,
		Vcpus:     int32(sb.VCPUs),
		MemoryMb:  int32(sb.MemoryMB),
		CreatedAt: sb.CreatedAt.Format(time.RFC3339),
	}
}
