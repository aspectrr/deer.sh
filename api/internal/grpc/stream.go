package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"

	"github.com/aspectrr/fluid.sh/api/internal/auth"
	"github.com/aspectrr/fluid.sh/api/internal/registry"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

// StreamHandler implements fluidv1.HostServiceServer.
type StreamHandler struct {
	fluidv1.UnimplementedHostServiceServer

	registry         *registry.Registry
	store            store.Store
	logger           *slog.Logger
	heartbeatTimeout time.Duration

	// pendingRequests maps request_id -> response channel.
	pendingRequests sync.Map // map[string]chan *fluidv1.HostMessage

	// streams maps host_id -> active server stream.
	streams sync.Map // map[string]fluidv1.HostService_ConnectServer

	// streamMu holds a per-host mutex to serialize stream.Send calls.
	streamMu sync.Map // map[string]*sync.Mutex

	// cancelFns maps host_id -> context.CancelFunc for the active connection.
	// Used to cancel old connections when a host reconnects.
	cancelFns sync.Map // map[string]context.CancelFunc
}

// NewStreamHandler creates a stream handler wired to the given dependencies.
func NewStreamHandler(
	reg *registry.Registry,
	st store.Store,
	logger *slog.Logger,
	heartbeatTimeout time.Duration,
) *StreamHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &StreamHandler{
		registry:         reg,
		store:            st,
		logger:           logger.With("component", "stream-handler"),
		heartbeatTimeout: heartbeatTimeout,
	}
}

// hostMu returns the per-host mutex, creating one if needed.
func (h *StreamHandler) hostMu(hostID string) *sync.Mutex {
	v, _ := h.streamMu.LoadOrStore(hostID, &sync.Mutex{})
	mu, ok := v.(*sync.Mutex)
	if !ok {
		h.logger.Error("streamMu contains non-Mutex value", "host_id", hostID)
		return &sync.Mutex{}
	}
	return mu
}

// Connect handles a single bidirectional stream from a sandbox host.
func (h *StreamHandler) Connect(stream fluidv1.HostService_ConnectServer) error {
	firstMsg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("recv registration: %w", err)
	}

	reg := firstMsg.GetRegistration()
	if reg == nil {
		return fmt.Errorf("first message must be HostRegistration")
	}

	hostID := reg.GetHostId()
	hostname := reg.GetHostname()
	orgID := auth.OrgIDFromContext(stream.Context())
	tokenID := auth.TokenIDFromContext(stream.Context())

	if tokenID == "" {
		return fmt.Errorf("missing token identity from auth context")
	}

	// Override daemon-supplied hostID with server-assigned identity derived
	// from the authenticated token so a daemon cannot impersonate another host.
	if hostID != tokenID {
		h.logger.Warn("daemon-supplied host_id differs from token, overriding",
			"daemon_host_id", hostID, "token_id", tokenID)
		hostID = tokenID
	}

	logger := h.logger.With("host_id", hostID, "hostname", hostname, "org_id", orgID)
	logger.Info("host connecting", "version", reg.GetVersion())

	// Send RegistrationAck.
	ack := &fluidv1.ControlMessage{
		RequestId: firstMsg.GetRequestId(),
		Payload: &fluidv1.ControlMessage_RegistrationAck{
			RegistrationAck: &fluidv1.RegistrationAck{
				Accepted:       true,
				AssignedHostId: hostID,
			},
		},
	}
	if err := stream.Send(ack); err != nil {
		return fmt.Errorf("send registration ack: %w", err)
	}

	// Cancel any existing connection for this host to avoid duplicate streams.
	if oldCancel, loaded := h.cancelFns.LoadAndDelete(hostID); loaded {
		if fn, ok := oldCancel.(context.CancelFunc); ok {
			fn()
		} else {
			logger.Error("cancelFns contains non-CancelFunc value")
		}
	}

	// Store the stream before registering so it is available immediately
	// when other goroutines observe the host in the registry.
	h.streams.Store(hostID, stream)
	if err := h.registry.Register(hostID, orgID, hostname, stream); err != nil {
		h.streams.Delete(hostID)
		return fmt.Errorf("register host: %w", err)
	}
	h.registry.SetRegistration(hostID, reg)
	h.registry.UpdateHeartbeatCounts(hostID, 0, int32(len(reg.GetSourceVms())))

	// Persist or update host in the database using a background context
	// so the write completes even if the stream context is cancelled.
	regCtx, regCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer regCancel()
	h.persistHostRegistration(regCtx, hostID, orgID, reg)

	logger.Info("host registered",
		"total_cpus", reg.GetTotalCpus(),
		"total_memory_mb", reg.GetTotalMemoryMb(),
		"base_images", reg.GetBaseImages(),
	)

	// Spawn heartbeat monitor.
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()
	h.cancelFns.Store(hostID, cancel)

	go h.monitorHeartbeat(ctx, cancel, hostID, logger)

	// Cleanup on disconnect.
	defer func() {
		// Only clean up if we still own the stream. A reconnecting host
		// stores its new stream before re-registering, so if CompareAndDelete
		// fails our state has already been replaced and cleanup would clobber
		// the new connection.
		if h.streams.CompareAndDelete(hostID, stream) {
			h.cancelFns.Delete(hostID)
			h.registry.Unregister(hostID)
			h.streamMu.Delete(hostID)
			logger.Info("host disconnected")
		} else {
			logger.Info("connection replaced, skipping stale cleanup")
		}
	}()

	// Main recv loop.
	for {
		msg, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				logger.Info("host stream closed by peer")
				return nil
			}
			logger.Error("stream recv error", "error", err)
			return err
		}

		h.handleHostMessage(ctx, hostID, msg, logger)
	}
}

