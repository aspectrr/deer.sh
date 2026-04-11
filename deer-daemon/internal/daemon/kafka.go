package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	deerv1 "github.com/aspectrr/deer.sh/proto/gen/go/deer/v1"

	"github.com/aspectrr/deer.sh/deer-daemon/internal/kafkastub"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/provider"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/redact"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newKafkaManager(baseDir string, redactor *redact.Redactor, logger *slog.Logger, store *state.Store) (*kafkastub.Manager, error) {
	manager, err := kafkastub.NewManager(baseDir, redactor, logger,
		kafkastub.WithTransport(kafkastub.NewKafkaGoTransport()),
		kafkastub.WithHooks(kafkastub.Hooks{
			OnCaptureStatus: func(item kafkastub.CaptureStatus) {
				if store == nil {
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := mergeCaptureStatus(ctx, store, item); err != nil {
					logger.Warn("merge capture status failed", "config_id", item.CaptureConfigID, "error", err)
				}
			},
			OnSandboxStub: func(stub *kafkastub.SandboxStub) {
				if store == nil {
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := store.UpsertSandboxKafkaStub(ctx, sandboxStubToState(stub)); err != nil {
					logger.Warn("upsert sandbox kafka stub failed", "stub_id", stub.ID, "error", err)
				}
			},
		}))
	if err != nil {
		return nil, err
	}
	if store != nil {
		if err := restoreKafkaRuntime(context.Background(), store, manager); err != nil {
			logger.Warn("failed to restore kafka runtime", "error", err)
		}
	}
	return manager, nil
}

func restoreKafkaRuntime(ctx context.Context, store *state.Store, manager *kafkastub.Manager) error {
	configRows, err := store.ListKafkaCaptureConfigs(ctx, nil)
	if err != nil {
		return err
	}
	configs := make([]kafkastub.CaptureConfig, 0, len(configRows))
	for _, row := range configRows {
		configs = append(configs, captureConfigFromState(row))
	}

	sandboxes, err := store.ListSandboxes(ctx)
	if err != nil {
		return err
	}
	var stubs []kafkastub.SandboxStub
	for _, sandbox := range sandboxes {
		rows, err := store.ListSandboxKafkaStubs(ctx, sandbox.ID)
		if err != nil {
			return err
		}
		for _, row := range rows {
			stubs = append(stubs, sandboxStubFromState(row))
		}
	}
	return manager.Restore(ctx, configs, stubs)
}

func (s *Server) attachKafkaDataSources(ctx context.Context, sandboxID, sandboxIP string, dataSources []*deerv1.DataSourceAttachment, fallback []*deerv1.KafkaCaptureConfigBinding) ([]*deerv1.SandboxKafkaStubInfo, error) {
	attachments := kafkaSandboxAttachmentsFromProto(dataSources, fallback)
	if s.kafkaMgr == nil || len(attachments) == 0 {
		return nil, nil
	}

	for _, attachment := range attachments {
		cfg := attachment.CaptureConfig
		if err := s.store.UpsertKafkaCaptureConfig(ctx, captureConfigToState(cfg)); err != nil {
			s.logger.Warn("failed to persist kafka capture config", "config_id", cfg.ID, "error", err)
		}
	}

	stubs, err := s.kafkaMgr.AttachSandbox(ctx, sandboxID, sandboxBrokerEndpoint(sandboxIP), attachments)
	if err != nil {
		return nil, err
	}

	out := make([]*deerv1.SandboxKafkaStubInfo, 0, len(stubs))
	for _, stub := range stubs {
		if err := s.store.UpsertSandboxKafkaStub(ctx, sandboxStubToState(stub)); err != nil {
			s.logger.Warn("failed to persist sandbox kafka stub", "stub_id", stub.ID, "error", err)
		}
		out = append(out, sandboxStubToProto(stub))
	}
	return out, nil
}

func (s *Server) removeKafkaStubs(ctx context.Context, sandboxID string) {
	if s.kafkaMgr != nil {
		if err := s.kafkaMgr.DetachSandbox(ctx, sandboxID); err != nil {
			s.logger.Warn("failed to detach sandbox kafka stubs", "sandbox_id", sandboxID, "error", err)
		}
	}
	if err := s.store.DeleteSandboxKafkaStubs(ctx, sandboxID); err != nil {
		s.logger.Warn("failed to delete sandbox kafka stubs from state", "sandbox_id", sandboxID, "error", err)
	}
}

func (s *Server) ListSandboxKafkaStubs(ctx context.Context, req *deerv1.ListSandboxKafkaStubsCommand) (*deerv1.ListSandboxKafkaStubsResponse, error) {
	if req.GetSandboxId() == "" {
		return nil, status.Error(codes.InvalidArgument, "sandbox_id is required")
	}
	if s.kafkaMgr == nil {
		return &deerv1.ListSandboxKafkaStubsResponse{}, nil
	}
	stubs, err := s.kafkaMgr.ListSandboxStubs(ctx, req.GetSandboxId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list sandbox kafka stubs: %v", err)
	}
	resp := &deerv1.ListSandboxKafkaStubsResponse{Stubs: make([]*deerv1.SandboxKafkaStubInfo, 0, len(stubs))}
	for _, stub := range stubs {
		_ = s.store.UpsertSandboxKafkaStub(ctx, sandboxStubToState(stub))
		resp.Stubs = append(resp.Stubs, sandboxStubToProto(stub))
	}
	return resp, nil
}

func (s *Server) GetSandboxKafkaStub(ctx context.Context, req *deerv1.GetSandboxKafkaStubCommand) (*deerv1.SandboxKafkaStubInfo, error) {
	if err := requireStubIdentifiers(req.GetSandboxId(), req.GetStubId()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if s.kafkaMgr == nil {
		return nil, status.Error(codes.NotFound, "kafka stub not found")
	}
	stub, err := s.kafkaMgr.GetSandboxStub(ctx, req.GetSandboxId(), req.GetStubId())
	if err != nil {
		if err == kafkastub.ErrNotFound {
			return nil, status.Error(codes.NotFound, "kafka stub not found")
		}
		return nil, status.Errorf(codes.Internal, "get sandbox kafka stub: %v", err)
	}
	_ = s.store.UpsertSandboxKafkaStub(ctx, sandboxStubToState(stub))
	return sandboxStubToProto(stub), nil
}

func (s *Server) StartSandboxKafkaStub(ctx context.Context, req *deerv1.StartSandboxKafkaStubCommand) (*deerv1.SandboxKafkaStubInfo, error) {
	return s.transitionSandboxKafkaStub(ctx, req.GetSandboxId(), req.GetStubId(), "start")
}

func (s *Server) StopSandboxKafkaStub(ctx context.Context, req *deerv1.StopSandboxKafkaStubCommand) (*deerv1.SandboxKafkaStubInfo, error) {
	return s.transitionSandboxKafkaStub(ctx, req.GetSandboxId(), req.GetStubId(), "stop")
}

func (s *Server) RestartSandboxKafkaStub(ctx context.Context, req *deerv1.RestartSandboxKafkaStubCommand) (*deerv1.SandboxKafkaStubInfo, error) {
	return s.transitionSandboxKafkaStub(ctx, req.GetSandboxId(), req.GetStubId(), "restart")
}

func (s *Server) transitionSandboxKafkaStub(ctx context.Context, sandboxID, stubID, action string) (*deerv1.SandboxKafkaStubInfo, error) {
	if err := requireStubIdentifiers(sandboxID, stubID); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if s.kafkaMgr == nil {
		return nil, status.Error(codes.NotFound, "kafka stub not found")
	}

	var (
		stub *kafkastub.SandboxStub
		err  error
	)
	switch action {
	case "start":
		stub, err = s.kafkaMgr.StartSandboxStub(ctx, sandboxID, stubID)
	case "stop":
		stub, err = s.kafkaMgr.StopSandboxStub(ctx, sandboxID, stubID)
	case "restart":
		stub, err = s.kafkaMgr.RestartSandboxStub(ctx, sandboxID, stubID)
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported action")
	}
	if err != nil {
		if err == kafkastub.ErrNotFound {
			return nil, status.Error(codes.NotFound, "kafka stub not found")
		}
		return nil, status.Errorf(codes.Internal, "%s sandbox kafka stub: %v", action, err)
	}
	_ = s.store.UpsertSandboxKafkaStub(ctx, sandboxStubToState(stub))
	return sandboxStubToProto(stub), nil
}

func (s *Server) GetKafkaCaptureStatus(ctx context.Context, req *deerv1.KafkaCaptureStatusRequest) (*deerv1.KafkaCaptureStatusResponse, error) {
	if s.kafkaMgr == nil {
		return &deerv1.KafkaCaptureStatusResponse{}, nil
	}
	statuses, err := s.kafkaMgr.ListCaptureStatuses(ctx, req.GetCaptureConfigIds())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list kafka capture statuses: %v", err)
	}
	resp := &deerv1.KafkaCaptureStatusResponse{Statuses: make([]*deerv1.KafkaCaptureStatus, 0, len(statuses))}
	for _, item := range statuses {
		_ = mergeCaptureStatus(ctx, s.store, item)
		resp.Statuses = append(resp.Statuses, &deerv1.KafkaCaptureStatus{
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
	return resp, nil
}

func kafkaBrokerConfigForDataSources(dataSources []*deerv1.DataSourceAttachment, fallback []*deerv1.KafkaCaptureConfigBinding, simpleKafkaBroker bool) *provider.KafkaBrokerConfig {
	if !simpleKafkaBroker && len(kafkaSandboxAttachmentsFromProto(dataSources, fallback)) == 0 {
		return nil
	}
	return &provider.KafkaBrokerConfig{
		Port: 9092,
	}
}

func elasticsearchBrokerConfig(simpleElasticsearchBroker bool) *provider.ElasticsearchBrokerConfig {
	if !simpleElasticsearchBroker {
		return nil
	}
	return &provider.ElasticsearchBrokerConfig{
		Port: 9200,
	}
}

func providerDataSourcesFromProto(dataSources []*deerv1.DataSourceAttachment, fallback []*deerv1.KafkaCaptureConfigBinding) []provider.DataSourceAttachment {
	attachments := kafkaSandboxAttachmentsFromProto(dataSources, fallback)
	out := make([]provider.DataSourceAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		out = append(out, provider.DataSourceAttachment{
			Type:      provider.DataSourceTypeKafka,
			ConfigRef: attachment.CaptureConfig.ID,
			Kafka: &provider.KafkaDataSourceConfig{
				CaptureConfigID: attachment.CaptureConfig.ID,
				Topics:          append([]string(nil), attachment.Topics...),
				ReplayWindow:    attachment.ReplayWindow,
			},
		})
	}
	return out
}

func kafkaSandboxAttachmentsFromProto(dataSources []*deerv1.DataSourceAttachment, fallback []*deerv1.KafkaCaptureConfigBinding) []kafkastub.SandboxAttachment {
	if len(dataSources) > 0 {
		attachments := make([]kafkastub.SandboxAttachment, 0, len(dataSources))
		for _, ds := range dataSources {
			if ds.GetType() != deerv1.DataSourceType_DATA_SOURCE_TYPE_KAFKA {
				continue
			}
			kafkaCfg := ds.GetKafka()
			if kafkaCfg == nil || kafkaCfg.GetCaptureConfig() == nil {
				continue
			}
			cfg := captureConfigFromProto(kafkaCfg.GetCaptureConfig())
			topics := append([]string(nil), kafkaCfg.GetTopics()...)
			if len(topics) == 0 {
				topics = append(topics, cfg.Topics...)
			}
			attachments = append(attachments, kafkastub.SandboxAttachment{
				CaptureConfig: cfg,
				Topics:        topics,
				ReplayWindow:  time.Duration(kafkaCfg.GetReplayWindowSeconds()) * time.Second,
			})
		}
		return attachments
	}

	attachments := make([]kafkastub.SandboxAttachment, 0, len(fallback))
	for _, binding := range fallback {
		cfg := captureConfigFromProto(binding)
		attachments = append(attachments, kafkastub.SandboxAttachment{
			CaptureConfig: cfg,
			Topics:        append([]string(nil), cfg.Topics...),
		})
	}
	return attachments
}

func sandboxBrokerEndpoint(sandboxIP string) string {
	if sandboxIP == "" {
		return "127.0.0.1:9092"
	}
	return fmt.Sprintf("%s:9092", sandboxIP)
}

func captureConfigFromProto(binding *deerv1.KafkaCaptureConfigBinding) kafkastub.CaptureConfig {
	return kafkastub.CaptureConfig{
		ID:                 binding.GetId(),
		SourceVM:           binding.GetSourceVm(),
		BootstrapServers:   append([]string(nil), binding.GetBootstrapServers()...),
		Topics:             append([]string(nil), binding.GetTopics()...),
		Username:           binding.GetUsername(),
		Password:           binding.GetPassword(),
		SASLMechanism:      binding.GetSaslMechanism(),
		TLSEnabled:         binding.GetTlsEnabled(),
		InsecureSkipVerify: binding.GetInsecureSkipVerify(),
		TLSCAPEM:           binding.GetTlsCaPem(),
		Codec:              binding.GetCodec(),
		RedactionRules:     append([]string(nil), binding.GetRedactionRules()...),
		MaxBufferAge:       time.Duration(binding.GetMaxBufferAgeSeconds()) * time.Second,
		MaxBufferBytes:     binding.GetMaxBufferBytes(),
		Enabled:            binding.GetEnabled(),
	}
}

func captureConfigFromState(row *state.KafkaCaptureConfig) kafkastub.CaptureConfig {
	return kafkastub.CaptureConfig{
		ID:                 row.ID,
		SourceVM:           row.SourceVM,
		BootstrapServers:   append([]string(nil), row.BootstrapServers...),
		Topics:             append([]string(nil), row.Topics...),
		Username:           row.Username,
		Password:           row.Password,
		SASLMechanism:      row.SASLMechanism,
		TLSEnabled:         row.TLSEnabled,
		InsecureSkipVerify: row.InsecureSkipVerify,
		TLSCAPEM:           row.TLSCAPEM,
		Codec:              row.Codec,
		RedactionRules:     append([]string(nil), row.RedactionRules...),
		MaxBufferAge:       time.Duration(row.MaxBufferAgeSecs) * time.Second,
		MaxBufferBytes:     row.MaxBufferBytes,
		Enabled:            row.Enabled,
	}
}

func captureConfigToState(cfg kafkastub.CaptureConfig) *state.KafkaCaptureConfig {
	return &state.KafkaCaptureConfig{
		ID:                 cfg.ID,
		SourceVM:           cfg.SourceVM,
		BootstrapServers:   append([]string(nil), cfg.BootstrapServers...),
		Topics:             append([]string(nil), cfg.Topics...),
		Username:           cfg.Username,
		Password:           cfg.Password,
		SASLMechanism:      cfg.SASLMechanism,
		TLSEnabled:         cfg.TLSEnabled,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		TLSCAPEM:           cfg.TLSCAPEM,
		Codec:              cfg.Codec,
		RedactionRules:     append([]string(nil), cfg.RedactionRules...),
		MaxBufferAgeSecs:   int(cfg.MaxBufferAge / time.Second),
		MaxBufferBytes:     cfg.MaxBufferBytes,
		Enabled:            cfg.Enabled,
		UpdatedAt:          time.Now().UTC(),
	}
}

func mergeCaptureStatus(ctx context.Context, store *state.Store, item kafkastub.CaptureStatus) error {
	rows, err := store.ListKafkaCaptureConfigs(ctx, []string{item.CaptureConfigID})
	if err != nil {
		return err
	}
	var row *state.KafkaCaptureConfig
	if len(rows) > 0 {
		row = rows[0]
	} else {
		row = &state.KafkaCaptureConfig{ID: item.CaptureConfigID, SourceVM: item.SourceVM}
	}
	row.SourceVM = item.SourceVM
	row.State = item.State
	row.BufferedBytes = item.BufferedBytes
	row.SegmentCount = item.SegmentCount
	row.LastError = item.LastError
	row.LastResumeCursor = item.LastResumeCursor
	row.UpdatedAt = item.UpdatedAt
	return store.UpsertKafkaCaptureConfig(ctx, row)
}

func sandboxStubToState(stub *kafkastub.SandboxStub) *state.SandboxKafkaStub {
	return &state.SandboxKafkaStub{
		ID:                  stub.ID,
		SandboxID:           stub.SandboxID,
		CaptureConfigID:     stub.CaptureConfigID,
		BrokerEndpoint:      stub.BrokerEndpoint,
		Topics:              append([]string(nil), stub.Topics...),
		ReplayWindowSeconds: int(stub.ReplayWindow / time.Second),
		State:               stub.State,
		LastReplayCursor:    stub.LastReplayCursor,
		LastError:           stub.LastError,
		AutoStart:           stub.AutoStart,
		CreatedAt:           stub.CreatedAt,
		UpdatedAt:           stub.UpdatedAt,
	}
}

func sandboxStubFromState(row *state.SandboxKafkaStub) kafkastub.SandboxStub {
	return kafkastub.SandboxStub{
		ID:               row.ID,
		SandboxID:        row.SandboxID,
		CaptureConfigID:  row.CaptureConfigID,
		BrokerEndpoint:   row.BrokerEndpoint,
		Topics:           append([]string(nil), row.Topics...),
		ReplayWindow:     time.Duration(row.ReplayWindowSeconds) * time.Second,
		State:            row.State,
		LastReplayCursor: row.LastReplayCursor,
		LastError:        row.LastError,
		AutoStart:        row.AutoStart,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func sandboxStubToProto(stub *kafkastub.SandboxStub) *deerv1.SandboxKafkaStubInfo {
	return &deerv1.SandboxKafkaStubInfo{
		StubId:              stub.ID,
		SandboxId:           stub.SandboxID,
		CaptureConfigId:     stub.CaptureConfigID,
		BrokerEndpoint:      stub.BrokerEndpoint,
		Topics:              append([]string(nil), stub.Topics...),
		ReplayWindowSeconds: int32(stub.ReplayWindow / time.Second),
		State:               stubStateToProto(stub.State),
		LastReplayCursor:    stub.LastReplayCursor,
		AutoStart:           stub.AutoStart,
		LastError:           stub.LastError,
	}
}

func stubStateToProto(v string) deerv1.KafkaStubState {
	switch v {
	case kafkastub.StateRunning:
		return deerv1.KafkaStubState_KAFKA_STUB_STATE_RUNNING
	case kafkastub.StatePaused:
		return deerv1.KafkaStubState_KAFKA_STUB_STATE_PAUSED
	case kafkastub.StateError:
		return deerv1.KafkaStubState_KAFKA_STUB_STATE_ERROR
	default:
		return deerv1.KafkaStubState_KAFKA_STUB_STATE_STOPPED
	}
}

func requireStubIdentifiers(sandboxID, stubID string) error {
	if sandboxID == "" || stubID == "" {
		return fmt.Errorf("sandbox_id and stub_id are required")
	}
	return nil
}
