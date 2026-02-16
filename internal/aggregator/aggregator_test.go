package aggregator

import (
	"math"
	"testing"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

func TestAggregatorAggregate(t *testing.T) {
	aggregator := New()
	ts := time.Date(2026, 2, 15, 13, 0, 0, 0, time.UTC)

	vaultReport := &models.VaultReport{
		Summary: models.VaultSummary{TotalReferences: 4},
		Secrets: map[string]*models.SecretInfo{
			"secret/missing": {Status: "missing", References: []models.VaultReference{{File: "a", Line: 1}}},
			"secret/ok":      {Status: "ok", References: []models.VaultReference{{File: "b", Line: 2}}},
		},
	}

	s3Report := &models.S3Report{
		Summary: models.S3Summary{TotalBuckets: 6},
		Buckets: map[string]*models.BucketAnalysis{
			"bucket-missing": {Status: "MISSING_BUCKET", Message: "missing"},
			"bucket-ok":      {Status: "OK"},
		},
	}

	tests := []struct {
		name            string
		reports         []models.ToolReport
		wantIssues      int
		wantTools       int
		wantSupported   int
		wantUnsupported int
		wantHealth      string
		wantScore       float64
		wantByTool      map[string]int
		wantByCategory  map[string]int
		wantBySeverity  map[string]int
	}{
		{
			name: "mixed supported and unsupported",
			reports: []models.ToolReport{
				{
					Tool:        string(models.ToolVault),
					Timestamp:   ts,
					IsSupported: true,
					RawData:     vaultReport,
				},
				{
					Tool:        string(models.ToolS3),
					Timestamp:   ts,
					IsSupported: true,
					RawData:     s3Report,
				},
				{
					Tool:        "unknownspectre",
					Timestamp:   ts,
					IsSupported: false,
				},
			},
			wantIssues:      2,
			wantTools:       3,
			wantSupported:   2,
			wantUnsupported: 1,
			wantHealth:      "warning",
			wantScore:       80.0,
			wantByTool: map[string]int{
				string(models.ToolVault): 1,
				string(models.ToolS3):    1,
			},
			wantByCategory: map[string]int{
				models.StatusMissing: 2,
			},
			wantBySeverity: map[string]int{
				models.SeverityCritical: 2,
			},
		},
		{
			name: "unsupported only",
			reports: []models.ToolReport{
				{
					Tool:        "unknownspectre",
					Timestamp:   ts,
					IsSupported: false,
				},
			},
			wantIssues:      0,
			wantTools:       1,
			wantSupported:   0,
			wantUnsupported: 1,
			wantHealth:      "unknown",
			wantScore:       0.0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			report, err := aggregator.Aggregate(tt.reports)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if report.Summary.TotalIssues != tt.wantIssues {
				t.Fatalf("expected %d total issues, got %d", tt.wantIssues, report.Summary.TotalIssues)
			}
			if report.Summary.TotalTools != tt.wantTools {
				t.Fatalf("expected %d tools, got %d", tt.wantTools, report.Summary.TotalTools)
			}
			if report.Summary.SupportedTools != tt.wantSupported {
				t.Fatalf("expected %d supported tools, got %d", tt.wantSupported, report.Summary.SupportedTools)
			}
			if report.Summary.UnsupportedTools != tt.wantUnsupported {
				t.Fatalf("expected %d unsupported tools, got %d", tt.wantUnsupported, report.Summary.UnsupportedTools)
			}
			if report.Summary.HealthScore != tt.wantHealth {
				t.Fatalf("expected health score %q, got %q", tt.wantHealth, report.Summary.HealthScore)
			}
			if math.Abs(report.Summary.ScorePercent-tt.wantScore) > 0.01 {
				t.Fatalf("expected score %.2f, got %.2f", tt.wantScore, report.Summary.ScorePercent)
			}
			for tool, want := range tt.wantByTool {
				if report.Summary.IssuesByTool[tool] != want {
					t.Fatalf("expected %d issues for %s, got %d", want, tool, report.Summary.IssuesByTool[tool])
				}
			}
			for category, want := range tt.wantByCategory {
				if report.Summary.IssuesByCategory[category] != want {
					t.Fatalf("expected %d issues for category %s, got %d", want, category, report.Summary.IssuesByCategory[category])
				}
			}
			for severity, want := range tt.wantBySeverity {
				if report.Summary.IssuesBySeverity[severity] != want {
					t.Fatalf("expected %d issues for severity %s, got %d", want, severity, report.Summary.IssuesBySeverity[severity])
				}
			}
		})
	}
}

func TestAggregatorAddTrend(t *testing.T) {
	aggregator := New()

	tests := []struct {
		name          string
		currentTotal  int
		previousTotal int
		previousNil   bool
		direction     string
		changePercent float64
		newIssues     int
		resolved      int
	}{
		{
			name:         "no previous",
			currentTotal: 2,
			previousNil:  true,
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
				Summary: models.CrossToolSummary{TotalIssues: tt.currentTotal},
			}

			var previous *models.AggregatedReport
			if !tt.previousNil {
				previous = &models.AggregatedReport{
					Summary: models.CrossToolSummary{TotalIssues: tt.previousTotal},
				}
			}

			aggregator.AddTrend(current, previous)
			if tt.previousNil {
				if current.Trend != nil {
					t.Fatalf("expected nil trend, got %+v", current.Trend)
				}
				return
			}

			if current.Trend == nil {
				t.Fatalf("expected trend, got nil")
			}
			if current.Trend.Direction != tt.direction {
				t.Fatalf("expected direction %s, got %s", tt.direction, current.Trend.Direction)
			}
			if math.Abs(current.Trend.ChangePercent-tt.changePercent) > 0.01 {
				t.Fatalf("expected change percent %.2f, got %.2f", tt.changePercent, current.Trend.ChangePercent)
			}
			if current.Trend.NewIssues != tt.newIssues {
				t.Fatalf("expected new issues %d, got %d", tt.newIssues, current.Trend.NewIssues)
			}
			if current.Trend.ResolvedIssues != tt.resolved {
				t.Fatalf("expected resolved issues %d, got %d", tt.resolved, current.Trend.ResolvedIssues)
			}
		})
	}
}
