package aggregator

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

func TestTrendAnalyzerCalculateTrend(t *testing.T) {
	analyzer := NewTrendAnalyzer()
	ts := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		currentTotal  int
		previousTotal int
		previousNil   bool
		wantNil       bool
		direction     string
		changePercent float64
		newIssues     int
		resolved      int
	}{
		{
			name:         "no previous",
			currentTotal: 3,
			previousNil:  true,
			wantNil:      true,
		},
		{
			name:          "improving",
			currentTotal:  3,
			previousTotal: 5,
			direction:     "improving",
			changePercent: -40.0,
			newIssues:     0,
			resolved:      2,
		},
		{
			name:          "degrading",
			currentTotal:  6,
			previousTotal: 4,
			direction:     "degrading",
			changePercent: 50.0,
			newIssues:     2,
			resolved:      0,
		},
		{
			name:          "stable",
			currentTotal:  4,
			previousTotal: 4,
			direction:     "stable",
			changePercent: 0.0,
			newIssues:     0,
			resolved:      0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			current := &models.AggregatedReport{
				Timestamp: ts.Add(24 * time.Hour),
				Summary:   models.CrossToolSummary{TotalIssues: tt.currentTotal},
			}

			var previous *models.AggregatedReport
			if !tt.previousNil {
				previous = &models.AggregatedReport{
					Timestamp: ts,
					Summary:   models.CrossToolSummary{TotalIssues: tt.previousTotal},
				}
			}

			trend := analyzer.CalculateTrend(current, previous)
			if tt.wantNil {
				if trend != nil {
					t.Fatalf("expected nil trend, got %+v", trend)
				}
				return
			}
			if trend == nil {
				t.Fatalf("expected trend, got nil")
			}
			if trend.Direction != tt.direction {
				t.Fatalf("expected direction %s, got %s", tt.direction, trend.Direction)
			}
			if math.Abs(trend.ChangePercent-tt.changePercent) > 0.01 {
				t.Fatalf("expected change percent %.2f, got %.2f", tt.changePercent, trend.ChangePercent)
			}
			if trend.NewIssues != tt.newIssues {
				t.Fatalf("expected new issues %d, got %d", tt.newIssues, trend.NewIssues)
			}
			if trend.ResolvedIssues != tt.resolved {
				t.Fatalf("expected resolved issues %d, got %d", tt.resolved, trend.ResolvedIssues)
			}
		})
	}
}

func TestTrendAnalyzerAnalyzeLastNRuns(t *testing.T) {
	analyzer := NewTrendAnalyzer()
	base := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		runs          []*models.AggregatedReport
		wantNil       bool
		wantRange     string
		wantSparkline []int
		wantByTool    map[string]*models.ToolTrend
	}{
		{
			name:    "no runs",
			runs:    nil,
			wantNil: true,
		},
		{
			name:      "single run",
			runs:      []*models.AggregatedReport{{Timestamp: base, Summary: models.CrossToolSummary{TotalIssues: 2}}},
			wantRange: "Single run",
			wantSparkline: []int{
				2,
			},
			wantByTool: map[string]*models.ToolTrend{},
		},
		{
			name: "multiple runs",
			runs: []*models.AggregatedReport{
				{
					Timestamp: base,
					Summary: models.CrossToolSummary{
						TotalIssues: 2,
						IssuesByTool: map[string]int{
							string(models.ToolVault): 2,
						},
					},
				},
				{
					Timestamp: base.Add(48 * time.Hour),
					Summary: models.CrossToolSummary{
						TotalIssues: 5,
						IssuesByTool: map[string]int{
							string(models.ToolVault): 1,
							string(models.ToolS3):    3,
						},
					},
				},
			},
			wantRange: "Last 2 days",
			wantSparkline: []int{
				2, 5,
			},
			wantByTool: map[string]*models.ToolTrend{
				string(models.ToolVault): {
					Name:           string(models.ToolVault),
					CurrentIssues:  1,
					PreviousIssues: 2,
					Change:         -1,
					ChangePercent:  -50.0,
				},
				string(models.ToolS3): {
					Name:           string(models.ToolS3),
					CurrentIssues:  3,
					PreviousIssues: 0,
					Change:         3,
					ChangePercent:  100.0,
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			summary := analyzer.AnalyzeLastNRuns(tt.runs)
			if tt.wantNil {
				if summary != nil {
					t.Fatalf("expected nil summary, got %+v", summary)
				}
				return
			}
			if summary == nil {
				t.Fatalf("expected summary, got nil")
			}
			if summary.TimeRange != tt.wantRange {
				t.Fatalf("expected time range %q, got %q", tt.wantRange, summary.TimeRange)
			}
			if len(summary.IssueSparkline) != len(tt.wantSparkline) {
				t.Fatalf("expected sparkline length %d, got %d", len(tt.wantSparkline), len(summary.IssueSparkline))
			}
			for i := range tt.wantSparkline {
				if summary.IssueSparkline[i] != tt.wantSparkline[i] {
					t.Fatalf("expected sparkline value %d at %d, got %d", tt.wantSparkline[i], i, summary.IssueSparkline[i])
				}
			}
			for tool, wantTrend := range tt.wantByTool {
				gotTrend, ok := summary.ByTool[tool]
				if !ok {
					t.Fatalf("missing trend for tool %s", tool)
				}
				if gotTrend.Change != wantTrend.Change || gotTrend.ChangePercent != wantTrend.ChangePercent ||
					gotTrend.CurrentIssues != wantTrend.CurrentIssues || gotTrend.PreviousIssues != wantTrend.PreviousIssues {
					t.Fatalf("unexpected trend for %s: %+v", tool, gotTrend)
				}
			}
		})
	}
}

