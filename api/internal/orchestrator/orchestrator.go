// Package orchestrator implements sandbox lifecycle management.
// It coordinates between the host registry, the gRPC stream handler,
// and the persistent store to create, manage, and destroy sandboxes.
package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	deerv1 "github.com/aspectrr/deer.sh/proto/gen/go/deer/v1"

	"github.com/aspectrr/deer.sh/api/internal/id"
	"github.com/aspectrr/deer.sh/api/internal/registry"
	"github.com/aspectrr/deer.sh/api/internal/store"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

const (
	timeoutCreateSandbox  = 5 * time.Minute
	timeoutDestroySandbox = 2 * time.Minute
	timeoutStartStop      = 2 * time.Minute
	timeoutSnapshot       = 5 * time.Minute
	timeoutListSources    = 30 * time.Second
	timeoutValidateVM     = 30 * time.Second
	timeoutPrepareVM      = 5 * time.Minute
	timeoutDiscoverHosts  = 2 * time.Minute
	timeoutReadFile       = 30 * time.Second
	commandTimeoutBuffer  = 30 * time.Second
)

// HostSender abstracts the ability to send a ControlMessage to a specific host
// and wait for a correlated response.
type HostSender interface {
	SendAndWait(ctx context.Context, hostID string, msg *deerv1.ControlMessage, timeout time.Duration) (*deerv1.HostMessage, error)
}

// Orchestrator coordinates sandbox lifecycle operations across connected hosts.
type Orchestrator struct {
	registry         *registry.Registry
	store            store.Store
	sender           HostSender
	logger           *slog.Logger
	defaultTTL       time.Duration
	heartbeatTimeout time.Duration
}

// New creates an Orchestrator.
func New(
	reg *registry.Registry,
	st store.Store,
	sender HostSender,
	logger *slog.Logger,
	defaultTTL time.Duration,
	heartbeatTimeout time.Duration,
) *Orchestrator {
	if logger == nil {
		logger = slog.Default()
	}
	return &Orchestrator{
		registry:         reg,
		store:            st,
		sender:           sender,
		logger:           logger.With("component", "orchestrator"),
		defaultTTL:       defaultTTL,
		heartbeatTimeout: heartbeatTimeout,
	}
}

// ---------------------------------------------------------------------------
// Sandbox lifecycle
// ---------------------------------------------------------------------------

