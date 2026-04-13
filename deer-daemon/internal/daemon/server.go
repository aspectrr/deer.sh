// Package daemon implements the DaemonService gRPC server for direct CLI access.
package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	deerv1 "github.com/aspectrr/deer.sh/proto/gen/go/deer/v1"

	"github.com/aspectrr/deer.sh/deer-daemon/internal/audit"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/config"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/kafkastub"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/provider"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/redact"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/snapshotpull"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/sshconfig"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/sshkeys"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/state"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/telemetry"

	genid "github.com/aspectrr/deer.sh/deer-daemon/internal/id"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const createSandboxStreamTotalSteps = 9

type sandboxCreatePuller interface {
	Pull(context.Context, snapshotpull.PullRequest, snapshotpull.SnapshotBackend) (*snapshotpull.PullResult, error)
}

type sandboxCreateProgressProvider interface {
	CreateSandboxWithProgress(context.Context, provider.CreateRequest, func(string, int, int)) (*provider.SandboxResult, error)
}

// Server implements the DaemonServiceServer interface.
type Server struct {
	deerv1.UnimplementedDaemonServiceServer

	cfg                      *config.Config
	prov                     provider.SandboxProvider
	store                    *state.Store
	puller                   sandboxCreatePuller
	keyMgr                   sshkeys.KeyProvider
	telemetry                telemetry.Service
	redactor                 *redact.Redactor
	auditLog                 *audit.Logger
	hostID                   string
	version                  string
	sshIdentityFile          string
	caPubKey                 string
	identityPubKey           string
	logger                   *slog.Logger
	kafkaMgr                 *kafkastub.Manager
	attachKafkaDataSourcesFn func(context.Context, string, string, []*deerv1.DataSourceAttachment, []*deerv1.KafkaCaptureConfigBinding) ([]*deerv1.SandboxKafkaStubInfo, error)

	vmHostMu    sync.RWMutex
	vmHostCache map[string]*deerv1.SourceHostConnection // VM name -> host connection
}

// NewServer creates a new DaemonService server.
func NewServer(cfg *config.Config, prov provider.SandboxProvider, store *state.Store, puller *snapshotpull.Puller, keyMgr sshkeys.KeyProvider, tele telemetry.Service, redactor *redact.Redactor, auditLog *audit.Logger, hostID, version, sshIdentityFile, caPubKey, identityPubKey string, logger *slog.Logger) *Server {
	kafkaBaseDir := filepath.Join(filepath.Dir(cfg.State.DBPath), "kafka-stub")
	kafkaMgr, err := newKafkaManager(kafkaBaseDir, redactor, logger, store)
	if err != nil && logger != nil {
		logger.Warn("failed to initialize kafka stub manager", "error", err)
	}
	return &Server{
		cfg:             cfg,
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
		identityPubKey:  identityPubKey,
		logger:          logger.With("component", "daemon-service"),
		kafkaMgr:        kafkaMgr,
		vmHostCache:     make(map[string]*deerv1.SourceHostConnection),
	}
}

func (s *Server) sendSandboxCreateProgress(stream deerv1.DaemonService_CreateSandboxStreamServer, sandboxID string, stepNum int, step string) error {
	return stream.Send(&deerv1.SandboxProgress{
		SandboxId:  sandboxID,
		Step:       step,
		StepNum:    int32(stepNum),
		TotalSteps: createSandboxStreamTotalSteps,
	})
}

func (s *Server) sendSandboxCreateError(stream deerv1.DaemonService_CreateSandboxStreamServer, sandboxID string, err error) {
	if err == nil {
		return
	}
	_ = stream.Send(&deerv1.SandboxProgress{
		SandboxId: sandboxID,
		Error:     err.Error(),
		Done:      true,
	})
}

func (s *Server) persistCreatedSandbox(ctx context.Context, result *provider.SandboxResult, req *deerv1.CreateSandboxCommand, baseImage string, vcpus, memMB int) {
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
}