func TestTrendAnalyzerGenerateComparisonReport(t *testing.T) {
	analyzer := NewTrendAnalyzer()
	base := time.Date(2026, 2, 12, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		current        *models.AggregatedReport
		previous       *models.AggregatedReport
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:     "no previous",
			current:  &models.AggregatedReport{Timestamp: base},
			previous: nil,
			wantContains: []string{
				"No previous run to compare with",
			},
		},
		{
			name: "new issues",
			current: &models.AggregatedReport{
				Timestamp: base.Add(24 * time.Hour),
				Summary: models.CrossToolSummary{
					TotalIssues: 4,
					IssuesByTool: map[string]int{
						string(models.ToolVault): 3,
						string(models.ToolS3):    1,
					},
				},
			},
			previous: &models.AggregatedReport{
				Timestamp: base,
				Summary: models.CrossToolSummary{
					TotalIssues: 2,
					IssuesByTool: map[string]int{
						string(models.ToolVault): 1,
						string(models.ToolS3):    1,
					},
				},
			},
			wantContains: []string{
				"Comparison: " + formatDate(base.Add(24*time.Hour)) + " vs " + formatDate(base),
				"Overall: 2 \u2192 4 issues",
				"vaultspectre:",
				"  1 \u2192 3 (+2)",
				"New Issues: 2",
			},
		},
		{
			name: "resolved issues",
			current: &models.AggregatedReport{
				Timestamp: base.Add(24 * time.Hour),
				Summary: models.CrossToolSummary{
					TotalIssues: 2,
					IssuesByTool: map[string]int{
						string(models.ToolVault): 1,
					},
				},
			},
			previous: &models.AggregatedReport{
				Timestamp: base,
				Summary: models.CrossToolSummary{
					TotalIssues: 5,
					IssuesByTool: map[string]int{
						string(models.ToolVault): 3,
					},
				},
			},
			wantContains: []string{
				"Overall: 5 \u2192 2 issues",
				"  3 \u2192 1 (-2)",
				"Resolved Issues: 3",
			},
			wantNotContain: []string{
				"New Issues:",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			report := analyzer.GenerateComparisonReport(tt.current, tt.previous)
			for _, expected := range tt.wantContains {
				if !strings.Contains(report, expected) {
					t.Fatalf("expected report to contain %q, got %q", expected, report)
				}
			}
			for _, unexpected := range tt.wantNotContain {
				if strings.Contains(report, unexpected) {
					t.Fatalf("expected report to not contain %q, got %q", unexpected, report)
				}
			}
		})
	}
}

func TestGetTrendIndicator(t *testing.T) {
	tests := []struct {
		direction string
		expected  string
	}{
		{"improving", "\u2193"},
		{"degrading", "\u2191"},
		{"stable", "\u2192"},
		{"unknown", "?"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.direction, func(t *testing.T) {
			if got := GetTrendIndicator(tt.direction); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
