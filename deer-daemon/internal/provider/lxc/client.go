package lxc

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is an HTTP client for the Proxmox VE LXC API.
// Authentication uses API tokens.
type Client struct {
	baseURL    string
	tokenID    string
	secret     string
	node       string
	httpClient *http.Client
	logger     *slog.Logger
	maxRetries int
}

// NewClient creates a new Proxmox LXC API client.
func NewClient(cfg Config, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !cfg.VerifySSL,
		},
	}
	if !cfg.VerifySSL {
		logger.Warn("TLS certificate verification is disabled")
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	return &Client{
		baseURL: strings.TrimRight(cfg.Host, "/"),
		tokenID: cfg.TokenID,
		secret:  cfg.Secret,
		node:    cfg.Node,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
		logger:     logger,
		maxRetries: 3,
	}
}

// do executes an HTTP request against the Proxmox API with retry logic.
func (c *Client) do(ctx context.Context, method, path string, body url.Values) (json.RawMessage, error) {
	apiURL := fmt.Sprintf("%s/api2/json%s", c.baseURL, path)

	var lastErr error
	delay := 1 * time.Second

	for attempt := 1; attempt <= c.maxRetries; attempt++ {
		var bodyReader io.Reader
		if body != nil {
			bodyReader = strings.NewReader(body.Encode())
		}

		req, err := http.NewRequestWithContext(ctx, method, apiURL, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s=%s", c.tokenID, c.secret))
		if body != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, fmt.Errorf("request %s %s: %w", method, path, err)
			}
			lastErr = fmt.Errorf("request %s %s: %w", method, path, err)
			if attempt < c.maxRetries {
				c.logger.Warn("retrying request", "method", method, "path", path, "attempt", attempt, "error", lastErr)
				jitteredDelay := time.Duration(float64(delay) * (0.9 + rand.Float64()*0.2))
				select {
				case <-ctx.Done():
					return nil, fmt.Errorf("request %s %s: %w", method, path, ctx.Err())
				case <-time.After(jitteredDelay):
				}
				delay *= 2
			}
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("API %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
			if attempt < c.maxRetries {
				c.logger.Warn("retrying request", "method", method, "path", path, "attempt", attempt, "error", lastErr)
				jitteredDelay := time.Duration(float64(delay) * (0.9 + rand.Float64()*0.2))
				select {
				case <-ctx.Done():
					return nil, fmt.Errorf("request %s %s: %w", method, path, ctx.Err())
				case <-time.After(jitteredDelay):
				}
				delay *= 2
			}
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("API %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
		}

		var envelope struct {
			Data   json.RawMessage `json:"data"`
			Errors json.RawMessage `json:"errors,omitempty"`
		}
		if err := json.Unmarshal(respBody, &envelope); err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}

		return envelope.Data, nil
	}

	return nil, lastErr
}

// ListCTs returns all LXC containers on the configured node.
func (c *Client) ListCTs(ctx context.Context) ([]CTListEntry, error) {
	path := fmt.Sprintf("/nodes/%s/lxc", c.node)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var cts []CTListEntry
	if err := json.Unmarshal(data, &cts); err != nil {
		return nil, fmt.Errorf("unmarshal CT list: %w", err)
	}
	return cts, nil
}

// GetCTStatus returns the status of a container by VMID.
func (c *Client) GetCTStatus(ctx context.Context, vmid int) (*CTStatus, error) {
	path := fmt.Sprintf("/nodes/%s/lxc/%d/status/current", c.node, vmid)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var status CTStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("unmarshal CT status: %w", err)
	}
	return &status, nil
}

// GetCTConfig returns the configuration of a container by VMID.
func (c *Client) GetCTConfig(ctx context.Context, vmid int) (*CTConfig, error) {
	path := fmt.Sprintf("/nodes/%s/lxc/%d/config", c.node, vmid)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var cfg CTConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal CT config: %w", err)
	}
	return &cfg, nil
}

// SetCTConfig updates container configuration parameters.
func (c *Client) SetCTConfig(ctx context.Context, vmid int, params url.Values) error {
	path := fmt.Sprintf("/nodes/%s/lxc/%d/config", c.node, vmid)
	_, err := c.do(ctx, http.MethodPut, path, params)
	return err
}

// CloneCT clones a container. Returns the UPID of the clone task.
func (c *Client) CloneCT(ctx context.Context, sourceVMID, newVMID int, hostname string, full bool) (string, error) {
	path := fmt.Sprintf("/nodes/%s/lxc/%d/clone", c.node, sourceVMID)
	params := url.Values{
		"newid":    {fmt.Sprintf("%d", newVMID)},
		"hostname": {hostname},
	}
	if full {
		params.Set("full", "1")
	}

	data, err := c.do(ctx, http.MethodPost, path, params)
	if err != nil {
		return "", err
	}

	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal UPID: %w", err)
	}
	return upid, nil
}

