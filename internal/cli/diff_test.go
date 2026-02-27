package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

func TestIssueKey(t *testing.T) {
	issue := models.NormalizedIssue{
		Tool:     "vaultspectre",
		Category: "missing",
		Resource: "secret/app/db",
	}
	got := issueKey(issue)
	want := "vaultspectre|missing|secret/app/db"
	if got != want {
		t.Errorf("issueKey() = %q, want %q", got, want)
	}
}

func TestComputeDiffNewIssues(t *testing.T) {
	baseline := &models.AggregatedReport{
		Timestamp: time.Now().Add(-1 * time.Hour),
		Issues: []models.NormalizedIssue{
			{Tool: "vaultspectre", Category: "missing", Severity: "critical", Resource: "secret/app/db"},
		},
	}
	current := &models.AggregatedReport{
		Timestamp: time.Now(),
		Issues: []models.NormalizedIssue{
			{Tool: "vaultspectre", Category: "missing", Severity: "critical", Resource: "secret/app/db"},
			{Tool: "s3spectre", Category: "unused", Severity: "low", Resource: "s3://old-bucket"},
		},
	}

	result := computeDiff(baseline, current)

	if result.Summary.NewCount != 1 {
		t.Errorf("expected 1 new issue, got %d", result.Summary.NewCount)
	}
	if result.Summary.ResolvedCount != 0 {
		t.Errorf("expected 0 resolved, got %d", result.Summary.ResolvedCount)
	}
	if result.Summary.Delta != 1 {
		t.Errorf("expected delta +1, got %d", result.Summary.Delta)
	}
	if result.NewIssues[0].Tool != "s3spectre" {
		t.Errorf("expected new issue from s3spectre, got %s", result.NewIssues[0].Tool)
	}
}

func TestComputeDiffResolvedIssues(t *testing.T) {
	baseline := &models.AggregatedReport{
		Timestamp: time.Now().Add(-1 * time.Hour),
		Issues: []models.NormalizedIssue{
			{Tool: "vaultspectre", Category: "missing", Severity: "critical", Resource: "secret/app/db"},
			{Tool: "s3spectre", Category: "unused", Severity: "low", Resource: "s3://old-bucket"},
		},
	}
	current := &models.AggregatedReport{
		Timestamp: time.Now(),
		Issues: []models.NormalizedIssue{
			{Tool: "vaultspectre", Category: "missing", Severity: "critical", Resource: "secret/app/db"},
		},
	}

	result := computeDiff(baseline, current)

	if result.Summary.NewCount != 0 {
		t.Errorf("expected 0 new issues, got %d", result.Summary.NewCount)
	}
	if result.Summary.ResolvedCount != 1 {
		t.Errorf("expected 1 resolved, got %d", result.Summary.ResolvedCount)
	}
	if result.Summary.Delta != -1 {
		t.Errorf("expected delta -1, got %d", result.Summary.Delta)
	}
}

func TestComputeDiffNoChange(t *testing.T) {
	issues := []models.NormalizedIssue{
		{Tool: "vaultspectre", Category: "missing", Severity: "critical", Resource: "secret/app/db"},
	}
	baseline := &models.AggregatedReport{
		Timestamp: time.Now().Add(-1 * time.Hour),
		Issues:    issues,
	}
	current := &models.AggregatedReport{
		Timestamp: time.Now(),
		Issues:    issues,
	}

	result := computeDiff(baseline, current)

	if result.Summary.NewCount != 0 {
		t.Errorf("expected 0 new, got %d", result.Summary.NewCount)
	}
	if result.Summary.ResolvedCount != 0 {
		t.Errorf("expected 0 resolved, got %d", result.Summary.ResolvedCount)
	}
	if result.Summary.Delta != 0 {
		t.Errorf("expected delta 0, got %d", result.Summary.Delta)
	}
}

func TestComputeDiffSummaryBreakdown(t *testing.T) {
	baseline := &models.AggregatedReport{
		Timestamp: time.Now().Add(-1 * time.Hour),
		Issues:    []models.NormalizedIssue{},
	}
	current := &models.AggregatedReport{
		Timestamp: time.Now(),
		Issues: []models.NormalizedIssue{
			{Tool: "vaultspectre", Category: "missing", Severity: "critical", Resource: "secret/a"},
			{Tool: "vaultspectre", Category: "stale", Severity: "low", Resource: "secret/b"},
			{Tool: "s3spectre", Category: "unused", Severity: "low", Resource: "s3://bucket"},
		},
	}

	result := computeDiff(baseline, current)

	if result.Summary.NewCount != 3 {
		t.Fatalf("expected 3 new, got %d", result.Summary.NewCount)
	}

	// By severity.
	if result.Summary.NewBySeverity["critical"] != 1 {
		t.Errorf("expected 1 critical, got %d", result.Summary.NewBySeverity["critical"])
	}
	if result.Summary.NewBySeverity["low"] != 2 {
		t.Errorf("expected 2 low, got %d", result.Summary.NewBySeverity["low"])
	}

	// By tool.
	if result.Summary.NewByTool["vaultspectre"] != 2 {
		t.Errorf("expected 2 from vaultspectre, got %d", result.Summary.NewByTool["vaultspectre"])
	}
	if result.Summary.NewByTool["s3spectre"] != 1 {
		t.Errorf("expected 1 from s3spectre, got %d", result.Summary.NewByTool["s3spectre"])
	}

	// By category.
	if result.Summary.NewByCategory["missing"] != 1 {
		t.Errorf("expected 1 missing, got %d", result.Summary.NewByCategory["missing"])
	}
}

