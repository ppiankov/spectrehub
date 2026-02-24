package cli

import (
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
