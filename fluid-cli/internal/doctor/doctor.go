package doctor

import (
	"context"
	"fmt"
	"io"

	"github.com/aspectrr/fluid.sh/fluid/internal/hostexec"
)

// CheckResult holds the outcome of a single doctor check.
type CheckResult struct {
	Name     string
	Category string // "connectivity", "binary", "service", "prerequisites", "storage", "config"
	Passed   bool
	Message  string
	FixCmd   string // empty if passed
}

// RunAll executes all doctor checks and returns results.
func RunAll(ctx context.Context, run hostexec.RunFunc) []CheckResult {
	checks := allChecks()
	results := make([]CheckResult, 0, len(checks))
	for _, c := range checks {
		result := c.fn(ctx, run)
		results = append(results, result)
	}
	return results
}

// PrintResults writes check results to w. Returns true if all checks passed.
func PrintResults(results []CheckResult, w io.Writer, color bool) bool {
	allPassed := true
	passed := 0
	failed := 0

	for _, r := range results {
		var icon, colorStart, colorEnd string
		if r.Passed {
			passed++
			icon = "v"
			if color {
				colorStart = "\033[32m" // green
				colorEnd = "\033[0m"
			}
		} else {
			failed++
			allPassed = false
			icon = "x"
			if color {
				colorStart = "\033[31m" // red
				colorEnd = "\033[0m"
			}
		}
		_, _ = fmt.Fprintf(w, "  %s%s %s%s\n", colorStart, icon, r.Message, colorEnd)
		if !r.Passed && r.FixCmd != "" {
			_, _ = fmt.Fprintf(w, "     Fix: %s\n", r.FixCmd)
		}
	}

	_, _ = fmt.Fprintln(w)
	if allPassed {
		_, _ = fmt.Fprintf(w, "  %d/%d passed\n", passed, passed+failed)
	} else {
		_, _ = fmt.Fprintf(w, "  %d/%d passed, %d failed\n", passed, passed+failed, failed)
	}

	return allPassed
}
