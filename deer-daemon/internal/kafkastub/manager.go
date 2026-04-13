package kafkastub

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/aspectrr/deer.sh/deer-daemon/internal/redact"
)

const (
	StateStopped = "stopped"
	StateRunning = "running"
	StatePaused  = "paused"
	StateError   = "error"
)

var ErrNotFound = errors.New("kafkastub: not found")

type CaptureConfig struct {
	ID                 string
	SourceVM           string
	BootstrapServers   []string
	Topics             []string
	Username           string
	Password           string
	SASLMechanism      string
	TLSEnabled         bool
	InsecureSkipVerify bool
	TLSCAPEM           string
	Codec              string
	RedactionRules     []string
	MaxBufferAge       time.Duration
	MaxBufferBytes     int64
	Enabled            bool
}

type SandboxStub struct {
	ID               string
	SandboxID        string
	CaptureConfigID  string
	BrokerEndpoint   string
	Topics           []string
	ReplayWindow     time.Duration
	State            string
	LastReplayCursor string
	LastError        string
	AutoStart        bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type SandboxAttachment struct {
	CaptureConfig CaptureConfig
	Topics        []string
	ReplayWindow  time.Duration
}

type CaptureStatus struct {
	CaptureConfigID      string
	SourceVM             string
	State                string
	BufferedBytes        int64
	SegmentCount         int
	UpdatedAt            time.Time
	AttachedSandboxCount int
	LastError            string
	LastResumeCursor     string
}

type Header struct {
	Key   string
	Value []byte
}

type Record struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       []byte
	Headers   []Header
	Timestamp time.Time
	Value     []byte
}

type Hooks struct {
	OnCaptureStatus func(CaptureStatus)
	OnSandboxStub   func(*SandboxStub)
}

type Option func(*Manager)

func WithTransport(transport Transport) Option {
	return func(m *Manager) {
		m.transport = transport
	}
}

func WithHooks(hooks Hooks) Option {
	return func(m *Manager) {
		m.hooks = hooks
	}
}

func WithSleep(fn func(context.Context, time.Duration) error) Option {
	return func(m *Manager) {
		m.sleep = fn
	}
}

type segment struct {
	path       string
	sizeBytes  int64
	capturedAt time.Time
}

type captureRuntime struct {
	cfg           CaptureConfig
	status        CaptureStatus
	segments      []segment
	attached      map[string]int
	captureCtx    context.Context
	captureCancel context.CancelFunc
	captureDone   chan struct{}
	loaded        bool
}

type replayRuntime struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type Manager struct {
	baseDir   string
	redactor  *redact.Redactor
	logger    *slog.Logger
	transport Transport
	hooks     Hooks
	sleep     func(context.Context, time.Duration) error

	mu       sync.Mutex
	captures map[string]*captureRuntime
	stubs    map[string]*SandboxStub
	replays  map[string]*replayRuntime
}

func NewManager(baseDir string, redactor *redact.Redactor, logger *slog.Logger, opts ...Option) (*Manager, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if redactor == nil {
		redactor = redact.New()
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create kafka stub base dir: %w", err)
	}
	m := &Manager{
		baseDir:   baseDir,
		redactor:  redactor,
		logger:    logger.With("component", "kafkastub"),
		transport: noopTransport{},
		hooks:     Hooks{},
		sleep: func(ctx context.Context, d time.Duration) error {
			t := time.NewTimer(d)
			defer t.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-t.C:
				return nil
			}
		},
		captures: make(map[string]*captureRuntime),
		stubs:    make(map[string]*SandboxStub),
		replays:  make(map[string]*replayRuntime),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m, nil
}

