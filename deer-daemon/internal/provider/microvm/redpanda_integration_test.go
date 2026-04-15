package microvm

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/aspectrr/deer.sh/deer-daemon/internal/daemon"
	imagestore "github.com/aspectrr/deer.sh/deer-daemon/internal/image"
	microvminternal "github.com/aspectrr/deer.sh/deer-daemon/internal/microvm"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/network"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/provider"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/sshca"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/sshkeys"
)

type liveE2EHarness struct {
	ctx       context.Context
	cfg       redpandaE2EConfig
	logger    *slog.Logger
	workDir   string
	vmMgr     *microvminternal.Manager
	readiness *daemon.ReadinessServer
	provider  *Provider
	bridgeIP  string
}

func TestProviderIntegration_RedpandaStartsInGuest(t *testing.T) {
	t.Helper()

	cfg := loadRedpandaE2EConfig(t)
	h := newLiveE2EHarness(t, cfg)
	archiveURL := startArchiveServer(t, h.bridgeIP, ensureRedpandaArchive(t, h.workDir, cfg))

	req := provider.CreateRequest{
		SandboxID: fmt.Sprintf("sbx-e2e-%d", time.Now().UnixNano()),
		Name:      "redpanda-e2e",
		BaseImage: "base",
		Network:   cfg.bridge,
		VCPUs:     2,
		MemoryMB:  2048,
		KafkaBroker: &provider.KafkaBrokerConfig{
			ArchiveURL: archiveURL,
			Port:       9092,
		},
	}

	result := createLiveSandbox(t, h, req)
	serialContent := waitForPhoneHomeNotification(t, h, result)
	if !strings.Contains(serialContent, "deer redpanda ready on attempt") {
		t.Fatalf("sandbox posted readiness without a successful in-guest Redpanda readiness check\nlast_stage: %s\nserial:\n%s", redpandaSerialStage(serialContent), serialContent)
	}

	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		cmdCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		run, err := h.provider.RunCommand(cmdCtx, result.SandboxID, redpandaProbeCommand, 45*time.Second)
		cancel()
		if err == nil && run.ExitCode == 0 {
			return
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("probe exit %d stderr=%s stdout=%s", run.ExitCode, run.Stderr, run.Stdout)
		}
		time.Sleep(10 * time.Second)
	}
	serialContent = serialLog(h.vmMgr.WorkDir(), result.SandboxID)
	t.Fatalf("guest posted readiness and in-guest Redpanda checks passed, but final host-side Kafka probe still failed: %v\nlast_stage: %s\nguest_diagnostics:\n%s\nhost_diagnostics:\n%s\nserial:\n%s", lastErr, redpandaSerialStage(serialContent), guestDiagnostics(h.ctx, h.provider, result.SandboxID), sandboxHostDiagnostics(h.vmMgr.WorkDir(), result.SandboxID, result.PID), serialContent)
}

