package paths

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigDir_XDGOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(tmp, "fluid")
	if dir != want {
		t.Errorf("ConfigDir() = %q, want %q", dir, want)
	}
}

func TestConfigDir_Default(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(dir, filepath.Join(".config", "fluid")) {
		t.Errorf("ConfigDir() = %q, want suffix %q", dir, filepath.Join(".config", "fluid"))
	}
}

func TestDataDir_XDGOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	dir, err := DataDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(tmp, "fluid")
	if dir != want {
		t.Errorf("DataDir() = %q, want %q", dir, want)
	}
}

func TestDataDir_Default(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")

	dir, err := DataDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(dir, filepath.Join(".local", "share", "fluid")) {
		t.Errorf("DataDir() = %q, want suffix %q", dir, filepath.Join(".local", "share", "fluid"))
	}
}

func TestConfigFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	f, err := ConfigFile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(f, "config.yaml") {
		t.Errorf("ConfigFile() = %q, want suffix config.yaml", f)
	}
}

func TestStateDB(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // separate from data

	f, err := StateDB()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(f, "state.db") {
		t.Errorf("StateDB() = %q, want suffix state.db", f)
	}

	dataDir, _ := DataDir()
	if !strings.HasPrefix(f, dataDir) {
		t.Errorf("StateDB() = %q, should be under DataDir %q", f, dataDir)
	}
}

func TestMaybeMigrate_OldDirExists(t *testing.T) {
	// Set up fake home with legacy ~/.fluid
	fakeHome := t.TempDir()
	oldDir := filepath.Join(fakeHome, ".fluid")
	if err := os.MkdirAll(filepath.Join(oldDir, "ssh-ca"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "config.yaml"), []byte("test: true"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "state.db"), []byte("sqlite"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "history"), []byte("cmd1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "ssh-ca", "ssh-ca"), []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Point XDG dirs to temp locations under fakeHome
	configBase := filepath.Join(fakeHome, "xdg-config")
	dataBase := filepath.Join(fakeHome, "xdg-data")
	t.Setenv("XDG_CONFIG_HOME", configBase)
	t.Setenv("XDG_DATA_HOME", dataBase)
	t.Setenv("HOME", fakeHome)

	if err := MaybeMigrate(); err != nil {
		t.Fatalf("MaybeMigrate() error: %v", err)
	}

	// Verify config files migrated
	if _, err := os.Stat(filepath.Join(configBase, "fluid", "config.yaml")); err != nil {
		t.Errorf("config.yaml not migrated: %v", err)
	}
	if _, err := os.Stat(filepath.Join(configBase, "fluid", "ssh-ca", "ssh-ca")); err != nil {
		t.Errorf("ssh-ca not migrated: %v", err)
	}

	// Verify data files migrated
	if _, err := os.Stat(filepath.Join(dataBase, "fluid", "state.db")); err != nil {
		t.Errorf("state.db not migrated: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataBase, "fluid", "history")); err != nil {
		t.Errorf("history not migrated: %v", err)
	}

	// Sentinel file should exist
	if _, err := os.Stat(filepath.Join(configBase, "fluid", sentinelName)); err != nil {
		t.Errorf("sentinel file not created: %v", err)
	}

	// Old dir should still exist
	if _, err := os.Stat(oldDir); err != nil {
		t.Errorf("old dir should not be deleted: %v", err)
	}
}

func TestMaybeMigrate_AlreadyMigrated(t *testing.T) {
	fakeHome := t.TempDir()
	oldDir := filepath.Join(fakeHome, ".fluid")
	if err := os.MkdirAll(oldDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "config.yaml"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	configBase := filepath.Join(fakeHome, "xdg-config")
	newConfigDir := filepath.Join(configBase, "fluid")
	if err := os.MkdirAll(newConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write sentinel to indicate migration already happened
	if err := os.WriteFile(filepath.Join(newConfigDir, sentinelName), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newConfigDir, "config.yaml"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_CONFIG_HOME", configBase)
	t.Setenv("XDG_DATA_HOME", filepath.Join(fakeHome, "xdg-data"))
	t.Setenv("HOME", fakeHome)

	if err := MaybeMigrate(); err != nil {
		t.Fatalf("MaybeMigrate() error: %v", err)
	}

	// New config should be unchanged
	data, err := os.ReadFile(filepath.Join(newConfigDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Errorf("config.yaml was overwritten, got %q want %q", string(data), "new")
	}
}

func TestMaybeMigrate_FreshInstall(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(fakeHome, "xdg-config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(fakeHome, "xdg-data"))
	t.Setenv("HOME", fakeHome)

	// No ~/.fluid, no new dirs - should be a no-op
	if err := MaybeMigrate(); err != nil {
		t.Fatalf("MaybeMigrate() error: %v", err)
	}

	// New dirs should NOT be created
	if _, err := os.Stat(filepath.Join(fakeHome, "xdg-config", "fluid")); !os.IsNotExist(err) {
		t.Errorf("config dir should not exist on fresh install")
	}
}

func TestMaybeMigrate_RetriableAfterPartialFailure(t *testing.T) {
	// If config dir exists but sentinel is missing, migration should be retried
	fakeHome := t.TempDir()
	oldDir := filepath.Join(fakeHome, ".fluid")
	if err := os.MkdirAll(oldDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "config.yaml"), []byte("test: true"), 0o600); err != nil {
		t.Fatal(err)
	}

	configBase := filepath.Join(fakeHome, "xdg-config")
	dataBase := filepath.Join(fakeHome, "xdg-data")
	// Pre-create config dir without sentinel (simulates partial failure)
	if err := os.MkdirAll(filepath.Join(configBase, "fluid"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_CONFIG_HOME", configBase)
	t.Setenv("XDG_DATA_HOME", dataBase)
	t.Setenv("HOME", fakeHome)

	if err := MaybeMigrate(); err != nil {
		t.Fatalf("MaybeMigrate() error: %v", err)
	}

	// Config should have been copied on retry
	if _, err := os.Stat(filepath.Join(configBase, "fluid", "config.yaml")); err != nil {
		t.Errorf("config.yaml not migrated on retry: %v", err)
	}
	// Sentinel should now exist
	if _, err := os.Stat(filepath.Join(configBase, "fluid", sentinelName)); err != nil {
		t.Errorf("sentinel not written after retry: %v", err)
	}
}
