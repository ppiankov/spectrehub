package reporter

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

func TestTextReporterGenerate(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextReporter(&buf)

	report := sampleReport()

	err := r.Generate(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	expectedFragments := []string{
		"SpectreHub Aggregated Report",
		"Overall Summary",
		"Total Tools: 1",
		"Total Issues: 1",
		"Health Score: WARNING",
		"vaultspectre",
	}

	for _, frag := range expectedFragments {
		if !strings.Contains(output, frag) {
			t.Errorf("expected output to contain %q", frag)
		}
	}
}

func TestTextReporterGenerateWithTrend(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextReporter(&buf)

	report := sampleReport()
	report.Trend = &models.Trend{
		Direction:      "improving",
		ChangePercent:  -20.0,
		PreviousIssues: 2,
		CurrentIssues:  1,
		ComparedWith:   time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC),
		ResolvedIssues: 1,
	}

	err := r.Generate(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Trend Analysis") {
		t.Error("expected Trend Analysis section")
	}
	if !strings.Contains(output, "improving") {
		t.Error("expected improving direction")
	}
	if !strings.Contains(output, "Resolved: 1") {
		t.Error("expected resolved count")
	}
}

func TestTextReporterGenerateWithNewIssues(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextReporter(&buf)

	report := sampleReport()
	report.Trend = &models.Trend{
		Direction:      "degrading",
		ChangePercent:  50.0,
		PreviousIssues: 1,
		CurrentIssues:  2,
		NewIssues:      1,
		ComparedWith:   time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC),
	}

	err := r.Generate(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "New Issues: 1") {
		t.Error("expected new issues count")
	}
}

func TestTextReporterVaultDetails(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextReporter(&buf)

	report := &models.AggregatedReport{
		Timestamp: time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC),
		Issues:    []models.NormalizedIssue{},
		ToolReports: map[string]models.ToolReport{
			"vaultspectre": {
				Tool:        "vaultspectre",
				Version:     "0.1.0",
				IsSupported: true,
				RawData: &models.VaultReport{
					Summary: models.VaultSummary{
						TotalReferences:    10,
						StatusOK:           7,
						StatusMissing:      1,
						StatusAccessDenied: 1,
						StaleSecrets:       1,
						StatusInvalid:      0,
						StatusError:        0,
						HealthScore:        "warning",
					},
				},
			},
		},
		Summary: models.CrossToolSummary{
			TotalIssues:      3,
			TotalTools:       1,
			SupportedTools:   1,
			IssuesByTool:     map[string]int{"vaultspectre": 3},
			IssuesByCategory: map[string]int{},
			IssuesBySeverity: map[string]int{},
		},
		Recommendations: []models.Recommendation{},
	}

	err := r.Generate(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Total References: 10") {
		t.Error("expected Total References")
	}
	if !strings.Contains(output, "Status OK: 7") {
		t.Error("expected Status OK")
	}
	if !strings.Contains(output, "Missing: 1") {
		t.Error("expected Missing count")
	}
}

func TestTextReporterS3Details(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextReporter(&buf)

	report := &models.AggregatedReport{
		Timestamp: time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC),
		Issues:    []models.NormalizedIssue{},
		ToolReports: map[string]models.ToolReport{
			"s3spectre": {
				Tool:        "s3spectre",
				Version:     "0.1.0",
				IsSupported: true,
				RawData: &models.S3Report{
					Summary: models.S3Summary{
						TotalBuckets:   5,
						OKBuckets:      3,
						MissingBuckets: []string{"b1"},
						UnusedBuckets:  []string{"b2"},
					},
				},
			},
		},
		Summary: models.CrossToolSummary{
			TotalIssues:      2,
			TotalTools:       1,
			SupportedTools:   1,
			IssuesByTool:     map[string]int{"s3spectre": 2},
			IssuesByCategory: map[string]int{},
			IssuesBySeverity: map[string]int{},
		},
		Recommendations: []models.Recommendation{},
	}

	err := r.Generate(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Total Buckets: 5") {
		t.Error("expected Total Buckets")
	}
}

