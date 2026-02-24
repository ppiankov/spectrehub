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

func TestConfigPathXDG(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := ConfigPath()
	if path == "" {
		t.Error("ConfigPath returned empty string")
	}
	expected := filepath.Join(dir, "spectrehub", "spectrehub.yaml")
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.StorageDir != ".spectre" {
		t.Errorf("expected storage_dir=.spectre, got %s", cfg.StorageDir)
	}
	if cfg.FailThreshold != 0 {
		t.Errorf("expected fail_threshold=0, got %d", cfg.FailThreshold)
	}
	if cfg.Format != "text" {
		t.Errorf("expected format=text, got %s", cfg.Format)
	}
	if cfg.LastRuns != 7 {
		t.Errorf("expected last_runs=7, got %d", cfg.LastRuns)
	}
	if cfg.Verbose {
		t.Error("expected verbose=false")
	}
	if cfg.Debug {
		t.Error("expected debug=false")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid defaults",
			cfg:     *DefaultConfig(),
			wantErr: false,
		},
		{
			name:    "valid json format",
			cfg:     Config{StorageDir: ".spectre", Format: "json", LastRuns: 7},
			wantErr: false,
		},
		{
			name:    "valid both format",
			cfg:     Config{StorageDir: ".spectre", Format: "both", LastRuns: 7},
			wantErr: false,
		},
		{
			name:    "invalid format",
			cfg:     Config{StorageDir: ".spectre", Format: "xml", LastRuns: 7},
			wantErr: true,
			errMsg:  "invalid format",
		},
		{
			name:    "negative threshold",
			cfg:     Config{StorageDir: ".spectre", Format: "text", FailThreshold: -1, LastRuns: 7},
			wantErr: true,
			errMsg:  "fail_threshold cannot be negative",
		},
		{
			name:    "zero last_runs",
			cfg:     Config{StorageDir: ".spectre", Format: "text", LastRuns: 0},
			wantErr: true,
			errMsg:  "last_runs must be positive",
		},
		{
			name:    "negative last_runs",
			cfg:     Config{StorageDir: ".spectre", Format: "text", LastRuns: -1},
			wantErr: true,
			errMsg:  "last_runs must be positive",
		},
		{
			name:    "empty storage_dir",
			cfg:     Config{Format: "text", LastRuns: 7},
			wantErr: true,
			errMsg:  "storage_dir cannot be empty",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Fatalf("expected error to contain %q, got %q", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestShouldFailOnThreshold(t *testing.T) {
	tests := []struct {
		name       string
		threshold  int
		issueCount int
		expected   bool
	}{
		{"disabled", 0, 100, false},
		{"below threshold", 10, 5, false},
		{"at threshold", 10, 10, false},
		{"above threshold", 10, 11, true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{FailThreshold: tt.threshold}
			if got := cfg.ShouldFailOnThreshold(tt.issueCount); got != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestGetStoragePath(t *testing.T) {
	tests := []struct {
		name       string
		storageDir string
		wantErr    bool
	}{
		{"relative path", ".spectre", false},
		{"home expansion", "~/spectre-data", false},
		{"absolute path", "/tmp/spectre", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{StorageDir: tt.storageDir}
			path, err := cfg.GetStoragePath()
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && path == "" {
				t.Fatal("expected non-empty path")
			}
		})
	}
}

func TestLoadFromFileWithConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spectrehub.yaml")

	content := `storage_dir: /custom/path
fail_threshold: 25
format: json
last_runs: 10
verbose: true
debug: true
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	if cfg.StorageDir != "/custom/path" {
		t.Errorf("expected storage_dir=/custom/path, got %s", cfg.StorageDir)
	}
	if cfg.FailThreshold != 25 {
		t.Errorf("expected fail_threshold=25, got %d", cfg.FailThreshold)
	}
	if cfg.Format != "json" {
		t.Errorf("expected format=json, got %s", cfg.Format)
	}
	if cfg.LastRuns != 10 {
		t.Errorf("expected last_runs=10, got %d", cfg.LastRuns)
	}
	if !cfg.Verbose {
		t.Error("expected verbose=true")
	}
	if !cfg.Debug {
		t.Error("expected debug=true")
	}
}

func TestLoadFromFileInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spectrehub.yaml")

	// Invalid format value
	content := `format: xml
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFromFile(path)
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestLoadFromFileNoFile(t *testing.T) {
	// Load with no config file should use defaults
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.StorageDir != ".spectre" {
		t.Errorf("expected default storage_dir, got %s", cfg.StorageDir)
	}
}

func TestGenerateSampleConfig(t *testing.T) {
	sample := GenerateSampleConfig()
	if sample == "" {
		t.Fatal("expected non-empty sample config")
	}
	expectedFragments := []string{
		"storage_dir",
		"fail_threshold",
		"format",
		"last_runs",
		"verbose",
		"debug",
		"license_key",
	}
	for _, frag := range expectedFragments {
		if !containsStr(sample, frag) {
			t.Errorf("expected sample config to contain %q", frag)
		}
	}
}

func TestLoadFromFileWithEnvVars(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SPECTREHUB_FORMAT", "json")
	t.Setenv("SPECTREHUB_VERBOSE", "true")

	cfg, err := LoadFromFile("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Format != "json" {
		t.Errorf("expected format=json from env, got %s", cfg.Format)
	}
	if !cfg.Verbose {
		t.Error("expected verbose=true from env")
	}
}