// CreateSandbox selects a host, sends a CreateSandboxCommand over the gRPC
// stream, waits for the SandboxCreated response, and persists the sandbox.
func (o *Orchestrator) CreateSandbox(ctx context.Context, req CreateSandboxRequest) (*store.Sandbox, error) {
	sandboxID, err := id.Generate("SBX-")
	if err != nil {
		return nil, fmt.Errorf("generate sandbox ID: %w", err)
	}

	vcpus := int32(req.VCPUs)
	if vcpus == 0 {
		vcpus = 2
	}
	memMB := int32(req.MemoryMB)
	if memMB == 0 {
		memMB = 2048
	}

	host, err := SelectHost(o.registry, req.SourceVM, req.OrgID, o.heartbeatTimeout, vcpus, memMB)
	if err != nil {
		// SourceVM is always set (validated in handler). Fall back to
		// source-VM-aware placement when base image matching fails.
		var fallbackErr error
		host, fallbackErr = SelectHostForSourceVM(o.registry, req.SourceVM, req.OrgID, o.heartbeatTimeout, vcpus, memMB)
		if fallbackErr != nil {
			return nil, fmt.Errorf("select host: image match: %v; source VM fallback: %w", err, fallbackErr)
		}
	}

	ttlSeconds := int32(req.TTLSeconds)
	if ttlSeconds == 0 && o.defaultTTL > 0 {
		ttlSeconds = int32(o.defaultTTL.Seconds())
	}

	name := req.Name
	if name == "" {
		short := sandboxID
		if len(short) > 12 {
			short = short[:12]
		} else if len(short) > 4 {
			short = short[4:]
		}
		name = "sbx-" + short
	}

	// Map live flag to snapshot mode
	var snapshotMode deerv1.SnapshotMode
	if req.Live {
		snapshotMode = deerv1.SnapshotMode_SNAPSHOT_MODE_FRESH
	}

	// Resolve source host connection if source_host_id is provided
	var sourceHostConn *deerv1.SourceHostConnection
	if req.SourceHostID != "" {
		sh, err := o.store.GetSourceHost(ctx, req.SourceHostID)
		if err != nil {
			return nil, fmt.Errorf("get source host: %w", err)
		}
		sourceHostConn = &deerv1.SourceHostConnection{
			Type:             sh.Type,
			SshHost:          sh.Hostname,
			SshPort:          int32(sh.SSHPort),
			SshUser:          sh.SSHUser,
			SshIdentityFile:  sh.SSHIdentityFile,
			ProxmoxHost:      sh.ProxmoxHost,
			ProxmoxTokenId:   sh.ProxmoxTokenID,
			ProxmoxSecret:    sh.ProxmoxSecret,
			ProxmoxNode:      sh.ProxmoxNode,
			ProxmoxVerifySsl: sh.ProxmoxVerifySSL,
		}
	}

	kafkaBindings := make([]*deerv1.KafkaCaptureConfigBinding, 0, len(req.KafkaCaptureConfigIDs))
	dataSources := make([]*deerv1.DataSourceAttachment, 0, len(req.DataSources)+len(req.KafkaCaptureConfigIDs))
	for _, ds := range mergeSandboxDataSources(req) {
		dsType := ds.Type
		if dsType == "" && ds.Kafka != nil {
			dsType = DataSourceTypeKafka
		}
		switch dsType {
		case DataSourceTypeKafka:
			configID := ds.ConfigRef
			if configID == "" && ds.Kafka != nil {
				configID = ds.Kafka.CaptureConfigID
			}
			if configID == "" {
				return nil, fmt.Errorf("kafka data source requires config_ref or kafka.capture_config_id")
			}

			cfg, err := o.store.GetKafkaCaptureConfig(ctx, configID)
			if err != nil {
				return nil, fmt.Errorf("get kafka capture config %s: %w", configID, err)
			}
			if cfg.OrgID != req.OrgID {
				return nil, fmt.Errorf("kafka capture config %s not found in org", configID)
			}

			binding := kafkaCaptureConfigBinding(cfg)
			kafkaBindings = append(kafkaBindings, binding)

			kafkaAttachment := &deerv1.KafkaDataSourceAttachment{
				CaptureConfig: binding,
			}
			if ds.Kafka != nil {
				if len(ds.Kafka.Topics) > 0 {
					for _, topic := range ds.Kafka.Topics {
						if !slices.Contains([]string(cfg.Topics), topic) {
							return nil, fmt.Errorf("kafka topic %q is not present in capture config %s", topic, cfg.ID)
						}
					}
					kafkaAttachment.Topics = append([]string(nil), ds.Kafka.Topics...)
				}
				if ds.Kafka.ReplayWindowSeconds > 0 {
					kafkaAttachment.ReplayWindowSeconds = int32(ds.Kafka.ReplayWindowSeconds)
				}
			}

			dataSources = append(dataSources, &deerv1.DataSourceAttachment{
				Type:      deerv1.DataSourceType_DATA_SOURCE_TYPE_KAFKA,
				ConfigRef: cfg.ID,
				Config: &deerv1.DataSourceAttachment_Kafka{
					Kafka: kafkaAttachment,
				},
			})
		default:
			return nil, fmt.Errorf("unsupported data source type %q", ds.Type)
		}
	}

	reqID := uuid.New().String()
	cmd := &deerv1.ControlMessage{
		RequestId: reqID,
		Payload: &deerv1.ControlMessage_CreateSandbox{
			CreateSandbox: &deerv1.CreateSandboxCommand{
				SandboxId:            sandboxID,
				BaseImage:            req.SourceVM,
				Name:                 name,
				Vcpus:                vcpus,
				MemoryMb:             memMB,
				TtlSeconds:           ttlSeconds,
				AgentId:              req.AgentID,
				Network:              req.Network,
				SourceVm:             req.SourceVM,
				SnapshotMode:         snapshotMode,
				SourceHostConnection: sourceHostConn,
				KafkaCaptureConfigs:  kafkaBindings,
				DataSources:          dataSources,
			},
		},
	}

	o.logger.Info("creating sandbox",
		"sandbox_id", sandboxID,
		"host_id", host.HostID,
		"org_id", req.OrgID,
		"source_vm", req.SourceVM,
		"live", req.Live,
	)

	resp, err := o.sender.SendAndWait(ctx, host.HostID, cmd, timeoutCreateSandbox)
	if err != nil {
		return nil, fmt.Errorf("create sandbox on host %s: %w", host.HostID, err)
	}

	created := resp.GetSandboxCreated()
	if created == nil {
		if errReport := resp.GetErrorReport(); errReport != nil {
			return nil, fmt.Errorf("host error: %s", errReport.GetError())
		}
		return nil, fmt.Errorf("unexpected response type from host")
	}

	sandbox := &store.Sandbox{
		ID:         sandboxID,
		OrgID:      req.OrgID,
		HostID:     host.HostID,
		Name:       created.GetName(),
		AgentID:    req.AgentID,
		BaseImage:  req.SourceVM,
		Bridge:     created.GetBridge(),
		MACAddress: created.GetMacAddress(),
		IPAddress:  created.GetIpAddress(),
		State:      store.SandboxState(created.GetState()),
		VCPUs:      vcpus,
		MemoryMB:   memMB,
		TTLSeconds: ttlSeconds,
		SourceVM:   req.SourceVM,
	}

	if err := o.store.CreateSandbox(ctx, sandbox); err != nil {
		// Compensating action: destroy the VM on the host to avoid orphan.
		// Uses context.Background() so this runs reliably even if the
		// caller's context is cancelled.
		o.logger.Warn("DB persist failed, issuing compensating destroy",
			"sandbox_id", sandboxID, "host_id", host.HostID, "error", err)
		compReqID := uuid.New().String()
		compCmd := &deerv1.ControlMessage{
			RequestId: compReqID,
			Payload: &deerv1.ControlMessage_DestroySandbox{
				DestroySandbox: &deerv1.DestroySandboxCommand{
					SandboxId: sandboxID,
				},
			},
		}
		if _, compErr := o.sender.SendAndWait(context.Background(), host.HostID, compCmd, timeoutDestroySandbox); compErr != nil {
			o.logger.Error("compensating destroy failed - orphaned VM on host",
				"sandbox_id", sandboxID, "host_id", host.HostID, "error", compErr)
		}
		return nil, fmt.Errorf("persist sandbox: %w", err)
	}

	for _, stub := range created.GetKafkaStubs() {
		if err := o.store.CreateSandboxKafkaStub(ctx, &store.SandboxKafkaStub{
			ID:                  stub.GetStubId(),
			OrgID:               req.OrgID,
			SandboxID:           sandboxID,
			CaptureConfigID:     stub.GetCaptureConfigId(),
			BrokerEndpoint:      stub.GetBrokerEndpoint(),
			Topics:              store.StringSlice(stub.GetTopics()),
			ReplayWindowSeconds: stub.GetReplayWindowSeconds(),
			State:               kafkaStubStateString(stub.GetState()),
			LastReplayCursor:    stub.GetLastReplayCursor(),
			AutoStart:           stub.GetAutoStart(),
		}); err != nil {
			o.logger.Error("persist sandbox kafka stub failed", "sandbox_id", sandboxID, "stub_id", stub.GetStubId(), "error", err)
		}
	}

	o.logger.Info("sandbox created",
		"sandbox_id", sandboxID,
		"host_id", host.HostID,
		"ip_address", created.GetIpAddress(),
	)

	return sandbox, nil
}