func TestTextReporterUnsupportedTool(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextReporter(&buf)

	report := &models.AggregatedReport{
		Timestamp: time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC),
		Issues:    []models.NormalizedIssue{},
		ToolReports: map[string]models.ToolReport{
			"customspectre": {
				Tool:        "customspectre",
				Version:     "0.1.0",
				IsSupported: false,
			},
		},
		Summary: models.CrossToolSummary{
			TotalIssues:      0,
			TotalTools:       1,
			UnsupportedTools: 1,
			IssuesByTool:     map[string]int{},
			IssuesByCategory: map[string]int{},
			IssuesBySeverity: map[string]int{},
		},
		Recommendations: []models.Recommendation{},
	}

	err := r.Generate(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "UNSUPPORTED") {
		t.Error("expected UNSUPPORTED label")
	}
}

func TestToTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"missing", "Missing"},
		{"", ""},
		{"a", "A"},
		{"ALREADY", "ALREADY"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			if got := toTitle(tt.input); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestFormatTimestamp(t *testing.T) {
	ts := time.Date(2026, 2, 15, 10, 30, 45, 0, time.UTC)
	expected := "2026-02-15 10:30:45"
	if got := formatTimestamp(ts); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestTextReporterKafkaDetails(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextReporter(&buf)

	report := &models.AggregatedReport{
		Timestamp: time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC),
		Issues:    []models.NormalizedIssue{},
		ToolReports: map[string]models.ToolReport{
			"kafkaspectre": {
				Tool:        "kafkaspectre",
				Version:     "0.1.0",
				IsSupported: true,
				RawData: &models.KafkaReport{
					Summary: &models.KafkaSummary{
						ClusterName:        "prod-cluster",
						TotalTopics:        50,
						ActiveTopics:       45,
						UnusedTopics:       5,
						HighRiskCount:      1,
						MediumRiskCount:    2,
						ClusterHealthScore: "good",
						UnusedPercentage:   10.0,
					},
				},
			},
		},
		Summary: models.CrossToolSummary{
			TotalIssues:      5,
			TotalTools:       1,
			SupportedTools:   1,
			IssuesByTool:     map[string]int{"kafkaspectre": 5},
			IssuesByCategory: map[string]int{},
			IssuesBySeverity: map[string]int{},
		},
		Recommendations: []models.Recommendation{},
	}

	err := r.Generate(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Cluster: prod-cluster") {
		t.Error("expected cluster name")
	}
	if !strings.Contains(output, "Total Topics: 50") {
		t.Error("expected total topics")
	}
}

func TestTextReporterClickHouseDetails(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextReporter(&buf)

	report := &models.AggregatedReport{
		Timestamp: time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC),
		Issues:    []models.NormalizedIssue{},
		ToolReports: map[string]models.ToolReport{
			"clickspectre": {
				Tool:        "clickspectre",
				Version:     "0.1.0",
				IsSupported: true,
				RawData: &models.ClickHouseReport{
					Metadata: models.ClickMetadata{
						ClickHouseHost: "ch-prod-1",
						LookbackDays:   30,
					},
					Tables: []models.ClickTable{
						{FullName: "db.t1", ZeroUsage: true},
						{FullName: "db.t2", ZeroUsage: false},
					},
					Anomalies: []models.ClickAnomaly{
						{Type: "latency", Description: "slow"},
					},
				},
			},
		},
		Summary: models.CrossToolSummary{
			TotalIssues:      2,
			TotalTools:       1,
			SupportedTools:   1,
			IssuesByTool:     map[string]int{"clickspectre": 2},
			IssuesByCategory: map[string]int{},
			IssuesBySeverity: map[string]int{},
		},
		Recommendations: []models.Recommendation{},
	}

	err := r.Generate(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Host: ch-prod-1") {
		t.Error("expected host")
	}
	if !strings.Contains(output, "Tables Analyzed: 2") {
		t.Error("expected tables analyzed")
	}
	if !strings.Contains(output, "Zero Usage: 1") {
		t.Error("expected zero usage count")
	}
	if !strings.Contains(output, "Anomalies: 1") {
		t.Error("expected anomalies count")
	}
}
