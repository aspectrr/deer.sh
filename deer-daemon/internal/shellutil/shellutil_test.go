package shellutil

import "testing"

func TestQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "'hello'"},
		{"", "''"},
		{"path with spaces", "'path with spaces'"},
		{"it's", "'it'\\''s'"},
		{"'already quoted'", "''\\''already quoted'\\'''"},
		{"$(whoami)", "'$(whoami)'"},
		{"`id`", "'`id`'"},
		{"foo;bar", "'foo;bar'"},
		{"a\nb", "'a\nb'"},
		{"/home/user/.ssh/id_rsa", "'/home/user/.ssh/id_rsa'"},
		{"user@host", "'user@host'"},
		{"hello'world'test", "'hello'\\''world'\\''test'"},
	}

	for _, tt := range tests {
		got := Quote(tt.input)
		if got != tt.expected {
			t.Errorf("Quote(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
