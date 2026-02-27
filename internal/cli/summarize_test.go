package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

// --- printSparkline tests ---

func TestPrintSparklineEmpty(t *testing.T) {
	output := captureStdout(t, func() {
		printSparkline([]int{})
	})
	if output != "" {
		t.Errorf("printSparkline(empty) = %q, want empty", output)
	}
}

func TestPrintSparklineSingleValue(t *testing.T) {
	output := captureStdout(t, func() {
		printSparkline([]int{5})
	})
	// Single value should use middle char and show range
	if !strings.Contains(output, "[5 → 5]") {
		t.Errorf("printSparkline([5]) = %q, want range [5 → 5]", output)
	}
}

func TestPrintSparklineAscending(t *testing.T) {
	output := captureStdout(t, func() {
		printSparkline([]int{0, 3, 7, 10})
	})
	if !strings.Contains(output, "[0 → 10]") {
		t.Errorf("printSparkline ascending = %q, want range [0 → 10]", output)
	}
	// First char should be lowest block, last should be highest
	if !strings.Contains(output, "▁") {
		t.Error("expected lowest block ▁ for min value")
	}
	if !strings.Contains(output, "█") {
		t.Error("expected highest block █ for max value")
	}
}

func TestPrintSparklineAllSame(t *testing.T) {
	output := captureStdout(t, func() {
		printSparkline([]int{5, 5, 5})
	})
	// All same → all middle char
	if !strings.Contains(output, "[5 → 5]") {
		t.Errorf("printSparkline(all same) = %q, want range [5 → 5]", output)
	}
}

// --- printTrendSummaryText tests ---

func TestPrintTrendSummaryTextBasic(t *testing.T) {
	summary := &models.TrendSummary{
		TimeRange:      "2026-01-01 to 2026-02-01",
		RunsAnalyzed:   3,
		IssueSparkline: []int{10, 8, 5},
		ByTool: map[string]*models.ToolTrend{
			"vaultspectre": {
				Name:          "vaultspectre",
				CurrentIssues: 5,
				Change:        -3,
				ChangePercent: -37.5,
			},
		},
	}

	reports := []*models.AggregatedReport{
		{
			Timestamp: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			Summary: models.CrossToolSummary{
				TotalIssues: 10,
				HealthScore: "warning",
			},
		},
		{
			Timestamp: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			Summary: models.CrossToolSummary{
				TotalIssues: 8,
				HealthScore: "good",
			},
		},
		{
			Timestamp: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
			Summary: models.CrossToolSummary{
				TotalIssues: 5,
				HealthScore: "good",
			},
			Recommendations: []models.Recommendation{
				{Severity: "high", Action: "Fix missing secrets"},
			},
		},
	}

	output := captureStdout(t, func() {
		printTrendSummaryText(summary, reports)
	})

	if !strings.Contains(output, "SpectreHub Trend Summary") {
		t.Error("missing header")
	}
	if !strings.Contains(output, "2026-01-01 to 2026-02-01") {
		t.Error("missing time range")
	}
	if !strings.Contains(output, "Runs Analyzed: 3") {
		t.Error("missing runs count")
	}
	if !strings.Contains(output, "↓") {
		t.Error("missing improvement indicator ↓")
	}
	if !strings.Contains(output, "improved") {
		t.Error("missing 'improved' text")
	}
	if !strings.Contains(output, "vaultspectre") {
		t.Error("missing tool trend")
	}
	if !strings.Contains(output, "Fix missing secrets") {
		t.Error("missing recommendation")
	}
}

func TestPrintTrendSummaryTextSingleRun(t *testing.T) {
	summary := &models.TrendSummary{
		TimeRange:    "2026-02-01",
		RunsAnalyzed: 1,
	}

	reports := []*models.AggregatedReport{
		{
			Timestamp: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
			Summary: models.CrossToolSummary{
				TotalIssues: 5,
				HealthScore: "good",
			},
		},
	}

	output := captureStdout(t, func() {
		printTrendSummaryText(summary, reports)
	})

	// Single run should not show comparison indicators
	if strings.Contains(output, "↑") || strings.Contains(output, "↓") {
		t.Error("single run should not show trend indicators")
	}
}

func TestPrintTrendSummaryTextDegradation(t *testing.T) {
	summary := &models.TrendSummary{
		TimeRange:    "2026-01-01 to 2026-02-01",
		RunsAnalyzed: 2,
	}

	reports := []*models.AggregatedReport{
		{
			Timestamp: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			Summary: models.CrossToolSummary{
				TotalIssues: 5,
				HealthScore: "good",
			},
		},
		{
			Timestamp: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
			Summary: models.CrossToolSummary{
				TotalIssues: 10,
				HealthScore: "warning",
			},
		},
	}

	output := captureStdout(t, func() {
		printTrendSummaryText(summary, reports)
	})

	if !strings.Contains(output, "↑") {
		t.Error("missing degradation indicator ↑")
	}
	if !strings.Contains(output, "degraded") {
		t.Error("missing 'degraded' text")
	}
}
