package setup

import (
	"context"
	"strings"
	"testing"

	"github.com/aspectrr/fluid.sh/fluid/internal/hostexec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectOSUbuntu(t *testing.T) {
	run := hostexec.RunFunc(func(ctx context.Context, command string) (string, string, int, error) {
		return `NAME="Ubuntu"
VERSION="22.04.3 LTS (Jammy Jellyfish)"
ID=ubuntu
PRETTY_NAME="Ubuntu 22.04.3 LTS"
`, "", 0, nil
	})

	distro, err := DetectOS(context.Background(), run)
	require.NoError(t, err)
	assert.Equal(t, "ubuntu", distro.ID)
	assert.Equal(t, "apt", distro.PkgManager)
	assert.Equal(t, "Ubuntu 22.04.3 LTS", distro.Name)
}

func TestDetectOSFedora(t *testing.T) {
	run := hostexec.RunFunc(func(ctx context.Context, command string) (string, string, int, error) {
		return `NAME="Fedora Linux"
ID=fedora
PRETTY_NAME="Fedora Linux 39"
`, "", 0, nil
	})

	distro, err := DetectOS(context.Background(), run)
	require.NoError(t, err)
	assert.Equal(t, "fedora", distro.ID)
	assert.Equal(t, "dnf", distro.PkgManager)
}

func TestDetectOSUnsupported(t *testing.T) {
	run := hostexec.RunFunc(func(ctx context.Context, command string) (string, string, int, error) {
		return `NAME="Arch Linux"
ID=arch
PRETTY_NAME="Arch Linux"
`, "", 0, nil
	})

	_, err := DetectOS(context.Background(), run)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported distribution")
}

func TestDetectOSNoFile(t *testing.T) {
	run := hostexec.RunFunc(func(ctx context.Context, command string) (string, string, int, error) {
		return "", "No such file", 1, nil
	})

	_, err := DetectOS(context.Background(), run)
	assert.Error(t, err)
}

func TestAllStepsCount(t *testing.T) {
	distro := DistroInfo{ID: "ubuntu", PkgManager: "apt"}
	steps := AllSteps(distro)
	assert.Len(t, steps, 8)
}

func TestStepsIdempotent(t *testing.T) {
	// Simulate a system where everything is already set up
	run := hostexec.RunFunc(func(ctx context.Context, command string) (string, string, int, error) {
		// systemctl is-active needs to return "active" for the enable-and-start check
		if command == "systemctl is-active fluid-daemon 2>/dev/null" {
			return "active\n", "", 0, nil
		}
		// id -nG check for libvirt group membership
		if strings.Contains(command, "id -nG fluid-daemon") {
			return "fluid-daemon libvirt\n", "", 0, nil
		}
		return "", "", 0, nil // everything else passes
	})

	distro := DistroInfo{ID: "ubuntu", PkgManager: "apt"}
	steps := AllSteps(distro)

	for _, step := range steps {
		done, err := step.Check(context.Background(), run)
		assert.NoError(t, err)
		assert.True(t, done, "step %s should report done on already-configured system", step.Name)
	}
}

func TestStepsFreshInstall(t *testing.T) {
	// Simulate a system where nothing is set up
	run := hostexec.RunFunc(func(ctx context.Context, command string) (string, string, int, error) {
		return "", "", 1, nil // everything fails
	})

	distro := DistroInfo{ID: "ubuntu", PkgManager: "apt"}
	steps := AllSteps(distro)

	for _, step := range steps {
		done, err := step.Check(context.Background(), run)
		assert.NoError(t, err)
		assert.False(t, done, "step %s should report not-done on fresh system", step.Name)
	}
}

func TestStepExecuteSuccess(t *testing.T) {
	executedCmds := make([]string, 0)
	sudoRun := hostexec.RunFunc(func(ctx context.Context, command string) (string, string, int, error) {
		executedCmds = append(executedCmds, command)
		return "", "", 0, nil
	})

	distro := DistroInfo{ID: "ubuntu", PkgManager: "apt"}
	steps := AllSteps(distro)

	// Execute the first step (install dependencies)
	err := steps[0].Execute(context.Background(), sudoRun)
	assert.NoError(t, err)
	assert.Greater(t, len(executedCmds), 0)
}

func TestStepsHaveCommands(t *testing.T) {
	steps := AllSteps(DistroInfo{ID: "ubuntu", PkgManager: "apt"})
	for _, step := range steps {
		assert.NotEmpty(t, step.Commands, "step %s should have Commands", step.Name)
	}
}

func TestStepExecuteFailure(t *testing.T) {
	sudoRun := hostexec.RunFunc(func(ctx context.Context, command string) (string, string, int, error) {
		return "", "E: Unable to locate package", 100, nil
	})

	distro := DistroInfo{ID: "ubuntu", PkgManager: "apt"}
	steps := AllSteps(distro)

	err := steps[0].Execute(context.Background(), sudoRun)
	assert.Error(t, err)
}
