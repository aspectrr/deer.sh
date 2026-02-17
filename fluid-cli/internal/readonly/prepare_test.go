package readonly

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// mockSSHRun records every command passed to it and returns preconfigured responses.
type mockSSHRun struct {
	mu       sync.Mutex
	commands []string
	// responses maps a call index to a response. If not set, returns success.
	responses map[int]sshResponse
}

type sshResponse struct {
	stdout   string
	stderr   string
	exitCode int
	err      error
}

func newMockSSHRun() *mockSSHRun {
	return &mockSSHRun{responses: make(map[int]sshResponse)}
}

func (m *mockSSHRun) failAt(index int, resp sshResponse) {
	m.responses[index] = resp
}

func (m *mockSSHRun) run(ctx context.Context, command string) (string, string, int, error) {
	m.mu.Lock()
	idx := len(m.commands)
	m.commands = append(m.commands, command)
	m.mu.Unlock()

	if resp, ok := m.responses[idx]; ok {
		return resp.stdout, resp.stderr, resp.exitCode, resp.err
	}
	return "", "", 0, nil
}

func (m *mockSSHRun) getCommands() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.commands))
	copy(out, m.commands)
	return out
}

// decodeBase64Command extracts the original command from the base64 wrapper.
// Prepare wraps every command as: echo <base64> | base64 -d | sudo bash
func decodeBase64Command(wrapped string) (string, error) {
	prefix := "echo "
	suffix := " | base64 -d | sudo bash"
	if !strings.HasPrefix(wrapped, prefix) || !strings.HasSuffix(wrapped, suffix) {
		return "", fmt.Errorf("command not in expected base64 wrapper format: %s", wrapped)
	}
	encoded := wrapped[len(prefix) : len(wrapped)-len(suffix)]
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %w", err)
	}
	return string(decoded), nil
}

func TestPrepare_Success(t *testing.T) {
	mock := newMockSSHRun()
	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"

	result, err := Prepare(context.Background(), mock.run, caPubKey, nil, nil)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if !result.ShellInstalled {
		t.Error("expected ShellInstalled=true")
	}
	if !result.UserCreated {
		t.Error("expected UserCreated=true")
	}
	if !result.CAKeyInstalled {
		t.Error("expected CAKeyInstalled=true")
	}
	if !result.SSHDConfigured {
		t.Error("expected SSHDConfigured=true")
	}
	if !result.PrincipalsCreated {
		t.Error("expected PrincipalsCreated=true")
	}
	if !result.SSHDRestarted {
		t.Error("expected SSHDRestarted=true")
	}
}

func TestPrepare_EmptyCAKey(t *testing.T) {
	mock := newMockSSHRun()

	tests := []string{"", "   ", "\n", "\t\n  "}
	for _, key := range tests {
		_, err := Prepare(context.Background(), mock.run, key, nil, nil)
		if err == nil {
			t.Errorf("expected error for CA key %q, got nil", key)
		}
		if !strings.Contains(err.Error(), "CA public key is required") {
			t.Errorf("expected 'CA public key is required' error, got: %v", err)
		}
	}

	// No SSH commands should have been issued
	if len(mock.getCommands()) != 0 {
		t.Errorf("expected no SSH commands for empty CA key, got %d", len(mock.getCommands()))
	}
}