// StartCT starts a container. Returns the UPID.
func (c *Client) StartCT(ctx context.Context, vmid int) (string, error) {
	path := fmt.Sprintf("/nodes/%s/lxc/%d/status/start", c.node, vmid)
	data, err := c.do(ctx, http.MethodPost, path, nil)
	if err != nil {
		return "", err
	}

	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal UPID: %w", err)
	}
	return upid, nil
}

// StopCT force-stops a container. Returns the UPID.
func (c *Client) StopCT(ctx context.Context, vmid int) (string, error) {
	path := fmt.Sprintf("/nodes/%s/lxc/%d/status/stop", c.node, vmid)
	data, err := c.do(ctx, http.MethodPost, path, nil)
	if err != nil {
		return "", err
	}

	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal UPID: %w", err)
	}
	return upid, nil
}

// ShutdownCT gracefully shuts down a container. Returns the UPID.
func (c *Client) ShutdownCT(ctx context.Context, vmid int) (string, error) {
	path := fmt.Sprintf("/nodes/%s/lxc/%d/status/shutdown", c.node, vmid)
	data, err := c.do(ctx, http.MethodPost, path, nil)
	if err != nil {
		return "", err
	}

	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal UPID: %w", err)
	}
	return upid, nil
}

// DeleteCT deletes a container with purge. Returns the UPID.
func (c *Client) DeleteCT(ctx context.Context, vmid int) (string, error) {
	path := fmt.Sprintf("/nodes/%s/lxc/%d", c.node, vmid)
	params := url.Values{
		"purge": {"1"},
		"force": {"1"},
	}
	data, err := c.do(ctx, http.MethodDelete, path+"?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}

	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal UPID: %w", err)
	}
	return upid, nil
}

// GetCTInterfaces returns network interfaces of a container.
func (c *Client) GetCTInterfaces(ctx context.Context, vmid int) ([]CTInterface, error) {
	path := fmt.Sprintf("/nodes/%s/lxc/%d/interfaces", c.node, vmid)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var ifaces []CTInterface
	if err := json.Unmarshal(data, &ifaces); err != nil {
		return nil, fmt.Errorf("unmarshal interfaces: %w", err)
	}
	return ifaces, nil
}

// CreateSnapshot creates a snapshot of a container.
func (c *Client) CreateSnapshot(ctx context.Context, vmid int, name string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/lxc/%d/snapshot", c.node, vmid)
	params := url.Values{
		"snapname": {name},
	}

	data, err := c.do(ctx, http.MethodPost, path, params)
	if err != nil {
		return "", err
	}

	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", nil
	}
	return upid, nil
}

// GetNodeStatus returns the resource status of the configured node.
func (c *Client) GetNodeStatus(ctx context.Context) (*NodeStatus, error) {
	path := fmt.Sprintf("/nodes/%s/status", c.node)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var status NodeStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("unmarshal node status: %w", err)
	}
	return &status, nil
}

// GetTaskStatus returns the status of a task by UPID.
func (c *Client) GetTaskStatus(ctx context.Context, upid string) (*TaskStatus, error) {
	path := fmt.Sprintf("/nodes/%s/tasks/%s/status", c.node, url.PathEscape(upid))
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var status TaskStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("unmarshal task status: %w", err)
	}
	return &status, nil
}

// WaitForTask polls a task until it completes or the context is cancelled.
func (c *Client) WaitForTask(ctx context.Context, upid string) error {
	if upid == "" {
		return nil
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			status, err := c.GetTaskStatus(ctx, upid)
			if err != nil {
				return fmt.Errorf("check task status: %w", err)
			}
			if status.Status == "stopped" {
				if status.ExitStatus != "OK" {
					return fmt.Errorf("task failed with status: %s", status.ExitStatus)
				}
				return nil
			}
		}
	}
}

// NextVMID finds the next available VMID in the configured range.
func (c *Client) NextVMID(ctx context.Context, start, end int) (int, error) {
	cts, err := c.ListCTs(ctx)
	if err != nil {
		return 0, fmt.Errorf("list CTs for VMID allocation: %w", err)
	}

	used := make(map[int]bool, len(cts))
	for _, ct := range cts {
		used[ct.VMID] = true
	}

	for id := start; id <= end; id++ {
		if !used[id] {
			return id, nil
		}
	}
	return 0, fmt.Errorf("no available VMID in range %d-%d", start, end)
}
