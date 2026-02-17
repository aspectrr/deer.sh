package mcp

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

const (
	// maxShellInputLength is the maximum allowed length for shell input arguments.
	maxShellInputLength = 32768

	// maxFileSize is the maximum allowed file size for edit/read operations (10 MB).
	maxFileSize int64 = 10 << 20
)

// Sentinel errors for shell input validation.
var (
	errShellInputEmpty       = errors.New("shell input is empty")
	errShellInputTooLong     = errors.New("shell input exceeds maximum length")
	errShellInputNullByte    = errors.New("shell input contains null byte")
	errShellInputControlChar = errors.New("shell input contains control character")
)

// validateShellArg checks a string for dangerous characters before shell escaping.
// Rejects empty strings, null bytes, control characters (except tab/newline/carriage return),
// and strings exceeding maxShellInputLength.
func validateShellArg(s string) error {
	if s == "" {
		return errShellInputEmpty
	}
	if len(s) > maxShellInputLength {
		return errShellInputTooLong
	}
	for _, r := range s {
		if r == 0 {
			return errShellInputNullByte
		}
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			return errShellInputControlChar
		}
	}
	return nil
}

// validateFilePath validates and cleans a file path for sandbox operations.
// Ensures the path is absolute and normalizes it with filepath.Clean.
// Returns the cleaned path or an error.
func validateFilePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	if strings.ContainsRune(path, 0) {
		return "", fmt.Errorf("path contains null byte")
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("path must be absolute: %s", path)
	}
	cleaned := filepath.Clean(path)
	// After Clean on an absolute path, ".." components are resolved.
	// Verify no ".." segments remain (would indicate traversal above root).
	for _, part := range strings.Split(cleaned, string(filepath.Separator)) {
		if part == ".." {
			return "", fmt.Errorf("path contains traversal: %s", path)
		}
	}
	return cleaned, nil
}

// checkFileSize validates that content size is within the allowed limit.
func checkFileSize(size int64) error {
	if size > maxFileSize {
		return fmt.Errorf("file size %d bytes exceeds maximum %d bytes (%.1f MB)", size, maxFileSize, float64(maxFileSize)/(1<<20))
	}
	return nil
}
