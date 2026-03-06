// Package redact provides PII/sensitive data redaction.
//
// NOTE: The daemon has a parallel copy at fluid-daemon/internal/redact/.
// This CLI copy includes additional detectors (config values, custom patterns).
// Changes to shared detectors should be mirrored in both locations.
package redact

import (
	"fmt"
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
		// Skip version-number-like strings where all octets <= 3 (e.g.
		// "1.2.3.4", "2.0.0.1"). Real public IPs like 8.8.8.8 have at
		// least one octet > 3 and are still redacted. Note: 1.1.1.1 has
		// all octets <= 3, so it is skipped by this heuristic.
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

// Matches full and compressed IPv6 addresses. This pattern covers the most
// common representations including :: shorthand.
var ipv6Re = regexp.MustCompile(
	`(?i)\b(?:` +
		// Full 8-group form or with :: compression
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
		// Must contain at least one colon to be an IPv6 address.
		if !strings.Contains(val, ":") {
			continue
		}
		// Skip the trivial "::" standalone unless it is clearly used as an address.
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

// -------------------------------------------------------------------
// Config value detector: matches injected known values from config.
// -------------------------------------------------------------------

type configValueDetector struct {
	entries []configEntry
}

type configEntry struct {
	value    string
	category string
}

func (d *configValueDetector) Name() string     { return "config_value" }
func (d *configValueDetector) Category() string { return "HOST" } // default; actual category comes from entry

func (d *configValueDetector) FindAll(text string) []Match {
	var matches []Match
	for _, e := range d.entries {
		if e.value == "" {
			continue
		}
		idx := 0
		for {
			pos := strings.Index(text[idx:], e.value)
			if pos < 0 {
				break
			}
			start := idx + pos
			end := start + len(e.value)
			matches = append(matches, Match{Value: e.value, Start: start, End: end})
			idx = end
		}
	}
	return matches
}

// categoryForConfigEntry returns the category for a config match.
func (d *configValueDetector) categoryForValue(value string) string {
	for _, e := range d.entries {
		if e.value == value {
			return e.category
		}
	}
	return "HOST"
}

// -------------------------------------------------------------------
// Custom regex pattern detector
// -------------------------------------------------------------------

type regexDetector struct {
	name     string
	category string
	re       *regexp.Regexp
}

func (d *regexDetector) Name() string     { return d.name }
func (d *regexDetector) Category() string { return d.category }

func (d *regexDetector) FindAll(text string) []Match {
	var matches []Match
	for _, loc := range d.re.FindAllStringIndex(text, -1) {
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

// newConfigValueDetector creates a detector from known config values.
func newConfigValueDetector(hosts, addresses, keyPaths []string) *configValueDetector {
	d := &configValueDetector{}
	for _, h := range hosts {
		if h != "" {
			d.entries = append(d.entries, configEntry{value: h, category: "HOST"})
		}
	}
	for _, a := range addresses {
		if a != "" {
			d.entries = append(d.entries, configEntry{value: a, category: "IP"})
		}
	}
	for _, k := range keyPaths {
		if k != "" {
			d.entries = append(d.entries, configEntry{value: k, category: "PATH"})
		}
	}
	return d
}

// newCustomPatternDetectors creates detectors from user-supplied regexes.
func newCustomPatternDetectors(patterns []string) []PatternDetector {
	var detectors []PatternDetector
	for i, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			continue
		}
		detectors = append(detectors, &regexDetector{
			name:     fmt.Sprintf("custom_%d", i),
			category: "SECRET",
			re:       re,
		})
	}
	return detectors
}
