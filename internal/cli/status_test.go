package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestWriteStatusTextWithLicense(t *testing.T) {
	result := statusResult{
		License: &statusLicense{
			Valid:     true,
			Tier:      "pro",
			MaxRepos:  10,
			UsedRepos: 3,
			Repos:     []string{"org/repo-a", "org/repo-b", "org/repo-c"},
			ExpiresAt: "2027-01-01",
		},
		Config: statusConfig{
			StorageDir: ".spectre",
			Format:     "text",
			Repo:       "org/main",
			HasKey:     true,
		},
	}

	output := captureStdout(t, func() {
		_ = writeStatusText(result)
	})

	if !strings.Contains(output, "pro") {
		t.Error("missing license tier")
	}
	if !strings.Contains(output, "3/10") {
		t.Error("missing repo usage")
	}
	if !strings.Contains(output, "org/repo-a") {
		t.Error("missing repo list")
	}
	if !strings.Contains(output, "2027-01-01") {
		t.Error("missing expiry")
	}
	if !strings.Contains(output, ".spectre") {
		t.Error("missing storage dir")
	}
	if !strings.Contains(output, "org/main") {
		t.Error("missing repo config")
	}
}

func TestWriteStatusTextUnlimitedRepos(t *testing.T) {
	result := statusResult{
		License: &statusLicense{
			Valid:     true,
			Tier:      "enterprise",
			MaxRepos:  0, // unlimited
			UsedRepos: 5,
			ExpiresAt: "2027-06-01",
		},
		Config: statusConfig{
			StorageDir: ".spectre",
			HasKey:     true,
		},
	}

	output := captureStdout(t, func() {
		_ = writeStatusText(result)
	})

	if !strings.Contains(output, "unlimited") {
		t.Error("expected 'unlimited' for MaxRepos=0")
	}
}

func TestWriteStatusTextNoLicense(t *testing.T) {
	result := statusResult{
		Config: statusConfig{
			StorageDir: ".spectre",
			HasKey:     false,
		},
	}

	output := captureStdout(t, func() {
		_ = writeStatusText(result)
	})

	if !strings.Contains(output, "not configured") {
		t.Error("expected 'not configured' for no license")
	}
}

func TestWriteStatusTextInvalidLicense(t *testing.T) {
	result := statusResult{
		Config: statusConfig{
			StorageDir: ".spectre",
			HasKey:     true,
		},
	}

	output := captureStdout(t, func() {
		_ = writeStatusText(result)
	})

	if !strings.Contains(output, "invalid or expired") {
		t.Error("expected 'invalid or expired' for HasKey=true but no License")
	}
}

func TestWriteStatusJSON(t *testing.T) {
	result := statusResult{
		License: &statusLicense{
			Valid:    true,
			Tier:     "pro",
			MaxRepos: 5,
		},
		Config: statusConfig{
			StorageDir: ".spectre",
			Format:     "json",
			HasKey:     true,
		},
		ConfigFile: "/home/.spectrehub.yaml",
	}

	output := captureStdout(t, func() {
		_ = writeStatusJSON(result)
	})

	var parsed statusResult
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.License.Tier != "pro" {
		t.Errorf("Tier = %q, want pro", parsed.License.Tier)
	}
	if parsed.Config.StorageDir != ".spectre" {
		t.Errorf("StorageDir = %q, want .spectre", parsed.Config.StorageDir)
	}
}

func TestWriteActivateJSONToStdout(t *testing.T) {
	output := captureStdout(t, func() {
		writeActivateJSON(os.Stdout, "activated", "", "pro", 10, "/home/.spectrehub.yaml")
	})

	var result activateResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Status != "activated" {
		t.Errorf("Status = %q, want activated", result.Status)
	}
}

func TestWriteActivateJSONViaFile(t *testing.T) {
	// writeActivateJSON writes to an *os.File parameter
	// We can test it by passing a temp file
	tmp, err := createTempFileForTest(t)
	if err != nil {
		t.Fatal(err)
	}

	writeActivateJSON(tmp, "activated", "", "pro", 10, "/home/.spectrehub.yaml")
	_ = tmp.Close()

	data, _ := readTempFile(t, tmp.Name())
	var result activateResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Status != "activated" {
		t.Errorf("Status = %q, want activated", result.Status)
	}
	if result.Plan != "pro" {
		t.Errorf("Plan = %q, want pro", result.Plan)
	}
	if result.MaxRepos != 10 {
		t.Errorf("MaxRepos = %d, want 10", result.MaxRepos)
	}
}

func TestWriteActivateJSONError(t *testing.T) {
	tmp, err := createTempFileForTest(t)
	if err != nil {
		t.Fatal(err)
	}

	writeActivateJSON(tmp, "error", "invalid key format", "", 0, "")
	_ = tmp.Close()

	data, _ := readTempFile(t, tmp.Name())
	var result activateResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Status != "error" {
		t.Errorf("Status = %q, want error", result.Status)
	}
	if result.Error != "invalid key format" {
		t.Errorf("Error = %q, want 'invalid key format'", result.Error)
	}
}

// helpers for temp file testing
func createTempFileForTest(t *testing.T) (*os.File, error) {
	t.Helper()
	return os.CreateTemp(t.TempDir(), "test-*.json")
}

func readTempFile(t *testing.T, path string) ([]byte, error) {
	t.Helper()
	return os.ReadFile(path)
}
