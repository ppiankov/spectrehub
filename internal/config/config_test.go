package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteActivationNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spectrehub.yaml")

	err := WriteActivation("sh_test_0123456789abcdef0123456789abcdef", "https://api.spectrehub.dev", path)
	if err != nil {
		t.Fatalf("WriteActivation: %v", err)
	}

	// Verify file exists and has correct permissions.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 permissions, got %o", info.Mode().Perm())
	}

	// Verify content can be loaded back.
	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if cfg.LicenseKey != "sh_test_0123456789abcdef0123456789abcdef" {
		t.Errorf("license_key mismatch: %s", cfg.LicenseKey)
	}
	if cfg.APIURL != "https://api.spectrehub.dev" {
		t.Errorf("api_url mismatch: %s", cfg.APIURL)
	}
}

func TestWriteActivationPreservesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spectrehub.yaml")

	// Write initial config with custom storage_dir.
	initial := []byte("storage_dir: /custom/path\nfail_threshold: 25\nformat: json\nlast_runs: 10\n")
	if err := os.WriteFile(path, initial, 0600); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	err := WriteActivation("sh_test_deadbeef12345678deadbeef12345678", "https://api.spectrehub.dev", path)
	if err != nil {
		t.Fatalf("WriteActivation: %v", err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	// License key should be set.
	if cfg.LicenseKey != "sh_test_deadbeef12345678deadbeef12345678" {
		t.Errorf("license_key mismatch: %s", cfg.LicenseKey)
	}

	// Existing values should be preserved.
	if cfg.StorageDir != "/custom/path" {
		t.Errorf("storage_dir not preserved: %s", cfg.StorageDir)
	}
	if cfg.FailThreshold != 25 {
		t.Errorf("fail_threshold not preserved: %d", cfg.FailThreshold)
	}
	if cfg.Format != "json" {
		t.Errorf("format not preserved: %s", cfg.Format)
	}
}

func TestWriteActivationCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "dir", "spectrehub.yaml")

	err := WriteActivation("sh_test_0123456789abcdef0123456789abcdef", "https://api.spectrehub.dev", path)
	if err != nil {
		t.Fatalf("WriteActivation: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestConfigPath(t *testing.T) {
	path := ConfigPath()
	if path == "" {
		t.Error("ConfigPath returned empty string")
	}
}
