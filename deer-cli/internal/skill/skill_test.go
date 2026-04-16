package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func bundledDefaultCount() int {
	loader := NewLoader()
	count, _ := loader.Discover()
	return count
}

func TestParseWithFrontmatter(t *testing.T) {
	data := []byte(`---
name: elasticsearch-deploy
description: "Deploy and configure ES clusters"
version: 1.0.0
source: github.com/example/es-skill
---
# Elasticsearch Deploy Skill

## When to Activate
- User asks to deploy Elasticsearch

## Instructions
Do the thing.
`)

	s, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if s.Name != "elasticsearch-deploy" {
		t.Errorf("Name = %q, want %q", s.Name, "elasticsearch-deploy")
	}
	if s.Description != "Deploy and configure ES clusters" {
		t.Errorf("Description = %q, want %q", s.Description, "Deploy and configure ES clusters")
	}
	if s.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", s.Version, "1.0.0")
	}
	if s.Source != "github.com/example/es-skill" {
		t.Errorf("Source = %q, want %q", s.Source, "github.com/example/es-skill")
	}
	if s.Content == "" {
		t.Error("Content should not be empty")
	}
	wantBody := "# Elasticsearch Deploy Skill"
	if s.Content[:len(wantBody)] != wantBody {
		t.Errorf("Content starts with %q, want %q", s.Content[:len(wantBody)], wantBody)
	}
}

func TestParseWithoutFrontmatter(t *testing.T) {
	data := []byte(`# My Skill

Some content here.
`)

	s, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if s.Name != "unnamed" {
		t.Errorf("Name = %q, want %q", s.Name, "unnamed")
	}
	if s.Description != "" {
		t.Errorf("Description should be empty, got %q", s.Description)
	}
	if s.Content == "" {
		t.Error("Content should not be empty")
	}
}

func TestParseMinimalFrontmatter(t *testing.T) {
	data := []byte(`---
name: kafka
description: Kafka management
---
Content here.
`)

	s, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if s.Name != "kafka" {
		t.Errorf("Name = %q, want %q", s.Name, "kafka")
	}
	if s.Description != "Kafka management" {
		t.Errorf("Description = %q, want %q", s.Description, "Kafka management")
	}
}

