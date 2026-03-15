// Package readonly provides client-side command validation for read-only
// golden VM access. Commands are parsed into pipeline segments and each
// segment's first token is checked against an allowlist.
package readonly

import (
	"github.com/aspectrr/fluid.sh/shared/readonly"
)

func AllowedCommandsList() []string {
	return readonly.AllowedCommandsList()
}

// ValidateCommand checks that every command in a pipeline is allowed for read-only mode.
func ValidateCommand(command string) error {
	return readonly.ValidateCommand(command)
}

// ValidateCommandWithExtra checks that every command in a pipeline is allowed,
// using both the default allowlist and extra user-configured commands.
func ValidateCommandWithExtra(command string, extraAllowed []string) error {
	return readonly.ValidateCommandWithExtra(command, extraAllowed)
}
