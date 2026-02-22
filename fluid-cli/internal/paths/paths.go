package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

// ConfigDir returns the fluid configuration directory.
//
// Resolution order:
//  1. $XDG_CONFIG_HOME/fluid (if set)
//  2. %AppData%/fluid (Windows)
//  3. ~/.config/fluid (macOS, Linux)
func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "fluid")
	}
	if runtime.GOOS == "windows" {
		if dir, err := os.UserConfigDir(); err == nil {
			return filepath.Join(dir, "fluid")
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "fluid")
}

// ConfigFile returns the path to the config.yaml file.
func ConfigFile() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

// StateDB returns the path to the SQLite state database.
func StateDB() string {
	return filepath.Join(ConfigDir(), "state.db")
}