func TestPrepare_Base64Wrapping(t *testing.T) {
	mock := newMockSSHRun()
	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"

	_, err := Prepare(context.Background(), mock.run, caPubKey, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	commands := mock.getCommands()
	if len(commands) == 0 {
		t.Fatal("expected at least one command")
	}

	for i, cmd := range commands {
		if !strings.HasPrefix(cmd, "echo ") || !strings.HasSuffix(cmd, " | base64 -d | sudo bash") {
			t.Errorf("command %d not base64-wrapped: %s", i, cmd)
		}
		_, err := decodeBase64Command(cmd)
		if err != nil {
			t.Errorf("command %d: %v", i, err)
		}
	}
}

func TestPrepare_CommandContent(t *testing.T) {
	mock := newMockSSHRun()
	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"

	_, err := Prepare(context.Background(), mock.run, caPubKey, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	commands := mock.getCommands()
	decoded := make([]string, len(commands))
	for i, cmd := range commands {
		d, err := decodeBase64Command(cmd)
		if err != nil {
			t.Fatalf("command %d decode failed: %v", i, err)
		}
		decoded[i] = d
	}

	// Step 1: install restricted shell - should contain the shell script path
	if !strings.Contains(decoded[0], "/usr/local/bin/fluid-readonly-shell") {
		t.Error("step 1 should install shell to /usr/local/bin/fluid-readonly-shell")
	}
	if !strings.Contains(decoded[0], "chmod 755") {
		t.Error("step 1 should chmod the shell script")
	}
	// Should contain the actual shell script content
	if !strings.Contains(decoded[0], "SSH_ORIGINAL_COMMAND") {
		t.Error("step 1 should contain the restricted shell script content")
	}

	// Step 2: create user - should reference fluid-readonly
	if !strings.Contains(decoded[1], "fluid-readonly") {
		t.Error("step 2 should create fluid-readonly user")
	}
	if !strings.Contains(decoded[1], "useradd") {
		t.Error("step 2 should contain useradd command")
	}

	// Step 3: usermod fixup
	if !strings.Contains(decoded[2], "usermod") {
		t.Error("step 3 should be usermod fixup")
	}

	// Step 4: install CA key - should contain our CA key content
	if !strings.Contains(decoded[3], caPubKey) {
		t.Error("step 4 should contain the CA public key")
	}
	if !strings.Contains(decoded[3], "/etc/ssh/fluid_ca.pub") {
		t.Error("step 4 should write to /etc/ssh/fluid_ca.pub")
	}

	// Steps 5-6: sshd config - TrustedUserCAKeys and AuthorizedPrincipalsFile
	sshdConfigCommands := decoded[4] + " " + decoded[5]
	if !strings.Contains(sshdConfigCommands, "TrustedUserCAKeys") {
		t.Error("sshd config steps should reference TrustedUserCAKeys")
	}
	if !strings.Contains(sshdConfigCommands, "AuthorizedPrincipalsFile") {
		t.Error("sshd config steps should reference AuthorizedPrincipalsFile")
	}

	// Steps 7-9: principals
	principalsAll := strings.Join(decoded[6:9], " ")
	if !strings.Contains(principalsAll, "/etc/ssh/authorized_principals") {
		t.Error("principals steps should reference /etc/ssh/authorized_principals")
	}

	// Last step: restart sshd
	lastCmd := decoded[len(decoded)-1]
	if !strings.Contains(lastCmd, "restart sshd") && !strings.Contains(lastCmd, "restart ssh") {
		t.Error("last step should restart sshd")
	}
}

func TestPrepare_CAKeyTrimmed(t *testing.T) {
	mock := newMockSSHRun()
	caPubKey := "  ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca  \n"

	_, err := Prepare(context.Background(), mock.run, caPubKey, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	commands := mock.getCommands()
	// Find the CA key install command (step 4, index 3)
	caCmd, err := decodeBase64Command(commands[3])
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	// Should contain trimmed key, not the whitespace-padded version
	if strings.Contains(caCmd, "  ssh-ed25519") {
		t.Error("CA key should be trimmed of leading whitespace")
	}
	if strings.Contains(caCmd, "test-ca  \n") {
		t.Error("CA key should be trimmed of trailing whitespace")
	}
}

func TestPrepare_ProgressReporting(t *testing.T) {
	mock := newMockSSHRun()
	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"

	var progress []PrepareProgress
	onProgress := func(p PrepareProgress) {
		progress = append(progress, p)
	}

	_, err := Prepare(context.Background(), mock.run, caPubKey, onProgress, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Each of 6 steps should report start (Done=false) and completion (Done=true)
	expectedSteps := []PrepareStep{
		StepInstallShell,
		StepCreateUser,
		StepInstallCAKey,
		StepConfigureSSHD,
		StepCreatePrincipals,
		StepRestartSSHD,
	}

	if len(progress) != 12 {
		t.Fatalf("expected 12 progress reports (6 steps x start+done), got %d", len(progress))
	}

	for i, step := range expectedSteps {
		startIdx := i * 2
		doneIdx := i*2 + 1

		if progress[startIdx].Step != step {
			t.Errorf("progress[%d]: expected step %d, got %d", startIdx, step, progress[startIdx].Step)
		}
		if progress[startIdx].Done {
			t.Errorf("progress[%d]: expected Done=false (start)", startIdx)
		}
		if progress[startIdx].Total != 6 {
			t.Errorf("progress[%d]: expected Total=6, got %d", startIdx, progress[startIdx].Total)
		}
		if progress[startIdx].StepName == "" {
			t.Errorf("progress[%d]: StepName should not be empty", startIdx)
		}

		if progress[doneIdx].Step != step {
			t.Errorf("progress[%d]: expected step %d, got %d", doneIdx, step, progress[doneIdx].Step)
		}
		if !progress[doneIdx].Done {
			t.Errorf("progress[%d]: expected Done=true (completed)", doneIdx)
		}
	}
}

func TestPrepare_NilProgressDoesNotPanic(t *testing.T) {
	mock := newMockSSHRun()
	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"

	// Should not panic with nil progress function
	_, err := Prepare(context.Background(), mock.run, caPubKey, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepare_FailAtInstallShell(t *testing.T) {
	mock := newMockSSHRun()
	mock.failAt(0, sshResponse{stderr: "permission denied", exitCode: 1})
	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"

	result, err := Prepare(context.Background(), mock.run, caPubKey, nil, nil)
	if err == nil {
		t.Fatal("expected error when shell install fails")
	}
	if !strings.Contains(err.Error(), "install restricted shell") {
		t.Errorf("error should mention install restricted shell: %v", err)
	}
	if result.ShellInstalled {
		t.Error("ShellInstalled should be false on failure")
	}
	if result.UserCreated {
		t.Error("UserCreated should be false (step not reached)")
	}

	// Only 1 command should have been attempted
	if len(mock.getCommands()) != 1 {
		t.Errorf("expected 1 command attempt, got %d", len(mock.getCommands()))
	}
}

func TestPrepare_FailAtCreateUser(t *testing.T) {
	mock := newMockSSHRun()
	// Step 1 succeeds (install shell), step 2 fails (create user)
	mock.failAt(1, sshResponse{stderr: "useradd failed", exitCode: 1})
	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"

	result, err := Prepare(context.Background(), mock.run, caPubKey, nil, nil)
	if err == nil {
		t.Fatal("expected error when user creation fails")
	}
	if !strings.Contains(err.Error(), "create fluid-readonly user") {
		t.Errorf("error should mention create fluid-readonly user: %v", err)
	}
	if !result.ShellInstalled {
		t.Error("ShellInstalled should be true (succeeded before failure)")
	}
	if result.UserCreated {
		t.Error("UserCreated should be false on failure")
	}
	if result.CAKeyInstalled {
		t.Error("CAKeyInstalled should be false (step not reached)")
	}
}

func TestPrepare_FailAtCAKeyInstall(t *testing.T) {
	mock := newMockSSHRun()
	// Steps 1-3 succeed (install shell, create user, usermod fixup), step 4 fails (CA key)
	mock.failAt(3, sshResponse{stderr: "write failed", exitCode: 1})
	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"

	result, err := Prepare(context.Background(), mock.run, caPubKey, nil, nil)
	if err == nil {
		t.Fatal("expected error when CA key install fails")
	}
	if !strings.Contains(err.Error(), "install CA pub key") {
		t.Errorf("error should mention install CA pub key: %v", err)
	}
	if !result.ShellInstalled {
		t.Error("ShellInstalled should be true")
	}
	if !result.UserCreated {
		t.Error("UserCreated should be true")
	}
	if result.CAKeyInstalled {
		t.Error("CAKeyInstalled should be false on failure")
	}
}

func TestPrepare_FailAtSSHDConfig(t *testing.T) {
	mock := newMockSSHRun()
	// Steps 1-4 succeed, step 5 (first sshd config command) fails
	mock.failAt(4, sshResponse{stderr: "sshd_config locked", exitCode: 1})
	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"

	result, err := Prepare(context.Background(), mock.run, caPubKey, nil, nil)
	if err == nil {
		t.Fatal("expected error when sshd config fails")
	}
	if !strings.Contains(err.Error(), "configure sshd") {
		t.Errorf("error should mention configure sshd: %v", err)
	}
	if !result.CAKeyInstalled {
		t.Error("CAKeyInstalled should be true")
	}
	if result.SSHDConfigured {
		t.Error("SSHDConfigured should be false on failure")
	}
}

func TestPrepare_FailAtPrincipals(t *testing.T) {
	mock := newMockSSHRun()
	// Steps 1-6 succeed, step 7 (first principals command) fails
	mock.failAt(6, sshResponse{stderr: "mkdir failed", exitCode: 1})
	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"

	result, err := Prepare(context.Background(), mock.run, caPubKey, nil, nil)
	if err == nil {
		t.Fatal("expected error when principals creation fails")
	}
	if !strings.Contains(err.Error(), "create principals") {
		t.Errorf("error should mention create principals: %v", err)
	}
	if !result.SSHDConfigured {
		t.Error("SSHDConfigured should be true")
	}
	if result.PrincipalsCreated {
		t.Error("PrincipalsCreated should be false on failure")
	}
}

func TestPrepare_FailAtRestartSSHD(t *testing.T) {
	mock := newMockSSHRun()
	// All steps succeed except the last one (restart sshd, index 9)
	mock.failAt(9, sshResponse{stderr: "sshd restart failed", exitCode: 1})
	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"

	result, err := Prepare(context.Background(), mock.run, caPubKey, nil, nil)
	if err == nil {
		t.Fatal("expected error when sshd restart fails")
	}
	if !strings.Contains(err.Error(), "restart sshd") {
		t.Errorf("error should mention restart sshd: %v", err)
	}
	if !result.PrincipalsCreated {
		t.Error("PrincipalsCreated should be true")
	}
	if result.SSHDRestarted {
		t.Error("SSHDRestarted should be false on failure")
	}
}

func TestPrepare_SSHRunError(t *testing.T) {
	mock := newMockSSHRun()
	mock.failAt(0, sshResponse{err: fmt.Errorf("connection refused")})
	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"

	result, err := Prepare(context.Background(), mock.run, caPubKey, nil, nil)
	if err == nil {
		t.Fatal("expected error when SSH connection fails")
	}
	if result.ShellInstalled {
		t.Error("ShellInstalled should be false on SSH error")
	}
	if len(mock.getCommands()) != 1 {
		t.Errorf("should stop after first failure, got %d commands", len(mock.getCommands()))
	}
}

func TestPrepare_UsermodFailureNonFatal(t *testing.T) {
	mock := newMockSSHRun()
	// usermod fixup is command index 2, make it fail
	mock.failAt(2, sshResponse{stderr: "usermod failed", exitCode: 1})
	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"

	result, err := Prepare(context.Background(), mock.run, caPubKey, nil, nil)
	if err != nil {
		t.Fatalf("usermod failure should be non-fatal, got error: %v", err)
	}

	// All steps should still complete
	if !result.ShellInstalled {
		t.Error("ShellInstalled should be true")
	}
	if !result.UserCreated {
		t.Error("UserCreated should be true")
	}
	if !result.CAKeyInstalled {
		t.Error("CAKeyInstalled should be true")
	}
	if !result.SSHDConfigured {
		t.Error("SSHDConfigured should be true")
	}
	if !result.PrincipalsCreated {
		t.Error("PrincipalsCreated should be true")
	}
	if !result.SSHDRestarted {
		t.Error("SSHDRestarted should be true")
	}
}

func TestPrepare_CommandCount(t *testing.T) {
	mock := newMockSSHRun()
	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"

	_, err := Prepare(context.Background(), mock.run, caPubKey, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	commands := mock.getCommands()
	// Expected commands:
	// 1: install shell script
	// 2: create user (useradd)
	// 3: usermod fixup
	// 4: install CA key
	// 5: sshd config - TrustedUserCAKeys
	// 6: sshd config - AuthorizedPrincipalsFile
	// 7: mkdir authorized_principals
	// 8: write principals file
	// 9: chmod principals file
	// 10: restart sshd
	if len(commands) != 10 {
		t.Errorf("expected 10 SSH commands, got %d", len(commands))
		for i, cmd := range commands {
			decoded, _ := decodeBase64Command(cmd)
			summary := decoded
			if len(summary) > 80 {
				summary = summary[:80] + "..."
			}
			t.Logf("  command %d: %s", i, summary)
		}
	}
}

func TestPrepare_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	callCount := 0
	sshRun := func(ctx context.Context, command string) (string, string, int, error) {
		callCount++
		if ctx.Err() != nil {
			return "", "", 0, ctx.Err()
		}
		return "", "", 0, nil
	}

	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"
	_, err := Prepare(ctx, sshRun, caPubKey, nil, nil)
	if err == nil {
		t.Fatal("expected error with cancelled context")
	}

	// Should fail on the first SSH call
	if callCount != 1 {
		t.Errorf("expected 1 call with cancelled context, got %d", callCount)
	}
}

func TestPrepare_ProgressOnFailure(t *testing.T) {
	mock := newMockSSHRun()
	// Fail at step 2 (create user, command index 1)
	mock.failAt(1, sshResponse{stderr: "fail", exitCode: 1})
	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"

	var progress []PrepareProgress
	onProgress := func(p PrepareProgress) {
		progress = append(progress, p)
	}

	_, _ = Prepare(context.Background(), mock.run, caPubKey, onProgress, nil)

	// Step 1 (install shell) should have start+done = 2 reports
	// Step 2 (create user) should have start only = 1 report
	// Total: 3
	if len(progress) != 3 {
		t.Errorf("expected 3 progress reports on step 2 failure, got %d", len(progress))
		for i, p := range progress {
			t.Logf("  progress[%d]: step=%d name=%q done=%v", i, p.Step, p.StepName, p.Done)
		}
	}

	if len(progress) >= 3 {
		// Last progress should be start of step 2
		if progress[2].Step != StepCreateUser {
			t.Errorf("last progress should be StepCreateUser, got %d", progress[2].Step)
		}
		if progress[2].Done {
			t.Error("last progress should be Done=false (step failed)")
		}
	}
}

func TestPrepare_ShellScriptContent(t *testing.T) {
	mock := newMockSSHRun()
	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"

	_, err := Prepare(context.Background(), mock.run, caPubKey, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	commands := mock.getCommands()
	shellCmd, err := decodeBase64Command(commands[0])
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	// The install shell command should contain the full RestrictedShellScript
	if !strings.Contains(shellCmd, RestrictedShellScript) {
		t.Error("install shell command should embed the full RestrictedShellScript content")
	}

	// Should use a heredoc with the FLUID_SHELL_EOF delimiter
	if !strings.Contains(shellCmd, "FLUID_SHELL_EOF") {
		t.Error("shell install should use FLUID_SHELL_EOF heredoc delimiter")
	}
}

func TestPrepare_IdempotentUserCreation(t *testing.T) {
	mock := newMockSSHRun()
	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"

	_, err := Prepare(context.Background(), mock.run, caPubKey, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	commands := mock.getCommands()
	userCmd, err := decodeBase64Command(commands[1])
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	// Should check if user exists before creating
	if !strings.Contains(userCmd, "id fluid-readonly") {
		t.Error("user creation should check if user already exists via 'id' command")
	}

	// Should use conditional creation (|| pattern)
	if !strings.Contains(userCmd, "||") {
		t.Error("user creation should use || for idempotent create")
	}
}

func TestPrepare_SSHDConfigIdempotent(t *testing.T) {
	mock := newMockSSHRun()
	caPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test-ca"

	_, err := Prepare(context.Background(), mock.run, caPubKey, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	commands := mock.getCommands()
	// sshd config commands are at indices 4 and 5
	for _, idx := range []int{4, 5} {
		cmd, err := decodeBase64Command(commands[idx])
		if err != nil {
			t.Fatalf("decode command %d failed: %v", idx, err)
		}
		// Each should use grep -q ... || echo to be idempotent
		if !strings.Contains(cmd, "grep -q") {
			t.Errorf("sshd config command %d should use grep -q for idempotency", idx)
		}
		if !strings.Contains(cmd, "||") {
			t.Errorf("sshd config command %d should use || for conditional append", idx)
		}
	}
}
