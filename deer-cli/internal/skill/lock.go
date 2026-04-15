package skill

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const lockFile = "skills-lock.json"

// LockFile represents the skills-lock.json that tracks installed skills.
type LockFile struct {
	Version int                  `json:"version"`
	Skills  map[string]LockEntry `json:"skills"`
}

// LockEntry tracks a single installed skill.
type LockEntry struct {
	Source       string `json:"source"`
	SourceType   string `json:"source_type"`
	ComputedHash string `json:"computed_hash,omitempty"`
}

// LoadLock reads the skills-lock.json from the config directory.
func LoadLock() (*LockFile, error) {
	path, err := lockPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &LockFile{Version: 1, Skills: make(map[string]LockEntry)}, nil
		}
		return nil, fmt.Errorf("read skills lock: %w", err)
	}

	var lf LockFile
	if err := json.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("parse skills lock: %w", err)
	}
	if lf.Skills == nil {
		lf.Skills = make(map[string]LockEntry)
	}
	return &lf, nil
}

// Save writes the lock file to disk.
func (lf *LockFile) Save() error {
	path, err := lockPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create lock dir: %w", err)
	}

	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal lock: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// Add adds or updates a skill entry in the lock file.
func (lf *LockFile) Add(name string, entry LockEntry) {
	lf.Skills[name] = entry
}

// Remove removes a skill entry from the lock file.
func (lf *LockFile) Remove(name string) bool {
	if _, ok := lf.Skills[name]; ok {
		delete(lf.Skills, name)
		return true
	}
	return false
}

func lockPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, lockFile), nil
}
