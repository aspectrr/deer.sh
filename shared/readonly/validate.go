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
	"curl": true,

	// TLS/cert diagnostics
	"openssl": true,

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
	"sed":  {"-i", "--in-place"},
	"curl": {"-X", "--request", "-d", "--data", "--data-raw", "--data-binary", "--data-urlencode", "-F", "--form", "-T", "--upload-file", "-o", "--output", "-O", "--remote-name", "-K", "--config", "-x", "--proxy", "--proxy-anyauth", "--proxy-basic", "--proxy-digest", "--proxy-negotiate", "--proxy-ntlm", "--proxy-header", "--proxy-insecure", "--proxy-key", "--proxy-user", "--proxy-pass", "--proxy1.0", "--preproxy"},
}

// commandArgValidators maps commands to functions that validate their arguments
// within a pipeline segment. For example, xargs can invoke arbitrary commands
// so we validate that the first non-flag argument (if any) is in the allowlist.
var commandArgValidators = map[string]func(tokens []string, allowed map[string]bool) error{
	"xargs":   ValidateXargsCommand,
	"openssl": ValidateOpenSSLArgs,
}

// ValidateXargsCommand checks that xargs does not invoke a disallowed command.
// xargs with no explicit command defaults to /bin/echo, which is safe.
func ValidateXargsCommand(tokens []string, allowed map[string]bool) error {
	for _, tok := range tokens[1:] {
		if strings.HasPrefix(tok, "-") {
			continue
		}
		base := tok
		if idx := strings.LastIndex(tok, "/"); idx >= 0 {
			base = tok[idx+1:]
		}
		if !allowed[base] {
			return fmt.Errorf("xargs command %q is not allowed in read-only mode", base)
		}
		return nil
	}
	return nil
}

// ValidateOpenSSLArgs checks that specific openssl subcommands don't use
// dangerous flags.
func ValidateOpenSSLArgs(tokens []string, allowed map[string]bool) error {
	var subCmd string
	for _, tok := range tokens[1:] {
		if !strings.HasPrefix(tok, "-") {
			subCmd = tok
			break
		}
	}
	if subCmd == "req" {
		for _, tok := range tokens {
			if tok == "-new" || tok == "-signkey" || tok == "-x509" {
				return fmt.Errorf("openssl req %s is not allowed in read-only mode", tok)
			}
		}
	}
	if subCmd == "s_client" {
		for i, tok := range tokens {
			if tok == "-proxy" {
				return fmt.Errorf("openssl s_client -proxy is not allowed in read-only mode")
			}
			if tok == "-connect" && i+1 < len(tokens) {
				hostPort := tokens[i+1]
				var host string
				if strings.HasPrefix(hostPort, "[") {
					if end := strings.Index(hostPort, "]"); end >= 0 {
						host = hostPort[1:end]
					} else {
						host = strings.TrimPrefix(hostPort, "[")
					}
				} else {
					host = hostPort
					if idx := strings.LastIndex(hostPort, ":"); idx >= 0 {
						host = hostPort[:idx]
					}
				}
				if host != "localhost" && host != "127.0.0.1" && host != "::1" && host != "" {
					return fmt.Errorf("openssl s_client -connect only allowed to localhost, got %q", host)
				}
			}
		}
	}
	return nil
}

// subcommandRestrictions maps commands to the set of allowed first arguments.
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
	"openssl": {
		"x509":     true,
		"verify":   true,
		"s_client": true,
		"crl":      true,
		"version":  true,
		"ciphers":  true,
		"req":      true,
	},
}

// envAssignRe matches shell env var assignments like FOO=bar or _VAR=value.
var envAssignRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// ValidateCommand checks that every command in a pipeline is allowed for
// read-only mode.
func ValidateCommand(command string) error {
	return validateCommand(command, allowedCommands)
}

