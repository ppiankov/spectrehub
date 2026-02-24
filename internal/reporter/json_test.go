package reporter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

func sampleReport() *models.AggregatedReport {
	return &models.AggregatedReport{
		Timestamp: time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC),
		Issues: []models.NormalizedIssue{
			{Tool: "vaultspectre", Category: "missing", Severity: "critical", Resource: "secret/db"},
		},
		ToolReports: map[string]models.ToolReport{
			"vaultspectre": {Tool: "vaultspectre", Version: "0.1.0"},
		},
		Summary: models.CrossToolSummary{
			TotalIssues:      1,
			TotalTools:       1,
			SupportedTools:   1,
			HealthScore:      "warning",
			ScorePercent:     75.0,
			IssuesByTool:     map[string]int{"vaultspectre": 1},
			IssuesByCategory: map[string]int{"missing": 1},
			IssuesBySeverity: map[string]int{"critical": 1},
		},
		Recommendations: []models.Recommendation{
			{Severity: "critical", Tool: "vaultspectre", Action: "Fix missing", Impact: "Broken", Count: 1},
		},
	}
}

func TestJSONReporterGenerate(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONReporter(&buf, false)

	err := r.Generate(sampleReport())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.HasSuffix(output, "\n") {
		t.Error("expected trailing newline")
	}

	// Should be valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}

func TestJSONReporterGeneratePretty(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONReporter(&buf, true)

	err := r.Generate(sampleReport())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Pretty JSON has indentation
	if !strings.Contains(output, "  ") {
		t.Error("expected pretty-printed JSON with indentation")
	}
}

func TestJSONReporterGenerateSummaryOnly(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONReporter(&buf, false)

	report := sampleReport()
	err := r.GenerateSummaryOnly(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Should be valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Should have summary but not raw tool data
	if _, ok := result["summary"]; !ok {
		t.Error("expected summary field")
	}
	if _, ok := result["timestamp"]; !ok {
		t.Error("expected timestamp field")
	}
}

func TestJSONReporterGenerateSummaryOnlyPretty(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONReporter(&buf, true)

	err := r.GenerateSummaryOnly(sampleReport())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "  ") {
		t.Error("expected pretty-printed JSON with indentation")
	}
}

func TestJSONReporterGenerateWithTrend(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONReporter(&buf, false)

	report := sampleReport()
	report.Trend = &models.Trend{
		Direction:      "improving",
		ChangePercent:  -20.0,
		PreviousIssues: 2,
		CurrentIssues:  1,
	}

	err := r.Generate(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "improving") {
		t.Error("expected trend direction in output")
	}
}
