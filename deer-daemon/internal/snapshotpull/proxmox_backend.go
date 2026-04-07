package snapshotpull

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ProxmoxBackend snapshots and pulls a VM disk from a Proxmox VE host via its REST API.
type ProxmoxBackend struct {
	host       string
	tokenID    string
	secret     string
	node       string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewProxmoxBackend creates a backend that uses the Proxmox API to snapshot and download VM disks.
func NewProxmoxBackend(host, tokenID, secret, node string, verifySSL bool, logger *slog.Logger) *ProxmoxBackend {
	if logger == nil {
		logger = slog.Default()
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !verifySSL,
		},
	}
	return &ProxmoxBackend{
		host:    strings.TrimRight(host, "/"),
		tokenID: tokenID,
		secret:  secret,
		node:    node,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   10 * time.Minute,
		},
		logger: logger.With("component", "proxmox-backend"),
	}
}

// SnapshotAndPull creates a snapshot on Proxmox, exports it via vzdump,
// downloads the dump, converts to qcow2, and cleans up.
func (b *ProxmoxBackend) SnapshotAndPull(ctx context.Context, vmName string, destPath string) error {
	b.logger.Info("starting proxmox snapshot-and-pull", "vm", vmName, "dest", destPath)

	// 1. Resolve VMID from name
	vmid, err := b.resolveVMID(ctx, vmName)
	if err != nil {
		return fmt.Errorf("resolve vmid: %w", err)
	}

	// 2. Create snapshot
	snapName := "fluid-tmp-snap"
	if err := b.createSnapshot(ctx, vmid, snapName); err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}
	defer func() {
		if err := b.deleteSnapshot(ctx, vmid, snapName); err != nil {
			b.logger.Warn("delete snapshot failed", "vmid", vmid, "error", err)
		}
	}()

	// 3. Create vzdump backup
	dumpFile, err := b.vzdump(ctx, vmid)
	if err != nil {
		return fmt.Errorf("vzdump: %w", err)
	}
	defer func() {
		// Clean up remote dump file
		_ = b.deleteFile(ctx, dumpFile)
	}()

	// 4. Download the dump
	tmpDump := destPath + ".vzdump.tmp"
	if err := b.downloadFile(ctx, dumpFile, tmpDump); err != nil {
		return fmt.Errorf("download dump: %w", err)
	}
	defer func() { _ = os.Remove(tmpDump) }()

	// 5. Convert to qcow2
	if err := convertToQcow2(ctx, tmpDump, destPath); err != nil {
		return fmt.Errorf("convert to qcow2: %w", err)
	}

	b.logger.Info("proxmox snapshot-and-pull complete", "vm", vmName, "dest", destPath)
	return nil
}

// resolveVMID finds the VMID for a given VM name.
func (b *ProxmoxBackend) resolveVMID(ctx context.Context, vmName string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu", b.node)
	data, err := b.apiGet(ctx, path)
	if err != nil {
		return "", err
	}

	var vms []struct {
		VMID json.Number `json:"vmid"`
		Name string      `json:"name"`
	}
	if err := json.Unmarshal(data, &vms); err != nil {
		return "", fmt.Errorf("parse vm list: %w", err)
	}

	for _, vm := range vms {
		if vm.Name == vmName {
			return vm.VMID.String(), nil
		}
	}
	return "", fmt.Errorf("VM %q not found on node %s", vmName, b.node)
}

// createSnapshot creates a VM snapshot via the Proxmox API.
func (b *ProxmoxBackend) createSnapshot(ctx context.Context, vmid, snapName string) error {
	path := fmt.Sprintf("/nodes/%s/qemu/%s/snapshot", b.node, vmid)
	_, err := b.apiPost(ctx, path, url.Values{"snapname": {snapName}})
	return err
}

// deleteSnapshot removes a VM snapshot via the Proxmox API.
func (b *ProxmoxBackend) deleteSnapshot(ctx context.Context, vmid, snapName string) error {
	path := fmt.Sprintf("/nodes/%s/qemu/%s/snapshot/%s", b.node, vmid, snapName)
	_, err := b.apiDelete(ctx, path)
	return err
}