func TestProviderIntegration_SandboxStartsWithoutRedpanda(t *testing.T) {
	t.Helper()

	cfg := loadRedpandaE2EConfig(t)
	h := newLiveE2EHarness(t, cfg)

	req := provider.CreateRequest{
		SandboxID: fmt.Sprintf("sbx-e2e-plain-%d", time.Now().UnixNano()),
		Name:      "plain-e2e",
		BaseImage: "base",
		Network:   cfg.bridge,
		VCPUs:     1,
		MemoryMB:  1024,
	}

	result := createLiveSandbox(t, h, req)
	serialContent := waitForPhoneHomeNotification(t, h, result)
	if strings.Contains(serialContent, "deer redpanda ready on attempt") {
		t.Fatalf("plain sandbox should not emit redpanda readiness markers\nlast_stage: %s\nserial:\n%s", redpandaSerialStage(serialContent), serialContent)
	}

	cmdCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	run, err := h.provider.RunCommand(cmdCtx, result.SandboxID, plainSandboxProbeCommand, 45*time.Second)
	if err != nil {
		serialContent = serialLog(h.vmMgr.WorkDir(), result.SandboxID)
		t.Fatalf("plain sandbox probe failed: %v\nlast_stage: %s\nguest_diagnostics:\n%s\nhost_diagnostics:\n%s\nserial:\n%s", err, redpandaSerialStage(serialContent), guestDiagnostics(h.ctx, h.provider, result.SandboxID), sandboxHostDiagnostics(h.vmMgr.WorkDir(), result.SandboxID, result.PID), serialContent)
	}
	if run.ExitCode != 0 {
		serialContent = serialLog(h.vmMgr.WorkDir(), result.SandboxID)
		t.Fatalf("plain sandbox probe exited with %d\nstdout:\n%s\nstderr:\n%s\nlast_stage: %s\nguest_diagnostics:\n%s\nhost_diagnostics:\n%s\nserial:\n%s", run.ExitCode, run.Stdout, run.Stderr, redpandaSerialStage(serialContent), guestDiagnostics(h.ctx, h.provider, result.SandboxID), sandboxHostDiagnostics(h.vmMgr.WorkDir(), result.SandboxID, result.PID), serialContent)
	}
}

