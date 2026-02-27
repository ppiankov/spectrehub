package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

// --- getStoragePath tests ---

func TestGetStoragePathRelative(t *testing.T) {
	got, err := getStoragePath(".spectre")
	if err != nil {
		t.Fatalf("getStoragePath: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("expected absolute path, got %q", got)
	}
	if !strings.HasSuffix(got, ".spectre") {
		t.Errorf("expected path ending with .spectre, got %q", got)
	}
}

func TestGetStoragePathAbsolute(t *testing.T) {
	got, err := getStoragePath("/tmp/spectre-test")
	if err != nil {
		t.Fatalf("getStoragePath: %v", err)
	}
	if got != "/tmp/spectre-test" {
		t.Errorf("getStoragePath(/tmp/spectre-test) = %q, want /tmp/spectre-test", got)
	}
}

func TestGetStoragePathTilde(t *testing.T) {
	got, err := getStoragePath("~/spectre-data")
	if err != nil {
		t.Fatalf("getStoragePath: %v", err)
	}

	home, _ := os.UserHomeDir()
	want := filepath.Join(home, "spectre-data")
	if got != want {
		t.Errorf("getStoragePath(~/spectre-data) = %q, want %q", got, want)
	}
}

// --- generateOutput tests ---

func minimalReport() *models.AggregatedReport {
	return &models.AggregatedReport{
		Timestamp: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
		Issues: []models.NormalizedIssue{
			{Tool: "vaultspectre", Category: "missing", Severity: "critical", Resource: "secret/db"},
		},
		ToolReports: map[string]models.ToolReport{},
		Summary: models.CrossToolSummary{
			TotalIssues: 1,
			TotalTools:  1,
			HealthScore: "warning",
		},
	}
}

func TestGenerateOutputText(t *testing.T) {
	out := filepath.Join(t.TempDir(), "output.txt")

	if err := generateOutput(minimalReport(), "text", out); err != nil {
		t.Fatalf("generateOutput(text): %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) == 0 {
		t.Error("text output is empty")
	}
}

func TestGenerateOutputJSON(t *testing.T) {
	out := filepath.Join(t.TempDir(), "output.json")

	if err := generateOutput(minimalReport(), "json", out); err != nil {
		t.Fatalf("generateOutput(json): %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "vaultspectre") {
		t.Error("JSON output missing tool name")
	}
}

func TestGenerateOutputBothToFile(t *testing.T) {
	out := filepath.Join(t.TempDir(), "output.txt")

	if err := generateOutput(minimalReport(), "both", out); err != nil {
		t.Fatalf("generateOutput(both): %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "JSON Output") {
		t.Error("'both' format missing JSON separator")
	}
}

func TestGenerateOutputUnsupported(t *testing.T) {
	err := generateOutput(minimalReport(), "xml", "")
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("error = %q, want 'unsupported format'", err.Error())
	}
}
