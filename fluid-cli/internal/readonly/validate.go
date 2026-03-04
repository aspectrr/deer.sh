// Package readonly provides client-side command validation for read-only
// golden VM access. Commands are parsed into pipeline segments and each
// segment's first token is checked against an allowlist.
package readonly

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// allowedCommands is the set of commands permitted in read-only mode.
var allowedCommands = map[string]bool{
	// File inspection
	"cat": true, "ls": true, "find": true, "head": true, "tail": true,
	"stat": true, "file": true, "wc": true, "du": true, "tree": true,
	"strings": true, "md5sum": true, "sha256sum": true, "readlink": true,
	"realpath": true, "basename": true, "dirname": true, "base64": true,

	// Process/system
	"ps": true, "top": true, "pgrep": true,
	"systemctl": true, "journalctl": true, "dmesg": true,

	// Network
	"ss": true, "netstat": true, "ip": true, "ifconfig": true,
	"dig": true, "nslookup": true, "ping": true,

	// Disk
	"df": true, "lsblk": true, "blkid": true,

	// Package query
	"dpkg": true, "rpm": true, "apt": true, "pip": true,

	// System info
	"uname": true, "hostname": true, "uptime": true, "free": true,
	"lscpu": true, "lsmod": true, "lspci": true, "lsusb": true,
	"arch": true, "nproc": true,

	// User
	"whoami": true, "id": true, "groups": true, "who": true,
	"w": true, "last": true,

	// Misc
	"env": true, "printenv": true, "date": true, "which": true,
	"type": true, "echo": true, "test": true,

	// Pipe targets
	// NOTE: awk can write files via internal '>' redirection (e.g. awk '{print > "file"}').
	// Blocking this requires parsing awk scripts, which is out of scope. Server-side
	// shell provides defense-in-depth.
	"grep": true, "awk": true, "sed": true, "sort": true, "uniq": true,
	"cut": true, "tr": true, "xargs": true,
}

// blockedFlags maps commands to flags that must not appear anywhere in the
// segment. For example, sed -i performs in-place editing which violates
// read-only mode. Also catches variants like -i.bak (starts with -i).
var blockedFlags = map[string][]string{
	"sed": {"-i", "--in-place"},
}

// commandArgValidators maps commands to functions that validate their arguments
// within a pipeline segment. For example, xargs can invoke arbitrary commands
// so we validate that the first non-flag argument (if any) is in the allowlist.
var commandArgValidators = map[string]func(tokens []string, allowed map[string]bool) error{
	"xargs": validateXargsCommand,
}

// validateXargsCommand checks that xargs does not invoke a disallowed command.
// xargs with no explicit command defaults to /bin/echo, which is safe.
func validateXargsCommand(tokens []string, allowed map[string]bool) error {
	// tokens[0] is "xargs" itself; scan remaining for first non-flag token
	for _, tok := range tokens[1:] {
		if strings.HasPrefix(tok, "-") {
			continue
		}
		// Handle path-qualified commands like /usr/bin/rm
		base := tok
		if idx := strings.LastIndex(tok, "/"); idx >= 0 {
			base = tok[idx+1:]
		}
		if !allowed[base] {
			return fmt.Errorf("xargs command %q is not allowed in read-only mode", base)
		}
		return nil
	}
	return nil // no explicit command; xargs defaults to /bin/echo
}

// subcommandRestrictions maps commands to the set of allowed first arguments.
// If a command appears here, its first argument must be in the allowed set.
var subcommandRestrictions = map[string]map[string]bool{
	"systemctl": {
		"status":     true,
		"show":       true,
		"list-units": true,
		"is-active":  true,
		"is-enabled": true,
	},
	"dpkg": {
		"-l":     true,
		"--list": true,
	},
	"rpm": {
		"-qa": true,
		"-q":  true,
	},
	"apt": {
		"list": true,
	},
	"pip": {
		"list": true,
	},
}

