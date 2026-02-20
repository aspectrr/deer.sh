package hostexec

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithSudo(t *testing.T) {
	var captured string
	mockRun := RunFunc(func(ctx context.Context, command string) (string, string, int, error) {
		captured = command
		return "", "", 0, nil
	})

	sudoRun := WithSudo(mockRun)
	_, _, _, err := sudoRun(context.Background(), "apt install nginx")
	require.NoError(t, err)

	// Should wrap with base64 + sudo
	assert.Contains(t, captured, "base64 -d | sudo bash")
	assert.Contains(t, captured, "echo ")

	// Decode the base64 part to verify the original command
	parts := strings.SplitN(captured, "echo ", 2)
	require.Len(t, parts, 2)
	b64Part := strings.Split(parts[1], " |")[0]
	decoded, err := base64.StdEncoding.DecodeString(b64Part)
	require.NoError(t, err)
	assert.Equal(t, "apt install nginx", string(decoded))
}

func TestNewLocal(t *testing.T) {
	run := NewLocal()
	stdout, _, code, err := run(context.Background(), "echo hello")
	require.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Equal(t, "hello\n", stdout)
}

func TestNewLocalExitCode(t *testing.T) {
	run := NewLocal()
	_, _, code, err := run(context.Background(), "exit 42")
	assert.NoError(t, err) // err is nil for non-zero exit codes via ExitError
	assert.Equal(t, 42, code)
}

func TestNewSSHCommandConstruction(t *testing.T) {
	// We can't test actual SSH, but we can verify the function is constructed
	run := NewSSH("192.168.1.100", "root", 22)
	assert.NotNil(t, run)

	// Test with non-standard port
	runCustomPort := NewSSH("192.168.1.100", "root", 2222)
	assert.NotNil(t, runCustomPort)
}