func waitForSerialMarker(serialPath, marker string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for {
		serialBytes, _ := os.ReadFile(serialPath)
		serialContent := string(serialBytes)
		if strings.Contains(serialContent, marker) {
			return serialContent, nil
		}
		if time.Now().After(deadline) {
			return serialContent, fmt.Errorf("serial marker %q not observed within %v", marker, timeout)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func newLiveE2EHarness(t *testing.T, cfg redpandaE2EConfig) *liveE2EHarness {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	workDir := e2eWorkDir(t)
	imageDir := filepath.Join(workDir, "images")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatalf("mkdir image dir: %v", err)
	}
	if err := os.Symlink(cfg.baseImagePath, filepath.Join(imageDir, "base.qcow2")); err != nil {
		t.Fatalf("symlink base image: %v", err)
	}

	vmMgr, err := microvminternal.NewManager(cfg.qemuBinary, filepath.Join(workDir, "sandboxes"), logger)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	imgStore, err := imagestore.NewStore(imageDir, logger)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	caKeyPath := filepath.Join(workDir, "sshca", "ca")
	if err := sshca.GenerateCA(caKeyPath, "deer-e2e-ca"); err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}
	ca, err := sshca.NewCA(sshca.Config{
		CAKeyPath:             caKeyPath,
		CAPubKeyPath:          caKeyPath + ".pub",
		WorkDir:               filepath.Join(workDir, "sshca-work"),
		DefaultTTL:            30 * time.Minute,
		MaxTTL:                60 * time.Minute,
		DefaultPrincipals:     []string{"sandbox"},
		EnforceKeyPermissions: true,
	}, sshca.WithTimeNow(time.Now))
	if err != nil {
		t.Fatalf("NewCA: %v", err)
	}
	if err := ca.Initialize(ctx); err != nil {
		t.Fatalf("Initialize CA: %v", err)
	}
	caPubKeyBytes, err := os.ReadFile(caKeyPath + ".pub")
	if err != nil {
		t.Fatalf("read CA pubkey: %v", err)
	}

	keyMgr, err := sshkeys.NewKeyManager(ca, sshkeys.Config{
		KeyDir:         filepath.Join(workDir, "keys"),
		CertificateTTL: 30 * time.Minute,
		RefreshMargin:  30 * time.Second,
	}, logger)
	if err != nil {
		t.Fatalf("NewKeyManager: %v", err)
	}
	t.Cleanup(func() {
		_ = keyMgr.Close()
	})

	var bridgeIP string
	var readiness *daemon.ReadinessServer
	if cfg.bridge != "" {
		var err error
		bridgeIP, err = network.GetBridgeIP(cfg.bridge)
		if err != nil {
			t.Fatalf("GetBridgeIP(%q): %v", cfg.bridge, err)
		}
		readiness = startReadinessServer(t, bridgeIP, logger)
	}

	p := New(
		vmMgr,
		network.NewNetworkManager(cfg.bridge, nil, cfg.dhcpMode, logger),
		imgStore,
		nil,
		keyMgr,
		cfg.kernelPath,
		cfg.initrdPath,
		cfg.rootDevice,
		cfg.accel,
		5*time.Minute,
		cfg.startupTimeout,
		strings.TrimSpace(string(caPubKeyBytes)),
		bridgeIP,
		readiness,
		"",
		false,
		cfg.socketVMNetClient,
		cfg.socketVMNetPath,
		logger,
	)

	return &liveE2EHarness{
		ctx:       ctx,
		cfg:       cfg,
		logger:    logger,
		workDir:   workDir,
		vmMgr:     vmMgr,
		readiness: readiness,
		provider:  p,
		bridgeIP:  bridgeIP,
	}
}

func createLiveSandbox(t *testing.T, h *liveE2EHarness, req provider.CreateRequest) *provider.SandboxResult {
	t.Helper()

	if h.readiness != nil {
		h.readiness.Register(req.SandboxID)
		t.Cleanup(func() {
			h.readiness.Unregister(req.SandboxID)
		})
	}

	result, err := h.provider.CreateSandboxWithProgress(h.ctx, req, func(string, int, int) {})
	if err != nil {
		serial := serialLog(h.vmMgr.WorkDir(), req.SandboxID)
		t.Fatalf("CreateSandboxWithProgress: %v\nlast_stage: %s\nhost_diagnostics:\n%s\nserial:\n%s", err, redpandaSerialStage(serial), sandboxHostDiagnostics(h.vmMgr.WorkDir(), req.SandboxID, 0), serial)
	}
	if os.Getenv("DEER_E2E_KEEP_SANDBOX") != "1" {
		t.Cleanup(func() {
			destroyCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			if err := h.provider.DestroySandbox(destroyCtx, result.SandboxID); err != nil {
				t.Logf("DestroySandbox(%s): %v", result.SandboxID, err)
			}
		})
	} else {
		t.Logf("preserving sandbox artifacts for %s under %s", result.SandboxID, h.vmMgr.WorkDir())
	}

	if result.IPAddress == "" {
		serial := serialLog(h.vmMgr.WorkDir(), result.SandboxID)
		t.Fatalf("CreateSandbox returned empty guest IP\nlast_stage: %s\nready_ip: %s\nhost_diagnostics:\n%s\nserial:\n%s", redpandaSerialStage(serial), h.readiness.ReadyIP(result.SandboxID), sandboxHostDiagnostics(h.vmMgr.WorkDir(), result.SandboxID, result.PID), serial)
	}

	return result
}

func waitForPhoneHomeNotification(t *testing.T, h *liveE2EHarness, result *provider.SandboxResult) string {
	t.Helper()

	serialPath := filepath.Join(h.vmMgr.WorkDir(), result.SandboxID, "serial.log")
	serialBytes, _ := os.ReadFile(serialPath)
	if h.readiness != nil {
		if err := h.readiness.WaitReady(result.SandboxID, h.cfg.startupTimeout); err != nil {
			serial := string(serialBytes)
			t.Fatalf("sandbox did not post readiness within %v: %v\nlast_stage: %s\nready_ip: %s\nhost_diagnostics:\n%s\nserial:\n%s", h.cfg.startupTimeout, err, redpandaSerialStage(serial), h.readiness.ReadyIP(result.SandboxID), sandboxHostDiagnostics(h.vmMgr.WorkDir(), result.SandboxID, result.PID), serial)
		}
		serialBytes, _ = os.ReadFile(serialPath)
		if !h.readiness.WasReady(result.SandboxID) {
			serial := string(serialBytes)
			t.Fatalf("sandbox never posted readiness\nlast_stage: %s\nhost_diagnostics:\n%s\nserial:\n%s", redpandaSerialStage(serial), sandboxHostDiagnostics(h.vmMgr.WorkDir(), result.SandboxID, result.PID), serial)
		}
	} else {
		t.Logf("no readiness server (socket_vmnet without bridge); waiting for serial marker only")
	}

	serialContent, err := waitForSerialMarker(serialPath, "deer notify ready complete", 10*time.Second)
	if err != nil {
		t.Fatalf("sandbox posted readiness without completing in-guest phone_home notification: %v\nlast_stage: %s\nserial:\n%s", err, redpandaSerialStage(serialContent), serialContent)
	}
	return serialContent
}

type redpandaE2EConfig struct {
	baseImagePath     string
	kernelPath        string
	initrdPath        string
	archiveURL        string
	qemuBinary        string
	bridge            string
	dhcpMode          string
	rootDevice        string
	accel             string
	startupTimeout    time.Duration
	socketVMNetClient string
	socketVMNetPath   string
}

func loadRedpandaE2EConfig(t *testing.T) redpandaE2EConfig {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping live guest integration test in short mode")
	}
	if os.Getenv("DEER_E2E_MICROVM") != "1" {
		t.Skip("set DEER_E2E_MICROVM=1 to run live guest microVM integration tests")
	}
	socketVMNetClient := strings.TrimSpace(os.Getenv("DEER_E2E_SOCKET_VMNET_CLIENT"))
	socketVMNetPath := strings.TrimSpace(os.Getenv("DEER_E2E_SOCKET_VMNET_PATH"))
	usingSocketVMNet := socketVMNetClient != ""

	if !usingSocketVMNet && os.Geteuid() != 0 {
		t.Skip("live guest microVM integration test requires root for TAP/bridge setup (or set DEER_E2E_SOCKET_VMNET_CLIENT for socket_vmnet)")
	}

	defaultAccel := "tcg"
	if runtime.GOOS == "darwin" {
		defaultAccel = "hvf"
	}

	cfg := redpandaE2EConfig{
		baseImagePath:     os.Getenv("DEER_E2E_BASE_IMAGE"),
		kernelPath:        os.Getenv("DEER_E2E_KERNEL"),
		initrdPath:        os.Getenv("DEER_E2E_INITRD"),
		archiveURL:        strings.TrimSpace(os.Getenv("DEER_E2E_REDPANDA_ARCHIVE_URL")),
		qemuBinary:        envOrDefault("DEER_E2E_QEMU_BINARY", defaultQEMUBinary()),
		bridge:            os.Getenv("DEER_E2E_BRIDGE"),
		dhcpMode:          envOrDefault("DEER_E2E_DHCP_MODE", "arp"),
		rootDevice:        envOrDefault("DEER_E2E_ROOT_DEVICE", "/dev/vda1"),
		accel:             envOrDefault("DEER_E2E_ACCEL", defaultAccel),
		startupTimeout:    25 * time.Minute,
		socketVMNetClient: socketVMNetClient,
		socketVMNetPath:   socketVMNetPath,
	}

	if timeoutRaw := os.Getenv("DEER_E2E_STARTUP_TIMEOUT"); timeoutRaw != "" {
		timeout, err := time.ParseDuration(timeoutRaw)
		if err != nil {
			t.Fatalf("invalid DEER_E2E_STARTUP_TIMEOUT %q: %v", timeoutRaw, err)
		}
		cfg.startupTimeout = timeout
	}

	missing := make([]string, 0, 3)
	if cfg.baseImagePath == "" {
		missing = append(missing, "DEER_E2E_BASE_IMAGE")
	}
	if cfg.kernelPath == "" {
		missing = append(missing, "DEER_E2E_KERNEL")
	}
	if !usingSocketVMNet && cfg.bridge == "" {
		missing = append(missing, "DEER_E2E_BRIDGE")
	}
	if len(missing) > 0 {
		t.Skipf("missing required env vars for live guest integration test: %s", strings.Join(missing, ", "))
	}

	for _, path := range []string{cfg.baseImagePath, cfg.kernelPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("required path %q: %v", path, err)
		}
	}
	if cfg.initrdPath != "" {
		if _, err := os.Stat(cfg.initrdPath); err != nil {
			t.Fatalf("configured initrd %q: %v", cfg.initrdPath, err)
		}
	}
	requiredBins := []string{cfg.qemuBinary, "qemu-img", "ssh-keygen"}
	if usingSocketVMNet {
		requiredBins = append(requiredBins, cfg.socketVMNetClient)
	} else if runtime.GOOS == "darwin" {
		requiredBins = append(requiredBins, "ifconfig", "arp")
	} else {
		requiredBins = append(requiredBins, "ip")
	}
	for _, bin := range requiredBins {
		if _, err := exec.LookPath(bin); err != nil {
			t.Fatalf("required binary %q not found: %v", bin, err)
		}
	}
	if cfg.archiveURL == "" {
		if _, err := exec.LookPath("dpkg-deb"); err != nil {
			t.Fatalf("required binary %q not found: %v (set DEER_E2E_REDPANDA_ARCHIVE_URL to provide a pre-built archive)", "dpkg-deb", err)
		}
	}
	if cfg.bridge != "" {
		if _, err := network.GetBridgeIP(cfg.bridge); err != nil {
			t.Fatalf("bridge %q is not usable for integration test: %v", cfg.bridge, err)
		}
	}
	return cfg
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func TestWaitForSerialMarker_PollsUntilMarkerAppears(t *testing.T) {
	serialPath := filepath.Join(t.TempDir(), "serial.log")
	if err := os.WriteFile(serialPath, []byte("deer phone_home start\n"), 0o644); err != nil {
		t.Fatalf("write serial log: %v", err)
	}

	done := make(chan struct{})
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = os.WriteFile(serialPath, []byte("deer phone_home start\ndeer notify ready complete\n"), 0o644)
		close(done)
	}()

	serialContent, err := waitForSerialMarker(serialPath, "deer notify ready complete", time.Second)
	<-done
	if err != nil {
		t.Fatalf("waitForSerialMarker: %v", err)
	}
	if !strings.Contains(serialContent, "deer notify ready complete") {
		t.Fatalf("serial content %q missing completion marker", serialContent)
	}
}