// ValidateCommand checks that every command in a pipeline is allowed for
// read-only mode. Returns nil if all commands are allowed, or an error
// describing the first violation found.
func ValidateCommand(command string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return fmt.Errorf("empty command")
	}

	// Block dangerous shell metacharacters that could be used for command injection.
	if err := checkDangerousMetacharacters(command); err != nil {
		return err
	}

	// Block output redirection (unquoted > or >>).
	if containsUnquotedRedirection(command) {
		return fmt.Errorf("output redirection is not allowed in read-only mode")
	}

	// Split on pipes to get pipeline segments.
	segments := splitPipeline(command)

	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		// Extract the base command (first token), skipping env var assignments.
		baseCmd := extractBaseCommand(seg)
		if baseCmd == "" {
			continue
		}

		if !allowedCommands[baseCmd] {
			return fmt.Errorf("command %q is not allowed in read-only mode", baseCmd)
		}

		// Check blocked flags (e.g. sed -i).
		if flags, ok := blockedFlags[baseCmd]; ok {
			tokens := tokenize(seg)
			for _, tok := range tokens[1:] {
				for _, blocked := range flags {
					if tok == blocked || strings.HasPrefix(tok, blocked) {
						return fmt.Errorf("%s flag %q is not allowed in read-only mode", baseCmd, blocked)
					}
				}
			}
		}

		// Check subcommand restrictions if applicable.
		if restrictions, ok := subcommandRestrictions[baseCmd]; ok {
			subCmd := extractSubcommand(seg, baseCmd)
			if subCmd != "" && !restrictions[subCmd] {
				return fmt.Errorf("%s subcommand %q is not allowed in read-only mode (allowed: %s)",
					baseCmd, subCmd, joinKeys(restrictions))
			}
		}

		// Validate command arguments (e.g. xargs must invoke allowed commands).
		if validator, ok := commandArgValidators[baseCmd]; ok {
			tokens := tokenize(seg)
			if err := validator(tokens, allowedCommands); err != nil {
				return err
			}
		}
	}

	return nil
}

// checkDangerousMetacharacters detects shell expansion primitives that could
// be used to smuggle commands past the allowlist. We block command substitution,
// process substitution, and newlines outside of quotes.
func checkDangerousMetacharacters(s string) error {
	inSingle := false
	inDouble := false
	prev := rune(0)

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		switch {
		case ch == '\'' && !inDouble && prev != '\\':
			inSingle = !inSingle
		case ch == '"' && !inSingle && prev != '\\':
			inDouble = !inDouble
		case !inSingle && !inDouble:
			// Check for command substitution: $(...)
			if ch == '$' && i+1 < len(runes) && runes[i+1] == '(' {
				return fmt.Errorf("command substitution $(...) is not allowed in read-only mode")
			}
			// Check for backticks (alternate command substitution)
			if ch == '`' {
				return fmt.Errorf("backtick command substitution is not allowed in read-only mode")
			}
			// Check for process substitution: <(...) or >(...)
			if (ch == '<' || ch == '>') && i+1 < len(runes) && runes[i+1] == '(' {
				return fmt.Errorf("process substitution is not allowed in read-only mode")
			}
			// Check for newlines (could be used to inject additional commands)
			if ch == '\n' || ch == '\r' {
				return fmt.Errorf("newline characters are not allowed in read-only mode")
			}
		}
		prev = ch
	}
	return nil
}

// containsUnquotedRedirection detects > or >> outside of quotes.
func containsUnquotedRedirection(s string) bool {
	inSingle := false
	inDouble := false
	prev := rune(0)

	for _, ch := range s {
		switch {
		case ch == '\'' && !inDouble && prev != '\\':
			inSingle = !inSingle
		case ch == '"' && !inSingle && prev != '\\':
			inDouble = !inDouble
		case ch == '>' && !inSingle && !inDouble:
			// Check it's not inside a process substitution like >(cmd)
			// A simple > or >> outside quotes is a redirect.
			return true
		}
		prev = ch
	}
	return false
}

