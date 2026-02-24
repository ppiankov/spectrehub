package aggregator

import (
	"fmt"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

// Aggregator merges reports from multiple tools
type Aggregator struct {
	normalizer *Normalizer
}

// New creates a new aggregator
func New() *Aggregator {
	return &Aggregator{
		normalizer: NewNormalizer(),
	}
}

// Aggregate combines multiple tool reports into a unified report
func (a *Aggregator) Aggregate(toolReports []models.ToolReport) (*models.AggregatedReport, error) {
	// Initialize aggregated report
	report := &models.AggregatedReport{
		Timestamp:   time.Now(),
		Issues:      []models.NormalizedIssue{},
		ToolReports: make(map[string]models.ToolReport),
		Summary: models.CrossToolSummary{
			IssuesByTool:     make(map[string]int),
			IssuesByCategory: make(map[string]int),
			IssuesBySeverity: make(map[string]int),
		},
		Recommendations: []models.Recommendation{},
	}

	// Process each tool report
	for _, toolReport := range toolReports {
		// Store raw tool report
		report.ToolReports[toolReport.Tool] = toolReport

		// Normalize if supported
		if toolReport.IsSupported {
			issues, err := a.normalizer.Normalize(&toolReport)
			if err != nil {
				return nil, fmt.Errorf("failed to normalize %s: %w", toolReport.Tool, err)
			}

			// Add normalized issues
			report.Issues = append(report.Issues, issues...)

			// Update tool-specific counts
			report.ToolReports[toolReport.Tool] = toolReport
			toolReport.IssueCount = len(issues)
		}
	}

	// Calculate summary statistics
	a.calculateSummary(report)

	// Generate health score
	a.calculateHealthScore(report)

	return report, nil
}

// calculateSummary computes summary statistics from normalized issues
func (a *Aggregator) calculateSummary(report *models.AggregatedReport) {
	// Count issues by tool
	for _, issue := range report.Issues {
		report.Summary.IssuesByTool[issue.Tool]++
		report.Summary.IssuesByCategory[issue.Category]++
		report.Summary.IssuesBySeverity[issue.Severity]++
	}

	// Total issues
	report.Summary.TotalIssues = len(report.Issues)

	// Count tools
	report.Summary.TotalTools = len(report.ToolReports)
	for _, toolReport := range report.ToolReports {
		if toolReport.IsSupported {
			report.Summary.SupportedTools++
		} else {
			report.Summary.UnsupportedTools++
		}
	}
}

// calculateHealthScore determines overall health based on issues
func (a *Aggregator) calculateHealthScore(report *models.AggregatedReport) {
	// Calculate total resources across all tools
	totalResources := 0
	for _, toolReport := range report.ToolReports {
		if !toolReport.IsSupported {
			continue
		}

		// Count resources from each tool
		switch models.ToolType(toolReport.Tool) {
		case models.ToolVault:
			if vr, ok := toolReport.RawData.(*models.VaultReport); ok {
				totalResources += vr.Summary.TotalReferences
			}
		case models.ToolS3:
			if sr, ok := toolReport.RawData.(*models.S3Report); ok {
				totalResources += sr.Summary.TotalBuckets
			}
		case models.ToolKafka:
			if kr, ok := toolReport.RawData.(*models.KafkaReport); ok {
				if kr.Summary != nil {
					totalResources += kr.Summary.TotalTopics
				}
			}
		case models.ToolClickHouse:
			if cr, ok := toolReport.RawData.(*models.ClickHouseReport); ok {
				totalResources += len(cr.Tables)
			}
		case models.ToolPg:
			if pr, ok := toolReport.RawData.(*models.PgReport); ok {
				totalResources += pr.Scanned.Tables
			}
		}
	}

	// Count distinct affected resources (resources with at least one issue).
	affected := make(map[string]bool)
	for _, issue := range report.Issues {
		if issue.Resource != "" {
			affected[issue.Resource] = true
		}
	}

	// Calculate health score: clean resources / total resources.
	healthLevel, scorePercent := models.CalculateHealthScore(len(affected), totalResources)
	report.Summary.HealthScore = healthLevel
	report.Summary.ScorePercent = scorePercent
}

// AddTrend adds trend information by comparing with a previous report
func (a *Aggregator) AddTrend(current *models.AggregatedReport, previous *models.AggregatedReport) {
	if previous == nil {
		return
	}

	trend := &models.Trend{
		PreviousIssues: previous.Summary.TotalIssues,
		CurrentIssues:  current.Summary.TotalIssues,
		ComparedWith:   previous.Timestamp,
	}

	// Calculate change
	change := current.Summary.TotalIssues - previous.Summary.TotalIssues

	// Determine direction
	if change < 0 {
		trend.Direction = "improving"
		trend.ChangePercent = float64(change) / float64(previous.Summary.TotalIssues) * 100.0
	} else if change > 0 {
		trend.Direction = "degrading"
		trend.ChangePercent = float64(change) / float64(previous.Summary.TotalIssues) * 100.0
	} else {
		trend.Direction = "stable"
		trend.ChangePercent = 0.0
	}

	// Calculate new and resolved issues (simple count-based for MVP)
	trend.NewIssues = max(0, change)
	trend.ResolvedIssues = max(0, -change)

	current.Trend = trend
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
