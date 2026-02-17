package grpc

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"

	"github.com/aspectrr/fluid.sh/api/internal/registry"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

// StreamHandler implements fluidv1.HostServiceServer.
type StreamHandler struct {
	fluidv1.UnimplementedHostServiceServer

	registry *registry.Registry
	store    store.Store
	logger   *slog.Logger

	// pendingRequests maps request_id -> response channel.
	pendingRequests sync.Map // map[string]chan *fluidv1.HostMessage

	// streams maps host_id -> active server stream.
	streams sync.Map // map[string]fluidv1.HostService_ConnectServer
}

// NewStreamHandler creates a stream handler wired to the given dependencies.
func NewStreamHandler(
	reg *registry.Registry,
	st store.Store,
	logger *slog.Logger,
) *StreamHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &StreamHandler{
		registry: reg,
		store:    st,
		logger:   logger.With("component", "stream-handler"),
	}
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

	logger := h.logger.With("host_id", hostID, "hostname", hostname)
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

	// Register host.
	if err := h.registry.Register(hostID, hostname, stream); err != nil {
		return fmt.Errorf("register host: %w", err)
	}
	h.registry.SetRegistration(hostID, reg)
	h.streams.Store(hostID, stream)

	// Persist or update host in the database.
	h.persistHostRegistration(stream.Context(), hostID, reg)

	logger.Info("host registered",
		"total_cpus", reg.GetTotalCpus(),
		"total_memory_mb", reg.GetTotalMemoryMb(),
		"base_images", reg.GetBaseImages(),
	)

	// Spawn heartbeat monitor.
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	go h.monitorHeartbeat(ctx, hostID, logger)

	// Cleanup on disconnect.
	defer func() {
		h.registry.Unregister(hostID)
		h.streams.Delete(hostID)
		logger.Info("host disconnected")
	}()

	// Main recv loop.
	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
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
		_ = h.store.UpdateHostHeartbeat(
			ctx,
			hostID,
			hb.GetAvailableCpus(),
			hb.GetAvailableMemoryMb(),
			hb.GetAvailableDiskMb(),
		)

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
			respCh := ch.(chan *fluidv1.HostMessage)
			select {
			case respCh <- msg:
			default:
				logger.Warn("response channel full, dropping", "request_id", reqID)
			}
		} else {
			logger.Warn("no pending request for response", "request_id", reqID)
		}
	}
}

// SendAndWait sends a ControlMessage to a specific host and blocks until the
// host responds with a matching request_id or the timeout expires.
func (h *StreamHandler) SendAndWait(hostID string, msg *fluidv1.ControlMessage, timeout time.Duration) (*fluidv1.HostMessage, error) {
	streamVal, ok := h.streams.Load(hostID)
	if !ok {
		return nil, fmt.Errorf("host %s is not connected", hostID)
	}
	stream := streamVal.(fluidv1.HostService_ConnectServer)

	reqID := msg.GetRequestId()
	if reqID == "" {
		return nil, fmt.Errorf("control message must have a request_id")
	}

	respCh := make(chan *fluidv1.HostMessage, 1)
	h.pendingRequests.Store(reqID, respCh)
	defer h.pendingRequests.Delete(reqID)

	if err := stream.Send(msg); err != nil {
		return nil, fmt.Errorf("send to host %s: %w", hostID, err)
	}

	select {
	case resp := <-respCh:
		return resp, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for response from host %s (request_id=%s)", hostID, reqID)
	}
}

// GetStream returns the active stream for a host, if connected.
func (h *StreamHandler) GetStream(hostID string) (fluidv1.HostService_ConnectServer, bool) {
	v, ok := h.streams.Load(hostID)
	if !ok {
		return nil, false
	}
	return v.(fluidv1.HostService_ConnectServer), true
}

func (h *StreamHandler) monitorHeartbeat(ctx context.Context, hostID string, logger *slog.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			host, ok := h.registry.GetHost(hostID)
			if !ok {
				return
			}
			if time.Since(host.LastHeartbeat) > 90*time.Second {
				logger.Warn("host heartbeat overdue",
					"last_heartbeat", host.LastHeartbeat,
					"overdue_by", time.Since(host.LastHeartbeat)-90*time.Second,
				)
			}
		}
	}
}

func (h *StreamHandler) persistHostRegistration(ctx context.Context, hostID string, reg *fluidv1.HostRegistration) {
	existing, err := h.store.GetHost(ctx, hostID)
	if err != nil {
		host := hostFromRegistration(hostID, reg)
		if createErr := h.store.CreateHost(ctx, host); createErr != nil {
			h.logger.Error("failed to create host in store", "host_id", hostID, "error", createErr)
		}
		return
	}

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

	sourceVMs := make(store.SourceVMSlice, 0, len(reg.GetSourceVms()))
	for _, vm := range reg.GetSourceVms() {
		sourceVMs = append(sourceVMs, store.SourceVMJSON{
			Name:      vm.GetName(),
			State:     vm.GetState(),
			IPAddress: vm.GetIpAddress(),
			Prepared:  vm.GetPrepared(),
		})
	}
	existing.SourceVMs = sourceVMs

	bridges := make(store.BridgeSlice, 0, len(reg.GetBridges()))
	for _, b := range reg.GetBridges() {
		bridges = append(bridges, store.BridgeJSON{
			Name:   b.GetName(),
			Subnet: b.GetSubnet(),
		})
	}
	existing.Bridges = bridges

	if err := h.store.UpdateHost(ctx, existing); err != nil {
		h.logger.Error("failed to update host in store", "host_id", hostID, "error", err)
	}
}

func hostFromRegistration(hostID string, reg *fluidv1.HostRegistration) *store.Host {
	sourceVMs := make(store.SourceVMSlice, 0, len(reg.GetSourceVms()))
	for _, vm := range reg.GetSourceVms() {
		sourceVMs = append(sourceVMs, store.SourceVMJSON{
			Name:      vm.GetName(),
			State:     vm.GetState(),
			IPAddress: vm.GetIpAddress(),
			Prepared:  vm.GetPrepared(),
		})
	}

	bridges := make(store.BridgeSlice, 0, len(reg.GetBridges()))
	for _, b := range reg.GetBridges() {
		bridges = append(bridges, store.BridgeJSON{
			Name:   b.GetName(),
			Subnet: b.GetSubnet(),
		})
	}

	return &store.Host{
		ID:                hostID,
		Hostname:          reg.GetHostname(),
		Version:           reg.GetVersion(),
		TotalCPUs:         reg.GetTotalCpus(),
		TotalMemoryMB:     reg.GetTotalMemoryMb(),
		TotalDiskMB:       reg.GetTotalDiskMb(),
		AvailableCPUs:     reg.GetAvailableCpus(),
		AvailableMemoryMB: reg.GetAvailableMemoryMb(),
		AvailableDiskMB:   reg.GetAvailableDiskMb(),
		BaseImages:        reg.GetBaseImages(),
		SourceVMs:         sourceVMs,
		Bridges:           bridges,
		Status:            store.HostStatusOnline,
		LastHeartbeat:     time.Now(),
	}
}
