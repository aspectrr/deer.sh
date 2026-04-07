package redact

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// RedactionStats holds aggregate redaction counts.
type RedactionStats struct {
	Total      int
	ByCategory map[string]int
}

// Redactor replaces sensitive values with deterministic tokens and can restore them.
// Note: the mapping and reverse maps grow unboundedly for the lifetime of the
// Redactor. For long-running sessions with many unique sensitive values this is
// a memory trade-off - acceptable for CLI session lifetime.
type Redactor struct {
	mapping   map[string]string // token -> original value
	reverse   map[string]string // original value -> token
	counters  map[string]int    // per-category counters
	patterns  []PatternDetector
	allowlist map[string]bool
	mu        sync.Mutex
}

// Option configures a Redactor.
type Option func(*Redactor)

// WithAllowlist adds values that should never be redacted.
func WithAllowlist(values []string) Option {
	return func(r *Redactor) {
		for _, v := range values {
			r.allowlist[v] = true
		}
	}
}

// WithConfigValues injects known configuration values (hostnames, addresses, key paths)
// so they are detected and redacted.
func WithConfigValues(hosts, addresses, keyPaths []string) Option {
	return func(r *Redactor) {
		d := newConfigValueDetector(hosts, addresses, keyPaths)
		if len(d.entries) > 0 {
			r.patterns = append(r.patterns, d)
		}
	}
}

// WithCustomPatterns adds additional regex patterns that produce SECRET-category tokens.
func WithCustomPatterns(patterns []string) Option {
	return func(r *Redactor) {
		r.patterns = append(r.patterns, newCustomPatternDetectors(patterns)...)
	}
}

// New creates a Redactor with the default built-in detectors plus any supplied options.
func New(opts ...Option) *Redactor {
	r := &Redactor{
		mapping:   make(map[string]string),
		reverse:   make(map[string]string),
		counters:  make(map[string]int),
		patterns:  defaultDetectors(),
		allowlist: make(map[string]bool),
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Redact replaces all detected sensitive values in text with deterministic tokens.
// The same original value always maps to the same token within the Redactor's lifetime.
func (r *Redactor) Redact(text string) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Collect all matches from every detector.
	type catMatch struct {
		Match
		category string
	}
	var all []catMatch
	for _, p := range r.patterns {
		for _, m := range p.FindAll(text) {
			cat := p.Category()
			// For configValueDetector, use per-entry category.
			if cvd, ok := p.(*configValueDetector); ok {
				cat = cvd.categoryForValue(m.Value)
			}
			all = append(all, catMatch{Match: m, category: cat})
		}
	}
	if len(all) == 0 {
		return text
	}

	// Sort by start position, then by length descending (longest first).
	sort.Slice(all, func(i, j int) bool {
		if all[i].Start != all[j].Start {
			return all[i].Start < all[j].Start
		}
		return len(all[i].Value) > len(all[j].Value)
	})

	// Deduplicate overlapping matches: longest match wins.
	var deduped []catMatch
	end := -1
	for _, m := range all {
		if m.Start < end {
			// This match overlaps with the previous winning match.
			// If it starts at the same position the longer one is already first
			// (sorted above), so skip. If it starts inside the previous match, skip.
			continue
		}
		deduped = append(deduped, m)
		end = m.End
	}

	// Build result by replacing matches back-to-front to preserve offsets.
	result := text
	for i := len(deduped) - 1; i >= 0; i-- {
		m := deduped[i]
		if r.allowlist[m.Value] {
			continue
		}
		token := r.tokenFor(m.Value, m.category)
		result = result[:m.Start] + token + result[m.End:]
	}
	return result
}

// Restore replaces all tokens in text with their original values.
func (r *Redactor) Restore(text string) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := text
	// Replace longest tokens first to avoid partial replacement issues.
	type kv struct {
		token string
		orig  string
	}
	var pairs []kv
	for token, orig := range r.mapping {
		pairs = append(pairs, kv{token, orig})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return len(pairs[i].token) > len(pairs[j].token)
	})
	for _, p := range pairs {
		result = strings.ReplaceAll(result, p.token, p.orig)
	}
	return result
}

// Stats returns redaction counts.
func (r *Redactor) Stats() RedactionStats {
	r.mu.Lock()
	defer r.mu.Unlock()

	s := RedactionStats{
		ByCategory: make(map[string]int),
	}
	for cat, n := range r.counters {
		s.ByCategory[cat] = n
		s.Total += n
	}
	return s
}

// RedactMap recursively redacts all string values in a map.
func (r *Redactor) RedactMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = r.RedactAny(v)
	}
	return out
}

// RedactAny redacts the value if it is a string, map, or slice.
// Other types are passed through unchanged.
func (r *Redactor) RedactAny(v any) any {
	return r.redactValue(v)
}

// redactValue recursively dispatches redaction by type.
func (r *Redactor) redactValue(v any) any {
	switch val := v.(type) {
	case string:
		return r.Redact(val)
	case map[string]any:
		return r.RedactMap(val)
	case []any:
		out := make([]any, len(val))
		for i, elem := range val {
			out[i] = r.redactValue(elem)
		}
		return out
	default:
		return v
	}
}

// tokenFor returns the token for a value, creating a new one if needed.
// Must be called with r.mu held.
func (r *Redactor) tokenFor(value, category string) string {
	if token, ok := r.reverse[value]; ok {
		return token
	}
	r.counters[category]++
	n := r.counters[category]
	token := fmt.Sprintf("[REDACTED_%s_%d]", category, n)
	r.mapping[token] = value
	r.reverse[value] = token
	return token
}
