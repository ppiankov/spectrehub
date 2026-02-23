package discovery

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/ppiankov/spectrehub/internal/models"
)

// mockLookPath returns a function that resolves only the listed binaries.
func mockLookPath(available map[string]string) LookPathFunc {
	return func(file string) (string, error) {
		if path, ok := available[file]; ok {
			return path, nil
		}
		return "", errors.New("not found")
	}
}

// mockGetenv returns a function that resolves only the listed env vars.
func mockGetenv(vars map[string]string) GetenvFunc {
	return func(key string) string {
		return vars[key]
	}
}

func TestDiscover_NoBinaries(t *testing.T) {
	d := New(mockLookPath(nil), mockGetenv(nil))
	plan := d.Discover()

	if plan.TotalFound != 0 {
		t.Errorf("expected 0 found, got %d", plan.TotalFound)
	}
	if plan.TotalRunnable != 0 {
		t.Errorf("expected 0 runnable, got %d", plan.TotalRunnable)
	}
	if len(plan.Tools) != len(Registry) {
		t.Errorf("expected %d tools, got %d", len(Registry), len(plan.Tools))
	}

	for _, td := range plan.Tools {
		if td.Available {
			t.Errorf("tool %s should not be available", td.Tool)
		}
		if td.Runnable {
			t.Errorf("tool %s should not be runnable", td.Tool)
		}
	}
}

func TestDiscover_BinaryButNoTarget(t *testing.T) {
	binaries := map[string]string{
		"vaultspectre": "/usr/local/bin/vaultspectre",
		"pgspectre":    "/usr/local/bin/pgspectre",
	}
	d := New(mockLookPath(binaries), mockGetenv(nil))
	plan := d.Discover()

	if plan.TotalFound != 2 {
		t.Errorf("expected 2 found, got %d", plan.TotalFound)
	}
	if plan.TotalRunnable != 0 {
		t.Errorf("expected 0 runnable (no targets), got %d", plan.TotalRunnable)
	}

	for _, td := range plan.Tools {
		if td.Available && td.Runnable {
			t.Errorf("tool %s should not be runnable without target", td.Tool)
		}
	}
}

func TestDiscover_BinaryWithEnvTarget(t *testing.T) {
	binaries := map[string]string{
		"vaultspectre": "/usr/local/bin/vaultspectre",
	}
	envVars := map[string]string{
		"VAULT_ADDR":  "https://vault.example.com",
		"VAULT_TOKEN": "s.abc123",
	}
	d := New(mockLookPath(binaries), mockGetenv(envVars))
	plan := d.Discover()

	if plan.TotalFound != 1 {
		t.Errorf("expected 1 found, got %d", plan.TotalFound)
	}
	if plan.TotalRunnable != 1 {
		t.Errorf("expected 1 runnable, got %d", plan.TotalRunnable)
	}

	runnable := plan.RunnableTools()
	if len(runnable) != 1 {
		t.Fatalf("expected 1 runnable tool, got %d", len(runnable))
	}
	if runnable[0].Tool != models.ToolVault {
		t.Errorf("expected vaultspectre, got %s", runnable[0].Tool)
	}
	if runnable[0].BinaryPath != "/usr/local/bin/vaultspectre" {
		t.Errorf("expected path /usr/local/bin/vaultspectre, got %s", runnable[0].BinaryPath)
	}
}

func TestDiscover_PartialEnvVars(t *testing.T) {
	binaries := map[string]string{
		"vaultspectre": "/usr/local/bin/vaultspectre",
	}
	envVars := map[string]string{
		"VAULT_ADDR": "https://vault.example.com",
		// VAULT_TOKEN not set
	}
	d := New(mockLookPath(binaries), mockGetenv(envVars))
	plan := d.Discover()

	// Still runnable — at least one env var is set
	if plan.TotalRunnable != 1 {
		t.Errorf("expected 1 runnable (partial env OK), got %d", plan.TotalRunnable)
	}
}

func TestDiscover_AllTools(t *testing.T) {
	binaries := map[string]string{
		"vaultspectre": "/usr/local/bin/vaultspectre",
		"s3spectre":    "/usr/local/bin/s3spectre",
		"kafkaspectre": "/usr/local/bin/kafkaspectre",
		"clickspectre": "/usr/local/bin/clickspectre",
		"pgspectre":    "/usr/local/bin/pgspectre",
		"mongospectre": "/usr/local/bin/mongospectre",
	}
	envVars := map[string]string{
		"VAULT_ADDR":       "https://vault.example.com",
		"VAULT_TOKEN":      "s.abc123",
		"AWS_PROFILE":      "production",
		"PGSPECTRE_DB_URL": "postgres://localhost:5432/db",
		"MONGODB_URI":      "mongodb://localhost:27017",
	}
	d := New(mockLookPath(binaries), mockGetenv(envVars))
	plan := d.Discover()

	if plan.TotalFound != 6 {
		t.Errorf("expected 6 found, got %d", plan.TotalFound)
	}

	// Kafka and ClickHouse have no env vars set and no config files present
	// so they should NOT be runnable
	if plan.TotalRunnable != 4 {
		t.Errorf("expected 4 runnable (vault, s3, pg, mongo), got %d", plan.TotalRunnable)
	}
}

