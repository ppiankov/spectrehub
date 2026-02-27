package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

func TestBuildExplanation(t *testing.T) {
	report := &models.AggregatedReport{
		Timestamp: time.Now(),
		Issues: []models.NormalizedIssue{
			{Tool: "vaultspectre", Category: "missing", Severity: "critical", Resource: "secret/db"},
			{Tool: "vaultspectre", Category: "stale", Severity: "low", Resource: "secret/cache"},
			{Tool: "s3spectre", Category: "unused", Severity: "medium", Resource: "s3://old-bucket"},
			{Tool: "vaultspectre", Category: "info", Severity: "low", Resource: ""}, // empty resource → skipped in affected count
		},
		ToolReports: map[string]models.ToolReport{
			"vaultspectre": {
				Tool:        "vaultspectre",
				IsSupported: true,
				RawData: &models.VaultReport{
					Summary: models.VaultSummary{TotalReferences: 10},
				},
			},
			"s3spectre": {
				Tool:        "s3spectre",
				IsSupported: true,
				RawData: &models.S3Report{
					Summary: models.S3Summary{TotalBuckets: 5},
				},
			},
			"unsupported-tool": {
				Tool:        "unsupported-tool",
				IsSupported: false,
			},
		},
		Summary: models.CrossToolSummary{
			TotalIssues:  3,
			HealthScore:  "warning",
			ScorePercent: 73.3,
			IssuesBySeverity: map[string]int{
				"critical": 1,
				"medium":   1,
				"low":      1,
			},
		},
	}

	result := buildExplanation(report)

	if result.TotalResources != 15 {
		t.Errorf("TotalResources = %d, want 15", result.TotalResources)
	}
	if result.AffectedCount != 3 {
		t.Errorf("AffectedCount = %d, want 3", result.AffectedCount)
	}
	if len(result.PerTool) != 2 {
		t.Fatalf("PerTool = %d tools, want 2", len(result.PerTool))
	}
	if result.Health != "warning" {
		t.Errorf("Health = %q, want %q", result.Health, "warning")
	}
	if len(result.Thresholds) != 5 {
		t.Errorf("Thresholds = %d, want 5", len(result.Thresholds))
	}
	if result.IssuesBySeverity["critical"] != 1 {
		t.Errorf("IssuesBySeverity[critical] = %d, want 1", result.IssuesBySeverity["critical"])
	}
	// PerTool sorted alphabetically
	if result.PerTool[0].Tool != "s3spectre" {
		t.Errorf("PerTool[0].Tool = %q, want s3spectre (sorted)", result.PerTool[0].Tool)
	}
}

func TestBuildExplanationEmptyReport(t *testing.T) {
	report := &models.AggregatedReport{
		Timestamp:   time.Now(),
		Issues:      []models.NormalizedIssue{},
		ToolReports: map[string]models.ToolReport{},
		Summary:     models.CrossToolSummary{},
	}

	result := buildExplanation(report)

	if result.TotalResources != 0 {
		t.Errorf("TotalResources = %d, want 0", result.TotalResources)
	}
	if result.AffectedCount != 0 {
		t.Errorf("AffectedCount = %d, want 0", result.AffectedCount)
	}
}

func TestCountToolResources(t *testing.T) {
	tests := []struct {
		name string
		tr   models.ToolReport
		want int
	}{
		{
			name: "vault",
			tr: models.ToolReport{
				Tool:    "vaultspectre",
				RawData: &models.VaultReport{Summary: models.VaultSummary{TotalReferences: 42}},
			},
			want: 42,
		},
		{
			name: "s3",
			tr: models.ToolReport{
				Tool:    "s3spectre",
				RawData: &models.S3Report{Summary: models.S3Summary{TotalBuckets: 7}},
			},
			want: 7,
		},
		{
			name: "kafka",
			tr: models.ToolReport{
				Tool:    "kafkaspectre",
				RawData: &models.KafkaReport{Summary: &models.KafkaSummary{TotalTopics: 15}},
			},
			want: 15,
		},
		{
			name: "clickhouse",
			tr: models.ToolReport{
				Tool: "clickspectre",
				RawData: &models.ClickHouseReport{
					Tables: []models.ClickTable{{Name: "a"}, {Name: "b"}, {Name: "c"}},
				},
			},
			want: 3,
		},
		{
			name: "pg",
			tr: models.ToolReport{
				Tool:    "pgspectre",
				RawData: &models.PgReport{Scanned: models.PgScanContext{Tables: 20}},
			},
			want: 20,
		},
		{
			name: "unknown_tool",
			tr: models.ToolReport{
				Tool:    "unknown",
				RawData: nil,
			},
			want: 0,
		},
		{
			name: "kafka_nil_summary",
			tr: models.ToolReport{
				Tool:    "kafkaspectre",
				RawData: &models.KafkaReport{Summary: nil},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countToolResources(tt.tr)
			if got != tt.want {
				t.Errorf("countToolResources(%s) = %d, want %d", tt.name, got, tt.want)
			}
		})
	}
}

func TestWriteExplainText(t *testing.T) {
	result := explainResult{
		PerTool: []toolContribution{
			{Tool: "vaultspectre", Resources: 10, Issues: 2, Affected: 2},
		},
		TotalResources: 10,
		AffectedCount:  2,
		AffectedList:   []string{"secret/db", "secret/cache"},
		Score:          80.0,
		Health:         "good",
		Formula:        "(10 - 2) / 10 * 100 = 80.0",
		Thresholds: []threshold{
			{Min: 95, Label: "excellent"},
			{Min: 85, Label: "good"},
			{Min: 70, Label: "warning"},
			{Min: 50, Label: "critical"},
			{Min: 0, Label: "severe"},
		},
		IssuesBySeverity: map[string]int{
			"critical": 1,
			"low":      1,
		},
	}

	output := captureStdout(t, func() {
		_ = writeExplainText(result)
	})

	if !strings.Contains(output, "Health Score Breakdown") {
		t.Error("missing header")
	}
	if !strings.Contains(output, "vaultspectre") {
		t.Error("missing tool name")
	}
	if !strings.Contains(output, "(10 - 2) / 10 * 100 = 80.0") {
		t.Error("missing formula")
	}
	if !strings.Contains(output, "secret/db") {
		t.Error("missing affected resource")
	}
	if !strings.Contains(output, "GOOD") {
		t.Error("missing health result")
	}
	// Arrow indicator for matching threshold
	if !strings.Contains(output, "→") {
		t.Error("missing threshold indicator")
	}
}

func TestWriteExplainTextManyAffected(t *testing.T) {
	// When >20 affected, should truncate to 15 + "... +N more"
	affected := make([]string, 25)
	for i := range affected {
		affected[i] = "resource-" + string(rune('a'+i))
	}

	result := explainResult{
		PerTool:        []toolContribution{},
		TotalResources: 100,
		AffectedCount:  25,
		AffectedList:   affected,
		Score:          75.0,
		Health:         "warning",
		Formula:        "(100 - 25) / 100 * 100 = 75.0",
		Thresholds: []threshold{
			{Min: 95, Label: "excellent"},
			{Min: 85, Label: "good"},
			{Min: 70, Label: "warning"},
			{Min: 50, Label: "critical"},
			{Min: 0, Label: "severe"},
		},
	}

	output := captureStdout(t, func() {
		_ = writeExplainText(result)
	})

	if !strings.Contains(output, "+10 more") {
		t.Error("expected truncation indicator '+10 more'")
	}
}