func (m *Manager) Restore(_ context.Context, configs []CaptureConfig, stubs []SandboxStub) error {
	for _, cfg := range configs {
		m.EnsureCaptureConfig(cfg)
	}

	var toStart []string
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, stub := range stubs {
		cp := stub
		if cp.Topics == nil {
			cp.Topics = []string{}
		}
		if cp.ReplayWindow == 0 {
			if runtime, ok := m.captures[cp.CaptureConfigID]; ok {
				cp.ReplayWindow = defaultReplayWindow(runtime.cfg.MaxBufferAge)
			} else {
				cp.ReplayWindow = defaultReplayWindow(0)
			}
		}
		m.stubs[cp.ID] = cloneStub(&cp)
		if runtime, ok := m.captures[cp.CaptureConfigID]; ok {
			runtime.attached[cp.SandboxID] = 1
			runtime.status.AttachedSandboxCount = len(runtime.attached)
			m.touchCaptureLocked(runtime)
		}
		if cp.State == StateRunning {
			toStart = append(toStart, cp.ID)
		}
	}
	go func(ids []string) {
		for _, stubID := range ids {
			if _, err := m.StartSandboxStub(context.Background(), stubForID(m, stubID), stubID); err != nil {
				m.logger.Warn("restore replay worker failed", "stub_id", stubID, "error", err)
			}
		}
	}(append([]string(nil), toStart...))
	return nil
}

func (m *Manager) EnsureCaptureConfig(cfg CaptureConfig) {
	runtime, status := m.ensureAndMaybeStartCapture(cfg)
	m.notifyCapture(status)
	if runtime == nil {
		return
	}
}

func (m *Manager) AttachSandbox(ctx context.Context, sandboxID, brokerEndpoint string, attachments []SandboxAttachment) ([]*SandboxStub, error) {
	now := time.Now().UTC()
	stubs := make([]*SandboxStub, 0, len(attachments))
	for _, attachment := range attachments {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		cfg := attachment.CaptureConfig
		m.EnsureCaptureConfig(cfg)
		topics := append([]string(nil), attachment.Topics...)
		if len(topics) == 0 {
			topics = append(topics, cfg.Topics...)
		}
		replayWindow := attachment.ReplayWindow
		if replayWindow <= 0 {
			replayWindow = defaultReplayWindow(cfg.MaxBufferAge)
		}

		stub := &SandboxStub{
			ID:               sanitizeID(fmt.Sprintf("%s-%s", sandboxID, cfg.ID)),
			SandboxID:        sandboxID,
			CaptureConfigID:  cfg.ID,
			BrokerEndpoint:   normalizeBrokerEndpoint(brokerEndpoint),
			Topics:           topics,
			ReplayWindow:     replayWindow,
			State:            StateStopped,
			LastReplayCursor: "head",
			AutoStart:        true,
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		m.mu.Lock()
		runtime := m.ensureCaptureLocked(cfg)
		runtime.attached[sandboxID] = 1
		runtime.status.AttachedSandboxCount = len(runtime.attached)
		m.touchCaptureLocked(runtime)
		m.stubs[stub.ID] = cloneStub(stub)
		m.mu.Unlock()

		m.notifyCapture(runtime.status)
		m.notifyStub(stub)

		if stub.AutoStart {
			started, err := m.StartSandboxStub(ctx, sandboxID, stub.ID)
			if err != nil {
				return nil, err
			}
			stub = started
		}
		stubs = append(stubs, cloneStub(stub))
	}
	return stubs, nil
}

func (m *Manager) DetachSandbox(_ context.Context, sandboxID string) error {
	var captureStatuses []CaptureStatus

	m.mu.Lock()
	for id, stub := range m.stubs {
		if stub.SandboxID != sandboxID {
			continue
		}
		if replay, ok := m.replays[id]; ok {
			replay.cancel()
			delete(m.replays, id)
		}
		if runtime, ok := m.captures[stub.CaptureConfigID]; ok {
			delete(runtime.attached, sandboxID)
			runtime.status.AttachedSandboxCount = len(runtime.attached)
			m.touchCaptureLocked(runtime)
			captureStatuses = append(captureStatuses, runtime.status)
		}
		delete(m.stubs, id)
	}
	m.mu.Unlock()

	for _, status := range captureStatuses {
		m.notifyCapture(status)
	}
	return nil
}

func (m *Manager) ListSandboxStubs(_ context.Context, sandboxID string) ([]*SandboxStub, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var stubs []*SandboxStub
	for _, stub := range m.stubs {
		if stub.SandboxID == sandboxID {
			stubs = append(stubs, cloneStub(stub))
		}
	}
	sort.Slice(stubs, func(i, j int) bool {
		return stubs[i].CreatedAt.Before(stubs[j].CreatedAt)
	})
	return stubs, nil
}

func (m *Manager) GetSandboxStub(_ context.Context, sandboxID, stubID string) (*SandboxStub, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stub, ok := m.stubs[stubID]
	if !ok || stub.SandboxID != sandboxID {
		return nil, ErrNotFound
	}
	return cloneStub(stub), nil
}

func (m *Manager) StartSandboxStub(ctx context.Context, sandboxID, stubID string) (*SandboxStub, error) {
	return m.transitionStub(ctx, sandboxID, stubID, "start")
}

func (m *Manager) StopSandboxStub(ctx context.Context, sandboxID, stubID string) (*SandboxStub, error) {
	return m.transitionStub(ctx, sandboxID, stubID, "stop")
}

func (m *Manager) RestartSandboxStub(ctx context.Context, sandboxID, stubID string) (*SandboxStub, error) {
	return m.transitionStub(ctx, sandboxID, stubID, "restart")
}

func (m *Manager) ListCaptureStatuses(_ context.Context, ids []string) ([]CaptureStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	allowed := make(map[string]bool, len(ids))
	for _, id := range ids {
		allowed[id] = true
	}

	var statuses []CaptureStatus
	for id, runtime := range m.captures {
		if len(allowed) > 0 && !allowed[id] {
			continue
		}
		statuses = append(statuses, runtime.status)
	}
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].CaptureConfigID < statuses[j].CaptureConfigID
	})
	return statuses, nil
}