// vzdump creates a backup of the VM and returns the dump file path.
func (b *ProxmoxBackend) vzdump(ctx context.Context, vmid string) (string, error) {
	params := url.Values{
		"vmid":     {vmid},
		"mode":     {"snapshot"},
		"compress": {"zstd"},
		"storage":  {"local"},
	}
	data, err := b.apiPost(ctx, "/nodes/"+b.node+"/vzdump", params)
	if err != nil {
		return "", err
	}

	// The response contains the UPID of the task. We need to wait for it.
	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("parse vzdump response: %w", err)
	}

	// Wait for task completion
	if err := b.waitForTask(ctx, upid); err != nil {
		return "", err
	}

	// Find the dump file in local storage
	return b.findLatestDump(ctx, vmid)
}

// waitForTask polls a Proxmox task until completion.
func (b *ProxmoxBackend) waitForTask(ctx context.Context, upid string) error {
	path := fmt.Sprintf("/nodes/%s/tasks/%s/status", b.node, url.PathEscape(upid))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}

		data, err := b.apiGet(ctx, path)
		if err != nil {
			return err
		}

		var status struct {
			Status     string `json:"status"`
			Exitstatus string `json:"exitstatus"`
		}
		if err := json.Unmarshal(data, &status); err != nil {
			return fmt.Errorf("parse task status: %w", err)
		}

		if status.Status == "stopped" {
			if status.Exitstatus != "OK" {
				return fmt.Errorf("task failed: %s", status.Exitstatus)
			}
			return nil
		}
	}
}

// findLatestDump finds the most recent vzdump file for a VMID.
func (b *ProxmoxBackend) findLatestDump(ctx context.Context, vmid string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/storage/local/content", b.node)
	data, err := b.apiGet(ctx, path)
	if err != nil {
		return "", err
	}

	var files []struct {
		Volid string `json:"volid"`
		CTime int64  `json:"ctime"`
	}
	if err := json.Unmarshal(data, &files); err != nil {
		return "", fmt.Errorf("parse storage content: %w", err)
	}

	var latest string
	var latestTime int64
	prefix := fmt.Sprintf("vzdump-qemu-%s-", vmid)
	for _, f := range files {
		if strings.Contains(f.Volid, prefix) && f.CTime > latestTime {
			latest = f.Volid
			latestTime = f.CTime
		}
	}

	if latest == "" {
		return "", fmt.Errorf("no vzdump found for vmid %s", vmid)
	}
	return latest, nil
}

// downloadFile downloads a file from Proxmox storage to a local path.
func (b *ProxmoxBackend) downloadFile(ctx context.Context, volid, localPath string) error {
	apiURL := fmt.Sprintf("%s/api2/json/nodes/%s/storage/local/file-restore/download",
		b.host, b.node)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Set("volume", volid)
	q.Set("filepath", "/")
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s=%s", b.tokenID, b.secret))

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download failed: %d: %s", resp.StatusCode, string(body))
	}

	out, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, resp.Body)
	return err
}

// deleteFile removes a file from Proxmox storage.
func (b *ProxmoxBackend) deleteFile(ctx context.Context, volid string) error {
	path := fmt.Sprintf("/nodes/%s/storage/local/content/%s", b.node, url.PathEscape(volid))
	_, err := b.apiDelete(ctx, path)
	return err
}

// apiGet performs a GET request against the Proxmox API.
func (b *ProxmoxBackend) apiGet(ctx context.Context, path string) (json.RawMessage, error) {
	return b.apiRequest(ctx, http.MethodGet, path, nil)
}

// apiPost performs a POST request against the Proxmox API.
func (b *ProxmoxBackend) apiPost(ctx context.Context, path string, params url.Values) (json.RawMessage, error) {
	return b.apiRequest(ctx, http.MethodPost, path, params)
}

// apiDelete performs a DELETE request against the Proxmox API.
func (b *ProxmoxBackend) apiDelete(ctx context.Context, path string) (json.RawMessage, error) {
	return b.apiRequest(ctx, http.MethodDelete, path, nil)
}

// apiRequest performs an authenticated HTTP request against the Proxmox API.
func (b *ProxmoxBackend) apiRequest(ctx context.Context, method, path string, params url.Values) (json.RawMessage, error) {
	apiURL := fmt.Sprintf("%s/api2/json%s", b.host, path)

	var body io.Reader
	if params != nil {
		body = strings.NewReader(params.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, apiURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s=%s", b.tokenID, b.secret))
	if params != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(respBody))
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return envelope.Data, nil
}

// convertToQcow2 converts a vzdump archive to a QCOW2 image.
func convertToQcow2(ctx context.Context, src, dest string) error {
	cmd := exec.CommandContext(ctx, "qemu-img", "convert", "-f", "raw", "-O", "qcow2", src, dest)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("qemu-img convert: %w: %s", err, string(output))
	}
	return nil
}