func (h *StreamHandler) handleHostMessage(ctx context.Context, hostID string, msg *fluidv1.HostMessage, logger *slog.Logger) {
	switch msg.Payload.(type) {
	case *fluidv1.HostMessage_Heartbeat:
		hb := msg.GetHeartbeat()
		h.registry.UpdateHeartbeat(hostID)
		h.registry.UpdateHeartbeatCounts(hostID, hb.GetActiveSandboxes(), hb.GetSourceVmCount())
		if err := h.store.UpdateHostHeartbeat(
			ctx,
			hostID,
			hb.GetAvailableCpus(),
			hb.GetAvailableMemoryMb(),
			hb.GetAvailableDiskMb(),
		); err != nil {
			h.logger.Warn("failed to update heartbeat", "host_id", hostID, "error", err)
		}
		h.registry.UpdateResources(hostID, hb.GetAvailableCpus(), hb.GetAvailableMemoryMb())

	case *fluidv1.HostMessage_ResourceReport:
		h.registry.UpdateHeartbeat(hostID)
		logger.Info("received resource report")

	case *fluidv1.HostMessage_ErrorReport:
		er := msg.GetErrorReport()
		logger.Error("host reported error",
			"sandbox_id", er.GetSandboxId(),
			"error", er.GetError(),
			"context", er.GetContext(),
		)

	default:
		reqID := msg.GetRequestId()
		if reqID == "" {
			logger.Warn("received message without request_id, dropping")
			return
		}
		if ch, ok := h.pendingRequests.LoadAndDelete(reqID); ok {
			respCh, ok := ch.(chan *fluidv1.HostMessage)
			if !ok {
				logger.Error("pendingRequests contains non-channel value", "request_id", reqID)
				return
			}
			respCh <- msg
		} else {
			logger.Warn("no pending request for response", "request_id", reqID)
		}
	}
}

// SendAndWait sends a ControlMessage to a specific host and blocks until the
// host responds with a matching request_id, the context is cancelled, or the
// timeout expires.
func (h *StreamHandler) SendAndWait(ctx context.Context, hostID string, msg *fluidv1.ControlMessage, timeout time.Duration) (*fluidv1.HostMessage, error) {
	streamVal, ok := h.streams.Load(hostID)
	if !ok {
		return nil, fmt.Errorf("host %s is not connected", hostID)
	}
	stream, ok := streamVal.(fluidv1.HostService_ConnectServer)
	if !ok {
		return nil, fmt.Errorf("host %s: stream has unexpected type", hostID)
	}

	reqID := msg.GetRequestId()
	if reqID == "" {
		return nil, fmt.Errorf("control message must have a request_id")
	}

	respCh := make(chan *fluidv1.HostMessage, 1)
	h.pendingRequests.Store(reqID, respCh)
	defer h.pendingRequests.Delete(reqID)

	mu := h.hostMu(hostID)
	mu.Lock()
	err := stream.Send(msg)
	mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("send to host %s: %w", hostID, err)
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case resp := <-respCh:
		return resp, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled waiting for response from host %s", hostID)
	case <-timer.C:
		return nil, fmt.Errorf("timeout waiting for response from host %s (request_id=%s)", hostID, reqID)
	}
}