func e2eWorkDir(t *testing.T) string {
	t.Helper()

	if dir := strings.TrimSpace(os.Getenv("DEER_E2E_WORKDIR")); dir != "" {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatalf("remove DEER_E2E_WORKDIR %q: %v", dir, err)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create DEER_E2E_WORKDIR %q: %v", dir, err)
		}
		t.Logf("using DEER_E2E_WORKDIR=%s", dir)
		return dir
	}

	candidates := []string{"/var/tmp", os.TempDir()}
	keepWorkDir := os.Getenv("DEER_E2E_KEEP_WORKDIR") == "1"
	for _, baseDir := range candidates {
		dir, err := os.MkdirTemp(baseDir, "deer-redpanda-e2e-")
		if err != nil {
			continue
		}
		if keepWorkDir {
			t.Logf("preserving E2E workdir=%s", dir)
		} else {
			t.Cleanup(func() {
				_ = os.RemoveAll(dir)
			})
		}
		t.Logf("using E2E workdir=%s", dir)
		return dir
	}

	t.Fatalf("could not create an E2E workdir in /var/tmp or %s", os.TempDir())
	return ""
}

func defaultQEMUBinary() string {
	if runtime.GOARCH == "arm64" {
		return "qemu-system-aarch64"
	}
	return "qemu-system-x86_64"
}

