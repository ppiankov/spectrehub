package aggregator

import (
	"testing"

	"github.com/ppiankov/spectrehub/internal/models"
)

func TestRecommendationGeneratorGenerateRecommendations(t *testing.T) {
	generator := NewRecommendationGenerator()

	tests := []struct {
		name           string
		issues         []models.NormalizedIssue
		wantCount      int
		wantFirst      models.Recommendation
		wantSecond     models.Recommendation
		wantThird      models.Recommendation
		expectOrdering bool
	}{
		{
			name: "grouping and sorting",
			issues: []models.NormalizedIssue{
				{Tool: string(models.ToolVault), Category: models.StatusMissing, Severity: models.SeverityCritical, Count: 2},
				{Tool: string(models.ToolVault), Category: models.StatusMissing, Severity: models.SeverityCritical, Count: 3},
				{Tool: string(models.ToolS3), Category: models.StatusUnused, Severity: models.SeverityHigh, Count: 1},
				{Tool: string(models.ToolClickHouse), Category: models.StatusMisconfig, Severity: models.SeverityMedium, Count: 4},
			},
			wantCount: 3,
			wantFirst: models.Recommendation{
				Severity: models.SeverityCritical,
				Tool:     string(models.ToolVault),
				Action:   "Fix 5 missing Vault secrets",
				Impact:   "Services may fail to start or operate incorrectly",
				Count:    5,
			},
			wantSecond: models.Recommendation{
				Severity: models.SeverityHigh,
				Tool:     string(models.ToolS3),
				Action:   "Clean up 1 unused S3 buckets/prefixes",
				Impact:   "Significant waste of resources and potential security risks",
				Count:    1,
			},
			wantThird: models.Recommendation{
				Severity: models.SeverityMedium,
				Tool:     string(models.ToolClickHouse),
				Action:   "Fix 4 misconfiguration(s) in clickspectre",
				Impact:   "Suboptimal performance or behavior",
				Count:    4,
			},
			expectOrdering: true,
		},
		{
			name:           "no issues",
			issues:         nil,
			wantCount:      0,
			expectOrdering: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			report := &models.AggregatedReport{Issues: tt.issues}
			recs := generator.GenerateRecommendations(report)
			if len(recs) != tt.wantCount {
				t.Fatalf("expected %d recommendations, got %d", tt.wantCount, len(recs))
			}
			if !tt.expectOrdering {
				return
			}
			if recs[0] != tt.wantFirst {
				t.Fatalf("unexpected first recommendation: %+v", recs[0])
			}
			if recs[1] != tt.wantSecond {
				t.Fatalf("unexpected second recommendation: %+v", recs[1])
			}
			if recs[2] != tt.wantThird {
				t.Fatalf("unexpected third recommendation: %+v", recs[2])
			}
		})
	}
}

