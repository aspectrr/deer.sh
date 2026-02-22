package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aspectrr/fluid.sh/fluid/internal/paths"
)

const maxBinarySize = 500 * 1024 * 1024 // 500MB limit for tar entry reads

const (
	releasesURL = "https://api.github.com/repos/aspectrr/fluid.sh/releases/latest"
	cacheFile   = ".last-update-check"
	cacheTTL    = 24 * time.Hour
)

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// CheckLatest queries GitHub API for the latest release.
// Returns (latestVersion, downloadURL, needsUpdate, error).
func CheckLatest(currentVersion string) (string, string, bool, error) {
	req, err := http.NewRequest("GET", releasesURL, nil)
	if err != nil {
		return "", "", false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", false, fmt.Errorf("fetch latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", "", false, fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", false, fmt.Errorf("decode release: %w", err)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	if current == "dev" || current == "" {
		// Dev builds always report as up to date
		return latest, "", false, nil
	}

	if latest == current {
		return latest, "", false, nil
	}

	// Find the right asset for this OS/arch
	assetName := fmt.Sprintf("fluid_%s_%s_%s.tar.gz", latest, runtime.GOOS, runtime.GOARCH)
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return latest, "", false, fmt.Errorf("no release asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	return latest, downloadURL, true, nil
}

// Update downloads the release archive from downloadURL and replaces the current executable.
func Update(downloadURL string) error {
	client := &http.Client{Timeout: 120 * time.Second}

	// Download the tar.gz to a temp file so we can checksum it
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	archiveData, err := io.ReadAll(io.LimitReader(resp.Body, maxBinarySize))
	if err != nil {
		return fmt.Errorf("read archive: %w", err)
	}

	// Download and verify checksum
	checksumURL := strings.TrimSuffix(downloadURL, filepath.Base(downloadURL)) + "checksums.txt"
	if err := verifyChecksum(client, checksumURL, filepath.Base(downloadURL), archiveData); err != nil {
		return fmt.Errorf("checksum verification: %w", err)
	}

	// Extract the "fluid" binary from the tar.gz archive
	gz, err := gzip.NewReader(bytes.NewReader(archiveData))
	if err != nil {
		return fmt.Errorf("open gzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	var binaryData []byte
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}
		// Look for the fluid binary (may be at root or in a subdirectory)
		base := filepath.Base(hdr.Name)
		if base == "fluid" && hdr.Typeflag == tar.TypeReg {
			binaryData, err = io.ReadAll(io.LimitReader(tr, maxBinarySize))
			if err != nil {
				return fmt.Errorf("read binary from archive: %w", err)
			}
			break
		}
	}

	if binaryData == nil {
		return fmt.Errorf("fluid binary not found in archive")
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolve symlinks: %w", err)
	}

	// Write to temp file in same directory (for atomic rename)
	dir := filepath.Dir(execPath)
	tmp, err := os.CreateTemp(dir, "fluid-update-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(binaryData); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp binary: %w", err)
	}
	if err := tmp.Chmod(0o755); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("chmod temp binary: %w", err)
	}
	_ = tmp.Close()

	// Atomic rename over the current executable
	if err := os.Rename(tmpPath, execPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace binary: %w", err)
	}

	return nil
}

// CacheDir returns the fluid data directory path for caching update checks.
func CacheDir() string {
	dir, err := paths.DataDir()
	if err != nil {
		return ""
	}
	return dir
}

// ShouldCheck returns true if enough time has passed since the last update check.
func ShouldCheck() bool {
	dir := CacheDir()
	if dir == "" {
		return false
	}
	path := filepath.Join(dir, cacheFile)
	info, err := os.Stat(path)
	if err != nil {
		return true // No cache file, should check
	}
	return time.Since(info.ModTime()) > cacheTTL
}

// MarkChecked updates the cache file timestamp.
func MarkChecked() {
	dir := CacheDir()
	if dir == "" {
		return
	}
	path := filepath.Join(dir, cacheFile)
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(path, []byte(time.Now().Format(time.RFC3339)), 0o644)
}

// verifyChecksum downloads checksums.txt from the release and verifies the archive SHA256.
func verifyChecksum(client *http.Client, checksumURL, assetName string, data []byte) error {
	resp, err := client.Get(checksumURL)
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checksums download returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1MB limit for checksums file
	if err != nil {
		return fmt.Errorf("read checksums: %w", err)
	}

	// Parse checksums file: each line is "sha256hash  filename"
	var expectedHash string
	for _, line := range strings.Split(string(body), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == assetName {
			expectedHash = parts[0]
			break
		}
	}

	if expectedHash == "" {
		return fmt.Errorf("no checksum found for %s in checksums.txt", assetName)
	}

	actualHash := sha256.Sum256(data)
	actualHex := hex.EncodeToString(actualHash[:])

	if actualHex != expectedHash {
		return fmt.Errorf("SHA256 mismatch: expected %s, got %s", expectedHash, actualHex)
	}

	return nil
}
