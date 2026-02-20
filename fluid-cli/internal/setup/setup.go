package setup

import (
	"context"

	"github.com/aspectrr/fluid.sh/fluid/internal/hostexec"
)

// StepDef defines a single setup step.
type StepDef struct {
	Name        string
	Description string
	Commands    []string // Human-readable commands shown before execution
	// Check returns true if this step is already done (skip).
	Check func(ctx context.Context, run hostexec.RunFunc) (done bool, err error)
	// Execute runs the step. Uses sudoRun for privileged operations.
	Execute func(ctx context.Context, sudoRun hostexec.RunFunc) error
}

// StepResult holds the outcome of executing a single setup step.
type StepResult struct {
	Name    string
	Skipped bool // already done per Check()
	Success bool
	Error   string
}

// AllSteps returns the ordered list of setup steps for the given distro.
func AllSteps(distro DistroInfo) []StepDef {
	return allSteps(distro)
}