func TestLoaderDiscover(t *testing.T) {
	dir := t.TempDir()

	esDir := filepath.Join(dir, "elasticsearch-deploy")
	if err := os.MkdirAll(esDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(esDir, "SKILL.md"), []byte(`---
name: elasticsearch-deploy
description: "Deploy ES clusters"
---
ES content.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	kafkaDir := filepath.Join(dir, "kafka-ops")
	if err := os.MkdirAll(kafkaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(kafkaDir, "SKILL.md"), []byte(`---
name: kafka-ops
description: "Kafka operations"
---
Kafka content.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(dir)
	count, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	defaultCount := bundledDefaultCount()
	expectedWithDefaults := 2 + defaultCount
	if count != expectedWithDefaults {
		t.Errorf("count = %d, want %d (2 user + %d defaults)", count, expectedWithDefaults, defaultCount)
	}

	skills := loader.List()
	if len(skills) != expectedWithDefaults {
		t.Errorf("List() = %d skills, want %d", len(skills), expectedWithDefaults)
	}

	es := loader.Get("elasticsearch-deploy")
	if es == nil {
		t.Fatal("Get(elasticsearch-deploy) = nil")
	}
	if es.Content != "ES content." {
		t.Errorf("Content = %q, want %q", es.Content, "ES content.")
	}

	kafka := loader.Get("kafka-ops")
	if kafka == nil {
		t.Fatal("Get(kafka-ops) = nil")
	}

	if loader.Get("nonexistent") != nil {
		t.Error("Get(nonexistent) should be nil")
	}
}

func TestLoaderCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "my-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: My-Skill
description: "Test"
---
Content.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(dir)
	if _, err := loader.Discover(); err != nil {
		t.Fatal(err)
	}

	if loader.Get("my-skill") == nil {
		t.Error("Get(my-skill) should find My-Skill (case-insensitive)")
	}
	if loader.Get("MY-SKILL") == nil {
		t.Error("Get(MY-SKILL) should find My-Skill (case-insensitive)")
	}
}

func TestLoaderMultipleDirs(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	if err := os.MkdirAll(filepath.Join(dir1, "skill-a"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir1, "skill-a", "SKILL.md"), []byte(`---
name: skill-a
description: "From dir1"
---
A content.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(dir2, "skill-b"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "skill-b", "SKILL.md"), []byte(`---
name: skill-b
description: "From dir2"
---
B content.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(dir1, dir2)
	count, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	defaultCount := bundledDefaultCount()
	expectedWithDefaults := 2 + defaultCount
	if count != expectedWithDefaults {
		t.Errorf("count = %d, want %d (2 user + %d defaults)", count, expectedWithDefaults, defaultCount)
	}
}

func TestLoaderNonexistentDir(t *testing.T) {
	loader := NewLoader("/nonexistent/path")
	count, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover should not error on nonexistent dir: %v", err)
	}
	defaultCount := bundledDefaultCount()
	if count != defaultCount {
		t.Errorf("count = %d, want %d (bundled defaults)", count, defaultCount)
	}
	if !loader.HasSkills() {
		t.Error("HasSkills() should be true even with only defaults")
	}
}

func TestLoaderCatalog(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "es"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "es", "SKILL.md"), []byte(`---
name: elasticsearch
description: "ES operations"
---
Content.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(dir)
	if _, err := loader.Discover(); err != nil {
		t.Fatal(err)
	}

	catalog := loader.Catalog()
	defaultCount := bundledDefaultCount()
	expectedCatalog := 1 + defaultCount
	if len(catalog) != expectedCatalog {
		t.Fatalf("Catalog() = %d entries, want %d (1 user + %d defaults)", len(catalog), expectedCatalog, defaultCount)
	}
	// Find the user skill in catalog
	found := false
	for _, e := range catalog {
		if e.Name == "elasticsearch" {
			found = true
			if e.Description != "ES operations" {
				t.Errorf("Description = %q, want %q", e.Description, "ES operations")
			}
		}
	}
	if !found {
		t.Error("elasticsearch skill not found in catalog")
	}
}

func TestLoaderNames(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a", "SKILL.md"), []byte(`---
name: alpha
description: "A"
---
A.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(dir)
	if _, err := loader.Discover(); err != nil {
		t.Fatal(err)
	}

	names := loader.Names()
	defaultCount := bundledDefaultCount()
	expectedNames := 1 + defaultCount
	if len(names) != expectedNames {
		t.Errorf("Names() = %d entries, want %d", len(names), expectedNames)
	}
	found := false
	for _, n := range names {
		if n == "alpha" {
			found = true
		}
	}
	if !found {
		t.Errorf("alpha not found in Names() = %v", names)
	}
}

func TestLoaderHasSkills(t *testing.T) {
	// Even with no user dirs, defaults are loaded
	loader := NewLoader("/nonexistent")
	if _, err := loader.Discover(); err != nil {
		t.Fatal(err)
	}
	if !loader.HasSkills() {
		t.Error("HasSkills() should be true (bundled defaults always loaded)")
	}

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "x", "SKILL.md"), []byte(`---
name: x
description: "X"
---
X.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	loader2 := NewLoader(dir)
	if _, err := loader2.Discover(); err != nil {
		t.Fatal(err)
	}
	if !loader2.HasSkills() {
		t.Error("HasSkills() should be true with skills loaded")
	}
}

func TestLoaderSkipNonDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("not a skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(dir)
	count, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	defaultCount := bundledDefaultCount()
	if count != defaultCount {
		t.Errorf("count = %d, want %d (bundled defaults only)", count, defaultCount)
	}
}

func TestLoaderSkipDirWithoutSkillFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "no-skill-file"), 0o755); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(dir)
	count, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	defaultCount := bundledDefaultCount()
	if count != defaultCount {
		t.Errorf("count = %d, want %d (bundled defaults only)", count, defaultCount)
	}
}

func TestBundledDefaults(t *testing.T) {
	loader := NewLoader()
	count, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	defaultCount := bundledDefaultCount()
	if count < defaultCount {
		t.Errorf("expected at least %d bundled defaults, got %d", defaultCount, count)
	}

	kafka := loader.Get("kafka")
	if kafka == nil {
		t.Fatal("bundled kafka skill should always be present")
	}
	if kafka.DirPath != "(builtin)" {
		t.Errorf("DirPath = %q, want (builtin)", kafka.DirPath)
	}

	logAgg := loader.Get("log-aggregation")
	if logAgg == nil {
		t.Fatal("bundled log-aggregation skill should always be present")
	}
	if logAgg.DirPath != "(builtin)" {
		t.Errorf("DirPath = %q, want (builtin)", logAgg.DirPath)
	}
}

func TestUserOverridesDefault(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "kafka"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "kafka", "SKILL.md"), []byte(`---
name: kafka
description: "Custom kafka override"
---
Custom kafka content.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(dir)
	if _, err := loader.Discover(); err != nil {
		t.Fatal(err)
	}

	kafka := loader.Get("kafka")
	if kafka == nil {
		t.Fatal("kafka skill should exist")
	}
	if kafka.Description != "Custom kafka override" {
		t.Errorf("Description = %q, user-installed should override default", kafka.Description)
	}
	if kafka.Content != "Custom kafka content." {
		t.Errorf("Content = %q, want custom content from user override", kafka.Content)
	}
}
