package cli

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ppiankov/spectrehub/internal/apiclient"
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

// --- submitUserActivity tests ---

func TestSubmitUserActivityNoLicenseKey(t *testing.T) {
	reports := []models.ToolReport{
		{Tool: "mongospectre", RawData: &models.SpectreV1Report{
			Tool: "mongospectre",
			Findings: []models.SpectreV1Finding{
				{ID: "INACTIVE_USER", Severity: "medium", Location: "admin.", Message: `user "x" has no auth`},
			},
		}},
	}

	err := submitUserActivity(reports, PipelineConfig{LicenseKey: ""})
	if err != nil {
		t.Fatalf("expected nil error for empty license key, got %v", err)
	}
}

func TestSubmitUserActivityNoMongoReports(t *testing.T) {
	reports := []models.ToolReport{
		{Tool: "pgspectre", RawData: &models.SpectreV1Report{Tool: "pgspectre"}},
	}

	// Even with a license key, no mongospectre reports means no API call
	err := submitUserActivity(reports, PipelineConfig{
		LicenseKey: "sh_test_0123456789abcdef0123456789abcdef",
		APIURL:     "http://localhost:0", // no server needed
	})
	if err != nil {
		t.Fatalf("expected nil error for no mongo reports, got %v", err)
	}
}

func TestSubmitUserActivityNoUserFindings(t *testing.T) {
	reports := []models.ToolReport{
		{Tool: "mongospectre", RawData: &models.SpectreV1Report{
			Tool: "mongospectre",
			Findings: []models.SpectreV1Finding{
				{ID: "UNUSED_COLLECTION", Severity: "medium", Location: "mydb.old", Message: "unused"},
			},
		}},
	}

	err := submitUserActivity(reports, PipelineConfig{
		LicenseKey: "sh_test_0123456789abcdef0123456789abcdef",
		APIURL:     "http://localhost:0",
	})
	if err != nil {
		t.Fatalf("expected nil error for no user findings, got %v", err)
	}
}

func TestSubmitUserActivitySuccess(t *testing.T) {
	var receivedPayload apiclient.UserActivityPayload
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/users/activity" && r.Method == "POST" {
			_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"count":2}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	reports := []models.ToolReport{
		{Tool: "mongospectre", RawData: &models.SpectreV1Report{
			Tool:   "mongospectre",
			Target: models.SpectreV1Target{Type: "mongodb", URIHash: "sha256:test123"},
			Findings: []models.SpectreV1Finding{
				{ID: "INACTIVE_USER", Severity: "medium", Location: "admin.", Message: `user "stale" has no authentication in the last 7 days`},
				{ID: "INACTIVE_PRIVILEGED_USER", Severity: "high", Location: "admin.", Message: `privileged user "old_admin" has no authentication in the last 7 days`},
			},
		}},
	}

	err := submitUserActivity(reports, PipelineConfig{
		LicenseKey: "sh_test_0123456789abcdef0123456789abcdef",
		APIURL:     server.URL,
	})
	if err != nil {
		t.Fatalf("submitUserActivity: %v", err)
	}

	if receivedPayload.TargetHash != "sha256:test123" {
		t.Fatalf("TargetHash = %s, want sha256:test123", receivedPayload.TargetHash)
	}
	if len(receivedPayload.Users) != 2 {
		t.Fatalf("Users count = %d, want 2", len(receivedPayload.Users))
	}
	if receivedPayload.Users[0].Username != "stale" {
		t.Fatalf("Users[0].Username = %s, want stale", receivedPayload.Users[0].Username)
	}
	if receivedPayload.Users[1].IsPrivileged != true {
		t.Fatal("Users[1].IsPrivileged should be true")
	}
}

func TestSubmitUserActivityAPIError(t *testing.T) {
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"user activity requires Team tier or higher"}`))
	}))
	defer server.Close()

	reports := []models.ToolReport{
		{Tool: "mongospectre", RawData: &models.SpectreV1Report{
			Tool:   "mongospectre",
			Target: models.SpectreV1Target{Type: "mongodb", URIHash: "sha256:test"},
			Findings: []models.SpectreV1Finding{
				{ID: "INACTIVE_USER", Severity: "medium", Location: "admin.", Message: `user "x" has no authentication in the last 7 days`},
			},
		}},
	}

	err := submitUserActivity(reports, PipelineConfig{
		LicenseKey: "sh_test_0123456789abcdef0123456789abcdef",
		APIURL:     server.URL,
	})
	if err == nil {
		t.Fatal("expected error for API 403")
	}
}

func TestSubmitUserActivityNonV1Report(t *testing.T) {
	reports := []models.ToolReport{
		{Tool: "mongospectre", RawData: "not-a-v1-report"},
	}

	err := submitUserActivity(reports, PipelineConfig{
		LicenseKey: "sh_test_0123456789abcdef0123456789abcdef",
		APIURL:     "http://localhost:0",
	})
	if err != nil {
		t.Fatalf("expected nil for non-v1 report, got %v", err)
	}
}