func (m *Manager) RecordCapture(ctx context.Context, configID string, payload string, capturedAt time.Time) (CaptureStatus, error) {
	m.mu.Lock()
	runtime, ok := m.captures[configID]
	if !ok {
		m.mu.Unlock()
		return CaptureStatus{}, ErrNotFound
	}
	cfg := runtime.cfg
	m.mu.Unlock()

	record := Record{
		Topic:     firstTopic(cfg.Topics),
		Partition: 0,
		Offset:    capturedAt.UTC().UnixNano(),
		Timestamp: capturedAt.UTC(),
		Value:     []byte(payload),
	}
	if err := m.persistRecord(configID, record); err != nil {
		return CaptureStatus{}, err
	}
	statuses, err := m.ListCaptureStatuses(ctx, []string{configID})
	if err != nil {
		return CaptureStatus{}, err
	}
	if len(statuses) == 0 {
		return CaptureStatus{}, ErrNotFound
	}
	return statuses[0], nil
}

func (m *Manager) ensureAndMaybeStartCapture(cfg CaptureConfig) (*captureRuntime, CaptureStatus) {
	m.mu.Lock()
	runtime := m.ensureCaptureLocked(cfg)
	status := runtime.status
	if cfg.Enabled && runtime.captureCancel == nil {
		// Wait for a previous worker to finish if one is still running.
		if runtime.captureDone != nil {
			<-runtime.captureDone
		}
		captureCtx, cancel := context.WithCancel(context.Background())
		runtime.captureCtx = captureCtx
		runtime.captureCancel = cancel
		runtime.captureDone = make(chan struct{})
		go func() {
			m.runCaptureWorker(captureCtx, cfg.ID)
			close(runtime.captureDone)
		}()
	}
	if !cfg.Enabled {
		if runtime.captureCancel != nil {
			runtime.captureCancel()
			runtime.captureCancel = nil
			runtime.captureCtx = nil
		}
		runtime.status.State = StateStopped
		runtime.status.LastError = ""
		m.touchCaptureLocked(runtime)
		status = runtime.status
	}
	m.mu.Unlock()
	return runtime, status
}

