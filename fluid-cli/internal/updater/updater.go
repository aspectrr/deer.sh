package updater

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

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
	defer resp.Body.Close()

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
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	// Extract the "fluid" binary from the tar.gz archive
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("open gzip: %w", err)
	}
	defer gz.Close()

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
			binaryData, err = io.ReadAll(tr)
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
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp binary: %w", err)
	}
	if err := tmp.Chmod(0o755); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("chmod temp binary: %w", err)
	}
	tmp.Close()

	// Atomic rename over the current executable
	if err := os.Rename(tmpPath, execPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("replace binary: %w", err)
	}

	return nil
}

// CacheDir returns the fluid config directory path for caching update checks.
func CacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".fluid")
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
