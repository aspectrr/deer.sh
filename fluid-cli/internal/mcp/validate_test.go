package mcp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateShellArg(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"valid simple", "hello", nil},
		{"valid with spaces", "hello world", nil},
		{"valid with quotes", "it's fine", nil},
		{"valid with tab", "col1\tcol2", nil},
		{"valid with newline", "line1\nline2", nil},
		{"valid with carriage return", "line1\r\nline2", nil},
		{"empty string", "", errShellInputEmpty},
		{"null byte", "hello\x00world", errShellInputNullByte},
		{"control char BEL", "hello\x07world", errShellInputControlChar},
		{"control char SOH", "\x01start", errShellInputControlChar},
		{"control char ESC", "prefix\x1bsuffix", errShellInputControlChar},
		{"at max length", strings.Repeat("a", maxShellInputLength), nil},
		{"over max length", strings.Repeat("a", maxShellInputLength+1), errShellInputTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateShellArg(tt.input)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantPath    string
		wantErr     bool
		errContains string
	}{
		{"valid absolute", "/etc/passwd", "/etc/passwd", false, ""},
		{"valid nested", "/var/lib/data/file.txt", "/var/lib/data/file.txt", false, ""},
		{"root path", "/", "/", false, ""},
		{"trailing slash cleaned", "/tmp/", "/tmp", false, ""},
		{"double slash cleaned", "/var//lib//file", "/var/lib/file", false, ""},
		{"dot segments cleaned", "/var/lib/./file", "/var/lib/file", false, ""},
		{"traversal cleaned to valid", "/var/lib/../../etc/passwd", "/etc/passwd", false, ""},
		{"relative path", "foo/bar", "", true, "path must be absolute"},
		{"empty path", "", "", true, "path is required"},
		{"dot only", ".", "", true, "path must be absolute"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateFilePath(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantPath, result)
			}
		})
	}
}

func TestCheckFileSize(t *testing.T) {
	tests := []struct {
		name    string
		size    int64
		wantErr bool
	}{
		{"zero", 0, false},
		{"small", 1024, false},
		{"at limit", maxFileSize, false},
		{"over limit", maxFileSize + 1, true},
		{"way over limit", 100 * 1024 * 1024, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkFileSize(tt.size)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "exceeds maximum")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