func validateCommand(command string, allowed map[string]bool) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return fmt.Errorf("empty command")
	}

	if err := CheckDangerousMetacharacters(command); err != nil {
		return err
	}

	if ContainsUnquotedRedirection(command) {
		return fmt.Errorf("output redirection is not allowed in read-only mode")
	}

	segments := SplitPipeline(command)

	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		baseCmd := ExtractBaseCommand(seg)
		if baseCmd == "" {
			continue
		}

		if !allowed[baseCmd] {
			return fmt.Errorf("command %q is not allowed in read-only mode", baseCmd)
		}

		if flags, ok := blockedFlags[baseCmd]; ok {
			tokens := Tokenize(seg)
			for _, tok := range tokens[1:] {
				for _, blocked := range flags {
					if tok == blocked || strings.HasPrefix(tok, blocked) {
						return fmt.Errorf("%s flag %q is not allowed in read-only mode", baseCmd, blocked)
					}
				}
			}
		}

		if restrictions, ok := subcommandRestrictions[baseCmd]; ok {
			subCmd := ExtractSubcommand(seg, baseCmd)
			if subCmd != "" && !restrictions[subCmd] {
				return fmt.Errorf("%s subcommand %q is not allowed in read-only mode (allowed: %s)",
					baseCmd, subCmd, JoinKeys(restrictions))
			}
		}

		if validator, ok := commandArgValidators[baseCmd]; ok {
			tokens := Tokenize(seg)
			if err := validator(tokens, allowed); err != nil {
				return err
			}
		}
	}

	return nil
}

// CheckDangerousMetacharacters detects shell expansion primitives.
func CheckDangerousMetacharacters(s string) error {
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
			if ch == '$' && i+1 < len(runes) && runes[i+1] == '(' {
				return fmt.Errorf("command substitution $(...) is not allowed in read-only mode")
			}
			if ch == '`' {
				return fmt.Errorf("backtick command substitution is not allowed in read-only mode")
			}
			if (ch == '<' || ch == '>') && i+1 < len(runes) && runes[i+1] == '(' {
				return fmt.Errorf("process substitution is not allowed in read-only mode")
			}
			if ch == '\n' || ch == '\r' {
				return fmt.Errorf("newline characters are not allowed in read-only mode")
			}
		}
		prev = ch
	}
	return nil
}

// ContainsUnquotedRedirection detects > or >> outside of quotes.
func ContainsUnquotedRedirection(s string) bool {
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
			return true
		}
		prev = ch
	}
	return false
}

// SplitPipeline splits a command string on unquoted pipe characters.
func SplitPipeline(s string) []string {
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
			if i+1 < len(runes) && runes[i+1] == '|' {
				segments = append(segments, current.String())
				current.Reset()
				i++
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
				i++
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

// ExtractBaseCommand returns the first actual command token from a segment,
// skipping leading environment variable assignments (VAR=value).
func ExtractBaseCommand(seg string) string {
	tokens := Tokenize(seg)
	for _, tok := range tokens {
		if envAssignRe.MatchString(tok) {
			continue
		}
		base := tok
		if idx := strings.LastIndex(tok, "/"); idx >= 0 {
			base = tok[idx+1:]
		}
		return base
	}
	return ""
}

// ExtractSubcommand returns the first argument after the base command.
func ExtractSubcommand(seg, baseCmd string) string {
	tokens := Tokenize(seg)
	foundBase := false
	for _, tok := range tokens {
		if !foundBase {
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

// Tokenize splits a command segment into shell-like tokens, respecting quotes.
func Tokenize(s string) []string {
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

// JoinKeys returns a comma-separated list of map keys.
func JoinKeys(m map[string]bool) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
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

// AllowedCommandsListMap returns the allowed commands map for external use.
func AllowedCommandsListMap() map[string]bool {
	result := make(map[string]bool)
	for k, v := range allowedCommands {
		result[k] = v
	}
	return result
}

// SubcommandRestrictions returns a map of commands to their allowed subcommands.
func SubcommandRestrictions() map[string]map[string]bool {
	result := make(map[string]map[string]bool)
	for k, v := range subcommandRestrictions {
		result[k] = v
	}
	return result
}

// ValidateCommandWithExtra checks that every command in a pipeline is allowed,
// using both the default allowlist and extra user-configured commands.
func ValidateCommandWithExtra(command string, extraAllowed []string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return fmt.Errorf("empty command")
	}

	if err := CheckDangerousMetacharacters(command); err != nil {
		return err
	}

	if ContainsUnquotedRedirection(command) {
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

	return validateCommand(command, merged)
}