func (m *Manager) ensureCaptureLocked(cfg CaptureConfig) *captureRuntime {
	runtime, ok := m.captures[cfg.ID]
	if ok {
		runtime.cfg = cfg
		if !runtime.loaded {
			m.loadSegmentsLocked(runtime)
		}
		return runtime
	}
	runtime = &captureRuntime{
		cfg:      cfg,
		attached: make(map[string]int),
		status: CaptureStatus{
			CaptureConfigID: cfg.ID,
			SourceVM:        cfg.SourceVM,
			State:           StateStopped,
			UpdatedAt:       time.Now().UTC(),
		},
	}
	m.loadSegmentsLocked(runtime)
	m.captures[cfg.ID] = runtime
	return runtime
}

func (m *Manager) loadSegmentsLocked(runtime *captureRuntime) {
	runtime.loaded = true
	segDir := filepath.Join(m.baseDir, sanitizeID(runtime.cfg.ID))
	entries, err := os.ReadDir(segDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		runtime.status.State = StateError
		runtime.status.LastError = err.Error()
		m.touchCaptureLocked(runtime)
		return
	}

	var segments []segment
	var lastCursor string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(segDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var stored persistedRecord
		if err := json.Unmarshal(data, &stored); err != nil {
			continue
		}
		ts := time.Unix(0, stored.TimestampUnixNano).UTC()
		segments = append(segments, segment{
			path:       path,
			sizeBytes:  int64(len(data)),
			capturedAt: ts,
		})
		lastCursor = stored.Cursor
	}
	sort.Slice(segments, func(i, j int) bool {
		return segments[i].capturedAt.Before(segments[j].capturedAt)
	})
	runtime.segments = segments
	runtime.status.BufferedBytes = 0
	for _, seg := range segments {
		runtime.status.BufferedBytes += seg.sizeBytes
	}
	runtime.status.SegmentCount = len(segments)
	runtime.status.LastResumeCursor = lastCursor
	m.touchCaptureLocked(runtime)
}

func (m *Manager) runCaptureWorker(ctx context.Context, configID string) {
	backoff := time.Second
	for {
		cfg, ok := m.captureConfigSnapshot(configID)
		if !ok {
			return
		}

		consumer, err := m.transport.NewConsumer(cfg)
		if err != nil {
			if !m.updateCaptureError(configID, err) {
				return
			}
			if m.sleep(ctx, backoff) != nil {
				return
			}
			continue
		}

		m.updateCaptureRunning(configID)
		for {
			record, err := consumer.ReadMessage(ctx)
			if err != nil {
				_ = consumer.Close()
				if ctx.Err() != nil {
					return
				}
				if !m.updateCaptureError(configID, err) {
					return
				}
				if m.sleep(ctx, backoff) != nil {
					return
				}
				break
			}
			if err := m.persistRecord(configID, record); err != nil {
				if !m.updateCaptureError(configID, err) {
					return
				}
				continue
			}
			m.updateCaptureCursor(configID, recordCursor(record))
		}
	}
}