func mergeSandboxDataSources(req CreateSandboxRequest) []DataSourceAttachmentRequest {
	merged := make([]DataSourceAttachmentRequest, 0, len(req.DataSources)+len(req.KafkaCaptureConfigIDs))
	merged = append(merged, req.DataSources...)

	existingKafka := make(map[string]struct{}, len(req.DataSources))
	for _, ds := range req.DataSources {
		dsType := ds.Type
		if dsType == "" && ds.Kafka != nil {
			dsType = DataSourceTypeKafka
		}
		if dsType != DataSourceTypeKafka {
			continue
		}
		configID := ds.ConfigRef
		if configID == "" && ds.Kafka != nil {
			configID = ds.Kafka.CaptureConfigID
		}
		if configID != "" {
			existingKafka[configID] = struct{}{}
		}
	}

	for _, configID := range req.KafkaCaptureConfigIDs {
		if _, ok := existingKafka[configID]; ok {
			continue
		}
		merged = append(merged, DataSourceAttachmentRequest{
			Type:      DataSourceTypeKafka,
			ConfigRef: configID,
			Kafka: &KafkaDataSourceAttachmentRequest{
				CaptureConfigID: configID,
			},
		})
	}
	return merged
}

func kafkaCaptureConfigBinding(cfg *store.KafkaCaptureConfig) *deerv1.KafkaCaptureConfigBinding {
	return &deerv1.KafkaCaptureConfigBinding{
		Id:                  cfg.ID,
		SourceVm:            cfg.SourceVM,
		BootstrapServers:    []string(cfg.BootstrapServers),
		Topics:              []string(cfg.Topics),
		Username:            cfg.Username,
		Password:            cfg.Password,
		SaslMechanism:       cfg.SASLMechanism,
		TlsEnabled:          cfg.TLSEnabled,
		InsecureSkipVerify:  cfg.InsecureSkipVerify,
		TlsCaPem:            cfg.TLSCAPEM,
		Codec:               cfg.Codec,
		RedactionRules:      []string(cfg.RedactionRules),
		MaxBufferAgeSeconds: cfg.MaxBufferAgeSecs,
		MaxBufferBytes:      cfg.MaxBufferBytes,
		Enabled:             cfg.Enabled,
	}
}

// GetSandbox retrieves a sandbox by ID, scoped to the given org.
func (o *Orchestrator) GetSandbox(ctx context.Context, orgID, id string) (*store.Sandbox, error) {
	return o.store.GetSandboxByOrg(ctx, orgID, id)
}

// ListSandboxesByOrg returns all non-deleted sandboxes for an org.
func (o *Orchestrator) ListSandboxesByOrg(ctx context.Context, orgID string) ([]*store.Sandbox, error) {
	sandboxes, err := o.store.ListSandboxesByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	result := make([]*store.Sandbox, len(sandboxes))
	for i := range sandboxes {
		result[i] = &sandboxes[i]
	}
	return result, nil
}

func (o *Orchestrator) ListSandboxKafkaStubs(ctx context.Context, orgID, sandboxID string) ([]*store.SandboxKafkaStub, error) {
	sandbox, err := o.store.GetSandboxByOrg(ctx, orgID, sandboxID)
	if err != nil {
		return nil, err
	}

	reqID := uuid.New().String()
	cmd := &deerv1.ControlMessage{
		RequestId: reqID,
		Payload: &deerv1.ControlMessage_ListSandboxKafkaStubs{
			ListSandboxKafkaStubs: &deerv1.ListSandboxKafkaStubsCommand{SandboxId: sandboxID},
		},
	}
	resp, err := o.sender.SendAndWait(ctx, sandbox.HostID, cmd, timeoutStartStop)
	if err != nil {
		return nil, fmt.Errorf("list sandbox kafka stubs on host %s: %w", sandbox.HostID, err)
	}

	list := resp.GetListSandboxKafkaStubsResponse()
	if list == nil {
		if errReport := resp.GetErrorReport(); errReport != nil {
			return nil, fmt.Errorf("host error: %s", errReport.GetError())
		}
		return nil, fmt.Errorf("unexpected response type from host")
	}

	stubs := make([]*store.SandboxKafkaStub, 0, len(list.GetStubs()))
	for _, stub := range list.GetStubs() {
		item := sandboxKafkaStubFromProto(orgID, stub)
		stubs = append(stubs, item)
		if err := o.store.UpdateSandboxKafkaStub(ctx, item); err != nil && !errors.Is(err, store.ErrNotFound) {
			o.logger.Warn("update sandbox kafka stub failed", "stub_id", item.ID, "error", err)
		} else if errors.Is(err, store.ErrNotFound) {
			if err := o.store.CreateSandboxKafkaStub(ctx, item); err != nil {
				o.logger.Error("create sandbox kafka stub failed", "stub_id", item.ID, "error", err)
			}
		}
	}
	return stubs, nil
}