func TestRecommendationGeneratorGenerateAction(t *testing.T) {
	generator := NewRecommendationGenerator()

	tests := []struct {
		name     string
		group    issueGroup
		expected string
	}{
		{"missing", issueGroup{tool: string(models.ToolVault), category: models.StatusMissing, count: 2}, "Fix 2 missing Vault secrets"},
		{"unused", issueGroup{tool: string(models.ToolS3), category: models.StatusUnused, count: 1}, "Clean up 1 unused S3 buckets/prefixes"},
		{"stale", issueGroup{tool: string(models.ToolKafka), category: models.StatusStale, count: 3}, "Review 3 stale Kafka topics"},
		{"error", issueGroup{tool: string(models.ToolKafka), category: models.StatusError, count: 4}, "Investigate 4 error(s) in kafkaspectre"},
		{"misconfig", issueGroup{tool: string(models.ToolS3), category: models.StatusMisconfig, count: 5}, "Fix 5 misconfiguration(s) in s3spectre"},
		{"access_deny", issueGroup{tool: string(models.ToolVault), category: models.StatusAccessDeny, count: 2}, "Restore access to 2 Vault secrets"},
		{"invalid", issueGroup{tool: string(models.ToolClickHouse), category: models.StatusInvalid, count: 1}, "Fix 1 invalid ClickHouse tables"},
		{"drift", issueGroup{tool: string(models.ToolClickHouse), category: models.StatusDrift, count: 7}, "Resolve 7 drift issue(s) in clickspectre"},
		{"default", issueGroup{tool: string(models.ToolVault), category: "other", count: 9}, "Address 9 issue(s) in vaultspectre"},
		{"resource_fallback", issueGroup{tool: "customspectre", category: models.StatusUnused, count: 2}, "Clean up 2 unused resources"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := generator.generateAction(&tt.group); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestRecommendationGeneratorGenerateImpact(t *testing.T) {
	generator := NewRecommendationGenerator()

	tests := []struct {
		name     string
		group    issueGroup
		expected string
	}{
		{"critical_missing", issueGroup{severity: models.SeverityCritical, category: models.StatusMissing}, "Services may fail to start or operate incorrectly"},
		{"critical_access", issueGroup{severity: models.SeverityCritical, category: models.StatusAccessDeny}, "Critical operations are blocked"},
		{"critical_error", issueGroup{severity: models.SeverityCritical, category: models.StatusError}, "System integrity is compromised"},
		{"critical_default", issueGroup{severity: models.SeverityCritical, category: "other"}, "Immediate action required to prevent outages"},
		{"high_missing", issueGroup{severity: models.SeverityHigh, category: models.StatusMissing}, "Important features may not work as expected"},
		{"high_unused", issueGroup{severity: models.SeverityHigh, category: models.StatusUnused}, "Significant waste of resources and potential security risks"},
		{"high_stale", issueGroup{severity: models.SeverityHigh, category: models.StatusStale}, "Data may be outdated or invalid"},
		{"high_misconfig", issueGroup{severity: models.SeverityHigh, category: models.StatusMisconfig}, "System behavior may be unpredictable"},
		{"high_default", issueGroup{severity: models.SeverityHigh, category: "other"}, "Significant impact on system reliability"},
		{"medium_unused", issueGroup{severity: models.SeverityMedium, category: models.StatusUnused}, "Resources are wasted but no immediate risk"},
		{"medium_stale", issueGroup{severity: models.SeverityMedium, category: models.StatusStale}, "Data quality may degrade over time"},
		{"medium_misconfig", issueGroup{severity: models.SeverityMedium, category: models.StatusMisconfig}, "Suboptimal performance or behavior"},
		{"medium_default", issueGroup{severity: models.SeverityMedium, category: "other"}, "Moderate impact on system efficiency"},
		{"low_unused", issueGroup{severity: models.SeverityLow, category: models.StatusUnused}, "Minor cleanup to improve maintainability"},
		{"low_stale", issueGroup{severity: models.SeverityLow, category: models.StatusStale}, "Consider updating or removing"},
		{"low_default", issueGroup{severity: models.SeverityLow, category: "other"}, "Low priority cleanup or optimization"},
		{"unknown_severity", issueGroup{severity: "unknown", category: models.StatusUnused}, "Review and address as needed"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := generator.generateImpact(&tt.group); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestRecommendationGeneratorGetResourceName(t *testing.T) {
	generator := NewRecommendationGenerator()

	tests := []struct {
		name     string
		tool     string
		expected string
	}{
		{"vault", string(models.ToolVault), "Vault secrets"},
		{"s3", string(models.ToolS3), "S3 buckets/prefixes"},
		{"kafka", string(models.ToolKafka), "Kafka topics"},
		{"clickhouse", string(models.ToolClickHouse), "ClickHouse tables"},
		{"default", "customspectre", "resources"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := generator.getResourceName(tt.tool); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestRecommendationGeneratorSeverityPriority(t *testing.T) {
	generator := NewRecommendationGenerator()

	tests := []struct {
		severity string
		expected int
	}{
		{models.SeverityCritical, 4},
		{models.SeverityHigh, 3},
		{models.SeverityMedium, 2},
		{models.SeverityLow, 1},
		{"unknown", 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.severity, func(t *testing.T) {
			if got := generator.severityPriority(tt.severity); got != tt.expected {
				t.Fatalf("expected %d, got %d", tt.expected, got)
			}
		})
	}
}

func TestRecommendationGeneratorGetTopRecommendations(t *testing.T) {
	generator := NewRecommendationGenerator()
	recs := []models.Recommendation{
		{Severity: models.SeverityCritical, Tool: string(models.ToolVault)},
		{Severity: models.SeverityHigh, Tool: string(models.ToolS3)},
		{Severity: models.SeverityLow, Tool: string(models.ToolKafka)},
	}

	tests := []struct {
		name      string
		n         int
		wantCount int
	}{
		{"all", 5, 3},
		{"subset", 2, 2},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			top := generator.GetTopRecommendations(recs, tt.n)
			if len(top) != tt.wantCount {
				t.Fatalf("expected %d recommendations, got %d", tt.wantCount, len(top))
			}
		})
	}
}

func TestRecommendationGeneratorGroupBySeverity(t *testing.T) {
	generator := NewRecommendationGenerator()

	recs := []models.Recommendation{
		{Severity: models.SeverityHigh, Tool: string(models.ToolVault)},
		{Severity: models.SeverityHigh, Tool: string(models.ToolS3)},
		{Severity: models.SeverityLow, Tool: string(models.ToolKafka)},
	}

	tests := []struct {
		name     string
		recs     []models.Recommendation
		expected map[string]int
	}{
		{
			name: "grouped",
			recs: recs,
			expected: map[string]int{
				models.SeverityHigh: 2,
				models.SeverityLow:  1,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			grouped := generator.GroupBySeverity(tt.recs)
			for severity, wantCount := range tt.expected {
				if len(grouped[severity]) != wantCount {
					t.Fatalf("expected %d for %s, got %d", wantCount, severity, len(grouped[severity]))
				}
			}
		})
	}
}
