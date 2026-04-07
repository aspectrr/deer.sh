// Package redact provides PII/sensitive data redaction.
//
// NOTE: The CLI has a parallel copy at deer-cli/internal/redact/ with
// additional detectors (config values, custom patterns). Changes to
// shared detectors should be mirrored in both locations.
package redact

import (
	"regexp"
	"strconv"
	"strings"
)

// PatternDetector finds sensitive data in text.
type PatternDetector interface {
	Name() string
	Category() string
	FindAll(text string) []Match
}

// Match represents a detected sensitive value and its position.
type Match struct {
	Value string
	Start int
	End   int
}

// -------------------------------------------------------------------
// IPv4
// -------------------------------------------------------------------

type ipv4Detector struct{}

func (d *ipv4Detector) Name() string     { return "ipv4" }
func (d *ipv4Detector) Category() string { return "IP" }

var ipv4Re = regexp.MustCompile(`\b(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})\b`)

func (d *ipv4Detector) FindAll(text string) []Match {
	var matches []Match
	for _, loc := range ipv4Re.FindAllStringSubmatchIndex(text, -1) {
		full := text[loc[0]:loc[1]]
		parts := strings.Split(full, ".")
		if len(parts) != 4 {
			continue
		}
		valid := true
		allSmall := true
		for _, p := range parts {
			n, err := strconv.Atoi(p)
			if err != nil || n < 0 || n > 255 {
				valid = false
				break
			}
			if n > 3 {
				allSmall = false
			}
		}
		if !valid {
			continue
		}
		// Skip version-number-like strings where all octets <= 3.
		if allSmall {
			continue
		}
		matches = append(matches, Match{Value: full, Start: loc[0], End: loc[1]})
	}
	return matches
}

// -------------------------------------------------------------------
// IPv6
// -------------------------------------------------------------------

type ipv6Detector struct{}

func (d *ipv6Detector) Name() string     { return "ipv6" }
func (d *ipv6Detector) Category() string { return "IP" }

var ipv6Re = regexp.MustCompile(
	`(?i)\b(?:` +
		`(?:[0-9a-f]{1,4}:){7}[0-9a-f]{1,4}` +
		`|(?:[0-9a-f]{1,4}:){1,7}:` +
		`|(?:[0-9a-f]{1,4}:){1,6}:[0-9a-f]{1,4}` +
		`|(?:[0-9a-f]{1,4}:){1,5}(?::[0-9a-f]{1,4}){1,2}` +
		`|(?:[0-9a-f]{1,4}:){1,4}(?::[0-9a-f]{1,4}){1,3}` +
		`|(?:[0-9a-f]{1,4}:){1,3}(?::[0-9a-f]{1,4}){1,4}` +
		`|(?:[0-9a-f]{1,4}:){1,2}(?::[0-9a-f]{1,4}){1,5}` +
		`|[0-9a-f]{1,4}:(?::[0-9a-f]{1,4}){1,6}` +
		`|:(?::[0-9a-f]{1,4}){1,7}` +
		`|::` +
		`)\b`,
)

func (d *ipv6Detector) FindAll(text string) []Match {
	var matches []Match
	for _, loc := range ipv6Re.FindAllStringIndex(text, -1) {
		val := text[loc[0]:loc[1]]
		if !strings.Contains(val, ":") {
			continue
		}
		if val == "::" {
			continue
		}
		matches = append(matches, Match{Value: val, Start: loc[0], End: loc[1]})
	}
	return matches
}

// -------------------------------------------------------------------
// API keys (sk-..., key-..., Bearer tokens)
// -------------------------------------------------------------------

type apiKeyDetector struct{}

func (d *apiKeyDetector) Name() string     { return "api_key" }
func (d *apiKeyDetector) Category() string { return "KEY" }

var apiKeyPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\bsk-[a-zA-Z0-9]{20,}\b`),
	regexp.MustCompile(`\bkey-[a-zA-Z0-9]{20,}\b`),
	regexp.MustCompile(`\bBearer\s+[A-Za-z0-9\-._~+/]+=*\b`),
}

func (d *apiKeyDetector) FindAll(text string) []Match {
	var matches []Match
	for _, re := range apiKeyPatterns {
		for _, loc := range re.FindAllStringIndex(text, -1) {
			matches = append(matches, Match{Value: text[loc[0]:loc[1]], Start: loc[0], End: loc[1]})
		}
	}
	return matches
}

// -------------------------------------------------------------------
// AWS access keys
// -------------------------------------------------------------------

type awsKeyDetector struct{}

func (d *awsKeyDetector) Name() string     { return "aws_key" }
func (d *awsKeyDetector) Category() string { return "KEY" }

var awsKeyRe = regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)

func (d *awsKeyDetector) FindAll(text string) []Match {
	var matches []Match
	for _, loc := range awsKeyRe.FindAllStringIndex(text, -1) {
		matches = append(matches, Match{Value: text[loc[0]:loc[1]], Start: loc[0], End: loc[1]})
	}
	return matches
}

// -------------------------------------------------------------------
// SSH private key blocks
// -------------------------------------------------------------------

type sshPrivateKeyDetector struct{}

func (d *sshPrivateKeyDetector) Name() string     { return "ssh_private_key" }
func (d *sshPrivateKeyDetector) Category() string { return "KEY" }

var sshKeyRe = regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`)

func (d *sshPrivateKeyDetector) FindAll(text string) []Match {
	var matches []Match
	for _, loc := range sshKeyRe.FindAllStringIndex(text, -1) {
		matches = append(matches, Match{Value: text[loc[0]:loc[1]], Start: loc[0], End: loc[1]})
	}
	return matches
}

// -------------------------------------------------------------------
// Connection strings
// -------------------------------------------------------------------

type connectionStringDetector struct{}

func (d *connectionStringDetector) Name() string     { return "connection_string" }
func (d *connectionStringDetector) Category() string { return "SECRET" }

var connStrRe = regexp.MustCompile(`\b(postgres|mysql|mongodb|redis)://[^\s]+`)

func (d *connectionStringDetector) FindAll(text string) []Match {
	var matches []Match
	for _, loc := range connStrRe.FindAllStringIndex(text, -1) {
		matches = append(matches, Match{Value: text[loc[0]:loc[1]], Start: loc[0], End: loc[1]})
	}
	return matches
}

// defaultDetectors returns the built-in pattern detectors.
func defaultDetectors() []PatternDetector {
	return []PatternDetector{
		&sshPrivateKeyDetector{},
		&connectionStringDetector{},
		&awsKeyDetector{},
		&apiKeyDetector{},
		&ipv6Detector{},
		&ipv4Detector{},
	}
}