func TestComputeDiffEmptyReports(t *testing.T) {
	baseline := &models.AggregatedReport{
		Timestamp: time.Now().Add(-1 * time.Hour),
		Issues:    nil,
	}
	current := &models.AggregatedReport{
		Timestamp: time.Now(),
		Issues:    nil,
	}

	result := computeDiff(baseline, current)

	if result.Summary.NewCount != 0 {
		t.Errorf("expected 0 new, got %d", result.Summary.NewCount)
	}
	if result.Summary.ResolvedCount != 0 {
		t.Errorf("expected 0 resolved, got %d", result.Summary.ResolvedCount)
	}
	if result.Summary.Delta != 0 {
		t.Errorf("expected delta 0, got %d", result.Summary.Delta)
	}
}

// --- outputDiff tests ---

func sampleDiffResult() *DiffResult {
	return &DiffResult{
		Baseline: "2026-01-01 10:00:00",
		Current:  "2026-02-01 10:00:00",
		NewIssues: []models.NormalizedIssue{
			{Tool: "vaultspectre", Category: "missing", Severity: "critical", Resource: "secret/new", Evidence: "key not found"},
		},
		ResolvedIssues: []models.NormalizedIssue{
			{Tool: "s3spectre", Category: "unused", Severity: "low", Resource: "s3://old-bucket"},
		},
		Summary: DiffSummary{
			BaselineTotal: 5,
			CurrentTotal:  5,
			NewCount:      1,
			ResolvedCount: 1,
			Delta:         0,
			NewBySeverity: map[string]int{"critical": 1},
			NewByTool:     map[string]int{"vaultspectre": 1},
			NewByCategory: map[string]int{"missing": 1},
		},
	}
}

func TestOutputDiffJSON(t *testing.T) {
	out := filepath.Join(t.TempDir(), "diff.json")

	if err := outputDiff(sampleDiffResult(), "json", out); err != nil {
		t.Fatalf("outputDiff(json): %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}

	var result DiffResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Summary.NewCount != 1 {
		t.Errorf("NewCount = %d, want 1", result.Summary.NewCount)
	}
	if result.Summary.ResolvedCount != 1 {
		t.Errorf("ResolvedCount = %d, want 1", result.Summary.ResolvedCount)
	}
}

func TestOutputDiffText(t *testing.T) {
	out := filepath.Join(t.TempDir(), "diff.txt")

	if err := outputDiff(sampleDiffResult(), "text", out); err != nil {
		t.Fatalf("outputDiff(text): %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "Drift Delta") {
		t.Error("missing header")
	}
	if !strings.Contains(content, "secret/new") {
		t.Error("missing new issue resource")
	}
	if !strings.Contains(content, "s3://old-bucket") {
		t.Error("missing resolved issue")
	}
}

func TestOutputDiffUnsupportedFormat(t *testing.T) {
	err := outputDiff(sampleDiffResult(), "xml", "")
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
}

func TestPrintDiffTextNoDrift(t *testing.T) {
	result := &DiffResult{
		Baseline:       "2026-01-01 10:00:00",
		Current:        "2026-02-01 10:00:00",
		NewIssues:      nil,
		ResolvedIssues: nil,
		Summary: DiffSummary{
			BaselineTotal: 3,
			CurrentTotal:  3,
			NewCount:      0,
			ResolvedCount: 0,
			Delta:         0,
		},
	}

	out := filepath.Join(t.TempDir(), "nodrift.txt")
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}

	if err := printDiffText(f, result); err != nil {
		t.Fatalf("printDiffText: %v", err)
	}
	_ = f.Close()

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "No drift detected") {
		t.Error("expected 'No drift detected' message")
	}
}

func TestPrintDiffTextOnlyResolved(t *testing.T) {
	result := &DiffResult{
		Baseline:  "2026-01-01 10:00:00",
		Current:   "2026-02-01 10:00:00",
		NewIssues: nil,
		ResolvedIssues: []models.NormalizedIssue{
			{Tool: "vaultspectre", Category: "missing", Resource: "secret/old"},
		},
		Summary: DiffSummary{
			BaselineTotal: 3,
			CurrentTotal:  2,
			NewCount:      0,
			ResolvedCount: 1,
			Delta:         -1,
		},
	}

	out := filepath.Join(t.TempDir(), "resolved.txt")
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}

	if err := printDiffText(f, result); err != nil {
		t.Fatalf("printDiffText: %v", err)
	}
	_ = f.Close()

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "No new issues") {
		t.Error("expected 'No new issues' message")
	}
}

// --- loadReportFromFile tests ---

func TestLoadReportFromFileValid(t *testing.T) {
	report := &models.AggregatedReport{
		Timestamp: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
		Issues: []models.NormalizedIssue{
			{Tool: "vaultspectre", Category: "missing", Severity: "critical", Resource: "secret/db"},
		},
		Summary: models.CrossToolSummary{TotalIssues: 1},
	}

	data, _ := json.Marshal(report)
	tmp := filepath.Join(t.TempDir(), "report.json")
	_ = os.WriteFile(tmp, data, 0644)

	loaded, err := loadReportFromFile(tmp)
	if err != nil {
		t.Fatalf("loadReportFromFile: %v", err)
	}
	if loaded.Summary.TotalIssues != 1 {
		t.Errorf("TotalIssues = %d, want 1", loaded.Summary.TotalIssues)
	}
}

func TestLoadReportFromFileInvalid(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "bad.json")
	_ = os.WriteFile(tmp, []byte("{not json"), 0644)

	_, err := loadReportFromFile(tmp)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadReportFromFileMissing(t *testing.T) {
	_, err := loadReportFromFile("/nonexistent/report.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
