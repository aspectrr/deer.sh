package agent

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
)

func newKafkaManager(baseDir string, logger *slog.Logger, localStore *state.Store) (*kafkastub.Manager, error) {
	manager, err := kafkastub.NewManager(baseDir, redact.New(), logger,
		kafkastub.WithTransport(kafkastub.NewKafkaGoTransport()),
		kafkastub.WithHooks(kafkastub.Hooks{
			OnCaptureStatus: func(item kafkastub.CaptureStatus) {
				if localStore == nil {
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := mergeCaptureStatus(ctx, localStore, item); err != nil {
					logger.Warn("merge capture status failed", "config_id", item.CaptureConfigID, "error", err)
				}
			},
			OnSandboxStub: func(stub *kafkastub.SandboxStub) {
				if localStore == nil {
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := localStore.UpsertSandboxKafkaStub(ctx, sandboxKafkaStubToLocal(stub)); err != nil {
					logger.Warn("upsert sandbox kafka stub failed", "stub_id", stub.ID, "error", err)
				}
			},
		}))
	if err != nil {
		return nil, err
	}
	if localStore != nil {
		if err := restoreKafkaRuntime(context.Background(), localStore, manager); err != nil {
			logger.Warn("failed to restore kafka runtime", "error", err)
		}
	}
	return manager, nil
}

func restoreKafkaRuntime(ctx context.Context, localStore *state.Store, manager *kafkastub.Manager) error {
	configRows, err := localStore.ListKafkaCaptureConfigs(ctx, nil)
	if err != nil {
		return err
	}
	configs := make([]kafkastub.CaptureConfig, 0, len(configRows))
	for _, row := range configRows {
		configs = append(configs, kafkaCaptureConfigFromLocal(row))
	}

	sandboxes, err := localStore.ListSandboxes(ctx)
	if err != nil {
		return err
	}
	var stubs []kafkastub.SandboxStub
	for _, sandbox := range sandboxes {
		rows, err := localStore.ListSandboxKafkaStubs(ctx, sandbox.ID)
		if err != nil {
			return err
		}
		for _, row := range rows {
			stubs = append(stubs, sandboxKafkaStubFromLocal(row))
		}
	}
	return manager.Restore(ctx, configs, stubs)
}

func kafkaBrokerConfigForDataSources(dataSources []*deerv1.DataSourceAttachment, fallback []*deerv1.KafkaCaptureConfigBinding) *provider.KafkaBrokerConfig {
	if len(kafkaSandboxAttachmentsFromProto(dataSources, fallback)) == 0 {
		return nil
	}
	return &provider.KafkaBrokerConfig{Port: 9092}
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
			cfg := kafkaCaptureConfigFromProto(kafkaCfg.GetCaptureConfig())
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
		cfg := kafkaCaptureConfigFromProto(binding)
		attachments = append(attachments, kafkastub.SandboxAttachment{
			CaptureConfig: cfg,
			Topics:        append([]string(nil), cfg.Topics...),
		})
	}
	return attachments
}

func kafkaCaptureConfigFromProto(binding *deerv1.KafkaCaptureConfigBinding) kafkastub.CaptureConfig {
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

func kafkaCaptureConfigToLocal(cfg kafkastub.CaptureConfig) *state.KafkaCaptureConfig {
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

func kafkaCaptureConfigFromLocal(row *state.KafkaCaptureConfig) kafkastub.CaptureConfig {
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

func mergeCaptureStatus(ctx context.Context, localStore *state.Store, item kafkastub.CaptureStatus) error {
	rows, err := localStore.ListKafkaCaptureConfigs(ctx, []string{item.CaptureConfigID})
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
	return localStore.UpsertKafkaCaptureConfig(ctx, row)
}

func sandboxKafkaStubToLocal(stub *kafkastub.SandboxStub) *state.SandboxKafkaStub {
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

func sandboxKafkaStubFromLocal(row *state.SandboxKafkaStub) kafkastub.SandboxStub {
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

func sandboxKafkaStubToProto(stub *kafkastub.SandboxStub) *deerv1.SandboxKafkaStubInfo {
	return &deerv1.SandboxKafkaStubInfo{
		StubId:              stub.ID,
		SandboxId:           stub.SandboxID,
		CaptureConfigId:     stub.CaptureConfigID,
		BrokerEndpoint:      stub.BrokerEndpoint,
		Topics:              append([]string(nil), stub.Topics...),
		ReplayWindowSeconds: int32(stub.ReplayWindow / time.Second),
		State:               sandboxKafkaStateToProto(stub.State),
		LastReplayCursor:    stub.LastReplayCursor,
		AutoStart:           stub.AutoStart,
		LastError:           stub.LastError,
	}
}

func sandboxKafkaStateToProto(v string) deerv1.KafkaStubState {
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

func sandboxBrokerEndpoint(sandboxIP string) string {
	if sandboxIP == "" {
		return "127.0.0.1:9092"
	}
	return fmt.Sprintf("%s:9092", sandboxIP)
}