func defaultRedpandaDebURLs() []string {
	baseURL := "https://dl.redpanda.com/public/redpanda/deb/ubuntu/pool/any-version/main/r/re"
	version := "25.3.11-1"
	arch := "amd64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}
	pkgs := []string{"redpanda", "redpanda-rpk", "redpanda-tuner"}
	urls := make([]string, 0, len(pkgs))
	for _, pkg := range pkgs {
		urls = append(urls, fmt.Sprintf("%s/%s_%s_%s.deb", baseURL, pkg, version, arch))
	}
	return urls
}

func startReadinessServer(t *testing.T, bridgeIP string, logger *slog.Logger) *daemon.ReadinessServer {
	t.Helper()

	addr := bridgeIP + ":9092"
	srv := daemon.NewReadinessServer(addr, logger)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("listen readiness server on %s: %v", addr, err)
	}

	done := make(chan error, 1)
	go func() {
		done <- srv.Serve(ln)
	}()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			t.Logf("shutdown readiness server: %v", err)
		}
		select {
		case err := <-done:
			if err != nil && err != http.ErrServerClosed {
				t.Logf("readiness server exited with error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Log("timed out waiting for readiness server shutdown")
		}
	})

	return srv
}

func ensureRedpandaArchive(t *testing.T, _ string, cfg redpandaE2EConfig) string {
	t.Helper()

	archiveDir := redpandaArchiveCacheDir(t)
	u, err := url.Parse(cfg.archiveURL)
	if cfg.archiveURL == "" {
		return buildRedpandaArchiveFromDebs(t, archiveDir, defaultRedpandaDebURLs())
	}
	if err != nil {
		t.Fatalf("parse archive URL %q: %v", cfg.archiveURL, err)
	}
	archiveName := filepath.Base(u.Path)
	if archiveName == "." || archiveName == "/" || archiveName == "" {
		archiveName = "redpanda.tar.gz"
	}
	return downloadArchiveToCache(t, archiveDir, archiveName, cfg.archiveURL)
}

