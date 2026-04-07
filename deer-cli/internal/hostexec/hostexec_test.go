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

func TestWithRelaxedHostKeys(t *testing.T) {
	args := []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=15",
		"-o", "BatchMode=yes",
		"-J", "user@jumphost",
	}

	opt := WithRelaxedHostKeys()
	result := opt(args)

	// Should replace StrictHostKeyChecking=accept-new with StrictHostKeyChecking=no
	foundStrict := false
	foundKnownHosts := false
	for i, a := range result {
		if a == "-o" && i+1 < len(result) {
			if result[i+1] == "StrictHostKeyChecking=no" {
				foundStrict = true
			}
			if result[i+1] == "UserKnownHostsFile=/dev/null" {
				foundKnownHosts = true
			}
			// Should NOT have accept-new anymore
			assert.NotEqual(t, "StrictHostKeyChecking=accept-new", result[i+1])
		}
	}
	assert.True(t, foundStrict, "should have StrictHostKeyChecking=no")
	assert.True(t, foundKnownHosts, "should have UserKnownHostsFile=/dev/null")

	// Other args should be preserved
	assert.Contains(t, result, "-J")
	assert.Contains(t, result, "user@jumphost")

	// Verify flags appear before "--" when used in full arg construction
	// (simulating how NewSSHWithJump builds args: opts first, then user@host -- command)
	fullArgs := append(result, "root@192.168.122.100", "--", "echo hello")
	dashDashIdx := -1
	knownHostsIdx := -1
	for i, a := range fullArgs {
		if a == "--" && dashDashIdx == -1 {
			dashDashIdx = i
		}
		if a == "UserKnownHostsFile=/dev/null" {
			knownHostsIdx = i
		}
	}
	assert.Greater(t, dashDashIdx, 0, "should have -- separator")
	assert.Greater(t, knownHostsIdx, 0, "should have UserKnownHostsFile")
	assert.Less(t, knownHostsIdx, dashDashIdx, "UserKnownHostsFile=/dev/null must appear before --")
}

func TestWithRelaxedHostKeys_NoExistingStrictCheck(t *testing.T) {
	// If no StrictHostKeyChecking is present, should still add both options
	args := []string{"-o", "ConnectTimeout=15", "root@host", "--", "cmd"}
	opt := WithRelaxedHostKeys()
	result := opt(args)

	foundKnownHosts := false
	foundStrictCheck := false
	for i, a := range result {
		if a == "-o" && i+1 < len(result) {
			if result[i+1] == "UserKnownHostsFile=/dev/null" {
				foundKnownHosts = true
			}
			if result[i+1] == "StrictHostKeyChecking=no" {
				foundStrictCheck = true
			}
		}
	}
	assert.True(t, foundKnownHosts, "should add UserKnownHostsFile even without existing StrictHostKeyChecking")
	assert.True(t, foundStrictCheck, "should add StrictHostKeyChecking=no even when not originally present")
}

func TestNewSSHWithJump_WithOptions(t *testing.T) {
	// Verify NewSSHWithJump accepts variadic options and returns non-nil
	run := NewSSHWithJump("192.168.122.100", "root", 22, "user@jumphost", WithRelaxedHostKeys())
	assert.NotNil(t, run)
}

func TestNewSSHWithJump_WithoutOptions(t *testing.T) {
	// Backward compatibility: no options
	run := NewSSHWithJump("192.168.122.100", "root", 22, "user@jumphost")
	assert.NotNil(t, run)
}