func (o *Orchestrator) GetSandboxKafkaStub(ctx context.Context, orgID, sandboxID, stubID string) (*store.SandboxKafkaStub, error) {
	sandbox, err := o.store.GetSandboxByOrg(ctx, orgID, sandboxID)
	if err != nil {
		return nil, err
	}

	reqID := uuid.New().String()
	cmd := &deerv1.ControlMessage{
		RequestId: reqID,
		Payload: &deerv1.ControlMessage_GetSandboxKafkaStub{
			GetSandboxKafkaStub: &deerv1.GetSandboxKafkaStubCommand{SandboxId: sandboxID, StubId: stubID},
		},
	}
	resp, err := o.sender.SendAndWait(ctx, sandbox.HostID, cmd, timeoutStartStop)
	if err != nil {
		return nil, fmt.Errorf("get sandbox kafka stub on host %s: %w", sandbox.HostID, err)
	}

	info := resp.GetSandboxKafkaStubInfo()
	if info == nil {
		if errReport := resp.GetErrorReport(); errReport != nil {
			return nil, fmt.Errorf("host error: %s", errReport.GetError())
		}
		return nil, fmt.Errorf("unexpected response type from host")
	}

	item := sandboxKafkaStubFromProto(orgID, info)
	if err := o.store.UpdateSandboxKafkaStub(ctx, item); err != nil && !errors.Is(err, store.ErrNotFound) {
		o.logger.Warn("update sandbox kafka stub failed", "stub_id", item.ID, "error", err)
	} else if errors.Is(err, store.ErrNotFound) {
		if err := o.store.CreateSandboxKafkaStub(ctx, item); err != nil {
			o.logger.Error("create sandbox kafka stub failed", "stub_id", item.ID, "error", err)
		}
	}
	return item, nil
}

func (o *Orchestrator) StartSandboxKafkaStub(ctx context.Context, orgID, sandboxID, stubID string) (*store.SandboxKafkaStub, error) {
	return o.transitionSandboxKafkaStub(ctx, orgID, sandboxID, stubID, "start")
}

func (o *Orchestrator) StopSandboxKafkaStub(ctx context.Context, orgID, sandboxID, stubID string) (*store.SandboxKafkaStub, error) {
	return o.transitionSandboxKafkaStub(ctx, orgID, sandboxID, stubID, "stop")
}

func (o *Orchestrator) RestartSandboxKafkaStub(ctx context.Context, orgID, sandboxID, stubID string) (*store.SandboxKafkaStub, error) {
	return o.transitionSandboxKafkaStub(ctx, orgID, sandboxID, stubID, "restart")
}

// DestroySandbox sends a destroy command to the host and marks the sandbox
// as destroyed in the store. The sandbox is looked up scoped to orgID for
// defense-in-depth authorization.
func (o *Orchestrator) DestroySandbox(ctx context.Context, orgID, sandboxID string) error {
	sandbox, err := o.store.GetSandboxByOrg(ctx, orgID, sandboxID)
	if err != nil {
		return fmt.Errorf("get sandbox: %w", err)
	}

	reqID := uuid.New().String()
	cmd := &deerv1.ControlMessage{
		RequestId: reqID,
		Payload: &deerv1.ControlMessage_DestroySandbox{
			DestroySandbox: &deerv1.DestroySandboxCommand{
				SandboxId: sandboxID,
			},
		},
	}

	resp, err := o.sender.SendAndWait(ctx, sandbox.HostID, cmd, timeoutDestroySandbox)
	if err != nil {
		return fmt.Errorf("destroy sandbox on host %s: %w", sandbox.HostID, err)
	}

	if destroyed := resp.GetSandboxDestroyed(); destroyed == nil {
		if errReport := resp.GetErrorReport(); errReport != nil {
			return fmt.Errorf("host error: %s", errReport.GetError())
		}
		return fmt.Errorf("unexpected response type from host")
	}

	if err := o.store.DeleteSandbox(ctx, sandboxID); err != nil {
		return fmt.Errorf("delete sandbox from store: %w", err)
	}

	o.logger.Info("sandbox destroyed", "sandbox_id", sandboxID)
	return nil
}

// RunCommand sends a command to execute in a sandbox and persists the result.
func (o *Orchestrator) RunCommand(ctx context.Context, orgID, sandboxID, command string, timeoutSec int) (*store.Command, error) {
	sandbox, err := o.store.GetSandboxByOrg(ctx, orgID, sandboxID)
	if err != nil {
		return nil, fmt.Errorf("get sandbox: %w", err)
	}

	if timeoutSec == 0 {
		timeoutSec = 300
	}

	reqID := uuid.New().String()
	cmd := &deerv1.ControlMessage{
		RequestId: reqID,
		Payload: &deerv1.ControlMessage_RunCommand{
			RunCommand: &deerv1.RunCommandCommand{
				SandboxId:      sandboxID,
				Command:        command,
				TimeoutSeconds: int32(timeoutSec),
			},
		},
	}

	startedAt := time.Now()

	resp, err := o.sender.SendAndWait(ctx, sandbox.HostID, cmd, time.Duration(timeoutSec)*time.Second+commandTimeoutBuffer)
	if err != nil {
		return nil, fmt.Errorf("run command on host %s: %w", sandbox.HostID, err)
	}

	result := resp.GetCommandResult()
	if result == nil {
		if errReport := resp.GetErrorReport(); errReport != nil {
			return nil, fmt.Errorf("host error: %s", errReport.GetError())
		}
		return nil, fmt.Errorf("unexpected response type from host")
	}

	cmdRecord := &store.Command{
		ID:         uuid.New().String(),
		SandboxID:  sandboxID,
		Command:    command,
		Stdout:     result.GetStdout(),
		Stderr:     result.GetStderr(),
		ExitCode:   result.GetExitCode(),
		DurationMS: result.GetDurationMs(),
		StartedAt:  startedAt,
		EndedAt:    time.Now(),
	}

	if err := o.store.CreateCommand(ctx, cmdRecord); err != nil {
		o.logger.Error("failed to persist command", "sandbox_id", sandboxID, "error", err)
	}

	return cmdRecord, nil
}