func redpandaArchiveCacheDir(t *testing.T) string {
	t.Helper()

	candidates := make([]string, 0, 2)
	if cacheDir, err := os.UserCacheDir(); err == nil && strings.TrimSpace(cacheDir) != "" {
		candidates = append(candidates, filepath.Join(cacheDir, "deer", "e2e", "redpanda"))
	}
	candidates = append(candidates, filepath.Join(os.TempDir(), "deer-redpanda-cache"))

	for _, dir := range candidates {
		if err := os.MkdirAll(dir, 0o755); err == nil {
			return dir
		}
	}

	t.Fatalf("could not create a writable redpanda archive cache directory")
	return ""
}

func buildRedpandaArchiveFromDebs(t *testing.T, cacheDir string, urls []string) string {
	t.Helper()

	archivePath := filepath.Join(cacheDir, fmt.Sprintf("redpanda-rootfs-%s.tar.gz", runtime.GOARCH))
	_ = os.Remove(archivePath)

	stageDir, err := os.MkdirTemp(cacheDir, "redpanda-rootfs-")
	if err != nil {
		t.Fatalf("create stage dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(stageDir) }()

	for _, pkgURL := range urls {
		pkgName := filepath.Base(pkgURL)
		pkgPath := downloadArchiveToCache(t, cacheDir, pkgName, pkgURL)
		cmd := exec.Command("dpkg-deb", "-x", pkgPath, stageDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("extract %s: %v\n%s", pkgName, err, string(output))
		}
	}

	payloadDir := filepath.Join(stageDir, "payload")
	if err := os.MkdirAll(payloadDir, 0o755); err != nil {
		t.Fatalf("create payload dir: %v", err)
	}
	for _, relPath := range []string{
		"opt/redpanda",
		"usr/bin/redpanda",
		"usr/bin/rpk",
		"usr/bin/iotune-redpanda",
	} {
		srcPath := filepath.Join(stageDir, relPath)
		if _, err := os.Lstat(srcPath); err != nil {
			continue
		}
		dstPath := filepath.Join(payloadDir, relPath)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			t.Fatalf("mkdir payload parent for %s: %v", relPath, err)
		}
		cmd := exec.Command("cp", "-a", srcPath, dstPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("stage %s into payload: %v\n%s", relPath, err, string(output))
		}
	}

	tmpArchive, err := os.CreateTemp(cacheDir, "redpanda-rootfs-*.tar.gz")
	if err != nil {
		t.Fatalf("create temp archive: %v", err)
	}
	tmpArchivePath := tmpArchive.Name()
	_ = tmpArchive.Close()
	defer func() { _ = os.Remove(tmpArchivePath) }()

	writeTarGzFromDir(t, tmpArchivePath, payloadDir)
	if err := os.Rename(tmpArchivePath, archivePath); err != nil {
		t.Fatalf("move archive into place: %v", err)
	}
	return archivePath
}

func downloadArchiveToCache(t *testing.T, cacheDir, archiveName, sourceURL string) string {
	t.Helper()

	archivePath := filepath.Join(cacheDir, archiveName)
	if _, err := os.Stat(archivePath); err == nil {
		return archivePath
	}

	resp, err := http.Get(sourceURL)
	if err != nil {
		t.Fatalf("download %s: %v", sourceURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("download %s: unexpected status %s", sourceURL, resp.Status)
	}

	out, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive file: %v", err)
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, resp.Body); err != nil {
		t.Fatalf("write archive file: %v", err)
	}
	return archivePath
}

func writeTarGzFromDir(t *testing.T, archivePath, root string) {
	t.Helper()

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive %s: %v", archivePath, err)
	}
	defer func() { _ = file.Close() }()

	gz := gzip.NewWriter(file)
	defer func() { _ = gz.Close() }()

	tw := tar.NewWriter(gz)
	defer func() { _ = tw.Close() }()

	if err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)
		linkTarget := ""
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err = os.Readlink(path)
			if err != nil {
				return err
			}
		}
		header, err := tar.FileInfoHeader(info, linkTarget)
		if err != nil {
			return err
		}
		header.Name = relPath
		if info.IsDir() {
			header.Name += "/"
		}
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = in.Close() }()
		_, err = io.Copy(tw, in)
		return err
	}); err != nil {
		t.Fatalf("write archive %s: %v", archivePath, err)
	}
}