func (s *Server) providerCreateRequest(req *deerv1.CreateSandboxCommand, sandboxID, baseImage string, vcpus, memMB int) provider.CreateRequest {
	createReq := provider.CreateRequest{
		SandboxID:           sandboxID,
		Name:                req.GetName(),
		BaseImage:           baseImage,
		SourceVM:            req.GetSourceVm(),
		Network:             req.GetNetwork(),
		VCPUs:               vcpus,
		MemoryMB:            memMB,
		TTLSeconds:          int(req.GetTtlSeconds()),
		AgentID:             req.GetAgentId(),
		SSHPublicKey:        req.GetSshPublicKey(),
		DataSources:         providerDataSourcesFromProto(req.GetDataSources(), req.GetKafkaCaptureConfigs()),
		KafkaBroker:         kafkaBrokerConfigForDataSources(req.GetDataSources(), req.GetKafkaCaptureConfigs(), req.GetSimpleKafkaBroker()),
		ElasticsearchBroker: elasticsearchBrokerConfig(req.GetSimpleElasticsearchBroker()),
	}
	normalized, clamped := provider.NormalizeCreateRequestResources(createReq, provider.DefaultSandboxVCPUs, provider.DefaultSandboxMemMB)
	if clamped {
		s.logger.Info("clamped sandbox resources",
			"sandbox_id", sandboxID,
			"effective_vcpus", normalized.VCPUs,
			"effective_memory_mb", normalized.MemoryMB,
		)
	}
	return normalized
}