func (m *Manager) persistRecord(configID string, record Record) error {
	m.mu.Lock()
	runtime, ok := m.captures[configID]
	if !ok {
		m.mu.Unlock()
		return ErrNotFound
	}
	cfg := runtime.cfg
	m.mu.Unlock()

	redacted, err := m.redactRecord(cfg, record)
	if err != nil {
		return err
	}
	data, err := json.Marshal(redacted)
	if err != nil {
		return fmt.Errorf("marshal record: %w", err)
	}

	segDir := filepath.Join(m.baseDir, sanitizeID(configID))
	if err := os.MkdirAll(segDir, 0o755); err != nil {
		return fmt.Errorf("create segment dir: %w", err)
	}
	segPath := filepath.Join(segDir, segmentFileName(record))
	if err := os.WriteFile(segPath, data, 0o600); err != nil {
		return fmt.Errorf("write segment: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	runtime, ok = m.captures[configID]
	if !ok {
		return ErrNotFound
	}
	runtime.segments = append(runtime.segments, segment{
		path:       segPath,
		sizeBytes:  int64(len(data)),
		capturedAt: record.Timestamp.UTC(),
	})
	sort.Slice(runtime.segments, func(i, j int) bool {
		return runtime.segments[i].capturedAt.Before(runtime.segments[j].capturedAt)
	})
	runtime.status.BufferedBytes += int64(len(data))
	runtime.status.SegmentCount = len(runtime.segments)
	runtime.status.State = StateRunning
	runtime.status.LastError = ""
	m.touchCaptureLocked(runtime)
	m.evictLocked(runtime, time.Now().UTC())
	status := runtime.status
	go m.notifyCapture(status)
	return nil
}

func (m *Manager) transitionStub(ctx context.Context, sandboxID, stubID, action string) (*SandboxStub, error) {
	switch action {
	case "stop":
		return m.stopReplay(stubID, sandboxID)
	case "start":
		return m.startReplay(ctx, stubID, sandboxID, false)
	case "restart":
		return m.startReplay(ctx, stubID, sandboxID, true)
	default:
		return nil, fmt.Errorf("unsupported action %q", action)
	}
}

func (m *Manager) stopReplay(stubID, sandboxID string) (*SandboxStub, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stub, ok := m.stubs[stubID]
	if !ok || stub.SandboxID != sandboxID {
		return nil, ErrNotFound
	}
	if replay, ok := m.replays[stubID]; ok {
		replay.cancel()
		delete(m.replays, stubID)
	}
	stub.State = StateStopped
	stub.LastError = ""
	stub.UpdatedAt = time.Now().UTC()
	cp := cloneStub(stub)
	go m.notifyStub(cp)
	return cp, nil
}

func (m *Manager) startReplay(ctx context.Context, stubID, sandboxID string, reset bool) (*SandboxStub, error) {
	m.mu.Lock()
	stub, ok := m.stubs[stubID]
	if !ok || stub.SandboxID != sandboxID {
		m.mu.Unlock()
		return nil, ErrNotFound
	}
	if replay, ok := m.replays[stubID]; ok {
		replay.cancel()
		delete(m.replays, stubID)
	}
	if reset {
		stub.LastReplayCursor = "head"
	}
	replayCtx, cancel := context.WithCancel(context.Background())
	m.replays[stubID] = &replayRuntime{ctx: replayCtx, cancel: cancel}
	stub.State = StateRunning
	stub.LastError = ""
	stub.UpdatedAt = time.Now().UTC()
	cp := cloneStub(stub)
	m.mu.Unlock()

	go m.runReplayWorker(replayCtx, stubID)
	m.notifyStub(cp)
	return cp, nil
}

func (m *Manager) runReplayWorker(ctx context.Context, stubID string) {
	stub, cfg, records, err := m.snapshotReplay(stubID)
	if err != nil {
		m.setStubError(stubID, err)
		return
	}
	if len(records) == 0 {
		m.setStubError(stubID, fmt.Errorf("no captured kafka records available for replay"))
		return
	}

	producer, err := m.transport.NewProducer(stub.BrokerEndpoint)
	if err != nil {
		m.setStubError(stubID, err)
		return
	}
	defer func() { _ = producer.Close() }()

	started := stub.LastReplayCursor == "" || stub.LastReplayCursor == "head"
	lastSentAt := time.Time{}
	for _, record := range records {
		if !started {
			if record.Cursor == stub.LastReplayCursor {
				started = true
			}
			continue
		}
		if !lastSentAt.IsZero() {
			delay := clampDelay(time.Unix(0, record.TimestampUnixNano).UTC().Sub(lastSentAt))
			if delay > 0 {
				if err := m.sleep(ctx, delay); err != nil {
					return
				}
			}
		}
		if err := producer.WriteMessage(ctx, Record{
			Topic:     record.Topic,
			Partition: record.Partition,
			Offset:    record.Offset,
			Key:       record.KeyBytes(),
			Headers:   record.HeadersList(),
			Timestamp: time.Unix(0, record.TimestampUnixNano).UTC(),
			Value:     record.ValueBytes(),
		}); err != nil {
			m.setStubError(stubID, err)
			return
		}
		lastSentAt = time.Unix(0, record.TimestampUnixNano).UTC()
		m.updateStubCursor(stubID, record.Cursor)
	}
	m.finishStubReplay(stubID)

	_ = cfg
}

func (m *Manager) snapshotReplay(stubID string) (*SandboxStub, CaptureConfig, []persistedRecord, error) {
	m.mu.Lock()
	stub, ok := m.stubs[stubID]
	if !ok {
		m.mu.Unlock()
		return nil, CaptureConfig{}, nil, ErrNotFound
	}
	runtime, ok := m.captures[stub.CaptureConfigID]
	if !ok {
		m.mu.Unlock()
		return nil, CaptureConfig{}, nil, ErrNotFound
	}
	cfg := runtime.cfg
	stubCopy := cloneStub(stub)
	m.mu.Unlock()

	records, err := m.loadRecords(cfg.ID)
	if err != nil {
		return nil, CaptureConfig{}, nil, err
	}
	if stubCopy.ReplayWindow > 0 {
		cutoff := time.Now().UTC().Add(-stubCopy.ReplayWindow)
		filtered := records[:0]
		for _, record := range records {
			if time.Unix(0, record.TimestampUnixNano).UTC().Before(cutoff) {
				continue
			}
			filtered = append(filtered, record)
		}
		records = filtered
	}
	if len(stubCopy.Topics) > 0 {
		filtered := records[:0]
		for _, record := range records {
			if !slices.Contains(stubCopy.Topics, record.Topic) {
				continue
			}
			filtered = append(filtered, record)
		}
		records = filtered
	}
	return stubCopy, cfg, records, nil
}

func (m *Manager) loadRecords(configID string) ([]persistedRecord, error) {
	segDir := filepath.Join(m.baseDir, sanitizeID(configID))
	entries, err := os.ReadDir(segDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	records := make([]persistedRecord, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(segDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var record persistedRecord
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].TimestampUnixNano != records[j].TimestampUnixNano {
			return records[i].TimestampUnixNano < records[j].TimestampUnixNano
		}
		if records[i].Partition != records[j].Partition {
			return records[i].Partition < records[j].Partition
		}
		return records[i].Offset < records[j].Offset
	})
	return records, nil
}

func (m *Manager) evictLocked(runtime *captureRuntime, now time.Time) {
	for len(runtime.segments) > 0 {
		head := runtime.segments[0]
		remove := false
		if runtime.cfg.MaxBufferAge > 0 && now.Sub(head.capturedAt) > runtime.cfg.MaxBufferAge {
			remove = true
		}
		if !remove && runtime.cfg.MaxBufferBytes > 0 && runtime.status.BufferedBytes > runtime.cfg.MaxBufferBytes {
			remove = true
		}
		if !remove {
			break
		}
		_ = os.Remove(head.path)
		runtime.status.BufferedBytes -= head.sizeBytes
		runtime.segments = runtime.segments[1:]
		runtime.status.SegmentCount = len(runtime.segments)
	}
	m.touchCaptureLocked(runtime)
}

func (m *Manager) captureConfigSnapshot(configID string) (CaptureConfig, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime, ok := m.captures[configID]
	if !ok || !runtime.cfg.Enabled {
		return CaptureConfig{}, false
	}
	return runtime.cfg, true
}

func (m *Manager) updateCaptureRunning(configID string) {
	m.mu.Lock()
	runtime, ok := m.captures[configID]
	if !ok {
		m.mu.Unlock()
		return
	}
	runtime.status.State = StateRunning
	runtime.status.LastError = ""
	m.touchCaptureLocked(runtime)
	status := runtime.status
	m.mu.Unlock()
	m.notifyCapture(status)
}

func (m *Manager) updateCaptureError(configID string, err error) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	runtime, ok := m.captures[configID]
	if !ok {
		return false
	}
	runtime.status.State = StateError
	runtime.status.LastError = err.Error()
	m.touchCaptureLocked(runtime)
	go m.notifyCapture(runtime.status)
	return runtime.captureCtx != nil
}

func (m *Manager) updateCaptureCursor(configID, cursor string) {
	m.mu.Lock()
	runtime, ok := m.captures[configID]
	if !ok {
		m.mu.Unlock()
		return
	}
	runtime.status.State = StateRunning
	runtime.status.LastResumeCursor = cursor
	runtime.status.LastError = ""
	m.touchCaptureLocked(runtime)
	status := runtime.status
	m.mu.Unlock()
	m.notifyCapture(status)
}

func (m *Manager) updateStubCursor(stubID, cursor string) {
	m.mu.Lock()
	stub, ok := m.stubs[stubID]
	if !ok {
		m.mu.Unlock()
		return
	}
	stub.LastReplayCursor = cursor
	stub.State = StateRunning
	stub.LastError = ""
	stub.UpdatedAt = time.Now().UTC()
	cp := cloneStub(stub)
	m.mu.Unlock()
	m.notifyStub(cp)
}

func (m *Manager) finishStubReplay(stubID string) {
	m.mu.Lock()
	stub, ok := m.stubs[stubID]
	if !ok {
		m.mu.Unlock()
		return
	}
	delete(m.replays, stubID)
	stub.State = StateStopped
	stub.LastError = ""
	stub.UpdatedAt = time.Now().UTC()
	cp := cloneStub(stub)
	m.mu.Unlock()
	m.notifyStub(cp)
}

func (m *Manager) setStubError(stubID string, err error) {
	m.mu.Lock()
	stub, ok := m.stubs[stubID]
	if !ok {
		m.mu.Unlock()
		return
	}
	delete(m.replays, stubID)
	stub.State = StateError
	stub.LastError = err.Error()
	stub.UpdatedAt = time.Now().UTC()
	cp := cloneStub(stub)
	m.mu.Unlock()
	m.notifyStub(cp)
}

func (m *Manager) touchCaptureLocked(runtime *captureRuntime) {
	runtime.status.SourceVM = runtime.cfg.SourceVM
	runtime.status.UpdatedAt = time.Now().UTC()
}

func (m *Manager) redactRecord(cfg CaptureConfig, record Record) (persistedRecord, error) {
	if err := validateCodec(cfg.Codec); err != nil {
		return persistedRecord{}, err
	}
	value := record.Value
	switch cfg.Codec {
	case "", "text":
		value = []byte(m.redactor.Redact(string(record.Value)))
	case "json":
		redacted, err := redactJSON(m.redactor, record.Value, cfg.RedactionRules)
		if err != nil {
			value = []byte(m.redactor.Redact(string(record.Value)))
		} else {
			value = redacted
		}
	}

	key := record.Key
	if utf8.Valid(key) {
		key = []byte(m.redactor.Redact(string(key)))
	}
	headers := make([]persistedHeader, 0, len(record.Headers))
	for _, header := range record.Headers {
		headerValue := header.Value
		if utf8.Valid(headerValue) {
			headerValue = []byte(m.redactor.Redact(string(headerValue)))
		}
		headers = append(headers, persistedHeader{
			Key:   header.Key,
			Value: base64.StdEncoding.EncodeToString(headerValue),
		})
	}

	return persistedRecord{
		Cursor:            recordCursor(record),
		Topic:             record.Topic,
		Partition:         record.Partition,
		Offset:            record.Offset,
		Key:               base64.StdEncoding.EncodeToString(key),
		Headers:           headers,
		TimestampUnixNano: record.Timestamp.UTC().UnixNano(),
		Value:             base64.StdEncoding.EncodeToString(value),
	}, nil
}

func (m *Manager) notifyCapture(status CaptureStatus) {
	if m.hooks.OnCaptureStatus != nil {
		m.hooks.OnCaptureStatus(status)
	}
}

func (m *Manager) notifyStub(stub *SandboxStub) {
	if m.hooks.OnSandboxStub != nil {
		m.hooks.OnSandboxStub(cloneStub(stub))
	}
}

type persistedHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type persistedRecord struct {
	Cursor            string            `json:"cursor"`
	Topic             string            `json:"topic"`
	Partition         int32             `json:"partition"`
	Offset            int64             `json:"offset"`
	Key               string            `json:"key"`
	Headers           []persistedHeader `json:"headers,omitempty"`
	TimestampUnixNano int64             `json:"timestamp_unix_nano"`
	Value             string            `json:"value"`
}

func (p persistedRecord) KeyBytes() []byte {
	data, _ := base64.StdEncoding.DecodeString(p.Key)
	return data
}

func (p persistedRecord) ValueBytes() []byte {
	data, _ := base64.StdEncoding.DecodeString(p.Value)
	return data
}

func (p persistedRecord) HeadersList() []Header {
	headers := make([]Header, 0, len(p.Headers))
	for _, header := range p.Headers {
		value, _ := base64.StdEncoding.DecodeString(header.Value)
		headers = append(headers, Header{Key: header.Key, Value: value})
	}
	return headers
}

func redactJSON(redactor *redact.Redactor, payload []byte, rules []string) ([]byte, error) {
	var doc any
	if err := json.Unmarshal(payload, &doc); err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		doc = redactor.RedactAny(doc)
	} else {
		for _, rule := range rules {
			applyJSONRule(redactor, doc, strings.Split(rule, "."))
		}
	}
	return json.Marshal(doc)
}