// splitPipeline splits a command string on unquoted pipe characters.
// It also splits on ; and && to handle chained commands.
func splitPipeline(s string) []string {
	var segments []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	prev := rune(0)

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		switch {
		case ch == '\'' && !inDouble && prev != '\\':
			inSingle = !inSingle
			current.WriteRune(ch)
		case ch == '"' && !inSingle && prev != '\\':
			inDouble = !inDouble
			current.WriteRune(ch)
		case ch == '|' && !inSingle && !inDouble:
			// Skip || (logical OR) - treat as separator anyway
			if i+1 < len(runes) && runes[i+1] == '|' {
				segments = append(segments, current.String())
				current.Reset()
				i++ // skip second |
			} else {
				segments = append(segments, current.String())
				current.Reset()
			}
		case ch == ';' && !inSingle && !inDouble:
			segments = append(segments, current.String())
			current.Reset()
		case ch == '&' && !inSingle && !inDouble:
			if i+1 < len(runes) && runes[i+1] == '&' {
				segments = append(segments, current.String())
				current.Reset()
				i++ // skip second &
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
		prev = ch
	}

	if current.Len() > 0 {
		segments = append(segments, current.String())
	}

	return segments
}

// envAssignRe matches shell env var assignments like FOO=bar or _VAR=value.
var envAssignRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// extractBaseCommand returns the first actual command token from a segment,
// skipping leading environment variable assignments (VAR=value).
func extractBaseCommand(seg string) string {
	tokens := tokenize(seg)
	for _, tok := range tokens {
		// Skip env var assignments like FOO=bar but not --config=/path
		if envAssignRe.MatchString(tok) {
			continue
		}
		// Handle path-qualified commands like /usr/bin/cat
		base := tok
		if idx := strings.LastIndex(tok, "/"); idx >= 0 {
			base = tok[idx+1:]
		}
		return base
	}
	return ""
}

// extractSubcommand returns the first argument after the base command,
// which for restricted commands is the subcommand to check.
func extractSubcommand(seg, baseCmd string) string {
	tokens := tokenize(seg)
	foundBase := false
	for _, tok := range tokens {
		if !foundBase {
			// Skip env assignments
			if envAssignRe.MatchString(tok) {
				continue
			}
			base := tok
			if idx := strings.LastIndex(tok, "/"); idx >= 0 {
				base = tok[idx+1:]
			}
			if base == baseCmd {
				foundBase = true
				continue
			}
		} else {
			return tok
		}
	}
	return ""
}

// tokenize splits a command segment into shell-like tokens, respecting quotes.
func tokenize(s string) []string {
	var tokens []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	prev := rune(0)

	for _, ch := range s {
		switch {
		case ch == '\'' && !inDouble && prev != '\\':
			inSingle = !inSingle
		case ch == '"' && !inSingle && prev != '\\':
			inDouble = !inDouble
		case (ch == ' ' || ch == '\t') && !inSingle && !inDouble:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
		prev = ch
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// ValidateCommandWithExtra checks that every command in a pipeline is allowed,
// using both the default allowlist and extra user-configured commands.
func ValidateCommandWithExtra(command string, extraAllowed []string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return fmt.Errorf("empty command")
	}

	if err := checkDangerousMetacharacters(command); err != nil {
		return err
	}

	if containsUnquotedRedirection(command) {
		return fmt.Errorf("output redirection is not allowed in read-only mode")
	}

	// Build merged allowlist
	merged := make(map[string]bool, len(allowedCommands)+len(extraAllowed))
	for k, v := range allowedCommands {
		merged[k] = v
	}
	for _, cmd := range extraAllowed {
		cmd = strings.TrimSpace(cmd)
		if cmd != "" {
			merged[cmd] = true
		}
	}

	segments := splitPipeline(command)

	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		baseCmd := extractBaseCommand(seg)
		if baseCmd == "" {
			continue
		}

		if !merged[baseCmd] {
			return fmt.Errorf("command %q is not allowed in read-only mode", baseCmd)
		}

		// Check blocked flags (e.g. sed -i).
		if flags, ok := blockedFlags[baseCmd]; ok {
			tokens := tokenize(seg)
			for _, tok := range tokens[1:] {
				for _, blocked := range flags {
					if tok == blocked || strings.HasPrefix(tok, blocked) {
						return fmt.Errorf("%s flag %q is not allowed in read-only mode", baseCmd, blocked)
					}
				}
			}
		}

		if restrictions, ok := subcommandRestrictions[baseCmd]; ok {
			subCmd := extractSubcommand(seg, baseCmd)
			if subCmd != "" && !restrictions[subCmd] {
				return fmt.Errorf("%s subcommand %q is not allowed in read-only mode (allowed: %s)",
					baseCmd, subCmd, joinKeys(restrictions))
			}
		}

		// Validate command arguments (e.g. xargs must invoke allowed commands).
		if validator, ok := commandArgValidators[baseCmd]; ok {
			tokens := tokenize(seg)
			if err := validator(tokens, merged); err != nil {
				return err
			}
		}
	}

	return nil
}

// AllowedCommandsList returns a sorted slice of default allowed command names.
func AllowedCommandsList() []string {
	cmds := make([]string, 0, len(allowedCommands))
	for k := range allowedCommands {
		cmds = append(cmds, k)
	}
	sort.Strings(cmds)
	return cmds
}

// joinKeys returns a comma-separated list of map keys.
func joinKeys(m map[string]bool) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}
