package aggregator

import (
	"fmt"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

// TrendAnalyzer analyzes trends across multiple runs
type TrendAnalyzer struct{}

// NewTrendAnalyzer creates a new trend analyzer
func NewTrendAnalyzer() *TrendAnalyzer {
	return &TrendAnalyzer{}
}

// CalculateTrend compares current report with previous one
func (t *TrendAnalyzer) CalculateTrend(current, previous *models.AggregatedReport) *models.Trend {
	if previous == nil {
		return nil
	}

	trend := &models.Trend{
		PreviousIssues: previous.Summary.TotalIssues,
		CurrentIssues:  current.Summary.TotalIssues,
		ComparedWith:   previous.Timestamp,
	}

	// Calculate change
	change := current.Summary.TotalIssues - previous.Summary.TotalIssues

	// Determine direction and percentage
	if previous.Summary.TotalIssues > 0 {
		trend.ChangePercent = float64(change) / float64(previous.Summary.TotalIssues) * 100.0
	}

	if change < 0 {
		trend.Direction = "improving"
		trend.ResolvedIssues = -change
	} else if change > 0 {
		trend.Direction = "degrading"
		trend.NewIssues = change
	} else {
		trend.Direction = "stable"
	}

	return trend
}

// AnalyzeLastNRuns analyzes trends across last N runs
func (t *TrendAnalyzer) AnalyzeLastNRuns(runs []*models.AggregatedReport) *models.TrendSummary {
	if len(runs) == 0 {
		return nil
	}

	summary := &models.TrendSummary{
		RunsAnalyzed: len(runs),
		ByTool:       make(map[string]*models.ToolTrend),
	}

	// Determine time range
	if len(runs) > 1 {
		earliest := runs[0].Timestamp
		latest := runs[len(runs)-1].Timestamp
		days := int(latest.Sub(earliest).Hours() / 24)
		summary.TimeRange = fmt.Sprintf("Last %d days", days)
	} else {
		summary.TimeRange = "Single run"
	}

	// Build issue sparkline (issue counts over time)
	summary.IssueSparkline = make([]int, len(runs))
	for i, run := range runs {
		summary.IssueSparkline[i] = run.Summary.TotalIssues
	}

	// Calculate per-tool trends
	if len(runs) >= 2 {
		t.calculateToolTrends(runs, summary)
	}

	return summary
}

// calculateToolTrends calculates trend for each tool
func (t *TrendAnalyzer) calculateToolTrends(runs []*models.AggregatedReport, summary *models.TrendSummary) {
	// Get earliest and latest runs
	earliest := runs[0]
	latest := runs[len(runs)-1]

	// Find all tools across both runs
	allTools := make(map[string]bool)
	for tool := range earliest.Summary.IssuesByTool {
		allTools[tool] = true
	}
	for tool := range latest.Summary.IssuesByTool {
		allTools[tool] = true
	}

	// Calculate trend for each tool
	for tool := range allTools {
		previousCount := earliest.Summary.IssuesByTool[tool]
		currentCount := latest.Summary.IssuesByTool[tool]
		change := currentCount - previousCount

		changePercent := 0.0
		if previousCount > 0 {
			changePercent = float64(change) / float64(previousCount) * 100.0
		} else if currentCount > 0 {
			// New tool appeared
			changePercent = 100.0
		}

		summary.ByTool[tool] = &models.ToolTrend{
			Name:           tool,
			CurrentIssues:  currentCount,
			PreviousIssues: previousCount,
			Change:         change,
			ChangePercent:  changePercent,
		}
	}
}

// GenerateComparisonReport creates a detailed comparison between two runs
func (t *TrendAnalyzer) GenerateComparisonReport(current, previous *models.AggregatedReport) string {
	if previous == nil {
		return "No previous run to compare with"
	}

	trend := t.CalculateTrend(current, previous)

	report := fmt.Sprintf("Comparison: %s vs %s\n\n",
		formatDate(current.Timestamp),
		formatDate(previous.Timestamp))

	// Overall change
	report += fmt.Sprintf("Overall: %d → %d issues (%.1f%% %s)\n\n",
		trend.PreviousIssues,
		trend.CurrentIssues,
		trend.ChangePercent,
		trend.Direction)

	// Per-tool changes
	for tool := range current.Summary.IssuesByTool {
		prevCount := previous.Summary.IssuesByTool[tool]
		currCount := current.Summary.IssuesByTool[tool]

		if prevCount == currCount {
			continue // Skip unchanged tools
		}

		report += fmt.Sprintf("%s:\n", tool)
		report += fmt.Sprintf("  %d → %d (", prevCount, currCount)

		if currCount > prevCount {
			report += fmt.Sprintf("+%d)\n", currCount-prevCount)
		} else {
			report += fmt.Sprintf("%d)\n", currCount-prevCount)
		}
	}

	// New issues
	if trend.NewIssues > 0 {
		report += fmt.Sprintf("\nNew Issues: %d\n", trend.NewIssues)
	}

	// Resolved issues
	if trend.ResolvedIssues > 0 {
		report += fmt.Sprintf("\nResolved Issues: %d\n", trend.ResolvedIssues)
	}

	return report
}

// formatDate formats a timestamp for display
func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// GetTrendIndicator returns a visual indicator for trend direction
func GetTrendIndicator(direction string) string {
	switch direction {
	case "improving":
		return "↓"
	case "degrading":
		return "↑"
	case "stable":
		return "→"
	default:
		return "?"
	}
}
