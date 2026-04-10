package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ConfigDir returns the deer configuration directory.
//
// Resolution order:
//  1. $XDG_CONFIG_HOME/deer (if set)
//  2. os.UserConfigDir()/deer (Windows)
//  3. ~/.config/deer (macOS, Linux)
func ConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "deer"), nil
	}
	if runtime.GOOS == "windows" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("paths: config dir: %w", err)
		}
		return filepath.Join(dir, "deer"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("paths: config dir: %w", err)
	}
	return filepath.Join(home, ".config", "deer"), nil
}

// DataDir returns the deer data directory for state, history, and logs.
//
// Resolution order:
//  1. $XDG_DATA_HOME/deer (if set)
//  2. %LOCALAPPDATA%/deer (Windows)
//  3. ~/.local/share/deer (macOS, Linux)
func DataDir() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "deer"), nil
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
		return filepath.Join(dir, "deer"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("paths: data dir: %w", err)
	}
	return filepath.Join(home, ".local", "share", "deer"), nil
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