func startArchiveServer(t *testing.T, bridgeIP, archivePath string) string {
	t.Helper()

	addr := bridgeIP + ":9088"
	mux := http.NewServeMux()
	mux.HandleFunc("/redpanda.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, archivePath)
	})
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("listen archive server on %s: %v", addr, err)
	}

	done := make(chan error, 1)
	go func() {
		done <- srv.Serve(ln)
	}()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			t.Logf("shutdown archive server: %v", err)
		}
		select {
		case err := <-done:
			if err != nil && err != http.ErrServerClosed {
				t.Logf("archive server exited with error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Log("timed out waiting for archive server shutdown")
		}
	})

	return "http://" + addr + "/redpanda.tar.gz"
}

func serialLog(workDir, sandboxID string) string {
	serialBytes, _ := os.ReadFile(filepath.Join(workDir, sandboxID, "serial.log"))
	return string(serialBytes)
}

func redpandaSerialStage(serial string) string {
	markers := []string{
		"deer redpanda install start",
		"deer redpanda archive download complete",
		"deer redpanda extraction complete",
		"deer redpanda binary resolution complete",
		"deer redpanda env file written",
		"deer redpanda install complete",
		"deer redpanda temp cleanup skipped for ephemeral sandbox",
		"deer redpanda enable start",
		"deer redpanda daemon reload complete",
		"deer redpanda service start invoked",
		"deer redpanda systemd enable complete",
		"deer redpanda readiness wait started",
		"deer redpanda readiness pending stage=",
		"deer redpanda readiness wait success",
		"deer redpanda readiness wait timeout",
		"deer redpanda readiness failure",
		"deer redpanda ready on attempt",
		"deer notify ready start",
		"deer notify ready checks complete",
		"deer phone_home start",
		"deer notify ready complete",
		"deer notify ready failure",
	}
	last := "unknown"
	for _, line := range strings.Split(serial, "\n") {
		for _, marker := range markers {
			if strings.Contains(line, marker) {
				last = strings.TrimSpace(line)
			}
		}
	}
	return last
}

