package paths

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// MaybeMigrate checks for the legacy ~/.fluid directory and copies files
// to the new XDG locations if the new config directory does not yet exist.
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

	// If the new config dir already exists, assume migration was done
	if _, err := os.Stat(configDir); err == nil {
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

	// Config dir files
	configFiles := []string{"config.yaml"}
	configDirs := []string{"ssh-ca", "sandbox-keys", "ansible"}

	for _, name := range configFiles {
		_ = copyFile(filepath.Join(oldDir, name), filepath.Join(configDir, name))
	}
	for _, name := range configDirs {
		_ = copyDir(filepath.Join(oldDir, name), filepath.Join(configDir, name))
	}

	// Data dir files
	dataFiles := []string{"state.db", "history"}
	for _, name := range dataFiles {
		_ = copyFile(filepath.Join(oldDir, name), filepath.Join(dataDir, name))
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

	_, err = io.Copy(out, in)
	return err
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