// StartSandbox sends a start command to the host.
func (o *Orchestrator) StartSandbox(ctx context.Context, orgID, sandboxID string) error {
	sandbox, err := o.store.GetSandboxByOrg(ctx, orgID, sandboxID)
	if err != nil {
		return fmt.Errorf("get sandbox: %w", err)
	}

	if sandbox.State == store.SandboxStateRunning {
		return fmt.Errorf("sandbox is already running")
	}
	if sandbox.State == store.SandboxStateDestroyed {
		return fmt.Errorf("sandbox is destroyed")
	}

	reqID := uuid.New().String()
	cmd := &deerv1.ControlMessage{
		RequestId: reqID,
		Payload: &deerv1.ControlMessage_StartSandbox{
			StartSandbox: &deerv1.StartSandboxCommand{
				SandboxId: sandboxID,
			},
		},
	}

	resp, err := o.sender.SendAndWait(ctx, sandbox.HostID, cmd, timeoutStartStop)
	if err != nil {
		return fmt.Errorf("start sandbox on host %s: %w", sandbox.HostID, err)
	}

	started := resp.GetSandboxStarted()
	if started == nil {
		if errReport := resp.GetErrorReport(); errReport != nil {
			return fmt.Errorf("host error: %s", errReport.GetError())
		}
		return fmt.Errorf("unexpected response type from host")
	}

	sandbox.State = store.SandboxState(started.GetState())
	sandbox.IPAddress = started.GetIpAddress()
	if err := o.store.UpdateSandbox(ctx, sandbox); err != nil {
		return fmt.Errorf("host operation succeeded but failed to persist state: %w", err)
	}

	return nil
}

// StopSandbox sends a stop command to the host.
func (o *Orchestrator) StopSandbox(ctx context.Context, orgID, sandboxID string) error {
	sandbox, err := o.store.GetSandboxByOrg(ctx, orgID, sandboxID)
	if err != nil {
		return fmt.Errorf("get sandbox: %w", err)
	}

	if sandbox.State == store.SandboxStateStopped {
		return fmt.Errorf("sandbox is already stopped")
	}
	if sandbox.State == store.SandboxStateDestroyed {
		return fmt.Errorf("sandbox is destroyed")
	}

	reqID := uuid.New().String()
	cmd := &deerv1.ControlMessage{
		RequestId: reqID,
		Payload: &deerv1.ControlMessage_StopSandbox{
			StopSandbox: &deerv1.StopSandboxCommand{
				SandboxId: sandboxID,
			},
		},
	}

	resp, err := o.sender.SendAndWait(ctx, sandbox.HostID, cmd, timeoutStartStop)
	if err != nil {
		return fmt.Errorf("stop sandbox on host %s: %w", sandbox.HostID, err)
	}

	stopped := resp.GetSandboxStopped()
	if stopped == nil {
		if errReport := resp.GetErrorReport(); errReport != nil {
			return fmt.Errorf("host error: %s", errReport.GetError())
		}
		return fmt.Errorf("unexpected response type from host")
	}

	sandbox.State = store.SandboxState(stopped.GetState())
	if err := o.store.UpdateSandbox(ctx, sandbox); err != nil {
		return fmt.Errorf("host operation succeeded but failed to persist state: %w", err)
	}

	return nil
}

// CreateSnapshot sends a snapshot command to the host.
func (o *Orchestrator) CreateSnapshot(ctx context.Context, orgID, sandboxID, name string) (*SnapshotResponse, error) {
	sandbox, err := o.store.GetSandboxByOrg(ctx, orgID, sandboxID)
	if err != nil {
		return nil, fmt.Errorf("get sandbox: %w", err)
	}

	reqID := uuid.New().String()
	cmd := &deerv1.ControlMessage{
		RequestId: reqID,
		Payload: &deerv1.ControlMessage_CreateSnapshot{
			CreateSnapshot: &deerv1.SnapshotCommand{
				SandboxId:    sandboxID,
				SnapshotName: name,
			},
		},
	}

	resp, err := o.sender.SendAndWait(ctx, sandbox.HostID, cmd, timeoutSnapshot)
	if err != nil {
		return nil, fmt.Errorf("create snapshot on host %s: %w", sandbox.HostID, err)
	}

	created := resp.GetSnapshotCreated()
	if created == nil {
		if errReport := resp.GetErrorReport(); errReport != nil {
			return nil, fmt.Errorf("host error: %s", errReport.GetError())
		}
		return nil, fmt.Errorf("unexpected response type from host")
	}

	return &SnapshotResponse{
		SnapshotID:   created.GetSnapshotId(),
		SandboxID:    sandboxID,
		SnapshotName: created.GetSnapshotName(),
		CreatedAt:    time.Now(),
	}, nil
}

