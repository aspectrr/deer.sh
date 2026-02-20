package doctor

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunAllAllPass(t *testing.T) {
	run := func(ctx context.Context, command string) (string, string, int, error) {
		// Everything succeeds
		if strings.Contains(command, "systemctl is-active fluid-daemon") {
			return "active\n", "", 0, nil
		}
		if strings.Contains(command, "systemctl is-enabled fluid-daemon") {
			return "enabled\n", "", 0, nil
		}
		if strings.Contains(command, "systemctl is-active libvirtd") {
			return "active\n", "", 0, nil
		}
		if strings.Contains(command, "which fluid-daemon") {
			return "/usr/local/bin/fluid-daemon\n", "", 0, nil
		}
		if strings.Contains(command, "which qemu-system") {
			return "/usr/bin/qemu-system\n", "", 0, nil
		}
		if strings.Contains(command, "ss -tlnp") {
			return "LISTEN 0 128 *:9091 *:*\n", "", 0, nil
		}
		if strings.Contains(command, "test -e /dev/kvm") {
			return "", "", 0, nil
		}
		if strings.Contains(command, "test -d /var/lib/fluid") {
			return "", "", 0, nil
		}
		if strings.Contains(command, "test -f") {
			return "", "", 0, nil
		}
		return "", "", 0, nil
	}

	results := RunAll(context.Background(), run)
	assert.Len(t, results, 9)
	for _, r := range results {
		assert.True(t, r.Passed, "check %s should pass", r.Name)
	}
}

func TestRunAllMixedFailures(t *testing.T) {
	run := func(ctx context.Context, command string) (string, string, int, error) {
		// Only daemon binary and KVM pass
		if strings.Contains(command, "which fluid-daemon") {
			return "/usr/local/bin/fluid-daemon\n", "", 0, nil
		}
		if strings.Contains(command, "test -e /dev/kvm") {
			return "", "", 0, nil
		}
		// Everything else fails
		if strings.Contains(command, "systemctl is-active") {
			return "inactive\n", "", 0, nil
		}
		if strings.Contains(command, "systemctl is-enabled") {
			return "disabled\n", "", 0, nil
		}
		return "", "", 1, nil
	}

	results := RunAll(context.Background(), run)
	assert.Len(t, results, 9)

	passCount := 0
	for _, r := range results {
		if r.Passed {
			passCount++
		} else {
			assert.NotEmpty(t, r.FixCmd, "failed check %s should have a fix command", r.Name)
		}
	}
	assert.Equal(t, 2, passCount)
}

func TestPrintResultsAllPass(t *testing.T) {
	results := []CheckResult{
		{Name: "test1", Passed: true, Message: "check 1 ok"},
		{Name: "test2", Passed: true, Message: "check 2 ok"},
	}

	var buf bytes.Buffer
	allPassed := PrintResults(results, &buf, false)
	assert.True(t, allPassed)
	assert.Contains(t, buf.String(), "2/2 passed")
}

func TestPrintResultsWithFailures(t *testing.T) {
	results := []CheckResult{
		{Name: "test1", Passed: true, Message: "check 1 ok"},
		{Name: "test2", Passed: false, Message: "check 2 failed", FixCmd: "fix it"},
	}

	var buf bytes.Buffer
	allPassed := PrintResults(results, &buf, false)
	assert.False(t, allPassed)
	assert.Contains(t, buf.String(), "1/2 passed, 1 failed")
	assert.Contains(t, buf.String(), "Fix: fix it")
}

func TestPrintResultsWithColor(t *testing.T) {
	results := []CheckResult{
		{Name: "test1", Passed: true, Message: "ok"},
		{Name: "test2", Passed: false, Message: "fail", FixCmd: "fix"},
	}

	var buf bytes.Buffer
	PrintResults(results, &buf, true)
	// Should contain ANSI escape codes
	assert.Contains(t, buf.String(), "\033[32m") // green
	assert.Contains(t, buf.String(), "\033[31m") // red
}