func applyJSONRule(redactor *redact.Redactor, node any, path []string) {
	if len(path) == 0 {
		return
	}
	obj, ok := node.(map[string]any)
	if !ok {
		return
	}
	value, ok := obj[path[0]]
	if !ok {
		return
	}
	if len(path) == 1 {
		obj[path[0]] = redactor.RedactAny(value)
		return
	}
	applyJSONRule(redactor, value, path[1:])
}

func segmentFileName(record Record) string {
	return fmt.Sprintf("%020d-%s-%06d-%020d.json",
		record.Timestamp.UTC().UnixNano(),
		sanitizeID(record.Topic),
		record.Partition,
		record.Offset,
	)
}

func recordCursor(record Record) string {
	return fmt.Sprintf("%s/%d/%d", record.Topic, record.Partition, record.Offset)
}

func defaultReplayWindow(maxAge time.Duration) time.Duration {
	if maxAge > 0 && maxAge < 5*time.Minute {
		return maxAge
	}
	return 5 * time.Minute
}

func clampDelay(delay time.Duration) time.Duration {
	if delay <= 0 {
		return 0
	}
	if delay > 2*time.Second {
		return 2 * time.Second
	}
	return delay
}

func sanitizeID(v string) string {
	v = strings.ToLower(v)
	v = strings.ReplaceAll(v, "/", "-")
	v = strings.ReplaceAll(v, ":", "-")
	v = strings.ReplaceAll(v, "_", "-")
	v = strings.ReplaceAll(v, "..", "-")
	v = strings.ReplaceAll(v, string(rune(0)), "")
	return v
}

func validateCodec(codec string) error {
	switch codec {
	case "", "json", "text":
		return nil
	default:
		return fmt.Errorf("unsupported codec %q", codec)
	}
}

func cloneStub(stub *SandboxStub) *SandboxStub {
	cp := *stub
	cp.Topics = append([]string(nil), stub.Topics...)
	return &cp
}

func stubForID(m *Manager, stubID string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if stub, ok := m.stubs[stubID]; ok {
		return stub.SandboxID
	}
	return ""
}

func firstTopic(topics []string) string {
	if len(topics) == 0 {
		return "logs"
	}
	return topics[0]
}

func normalizeBrokerEndpoint(endpoint string) string {
	if endpoint == "" {
		return "127.0.0.1:9092"
	}
	if strings.Contains(endpoint, ":") {
		return endpoint
	}
	return endpoint + ":9092"
}