// ListCommands returns all commands for a given sandbox.
func (o *Orchestrator) ListCommands(ctx context.Context, orgID, sandboxID string) ([]*store.Command, error) {
	// Verify sandbox belongs to org.
	if _, err := o.store.GetSandboxByOrg(ctx, orgID, sandboxID); err != nil {
		return nil, fmt.Errorf("get sandbox: %w", err)
	}

	commands, err := o.store.ListSandboxCommands(ctx, sandboxID)
	if err != nil {
		return nil, err
	}
	result := make([]*store.Command, len(commands))
	for i := range commands {
		result[i] = &commands[i]
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Host operations
// ---------------------------------------------------------------------------

// ListHosts returns info about all connected hosts.
func (o *Orchestrator) ListHosts(ctx context.Context, orgID string) ([]*HostInfo, error) {
	connected := o.registry.ListConnectedByOrg(orgID)
	result := make([]*HostInfo, 0, len(connected))

	// Batch-fetch sandbox counts to avoid N+1 queries.
	hostIDs := make([]string, len(connected))
	for i, h := range connected {
		hostIDs[i] = h.HostID
	}
	counts, err := o.store.CountSandboxesByHostIDs(ctx, hostIDs)
	if err != nil {
		o.logger.Warn("failed to batch-count sandboxes by host", "error", err)
		counts = map[string]int{}
	}

	for _, h := range connected {
		info := &HostInfo{
			HostID:          h.HostID,
			Hostname:        h.Hostname,
			Status:          "ONLINE",
			LastHeartbeat:   h.LastHeartbeat.Format(time.RFC3339),
			ActiveSandboxes: counts[h.HostID],
		}
		if h.Registration != nil {
			info.AvailableCPUs = h.Registration.GetAvailableCpus()
			info.AvailableMemMB = h.Registration.GetAvailableMemoryMb()
			info.AvailableDiskMB = h.Registration.GetAvailableDiskMb()
			info.BaseImages = h.Registration.GetBaseImages()
		}

		result = append(result, info)
	}

	return result, nil
}

// GetHost returns info about a specific connected host.
func (o *Orchestrator) GetHost(ctx context.Context, id, orgID string) (*HostInfo, error) {
	h, ok := o.registry.GetHost(id)
	if !ok {
		return nil, fmt.Errorf("host %s not found or not connected", id)
	}
	if h.OrgID != orgID {
		return nil, fmt.Errorf("host %s not found or not connected", id)
	}

	info := &HostInfo{
		HostID:        h.HostID,
		Hostname:      h.Hostname,
		Status:        "ONLINE",
		LastHeartbeat: h.LastHeartbeat.Format(time.RFC3339),
	}
	if h.Registration != nil {
		info.AvailableCPUs = h.Registration.GetAvailableCpus()
		info.AvailableMemMB = h.Registration.GetAvailableMemoryMb()
		info.AvailableDiskMB = h.Registration.GetAvailableDiskMb()
		info.BaseImages = h.Registration.GetBaseImages()
	}

	counts, err := o.store.CountSandboxesByHostIDs(ctx, []string{h.HostID})
	if err != nil {
		o.logger.Warn("failed to count sandboxes for host", "host_id", h.HostID, "error", err)
	} else {
		info.ActiveSandboxes = counts[h.HostID]
	}

	return info, nil
}

// ---------------------------------------------------------------------------
// Source VM operations
// ---------------------------------------------------------------------------

func (o *Orchestrator) transitionSandboxKafkaStub(ctx context.Context, orgID, sandboxID, stubID, action string) (*store.SandboxKafkaStub, error) {
	sandbox, err := o.store.GetSandboxByOrg(ctx, orgID, sandboxID)
	if err != nil {
		return nil, err
	}

	cmd := &deerv1.ControlMessage{RequestId: uuid.New().String()}
	switch action {
	case "start":
		cmd.Payload = &deerv1.ControlMessage_StartSandboxKafkaStub{
			StartSandboxKafkaStub: &deerv1.StartSandboxKafkaStubCommand{SandboxId: sandboxID, StubId: stubID},
		}
	case "stop":
		cmd.Payload = &deerv1.ControlMessage_StopSandboxKafkaStub{
			StopSandboxKafkaStub: &deerv1.StopSandboxKafkaStubCommand{SandboxId: sandboxID, StubId: stubID},
		}
	case "restart":
		cmd.Payload = &deerv1.ControlMessage_RestartSandboxKafkaStub{
			RestartSandboxKafkaStub: &deerv1.RestartSandboxKafkaStubCommand{SandboxId: sandboxID, StubId: stubID},
		}
	default:
		return nil, fmt.Errorf("unsupported kafka stub action %q", action)
	}

	resp, err := o.sender.SendAndWait(ctx, sandbox.HostID, cmd, timeoutStartStop)
	if err != nil {
		return nil, fmt.Errorf("%s sandbox kafka stub on host %s: %w", action, sandbox.HostID, err)
	}

	info := resp.GetSandboxKafkaStubInfo()
	if info == nil {
		if errReport := resp.GetErrorReport(); errReport != nil {
			return nil, fmt.Errorf("host error: %s", errReport.GetError())
		}
		return nil, fmt.Errorf("unexpected response type from host")
	}

	item := sandboxKafkaStubFromProto(orgID, info)
	if err := o.store.UpdateSandboxKafkaStub(ctx, item); err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, err
	} else if errors.Is(err, store.ErrNotFound) {
		if err := o.store.CreateSandboxKafkaStub(ctx, item); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func sandboxKafkaStubFromProto(orgID string, stub *deerv1.SandboxKafkaStubInfo) *store.SandboxKafkaStub {
	return &store.SandboxKafkaStub{
		ID:                  stub.GetStubId(),
		OrgID:               orgID,
		SandboxID:           stub.GetSandboxId(),
		CaptureConfigID:     stub.GetCaptureConfigId(),
		BrokerEndpoint:      stub.GetBrokerEndpoint(),
		Topics:              store.StringSlice(stub.GetTopics()),
		ReplayWindowSeconds: stub.GetReplayWindowSeconds(),
		State:               kafkaStubStateString(stub.GetState()),
		LastReplayCursor:    stub.GetLastReplayCursor(),
		AutoStart:           stub.GetAutoStart(),
	}
}

func kafkaStubStateString(state deerv1.KafkaStubState) string {
	switch state {
	case deerv1.KafkaStubState_KAFKA_STUB_STATE_RUNNING:
		return "running"
	case deerv1.KafkaStubState_KAFKA_STUB_STATE_PAUSED:
		return "paused"
	case deerv1.KafkaStubState_KAFKA_STUB_STATE_ERROR:
		return "error"
	default:
		return "stopped"
	}
}

// ListVMs aggregates source VMs from all connected hosts in parallel.
func (o *Orchestrator) ListVMs(ctx context.Context, orgID string) ([]*VMInfo, error) {
	connected := o.registry.ListConnectedByOrg(orgID)

	var mu sync.Mutex
	var result []*VMInfo

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(10)
	for _, h := range connected {
		if h.Registration == nil {
			continue
		}

		g.Go(func() error {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			default:
			}

			reqID := uuid.New().String()
			cmd := &deerv1.ControlMessage{
				RequestId: reqID,
				Payload: &deerv1.ControlMessage_ListSourceVms{
					ListSourceVms: &deerv1.ListSourceVMsCommand{},
				},
			}

			var vms []*VMInfo
			resp, err := o.sender.SendAndWait(gCtx, h.HostID, cmd, timeoutListSources)
			if err != nil {
				o.logger.Warn("failed to list VMs from host", "host_id", h.HostID, "error", err)
				for _, vm := range h.Registration.GetSourceVms() {
					vms = append(vms, &VMInfo{
						Name:      vm.GetName(),
						State:     vm.GetState(),
						IPAddress: vm.GetIpAddress(),
						Prepared:  vm.GetPrepared(),
						HostID:    h.HostID,
					})
				}
			} else if vmList := resp.GetSourceVmsList(); vmList != nil {
				for _, vm := range vmList.GetVms() {
					vms = append(vms, &VMInfo{
						Name:      vm.GetName(),
						State:     vm.GetState(),
						IPAddress: vm.GetIpAddress(),
						Prepared:  vm.GetPrepared(),
						HostID:    h.HostID,
					})
				}
			}

			mu.Lock()
			result = append(result, vms...)
			mu.Unlock()
			return nil
		})
	}

	// g.Wait always returns nil because goroutines handle errors internally
	// (log warning + fall back to cached registration data). Partial results
	// are returned intentionally.
	_ = g.Wait()
	return result, nil
}

// PrepareSourceVM sends a prepare command to the host that owns the source VM.
func (o *Orchestrator) PrepareSourceVM(ctx context.Context, orgID, vmName, sshUser, keyPath string) (*deerv1.SourceVMPrepared, error) {
	host, err := SelectHostForSourceVM(o.registry, vmName, orgID, o.heartbeatTimeout, 0, 0)
	if err != nil {
		return nil, err
	}

	reqID := uuid.New().String()
	cmd := &deerv1.ControlMessage{
		RequestId: reqID,
		Payload: &deerv1.ControlMessage_PrepareSourceVm{
			PrepareSourceVm: &deerv1.PrepareSourceVMCommand{
				SourceVm:   vmName,
				SshUser:    sshUser,
				SshKeyPath: keyPath,
			},
		},
	}

	resp, err := o.sender.SendAndWait(ctx, host.HostID, cmd, timeoutPrepareVM)
	if err != nil {
		return nil, fmt.Errorf("prepare source VM on host %s: %w", host.HostID, err)
	}

	prepared := resp.GetSourceVmPrepared()
	if prepared == nil {
		if errReport := resp.GetErrorReport(); errReport != nil {
			return nil, fmt.Errorf("host error: %s", errReport.GetError())
		}
		return nil, fmt.Errorf("unexpected response type from host")
	}

	return prepared, nil
}

// ValidateSourceVM sends a validate command to the host that owns the source VM.
func (o *Orchestrator) ValidateSourceVM(ctx context.Context, orgID, vmName string) (*deerv1.SourceVMValidation, error) {
	host, err := SelectHostForSourceVM(o.registry, vmName, orgID, o.heartbeatTimeout, 0, 0)
	if err != nil {
		return nil, err
	}

	reqID := uuid.New().String()
	cmd := &deerv1.ControlMessage{
		RequestId: reqID,
		Payload: &deerv1.ControlMessage_ValidateSourceVm{
			ValidateSourceVm: &deerv1.ValidateSourceVMCommand{
				SourceVm: vmName,
			},
		},
	}

	resp, err := o.sender.SendAndWait(ctx, host.HostID, cmd, timeoutValidateVM)
	if err != nil {
		return nil, fmt.Errorf("validate source VM on host %s: %w", host.HostID, err)
	}

	validation := resp.GetSourceVmValidation()
	if validation == nil {
		if errReport := resp.GetErrorReport(); errReport != nil {
			return nil, fmt.Errorf("host error: %s", errReport.GetError())
		}
		return nil, fmt.Errorf("unexpected response type from host")
	}

	return validation, nil
}

// RunSourceCommand executes a read-only command on a source VM via the host.
func (o *Orchestrator) RunSourceCommand(ctx context.Context, orgID, vmName, command string, timeoutSec int) (*SourceCommandResult, error) {
	host, err := SelectHostForSourceVM(o.registry, vmName, orgID, o.heartbeatTimeout, 0, 0)
	if err != nil {
		return nil, err
	}

	if timeoutSec == 0 {
		timeoutSec = 30
	}

	reqID := uuid.New().String()
	cmd := &deerv1.ControlMessage{
		RequestId: reqID,
		Payload: &deerv1.ControlMessage_RunSourceCommand{
			RunSourceCommand: &deerv1.RunSourceCommandCommand{
				SourceVm:       vmName,
				Command:        command,
				TimeoutSeconds: int32(timeoutSec),
			},
		},
	}

	resp, err := o.sender.SendAndWait(ctx, host.HostID, cmd, time.Duration(timeoutSec)*time.Second+commandTimeoutBuffer)
	if err != nil {
		return nil, fmt.Errorf("run source command on host %s: %w", host.HostID, err)
	}

	result := resp.GetSourceCommandResult()
	if result == nil {
		if errReport := resp.GetErrorReport(); errReport != nil {
			return nil, fmt.Errorf("host error: %s", errReport.GetError())
		}
		return nil, fmt.Errorf("unexpected response type from host")
	}

	return &SourceCommandResult{
		SourceVM: vmName,
		ExitCode: int(result.GetExitCode()),
		Stdout:   result.GetStdout(),
		Stderr:   result.GetStderr(),
	}, nil
}

// ReadSourceFile reads a file from a source VM via the host.
func (o *Orchestrator) ReadSourceFile(ctx context.Context, orgID, vmName, path string) (*SourceFileResult, error) {
	host, err := SelectHostForSourceVM(o.registry, vmName, orgID, o.heartbeatTimeout, 0, 0)
	if err != nil {
		return nil, err
	}

	reqID := uuid.New().String()
	cmd := &deerv1.ControlMessage{
		RequestId: reqID,
		Payload: &deerv1.ControlMessage_ReadSourceFile{
			ReadSourceFile: &deerv1.ReadSourceFileCommand{
				SourceVm: vmName,
				Path:     path,
			},
		},
	}

	resp, err := o.sender.SendAndWait(ctx, host.HostID, cmd, timeoutReadFile)
	if err != nil {
		return nil, fmt.Errorf("read source file on host %s: %w", host.HostID, err)
	}

	result := resp.GetSourceFileResult()
	if result == nil {
		if errReport := resp.GetErrorReport(); errReport != nil {
			return nil, fmt.Errorf("host error: %s", errReport.GetError())
		}
		return nil, fmt.Errorf("unexpected response type from host")
	}

	return &SourceFileResult{
		SourceVM: vmName,
		Path:     result.GetPath(),
		Content:  result.GetContent(),
	}, nil
}

// ---------------------------------------------------------------------------
// Source host operations
// ---------------------------------------------------------------------------

// DiscoverSourceHosts sends SSH config to a connected daemon for parsing and probing.
func (o *Orchestrator) DiscoverSourceHosts(ctx context.Context, orgID, sshConfigContent string) ([]*DiscoveredHost, error) {
	connected := o.registry.ListConnectedByOrg(orgID)
	if len(connected) == 0 {
		return nil, fmt.Errorf("no connected daemon hosts available for discovery")
	}

	// Pick a random connected host to distribute probing load.
	host := connected[rand.IntN(len(connected))]

	reqID := uuid.New().String()
	cmd := &deerv1.ControlMessage{
		RequestId: reqID,
		Payload: &deerv1.ControlMessage_DiscoverHosts{
			DiscoverHosts: &deerv1.DiscoverHostsCommand{
				SshConfigContent: sshConfigContent,
			},
		},
	}

	resp, err := o.sender.SendAndWait(ctx, host.HostID, cmd, timeoutDiscoverHosts)
	if err != nil {
		return nil, fmt.Errorf("discover hosts via %s: %w", host.HostID, err)
	}

	result := resp.GetDiscoverHostsResult()
	if result == nil {
		if errReport := resp.GetErrorReport(); errReport != nil {
			return nil, fmt.Errorf("host error: %s", errReport.GetError())
		}
		return nil, fmt.Errorf("unexpected response type from host")
	}

	discovered := make([]*DiscoveredHost, 0, len(result.GetHosts()))
	for _, h := range result.GetHosts() {
		discovered = append(discovered, &DiscoveredHost{
			Name:         h.GetName(),
			Hostname:     h.GetHostname(),
			User:         h.GetUser(),
			Port:         int(h.GetPort()),
			IdentityFile: h.GetIdentityFile(),
			Reachable:    h.GetReachable(),
			HasLibvirt:   h.GetHasLibvirt(),
			HasProxmox:   h.GetHasProxmox(),
			VMs:          h.GetVms(),
			Error:        h.GetError(),
		})
	}

	return discovered, nil
}
