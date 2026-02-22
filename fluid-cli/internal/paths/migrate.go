package paths

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// sentinelName is the file written to the config dir after a successful migration.
const sentinelName = ".migrated-from-dot-fluid"

// MaybeMigrate checks for the legacy ~/.fluid directory and copies files
// to the new XDG locations if migration has not already completed.
// It does NOT delete ~/.fluid - the user can do that manually.
func MaybeMigrate() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil // can't determine home, skip migration
	}

	oldDir := filepath.Join(home, ".fluid")
	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		return nil // no legacy dir, nothing to migrate
	}

	configDir, err := ConfigDir()
	if err != nil {
		return nil // can't determine new dir, skip
	}

	// If the sentinel file exists, migration was already completed successfully
	if _, err := os.Stat(filepath.Join(configDir, sentinelName)); err == nil {
		return nil
	}

	dataDir, err := DataDir()
	if err != nil {
		return nil
	}

	// Create new directories
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("migrate: create config dir: %w", err)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("migrate: create data dir: %w", err)
	}

	// Collect errors from individual copies
	var copyErrors []error

	// Config dir files
	configFiles := []string{"config.yaml"}
	configDirs := []string{"ssh-ca", "sandbox-keys", "ansible"}

	for _, name := range configFiles {
		if err := copyFile(filepath.Join(oldDir, name), filepath.Join(configDir, name)); err != nil && !os.IsNotExist(err) {
			copyErrors = append(copyErrors, fmt.Errorf("copy %s: %w", name, err))
		}
	}
	for _, name := range configDirs {
		if err := copyDir(filepath.Join(oldDir, name), filepath.Join(configDir, name)); err != nil && !os.IsNotExist(err) {
			copyErrors = append(copyErrors, fmt.Errorf("copy %s/: %w", name, err))
		}
	}

	// Data dir files
	dataFiles := []string{"state.db", "state.db-wal", "state.db-shm", "history"}
	for _, name := range dataFiles {
		if err := copyFile(filepath.Join(oldDir, name), filepath.Join(dataDir, name)); err != nil && !os.IsNotExist(err) {
			copyErrors = append(copyErrors, fmt.Errorf("copy %s: %w", name, err))
		}
	}

	if len(copyErrors) > 0 {
		for _, e := range copyErrors {
			fmt.Fprintf(os.Stderr, "Warning: migration: %v\n", e)
		}
		return fmt.Errorf("migrate: %d file(s) failed to copy", len(copyErrors))
	}

	// All copies succeeded - write sentinel
	if err := os.WriteFile(filepath.Join(configDir, sentinelName), nil, 0o600); err != nil {
		return fmt.Errorf("migrate: write sentinel: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Migrated config from ~/.fluid to %s and %s\n", configDir, dataDir)
	fmt.Fprintf(os.Stderr, "You can safely remove ~/.fluid after verifying the migration.\n")
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}

	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}
