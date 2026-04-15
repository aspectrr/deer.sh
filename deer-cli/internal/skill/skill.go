package skill

import (
	"bufio"
	"bytes"
	"strings"
)

// Skill represents a loaded skill from a SKILL.md file.
type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version,omitempty"`
	Source      string `json:"source,omitempty"`
	DirPath     string `json:"dir_path"`
	Content     string `json:"-"`
}

// CatalogEntry is a lightweight representation for the skills catalog
// shown to the LLM in the system prompt.
type CatalogEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Parse parses a SKILL.md file with optional YAML frontmatter.
//
// Format:
//
//	---
//	name: elasticsearch-deploy
//	description: "Deploy and configure ES clusters"
//	version: 1.0.0
//	---
//	# Skill content here
func Parse(data []byte) (*Skill, error) {
	s := &Skill{}

	content := string(data)
	body := content

	if strings.HasPrefix(content, "---") {
		end := strings.Index(content[3:], "---")
		if end >= 0 {
			frontmatter := strings.TrimSpace(content[3 : end+3])
			body = strings.TrimSpace(content[end+6:])
			parseFrontmatter(frontmatter, s)
		}
	}

	s.Content = body

	if s.Name == "" {
		s.Name = "unnamed"
	}

	return s, nil
}

func parseFrontmatter(text string, s *Skill) {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := parseKV(line)
		if !ok {
			continue
		}
		switch key {
		case "name":
			s.Name = unquote(value)
		case "description":
			s.Description = unquote(value)
		case "version":
			s.Version = unquote(value)
		case "source":
			s.Source = unquote(value)
		}
	}
}

func parseKV(line string) (string, string, bool) {
	idx := bytes.IndexByte([]byte(line), ':')
	if idx < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	return key, value, true
}

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