func guestDiagnostics(ctx context.Context, p *Provider, sandboxID string) string {
	cmdCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	run, err := p.RunCommand(cmdCtx, sandboxID, redpandaDiagnosticsCommand, 90*time.Second)
	if err != nil {
		return fmt.Sprintf("guest diagnostics unavailable: %v", err)
	}
	return fmt.Sprintf("exit=%d\nstdout:\n%s\nstderr:\n%s", run.ExitCode, run.Stdout, run.Stderr)
}

const redpandaProbeCommand = `bash -lc 'set -euo pipefail
listener_ready() {
  ss -H -ltn "( sport = :9092 )" | awk '"'"'END { exit(NR==0) }'"'"'
}
systemctl is-active --quiet deer-redpanda.service
systemctl is-enabled --quiet deer-redpanda.service
test -x /usr/bin/redpanda
test -s /etc/default/deer-redpanda
listener_ready
'`

const plainSandboxProbeCommand = `bash -lc 'set -euo pipefail
systemctl is-active --quiet ssh || systemctl is-active --quiet sshd
test -f /etc/ssh/deer_ca.pub
test -f /etc/ssh/authorized_principals/sandbox
test ! -f /etc/default/deer-redpanda
id sandbox >/dev/null
'`

const redpandaDiagnosticsCommand = `bash -lc 'set +e
echo "--- systemctl status ---"
systemctl status deer-redpanda.service --no-pager || true
echo "--- cloud-init status ---"
cloud-init status --long || true
echo "--- cloud-final journal ---"
journalctl -u cloud-final --no-pager -n 200 || true
echo "--- kernel journal ---"
journalctl -k --no-pager -n 200 || true
echo "--- redpanda journal ---"
journalctl -u deer-redpanda.service --no-pager -n 200 || true
echo "--- sockets ---"
ss -ltn || true
echo "--- sockets 9092 ---"
ss -H -ltn "( sport = :9092 )" || true
echo "--- sockets 9092 with pid ---"
ss -H -ltnp "( sport = :9092 )" || true
echo "--- redpanda env ---"
cat /etc/default/deer-redpanda || true
if [ -f /etc/default/deer-redpanda ]; then
  . /etc/default/deer-redpanda
fi
if [ -n "${RPK_BIN:-}" ] && [ -x "${RPK_BIN:-}" ]; then
  echo "--- rpk cluster info ---"
  timeout 10s "${RPK_BIN}" cluster info --brokers 127.0.0.1:9092 || true
  echo "--- rpk topic list ---"
  timeout 10s "${RPK_BIN}" topic list --brokers 127.0.0.1:9092 || true
fi
echo "--- redpanda config ---"
cat /etc/redpanda/redpanda.yaml || true
echo "--- redpanda tree ---"
find /opt/deer-redpanda-root -maxdepth 6 -type f | sort || true
echo "--- redpanda logs ---"
find /var/log /var/lib/redpanda -maxdepth 4 -type f \( -iname '*redpanda*.log' -o -iname '*redpanda*' -o -path '/var/log/deer/*' \) 2>/dev/null | sort | while read -r log_path; do
  echo "--- $log_path ---"
  cat "$log_path" || true
done
echo "--- os-release ---"
cat /etc/os-release || true
'`