// monitorHeartbeat checks for heartbeat timeouts on a connected host.
//
// Timing: the check interval is heartbeatTimeout/3 (default 90s/3 = 30s).
// A disconnect requires 3 consecutive misses, so the effective disconnect
// window is ~2-3 minutes after the last successful heartbeat. This
// intentional buffer tolerates transient network issues. For tighter SLAs,
// reduce ORCHESTRATOR_HEARTBEAT_TIMEOUT.
func (h *StreamHandler) monitorHeartbeat(ctx context.Context, cancel context.CancelFunc, hostID string, logger *slog.Logger) {
	interval := h.heartbeatTimeout / 3
	if interval < 10*time.Second {
		interval = 10 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	consecutiveMisses := 0
	const maxMisses = 3

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			host, ok := h.registry.GetHost(hostID)
			if !ok {
				return
			}
			if time.Since(host.LastHeartbeat) > h.heartbeatTimeout {
				consecutiveMisses++
				logger.Warn("host heartbeat overdue",
					"last_heartbeat", host.LastHeartbeat,
					"overdue_by", time.Since(host.LastHeartbeat)-h.heartbeatTimeout,
					"consecutive_misses", consecutiveMisses,
				)
				if consecutiveMisses >= maxMisses {
					logger.Error("host heartbeat missed too many times, disconnecting", "consecutive_misses", consecutiveMisses)
					cancel()
					return
				}
			} else {
				consecutiveMisses = 0
			}
		}
	}
}

func (h *StreamHandler) persistHostRegistration(ctx context.Context, hostID, orgID string, reg *fluidv1.HostRegistration) {
	existing, err := h.store.GetHost(ctx, hostID)
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			h.logger.Error("failed to look up host in store", "host_id", hostID, "error", err)
			return
		}
		host := hostFromRegistration(hostID, orgID, reg)
		if createErr := h.store.CreateHost(ctx, host); createErr != nil {
			h.logger.Error("failed to create host in store", "host_id", hostID, "error", createErr)
		}
		return
	}

	existing.OrgID = orgID
	existing.Hostname = reg.GetHostname()
	existing.Version = reg.GetVersion()
	existing.TotalCPUs = reg.GetTotalCpus()
	existing.TotalMemoryMB = reg.GetTotalMemoryMb()
	existing.TotalDiskMB = reg.GetTotalDiskMb()
	existing.AvailableCPUs = reg.GetAvailableCpus()
	existing.AvailableMemoryMB = reg.GetAvailableMemoryMb()
	existing.AvailableDiskMB = reg.GetAvailableDiskMb()
	existing.BaseImages = reg.GetBaseImages()
	existing.Status = store.HostStatusOnline
	existing.LastHeartbeat = time.Now()

	existing.SourceVMs = sourceVMsFromProto(reg.GetSourceVms())
	existing.Bridges = bridgesFromProto(reg.GetBridges())

	if err := h.store.UpdateHost(ctx, existing); err != nil {
		h.logger.Error("failed to update host in store", "host_id", hostID, "error", err)
	}
}

func sourceVMsFromProto(vms []*fluidv1.SourceVMInfo) store.SourceVMSlice {
	result := make(store.SourceVMSlice, 0, len(vms))
	for _, vm := range vms {
		result = append(result, store.SourceVMJSON{
			Name:      vm.GetName(),
			State:     vm.GetState(),
			IPAddress: vm.GetIpAddress(),
			Prepared:  vm.GetPrepared(),
		})
	}
	return result
}

func bridgesFromProto(bridges []*fluidv1.BridgeInfo) store.BridgeSlice {
	result := make(store.BridgeSlice, 0, len(bridges))
	for _, b := range bridges {
		result = append(result, store.BridgeJSON{
			Name:   b.GetName(),
			Subnet: b.GetSubnet(),
		})
	}
	return result
}

func hostFromRegistration(hostID, orgID string, reg *fluidv1.HostRegistration) *store.Host {
	return &store.Host{
		ID:                hostID,
		OrgID:             orgID,
		Hostname:          reg.GetHostname(),
		Version:           reg.GetVersion(),
		TotalCPUs:         reg.GetTotalCpus(),
		TotalMemoryMB:     reg.GetTotalMemoryMb(),
		TotalDiskMB:       reg.GetTotalDiskMb(),
		AvailableCPUs:     reg.GetAvailableCpus(),
		AvailableMemoryMB: reg.GetAvailableMemoryMb(),
		AvailableDiskMB:   reg.GetAvailableDiskMb(),
		BaseImages:        reg.GetBaseImages(),
		SourceVMs:         sourceVMsFromProto(reg.GetSourceVms()),
		Bridges:           bridgesFromProto(reg.GetBridges()),
		Status:            store.HostStatusOnline,
		LastHeartbeat:     time.Now(),
	}
}
