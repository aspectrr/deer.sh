package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ConfigDir returns the fluid configuration directory.
//
// Resolution order:
//  1. $XDG_CONFIG_HOME/fluid (if set)
//  2. os.UserConfigDir()/fluid (Windows)
//  3. ~/.config/fluid (macOS, Linux)
func ConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "fluid"), nil
	}
	if runtime.GOOS == "windows" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("paths: config dir: %w", err)
		}
		return filepath.Join(dir, "fluid"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("paths: config dir: %w", err)
	}
	return filepath.Join(home, ".config", "fluid"), nil
}

// DataDir returns the fluid data directory for state, history, and logs.
//
// Resolution order:
//  1. $XDG_DATA_HOME/fluid (if set)
//  2. %LOCALAPPDATA%/fluid (Windows)
//  3. ~/.local/share/fluid (macOS, Linux)
func DataDir() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "fluid"), nil
	}
	if runtime.GOOS == "windows" {
		dir := os.Getenv("LOCALAPPDATA")
		if dir == "" {
			var err error
			dir, err = os.UserCacheDir()
			if err != nil {
				return "", fmt.Errorf("paths: data dir: %w", err)
			}
		}
		return filepath.Join(dir, "fluid"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("paths: data dir: %w", err)
	}
	return filepath.Join(home, ".local", "share", "fluid"), nil
}

// ConfigFile returns the path to the config.yaml file.
func ConfigFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// StateDB returns the path to the SQLite state database.
func StateDB() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.db"), nil
}
