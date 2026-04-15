package skill

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed defaults/*/SKILL.md
var defaultSkills embed.FS

const (
	skillFile = "SKILL.md"
)

// Loader discovers and loads skills from filesystem directories and embedded defaults.
type Loader struct {
	dirs   []string
	mu     sync.RWMutex
	skills map[string]*Skill
}

// NewLoader creates a skill loader that scans the given directories.
// Bundled defaults are always loaded first; user directories override by name.
func NewLoader(dirs ...string) *Loader {
	return &Loader{
		dirs:   dirs,
		skills: make(map[string]*Skill),
	}
}

// Discover loads bundled defaults then scans configured directories.
// Returns the total number of skills loaded.
func (l *Loader) Discover() (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.loadDefaults()

	for _, dir := range l.dirs {
		if _, err := l.scanDir(dir); err != nil {
			return len(l.skills), fmt.Errorf("scan %s: %w", dir, err)
		}
	}
	return len(l.skills), nil
}

func (l *Loader) loadDefaults() {
	entries, err := fs.ReadDir(defaultSkills, "defaults")
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, err := fs.ReadFile(defaultSkills, filepath.Join("defaults", entry.Name(), skillFile))
		if err != nil {
			continue
		}
		s, err := Parse(data)
		if err != nil {
			continue
		}
		s.DirPath = "(builtin)"
		l.skills[strings.ToLower(s.Name)] = s
	}
}

func (l *Loader) scanDir(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	loaded := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(dir, entry.Name(), skillFile)
		data, err := os.ReadFile(skillPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return loaded, fmt.Errorf("read %s: %w", skillPath, err)
		}

		s, err := Parse(data)
		if err != nil {
			continue
		}
		s.DirPath = filepath.Join(dir, entry.Name())
		l.skills[strings.ToLower(s.Name)] = s
		loaded++
	}
	return loaded, nil
}

// List returns all discovered skills.
func (l *Loader) List() []*Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]*Skill, 0, len(l.skills))
	for _, s := range l.skills {
		result = append(result, s)
	}
	return result
}

// Catalog returns lightweight entries for system prompt injection.
func (l *Loader) Catalog() []CatalogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	entries := make([]CatalogEntry, 0, len(l.skills))
	for _, s := range l.skills {
		entries = append(entries, CatalogEntry{
			Name:        s.Name,
			Description: s.Description,
		})
	}
	return entries
}

// Get returns a skill by name (case-insensitive). Returns nil if not found.
func (l *Loader) Get(name string) *Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.skills[strings.ToLower(name)]
}

// Names returns all skill names.
func (l *Loader) Names() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	names := make([]string, 0, len(l.skills))
	for _, s := range l.skills {
		names = append(names, s.Name)
	}
	return names
}

// HasSkills returns true if any skills are loaded.
func (l *Loader) HasSkills() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.skills) > 0
}

// SkillsDir returns the path to the skills directory within the config dir.
func SkillsDir() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "skills"), nil
}

// EnsureSkillsDir creates the skills directory if it doesn't exist.
func EnsureSkillsDir() (string, error) {
	dir, err := SkillsDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create skills dir: %w", err)
	}
	return dir, nil
}

func configDir() (string, error) {
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg != "" {
		return filepath.Join(xdg, "deer"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home dir: %w", err)
	}
	return filepath.Join(home, ".config", "deer"), nil
}