func (s *Server) rollbackCreateFailure(ctx context.Context, sandboxID string) error {
	var errs []string
	s.removeKafkaStubs(ctx, sandboxID)
	if s.prov != nil {
		if err := s.prov.DestroySandbox(ctx, sandboxID); err != nil {
			errs = append(errs, fmt.Sprintf("destroy sandbox: %v", err))
		}
	}
	if s.store != nil {
		if err := s.store.DeleteSandbox(ctx, sandboxID); err != nil {
			errs = append(errs, fmt.Sprintf("delete sandbox state: %v", err))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(errs, "; "))
}

func (s *Server) attachKafkaDataSourcesForCreate(ctx context.Context, result *provider.SandboxResult, req *deerv1.CreateSandboxCommand) ([]*deerv1.SandboxKafkaStubInfo, error) {
	attachFn := s.attachKafkaDataSources
	if s.attachKafkaDataSourcesFn != nil {
		attachFn = s.attachKafkaDataSourcesFn
	}
	kafkaStubs, err := attachFn(ctx, result.SandboxID, result.IPAddress, req.GetDataSources(), req.GetKafkaCaptureConfigs())
	if err == nil {
		return kafkaStubs, nil
	}
	cleanupErr := s.rollbackCreateFailure(ctx, result.SandboxID)
	if cleanupErr != nil {
		return nil, fmt.Errorf("attach kafka data sources: %w (cleanup: %v)", err, cleanupErr)
	}
	return nil, fmt.Errorf("attach kafka data sources: %w", err)
}

func (s *Server) CreateSandbox(ctx context.Context, req *deerv1.CreateSandboxCommand) (*deerv1.SandboxCreated, error) {
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

	// Resolve source host connection: use provided, or resolve from config
	baseImage := req.GetBaseImage()
	conn := req.GetSourceHostConnection()
	if conn == nil && req.GetSourceVm() != "" && s.puller != nil && len(s.cfg.SourceHosts) > 0 {
		resolved, err := s.resolveSourceHost(ctx, req.GetSourceVm())
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "resolve source host: %v", err)
		}
		conn = resolved
	}
	if conn != nil && req.GetSourceVm() != "" && s.puller != nil {
		var backend snapshotpull.SnapshotBackend
		switch conn.GetType() {
		case "libvirt":
			backend = snapshotpull.NewLibvirtBackend(
				conn.GetSshHost(), int(conn.GetSshPort()),
				conn.GetSshUser(), s.sshIdentityFile, s.logger)
		case "proxmox":
			backend = snapshotpull.NewProxmoxBackend(
				conn.GetProxmoxHost(), conn.GetProxmoxTokenId(),
				conn.GetProxmoxSecret(), conn.GetProxmoxNode(),
				conn.GetProxmoxVerifySsl(), s.logger)
		}
		if backend != nil {
			mode := "cached"
			if req.GetSnapshotMode() == deerv1.SnapshotMode_SNAPSHOT_MODE_FRESH {
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

	createReq := s.providerCreateRequest(req, sandboxID, baseImage, vcpus, memMB)
	result, err := s.prov.CreateSandbox(ctx, createReq)
	if err != nil {
		s.logger.Error("CreateSandbox failed", "error", err)
		return nil, status.Errorf(codes.Internal, "create sandbox: %v", err)
	}

	s.persistCreatedSandbox(ctx, result, req, baseImage, createReq.VCPUs, createReq.MemoryMB)
	kafkaStubs, err := s.attachKafkaDataSourcesForCreate(ctx, result, req)
	if err != nil {
		s.logger.Error("CreateSandbox kafka attach failed", "sandbox_id", result.SandboxID, "error", err)
		return nil, status.Errorf(codes.Internal, "create sandbox: %v", err)
	}

	s.logAudit(audit.TypeSandboxCreated, map[string]any{
		"sandbox_id": result.SandboxID,
		"source_vm":  req.GetSourceVm(),
		"vcpus":      createReq.VCPUs,
		"memory_mb":  createReq.MemoryMB,
	}, nil, time.Since(start).Milliseconds())

	return &deerv1.SandboxCreated{
		SandboxId:  result.SandboxID,
		Name:       result.Name,
		State:      result.State,
		IpAddress:  result.IPAddress,
		MacAddress: result.MACAddress,
		Bridge:     result.Bridge,
		Pid:        int32(result.PID),
		KafkaStubs: kafkaStubs,
	}, nil
}

func (s *Server) CreateSandboxStream(req *deerv1.CreateSandboxCommand, stream deerv1.DaemonService_CreateSandboxStreamServer) error {
	ctx := stream.Context()
	start := time.Now()
	s.telemetry.Track("daemon_sandbox_created_stream", nil)
	s.logger.Info("CreateSandboxStream", "base_image", req.GetBaseImage(), "source_vm", req.GetSourceVm(), "name", req.GetName())

	sandboxID := req.GetSandboxId()
	if sandboxID == "" {
		var err error
		sandboxID, err = genid.Generate("sbx-")
		if err != nil {
			return status.Errorf(codes.Internal, "generate sandbox ID: %v", err)
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

	// Resolve source host connection: use provided, or resolve from config
	baseImage := req.GetBaseImage()
	conn := req.GetSourceHostConnection()
	switch {
	case conn != nil:
		if err := s.sendSandboxCreateProgress(stream, sandboxID, 1, "Using provided source host"); err != nil {
			return err
		}
	case req.GetSourceVm() != "" && s.puller != nil && len(s.cfg.SourceHosts) > 0:
		if err := s.sendSandboxCreateProgress(stream, sandboxID, 1, "Resolving source host"); err != nil {
			return err
		}
		resolved, err := s.resolveSourceHost(ctx, req.GetSourceVm())
		if err != nil {
			s.sendSandboxCreateError(stream, sandboxID, err)
			return status.Errorf(codes.NotFound, "resolve source host: %v", err)
		}
		conn = resolved
	default:
		if err := s.sendSandboxCreateProgress(stream, sandboxID, 1, "No source host resolution needed"); err != nil {
			return err
		}
	}

	backend := snapshotpull.SnapshotBackend(nil)
	if conn != nil && req.GetSourceVm() != "" && s.puller != nil {
		switch conn.GetType() {
		case "libvirt":
			backend = snapshotpull.NewLibvirtBackend(
				conn.GetSshHost(), int(conn.GetSshPort()),
				conn.GetSshUser(), s.sshIdentityFile, s.logger)
		case "proxmox":
			backend = snapshotpull.NewProxmoxBackend(
				conn.GetProxmoxHost(), conn.GetProxmoxTokenId(),
				conn.GetProxmoxSecret(), conn.GetProxmoxNode(),
				conn.GetProxmoxVerifySsl(), s.logger)
		}
	}

	if backend != nil {
		stepLabel := "Preparing base image"
		mode := "cached"
		if req.GetSnapshotMode() == deerv1.SnapshotMode_SNAPSHOT_MODE_FRESH {
			mode = "fresh"
			stepLabel = "Pulling fresh snapshot"
		}
		if err := s.sendSandboxCreateProgress(stream, sandboxID, 2, stepLabel); err != nil {
			return err
		}
		pullResult, err := s.puller.Pull(ctx, snapshotpull.PullRequest{
			SourceHost:   conn.GetSshHost(),
			VMName:       req.GetSourceVm(),
			SnapshotMode: mode,
		}, backend)
		if err != nil {
			s.sendSandboxCreateError(stream, sandboxID, err)
			return status.Errorf(codes.Internal, "pull snapshot: %v", err)
		}
		baseImage = pullResult.ImageName
		s.logger.Info("snapshot pulled", "image", baseImage, "cached", pullResult.Cached)
	} else {
		if err := s.sendSandboxCreateProgress(stream, sandboxID, 2, "Using requested base image"); err != nil {
			return err
		}
	}

	// Register for readiness signaling if supported
	if rp, ok := s.prov.(sandboxCreateProgressProvider); ok {
		// Use streaming provider
		createReq := s.providerCreateRequest(req, sandboxID, baseImage, vcpus, memMB)
		result, err := rp.CreateSandboxWithProgress(ctx, createReq, func(step string, stepNum, total int) {
			_ = s.sendSandboxCreateProgress(stream, sandboxID, stepNum+2, step)
		})
		if err != nil {
			s.logger.Error("CreateSandboxStream failed", "error", err)
			s.sendSandboxCreateError(stream, sandboxID, err)
			return status.Errorf(codes.Internal, "create sandbox: %v", err)
		}

		s.persistCreatedSandbox(ctx, result, req, baseImage, createReq.VCPUs, createReq.MemoryMB)
		kafkaStubs, err := s.attachKafkaDataSourcesForCreate(ctx, result, req)
		if err != nil {
			s.logger.Error("CreateSandboxStream kafka attach failed", "sandbox_id", result.SandboxID, "error", err)
			s.sendSandboxCreateError(stream, sandboxID, err)
			return status.Errorf(codes.Internal, "create sandbox: %v", err)
		}

		s.logAudit(audit.TypeSandboxCreated, map[string]any{
			"sandbox_id": result.SandboxID,
			"source_vm":  req.GetSourceVm(),
			"vcpus":      createReq.VCPUs,
			"memory_mb":  createReq.MemoryMB,
		}, nil, time.Since(start).Milliseconds())

		// Send final done message
		return stream.Send(&deerv1.SandboxProgress{
			SandboxId: sandboxID,
			Done:      true,
			Result: &deerv1.SandboxCreated{
				SandboxId:  result.SandboxID,
				Name:       result.Name,
				State:      result.State,
				IpAddress:  result.IPAddress,
				MacAddress: result.MACAddress,
				Bridge:     result.Bridge,
				Pid:        int32(result.PID),
				KafkaStubs: kafkaStubs,
			},
		})
	}

	// Fallback: provider doesn't support progress, use unary
	if err := s.sendSandboxCreateProgress(stream, sandboxID, 3, "Creating sandbox"); err != nil {
		return err
	}
	createReq := s.providerCreateRequest(req, sandboxID, baseImage, vcpus, memMB)
	result, err := s.prov.CreateSandbox(ctx, createReq)
	if err != nil {
		s.logger.Error("CreateSandboxStream (unary fallback) failed", "error", err)
		s.sendSandboxCreateError(stream, sandboxID, err)
		return status.Errorf(codes.Internal, "create sandbox: %v", err)
	}

	s.persistCreatedSandbox(ctx, result, req, baseImage, createReq.VCPUs, createReq.MemoryMB)
	kafkaStubs, err := s.attachKafkaDataSourcesForCreate(ctx, result, req)
	if err != nil {
		s.logger.Error("CreateSandboxStream kafka attach failed", "sandbox_id", result.SandboxID, "error", err)
		s.sendSandboxCreateError(stream, sandboxID, err)
		return status.Errorf(codes.Internal, "create sandbox: %v", err)
	}

	s.logAudit(audit.TypeSandboxCreated, map[string]any{
		"sandbox_id": result.SandboxID,
		"source_vm":  req.GetSourceVm(),
		"vcpus":      createReq.VCPUs,
		"memory_mb":  createReq.MemoryMB,
	}, nil, time.Since(start).Milliseconds())

	return stream.Send(&deerv1.SandboxProgress{
		SandboxId: sandboxID,
		Done:      true,
		Result: &deerv1.SandboxCreated{
			SandboxId:  result.SandboxID,
			Name:       result.Name,
			State:      result.State,
			IpAddress:  result.IPAddress,
			MacAddress: result.MACAddress,
			Bridge:     result.Bridge,
			Pid:        int32(result.PID),
			KafkaStubs: kafkaStubs,
		},
	})
}

func (s *Server) GetSandbox(ctx context.Context, req *deerv1.GetSandboxRequest) (*deerv1.SandboxInfo, error) {
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

func (s *Server) ListSandboxes(ctx context.Context, _ *deerv1.ListSandboxesRequest) (*deerv1.ListSandboxesResponse, error) {
	sandboxes, err := s.store.ListSandboxes(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list sandboxes: %v", err)
	}

	infos := make([]*deerv1.SandboxInfo, 0, len(sandboxes))
	for _, sb := range sandboxes {
		infos = append(infos, sandboxToInfo(sb))
	}

	return &deerv1.ListSandboxesResponse{
		Sandboxes: infos,
		Count:     int32(len(infos)),
	}, nil
}

func (s *Server) DestroySandbox(ctx context.Context, req *deerv1.DestroySandboxCommand) (*deerv1.SandboxDestroyed, error) {
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
	s.removeKafkaStubs(ctx, id)

	s.logAudit(audit.TypeSandboxDestroyed, map[string]any{
		"sandbox_id": id,
	}, nil, time.Since(start).Milliseconds())

	return &deerv1.SandboxDestroyed{SandboxId: id}, nil
}

func (s *Server) StartSandbox(ctx context.Context, req *deerv1.StartSandboxCommand) (*deerv1.SandboxStarted, error) {
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

	return &deerv1.SandboxStarted{
		SandboxId: id,
		State:     result.State,
		IpAddress: result.IPAddress,
	}, nil
}

func (s *Server) StopSandbox(ctx context.Context, req *deerv1.StopSandboxCommand) (*deerv1.SandboxStopped, error) {
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

	return &deerv1.SandboxStopped{
		SandboxId: id,
		State:     "STOPPED",
	}, nil
}

func (s *Server) RunCommand(ctx context.Context, req *deerv1.RunCommandCommand) (*deerv1.CommandResult, error) {
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
	if req.GetTimeoutSeconds() > 3600 {
		timeout = time.Hour
	}
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

	return &deerv1.CommandResult{
		SandboxId:  id,
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
		ExitCode:   int32(result.ExitCode),
		DurationMs: result.DurationMS,
	}, nil
}

func (s *Server) CreateSnapshot(ctx context.Context, req *deerv1.SnapshotCommand) (*deerv1.SnapshotCreated, error) {
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

	return &deerv1.SnapshotCreated{
		SandboxId:    id,
		SnapshotId:   result.SnapshotID,
		SnapshotName: result.SnapshotName,
	}, nil
}

func (s *Server) ListSourceVMs(ctx context.Context, req *deerv1.ListSourceVMsCommand) (*deerv1.SourceVMsList, error) {
	if conn := req.GetSourceHostConnection(); conn != nil {
		adhoc, err := s.adhocSourceVMManager(conn)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "create provider for host: %v", err)
		}
		vms, err := adhoc.ListVMs(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "list source VMs: %v", err)
		}
		entries := make([]*deerv1.SourceVMListEntry, 0, len(vms))
		for _, vm := range vms {
			entries = append(entries, &deerv1.SourceVMListEntry{
				Name:      vm.Name,
				State:     vm.State,
				IpAddress: vm.IPAddress,
				Prepared:  vm.Prepared,
				Host:      conn.GetSshHost(),
			})
		}
		return &deerv1.SourceVMsList{Vms: entries}, nil
	}

	// Query all configured source hosts
	if len(s.cfg.SourceHosts) > 0 {
		var allEntries []*deerv1.SourceVMListEntry
		var lastErr error
		for _, conn := range s.sourceHostConns() {
			adhoc, err := s.adhocSourceVMManager(conn)
			if err != nil {
				lastErr = err
				continue
			}
			vms, err := adhoc.ListVMs(ctx)
			if err != nil {
				lastErr = err
				continue
			}
			s.vmHostMu.Lock()
			for _, vm := range vms {
				s.vmHostCache[vm.Name] = conn
				allEntries = append(allEntries, &deerv1.SourceVMListEntry{
					Name:      vm.Name,
					State:     vm.State,
					IpAddress: vm.IPAddress,
					Prepared:  vm.Prepared,
					Host:      conn.SshHost,
				})
			}
			s.vmHostMu.Unlock()
		}
		if len(allEntries) == 0 && lastErr != nil {
			return nil, status.Errorf(codes.Internal, "list source VMs: %v", lastErr)
		}
		return &deerv1.SourceVMsList{Vms: allEntries}, nil
	}

	// Fall back to local provider
	vms, err := s.prov.ListSourceVMs(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list source VMs: %v", err)
	}

	entries := make([]*deerv1.SourceVMListEntry, 0, len(vms))
	for _, vm := range vms {
		entries = append(entries, &deerv1.SourceVMListEntry{
			Name:      vm.Name,
			State:     vm.State,
			IpAddress: vm.IPAddress,
			Prepared:  vm.Prepared,
		})
	}

	return &deerv1.SourceVMsList{Vms: entries}, nil
}

func (s *Server) ValidateSourceVM(ctx context.Context, req *deerv1.ValidateSourceVMCommand) (*deerv1.SourceVMValidation, error) {
	if req.GetSourceVm() == "" {
		return nil, status.Error(codes.InvalidArgument, "source_vm is required")
	}

	conn := req.GetSourceHostConnection()
	if conn == nil && len(s.cfg.SourceHosts) > 0 {
		resolved, err := s.resolveSourceHost(ctx, req.GetSourceVm())
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "resolve source host: %v", err)
		}
		conn = resolved
	}

	if conn != nil {
		adhoc, err := s.adhocSourceVMManager(conn)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "create provider for host: %v", err)
		}
		result, err := adhoc.ValidateSourceVM(ctx, req.GetSourceVm())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "validate source VM: %v", err)
		}
		return &deerv1.SourceVMValidation{
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

	return &deerv1.SourceVMValidation{
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

func (s *Server) PrepareSourceVM(ctx context.Context, req *deerv1.PrepareSourceVMCommand) (*deerv1.SourceVMPrepared, error) {
	if req.GetSourceVm() == "" {
		return nil, status.Error(codes.InvalidArgument, "source_vm is required")
	}

	conn := req.GetSourceHostConnection()
	if conn == nil && len(s.cfg.SourceHosts) > 0 {
		resolved, err := s.resolveSourceHost(ctx, req.GetSourceVm())
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "resolve source host: %v", err)
		}
		conn = resolved
	}

	if conn != nil {
		adhoc, err := s.adhocSourceVMManager(conn)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "create provider for host: %v", err)
		}
		result, err := adhoc.PrepareSourceVM(ctx, req.GetSourceVm(), req.GetSshUser(), req.GetSshKeyPath())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "prepare source VM: %v", err)
		}
		return &deerv1.SourceVMPrepared{
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

	return &deerv1.SourceVMPrepared{
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

func (s *Server) RunSourceCommand(ctx context.Context, req *deerv1.RunSourceCommandCommand) (*deerv1.SourceCommandResult, error) {
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

	conn := req.GetSourceHostConnection()
	if conn == nil && len(s.cfg.SourceHosts) > 0 {
		resolved, err := s.resolveSourceHost(ctx, req.GetSourceVm())
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "resolve source host: %v", err)
		}
		conn = resolved
	}

	if conn != nil {
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
		return &deerv1.SourceCommandResult{
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

	return &deerv1.SourceCommandResult{
		SourceVm: req.GetSourceVm(),
		ExitCode: int32(result.ExitCode),
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
	}, nil
}

func (s *Server) ReadSourceFile(ctx context.Context, req *deerv1.ReadSourceFileCommand) (*deerv1.SourceFileResult, error) {
	start := time.Now()
	if req.GetSourceVm() == "" {
		return nil, status.Error(codes.InvalidArgument, "source_vm is required")
	}
	if req.GetPath() == "" {
		return nil, status.Error(codes.InvalidArgument, "path is required")
	}

	conn := req.GetSourceHostConnection()
	if conn == nil && len(s.cfg.SourceHosts) > 0 {
		resolved, err := s.resolveSourceHost(ctx, req.GetSourceVm())
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "resolve source host: %v", err)
		}
		conn = resolved
	}

	if conn != nil {
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
		return &deerv1.SourceFileResult{
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

	return &deerv1.SourceFileResult{
		SourceVm: req.GetSourceVm(),
		Path:     req.GetPath(),
		Content:  content,
	}, nil
}

func (s *Server) GetHostInfo(ctx context.Context, _ *deerv1.GetHostInfoRequest) (*deerv1.HostInfoResponse, error) {
	hostname, _ := os.Hostname()

	caps, err := s.prov.Capabilities(ctx)
	if err != nil {
		s.logger.Warn("failed to get capabilities", "error", err)
	}

	var sourceHosts []*deerv1.SourceHostInfo
	for _, h := range s.cfg.SourceHosts {
		port := int32(h.SSHPort)
		if port == 0 {
			port = 22
		}
		user := h.SSHUser
		if user == "" {
			user = "deer-daemon"
		}
		sourceHosts = append(sourceHosts, &deerv1.SourceHostInfo{
			Address: h.Address,
			SshUser: user,
			SshPort: port,
		})
	}

	resp := &deerv1.HostInfoResponse{
		HostId:            s.hostID,
		Hostname:          hostname,
		Version:           s.version,
		ActiveSandboxes:   int32(s.prov.ActiveSandboxCount()),
		SshCaPubKey:       s.caPubKey,
		SshIdentityPubKey: s.identityPubKey,
		SourceHosts:       sourceHosts,
	}

	if caps != nil {
		resp.TotalCpus = int32(caps.TotalCPUs)
		resp.TotalMemoryMb = int64(caps.TotalMemoryMB)
		resp.BaseImages = caps.BaseImages
	}

	return resp, nil
}

func (s *Server) Health(_ context.Context, _ *deerv1.HealthRequest) (*deerv1.HealthResponse, error) {
	return &deerv1.HealthResponse{Status: "ok"}, nil
}

func (s *Server) DiscoverHosts(ctx context.Context, req *deerv1.DiscoverHostsCommand) (*deerv1.DiscoverHostsResult, error) {
	s.logger.Info("DiscoverHosts", "config_length", len(req.GetSshConfigContent()))

	hosts, err := sshconfig.Parse(strings.NewReader(req.GetSshConfigContent()))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "parse ssh config: %v", err)
	}

	if len(hosts) == 0 {
		return &deerv1.DiscoverHostsResult{}, nil
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

	return &deerv1.DiscoverHostsResult{Hosts: discovered}, nil
}

func (s *Server) ScanSourceHostKeys(ctx context.Context, _ *deerv1.ScanSourceHostKeysRequest) (*deerv1.ScanSourceHostKeysResponse, error) {
	if len(s.cfg.SourceHosts) == 0 {
		return &deerv1.ScanSourceHostKeysResponse{}, nil
	}

	// Resolve deer-daemon home directory
	homeDir := "/home/deer-daemon"
	if u, err := user.Lookup("deer-daemon"); err == nil {
		homeDir = u.HomeDir
	}
	sshDir := filepath.Join(homeDir, ".ssh")
	knownHostsPath := filepath.Join(sshDir, "known_hosts")

	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		s.logger.Warn("failed to create .ssh dir for deer-daemon", "path", sshDir, "error", err)
	}

	// Read existing known_hosts to skip duplicates
	existing, _ := os.ReadFile(knownHostsPath)
	existingStr := string(existing)

	var results []*deerv1.ScanSourceHostKeysResult
	for _, h := range s.cfg.SourceHosts {
		addr := h.Address
		scanCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		out, err := exec.CommandContext(scanCtx, "ssh-keyscan", "-H", addr).Output()
		cancel()

		if err != nil {
			s.logger.Warn("ssh-keyscan failed", "address", addr, "error", err)
			results = append(results, &deerv1.ScanSourceHostKeysResult{
				Address: addr,
				Success: false,
				Error:   fmt.Sprintf("ssh-keyscan: %v", err),
			})
			continue
		}

		// Append new keys (ssh-keyscan -H hashes the hostname so exact dedup
		// isn't practical; just skip if we already have content from this scan)
		newKeys := strings.TrimSpace(string(out))
		if newKeys == "" {
			results = append(results, &deerv1.ScanSourceHostKeysResult{
				Address: addr,
				Success: false,
				Error:   "ssh-keyscan returned no keys",
			})
			continue
		}

		// Append to known_hosts
		toAppend := newKeys + "\n"
		if len(existingStr) > 0 && !strings.HasSuffix(existingStr, "\n") {
			toAppend = "\n" + toAppend
		}

		f, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			results = append(results, &deerv1.ScanSourceHostKeysResult{
				Address: addr,
				Success: false,
				Error:   fmt.Sprintf("open known_hosts: %v", err),
			})
			continue
		}
		_, writeErr := f.WriteString(toAppend)
		closeErr := f.Close()
		if writeErr != nil {
			results = append(results, &deerv1.ScanSourceHostKeysResult{
				Address: addr,
				Success: false,
				Error:   fmt.Sprintf("write known_hosts: %v", writeErr),
			})
			continue
		}
		if closeErr != nil {
			results = append(results, &deerv1.ScanSourceHostKeysResult{
				Address: addr,
				Success: false,
				Error:   fmt.Sprintf("close known_hosts: %v", closeErr),
			})
			continue
		}
		existingStr += toAppend

		s.logger.Info("scanned host keys", "address", addr)
		results = append(results, &deerv1.ScanSourceHostKeysResult{
			Address: addr,
			Success: true,
		})
	}

	return &deerv1.ScanSourceHostKeysResponse{Results: results}, nil
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
func sandboxToInfo(sb *state.Sandbox) *deerv1.SandboxInfo {
	return &deerv1.SandboxInfo{
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
