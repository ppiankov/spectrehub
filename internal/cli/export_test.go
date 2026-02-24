package cli

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

func sampleReports() []*models.AggregatedReport {
	return []*models.AggregatedReport{
		{
			Timestamp: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
			Issues: []models.NormalizedIssue{
				{Tool: "vaultspectre", Category: "missing", Severity: "critical", Resource: "secret/db", Evidence: "key not found"},
				{Tool: "s3spectre", Category: "unused", Severity: "low", Resource: "s3://old-bucket"},
			},
			Summary: models.CrossToolSummary{
				TotalIssues:  2,
				HealthScore:  "warning",
				ScorePercent: 72.5,
				TotalTools:   2,
			},
		},
	}
}

func TestBuildComplianceExport(t *testing.T) {
	export := buildComplianceExport(sampleReports())

	if export.RunCount != 1 {
		t.Errorf("expected 1 run, got %d", export.RunCount)
	}
	if export.IssueCount != 2 {
		t.Errorf("expected 2 issues, got %d", export.IssueCount)
	}
	if export.Framework != "SOC2/ISO27001" {
		t.Errorf("expected SOC2/ISO27001, got %s", export.Framework)
	}

	// Critical should come first after sort.
	if export.Records[0].Severity != "critical" {
		t.Errorf("expected critical first, got %s", export.Records[0].Severity)
	}
}

func TestWriteCSV(t *testing.T) {
	export := buildComplianceExport(sampleReports())

	tmp, err := os.CreateTemp(t.TempDir(), "export-*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tmp.Close() }()

	if err := writeCSV(tmp, export); err != nil {
		t.Fatalf("writeCSV: %v", err)
	}

	// Read back.
	_ = tmp.Close()
	f, err := os.Open(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	// Header + 2 data rows.
	if len(records) != 3 {
		t.Errorf("expected 3 rows (header + 2), got %d", len(records))
	}
	if records[0][0] != "run_timestamp" {
		t.Errorf("expected header run_timestamp, got %s", records[0][0])
	}
}

func TestWriteExportJSON(t *testing.T) {
	export := buildComplianceExport(sampleReports())

	tmp, err := os.CreateTemp(t.TempDir(), "export-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tmp.Close() }()

	if err := writeExportJSON(tmp, export); err != nil {
		t.Fatalf("writeExportJSON: %v", err)
	}

	// Read back.
	_ = tmp.Close()
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}

	var parsed ComplianceExport
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.IssueCount != 2 {
		t.Errorf("expected 2 issues, got %d", parsed.IssueCount)
	}
}

func TestWriteSARIF(t *testing.T) {
	reports := sampleReports()

	tmp, err := os.CreateTemp(t.TempDir(), "export-*.sarif")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tmp.Close() }()

	if err := writeSARIF(tmp, reports); err != nil {
		t.Fatalf("writeSARIF: %v", err)
	}

	// Read back and validate structure.
	_ = tmp.Close()
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}

	var log sarifLog
	if err := json.Unmarshal(data, &log); err != nil {
		t.Fatalf("unmarshal sarif: %v", err)
	}

	if log.Version != "2.1.0" {
		t.Errorf("expected SARIF 2.1.0, got %s", log.Version)
	}
	if len(log.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(log.Runs))
	}
	if log.Runs[0].Tool.Driver.Name != "spectrehub" {
		t.Errorf("expected tool spectrehub, got %s", log.Runs[0].Tool.Driver.Name)
	}
	if len(log.Runs[0].Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(log.Runs[0].Results))
	}
}

func TestSarifLevel(t *testing.T) {
	tests := []struct {
		severity string
		want     string
	}{
		{"critical", "error"},
		{"high", "error"},
		{"medium", "warning"},
		{"low", "note"},
		{"", "note"},
	}
	for _, tt := range tests {
		got := sarifLevel(tt.severity)
		if got != tt.want {
			t.Errorf("sarifLevel(%q) = %q, want %q", tt.severity, got, tt.want)
		}
	}
}

func TestFormatEvidence(t *testing.T) {
	issue := models.NormalizedIssue{
		Tool:     "vaultspectre",
		Category: "missing",
		Resource: "secret/db",
		Evidence: "key not found",
	}
	got := formatEvidence(issue)
	if !strings.Contains(got, "vaultspectre") {
		t.Errorf("expected tool in evidence, got %s", got)
	}
	if !strings.Contains(got, "key not found") {
		t.Errorf("expected evidence text, got %s", got)
	}
}

func TestBuildComplianceExportMultipleRuns(t *testing.T) {
	reports := []*models.AggregatedReport{
		{
			Timestamp: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			Issues: []models.NormalizedIssue{
				{Tool: "vaultspectre", Category: "stale", Severity: "low", Resource: "secret/old"},
			},
			Summary: models.CrossToolSummary{TotalIssues: 1, HealthScore: "good", ScorePercent: 90},
		},
		{
			Timestamp: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
			Issues: []models.NormalizedIssue{
				{Tool: "s3spectre", Category: "missing", Severity: "critical", Resource: "s3://bucket"},
			},
			Summary: models.CrossToolSummary{TotalIssues: 1, HealthScore: "warning", ScorePercent: 70},
		},
	}

	export := buildComplianceExport(reports)

	if export.RunCount != 2 {
		t.Errorf("expected 2 runs, got %d", export.RunCount)
	}
	if export.IssueCount != 2 {
		t.Errorf("expected 2 issues, got %d", export.IssueCount)
	}
	// Critical should sort first.
	if export.Records[0].Severity != "critical" {
		t.Errorf("expected critical first, got %s", export.Records[0].Severity)
	}
}