func TestDiscover_MissingBinaryWithEnv(t *testing.T) {
	// Env vars set but binary not installed
	envVars := map[string]string{
		"VAULT_ADDR":  "https://vault.example.com",
		"VAULT_TOKEN": "s.abc123",
	}
	d := New(mockLookPath(nil), mockGetenv(envVars))
	plan := d.Discover()

	if plan.TotalRunnable != 0 {
		t.Errorf("expected 0 runnable (no binary), got %d", plan.TotalRunnable)
	}

	// But HasTarget should be true for vault
	for _, td := range plan.Tools {
		if td.Tool == models.ToolVault {
			if !td.HasTarget {
				t.Error("vaultspectre should have target (env vars set)")
			}
			if td.Available {
				t.Error("vaultspectre should not be available (no binary)")
			}
		}
	}
}

func TestDiscover_DeterministicOrder(t *testing.T) {
	d := New(mockLookPath(nil), mockGetenv(nil))

	plan1 := d.Discover()
	plan2 := d.Discover()

	if len(plan1.Tools) != len(plan2.Tools) {
		t.Fatal("plans have different tool counts")
	}

	for i := range plan1.Tools {
		if plan1.Tools[i].Tool != plan2.Tools[i].Tool {
			t.Errorf("order mismatch at %d: %s vs %s", i, plan1.Tools[i].Tool, plan2.Tools[i].Tool)
		}
	}
}

func TestDiscoveryPlan_JSON(t *testing.T) {
	binaries := map[string]string{
		"vaultspectre": "/usr/local/bin/vaultspectre",
	}
	envVars := map[string]string{
		"VAULT_ADDR": "https://vault.example.com",
	}
	d := New(mockLookPath(binaries), mockGetenv(envVars))
	plan := d.Discover()

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("failed to marshal plan: %v", err)
	}

	// Verify it round-trips
	var decoded DiscoveryPlan
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal plan: %v", err)
	}

	if decoded.TotalFound != plan.TotalFound {
		t.Errorf("TotalFound mismatch: %d vs %d", decoded.TotalFound, plan.TotalFound)
	}
	if decoded.TotalRunnable != plan.TotalRunnable {
		t.Errorf("TotalRunnable mismatch: %d vs %d", decoded.TotalRunnable, plan.TotalRunnable)
	}
}

func TestRunnableTools_Empty(t *testing.T) {
	plan := &DiscoveryPlan{}
	runnable := plan.RunnableTools()
	if len(runnable) != 0 {
		t.Errorf("expected 0 runnable, got %d", len(runnable))
	}
}

func TestEnvVarStatus_Tracking(t *testing.T) {
	binaries := map[string]string{
		"vaultspectre": "/usr/local/bin/vaultspectre",
	}
	envVars := map[string]string{
		"VAULT_ADDR": "https://vault.example.com",
		// VAULT_TOKEN deliberately not set
	}
	d := New(mockLookPath(binaries), mockGetenv(envVars))
	plan := d.Discover()

	for _, td := range plan.Tools {
		if td.Tool != models.ToolVault {
			continue
		}
		if len(td.EnvVars) != 2 {
			t.Fatalf("expected 2 env var statuses, got %d", len(td.EnvVars))
		}
		for _, ev := range td.EnvVars {
			switch ev.Name {
			case "VAULT_ADDR":
				if !ev.Set {
					t.Error("VAULT_ADDR should be set")
				}
			case "VAULT_TOKEN":
				if ev.Set {
					t.Error("VAULT_TOKEN should not be set")
				}
			default:
				t.Errorf("unexpected env var: %s", ev.Name)
			}
		}
	}
}

func TestRegistryCompleteness(t *testing.T) {
	// Every tool in models.SupportedTools should be in Registry
	for toolType := range models.SupportedTools {
		if _, ok := Registry[toolType]; !ok {
			t.Errorf("tool %s is in SupportedTools but missing from discovery Registry", toolType)
		}
	}

	// Every tool in Registry should be in models.SupportedTools
	for toolType := range Registry {
		if _, ok := models.SupportedTools[toolType]; !ok {
			t.Errorf("tool %s is in discovery Registry but missing from SupportedTools", toolType)
		}
	}
}

func TestRegistryFields(t *testing.T) {
	for toolType, info := range Registry {
		if info.Binary == "" {
			t.Errorf("tool %s has empty binary name", toolType)
		}
		if info.Subcommand == "" {
			t.Errorf("tool %s has empty subcommand", toolType)
		}
		if info.JSONFlag == "" {
			t.Errorf("tool %s has empty JSON flag", toolType)
		}
		// Each tool must have at least one discovery signal (env vars OR config files)
		if len(info.EnvVars) == 0 && len(info.ConfigFiles) == 0 {
			t.Errorf("tool %s has no env vars or config files for target detection", toolType)
		}
	}
}
