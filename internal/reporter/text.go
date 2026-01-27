package reporter

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ppiankov/spectrehub/internal/aggregator"
	"github.com/ppiankov/spectrehub/internal/models"
)

// TextReporter generates human-readable text reports
type TextReporter struct {
	writer io.Writer
}

// NewTextReporter creates a new text reporter
func NewTextReporter(writer io.Writer) *TextReporter {
	return &TextReporter{
		writer: writer,
	}
}

// Generate creates a text report from the aggregated data
func (r *TextReporter) Generate(report *models.AggregatedReport) error {
	// Header
	r.printHeader()
	r.printf("Timestamp: %s\n\n", formatTimestamp(report.Timestamp))

	// Overall Summary
	r.printOverallSummary(report)

	// Per-tool breakdown
	r.printToolBreakdown(report)

	// Recommendations
	if len(report.Recommendations) > 0 {
		r.printRecommendations(report.Recommendations)
	}

	// Trend summary if available
	if report.Trend != nil {
		r.printf("\n")
		r.printTrendInfo(report.Trend)
	}

	return nil
}

// printHeader prints the report header
func (r *TextReporter) printHeader() {
	r.printf("╔════════════════════════════════════════════╗\n")
	r.printf("║       SpectreHub Aggregated Report        ║\n")
	r.printf("╚════════════════════════════════════════════╝\n\n")
}

// printOverallSummary prints the overall summary section
func (r *TextReporter) printOverallSummary(report *models.AggregatedReport) {
	r.printf("Overall Summary:\n")
	r.printf("--------------------------------------------------\n")
	r.printf("  Total Tools: %d (%d supported, %d unsupported)\n",
		report.Summary.TotalTools,
		report.Summary.SupportedTools,
		report.Summary.UnsupportedTools)
	r.printf("  Total Issues: %d\n", report.Summary.TotalIssues)
	r.printf("  Health Score: %s", strings.ToUpper(report.Summary.HealthScore))

	// Add percentage if available
	if report.Summary.ScorePercent > 0 {
		r.printf(" (%.1f%%)", report.Summary.ScorePercent)
	}

	// Add trend indicator if available
	if report.Trend != nil {
		indicator := aggregator.GetTrendIndicator(report.Trend.Direction)
		r.printf(" %s %.1f%% from previous run", indicator, report.Trend.ChangePercent)
	}

	r.printf("\n\n")

	// Issues by category
	if len(report.Summary.IssuesByCategory) > 0 {
		r.printf("Issues by Category:\n")
		for category, count := range report.Summary.IssuesByCategory {
			r.printf("  %s: %d\n", strings.Title(category), count)
		}
		r.printf("\n")
	}

	// Issues by severity
	if len(report.Summary.IssuesBySeverity) > 0 {
		r.printf("Issues by Severity:\n")
		for severity, count := range report.Summary.IssuesBySeverity {
			r.printf("  %s: %d\n", strings.Title(severity), count)
		}
		r.printf("\n")
	}
}

// printToolBreakdown prints detailed breakdown for each tool
func (r *TextReporter) printToolBreakdown(report *models.AggregatedReport) {
	for toolName, toolReport := range report.ToolReports {
		r.printf("\n%s (v%s)\n", toolName, toolReport.Version)
		r.printf("--------------------------------------------------\n")

		issueCount := report.Summary.IssuesByTool[toolName]

		// Tool-specific details
		switch models.ToolType(toolName) {
		case models.ToolVault:
			r.printVaultDetails(toolReport, issueCount)
		case models.ToolS3:
			r.printS3Details(toolReport, issueCount)
		case models.ToolKafka:
			r.printKafkaDetails(toolReport, issueCount)
		case models.ToolClickHouse:
			r.printClickHouseDetails(toolReport, issueCount)
		default:
			if toolReport.IsSupported {
				r.printf("  Issues Found: %d\n", issueCount)
			} else {
				r.printf("  Status: UNSUPPORTED (data stored as raw)\n")
			}
		}

		r.printf("\n")
	}
}

// printVaultDetails prints VaultSpectre-specific details
func (r *TextReporter) printVaultDetails(toolReport models.ToolReport, issueCount int) {
	if vaultReport, ok := toolReport.RawData.(*models.VaultReport); ok {
		r.printf("  Total References: %d\n", vaultReport.Summary.TotalReferences)
		r.printf("  Status OK: %d\n", vaultReport.Summary.StatusOK)
		if vaultReport.Summary.StatusMissing > 0 {
			r.printf("  Missing: %d\n", vaultReport.Summary.StatusMissing)
		}
		if vaultReport.Summary.StatusAccessDenied > 0 {
			r.printf("  Access Denied: %d\n", vaultReport.Summary.StatusAccessDenied)
		}
		if vaultReport.Summary.StaleSecrets > 0 {
			r.printf("  Stale: %d\n", vaultReport.Summary.StaleSecrets)
		}
		if vaultReport.Summary.StatusInvalid > 0 {
			r.printf("  Invalid: %d\n", vaultReport.Summary.StatusInvalid)
		}
		if vaultReport.Summary.StatusError > 0 {
			r.printf("  Errors: %d\n", vaultReport.Summary.StatusError)
		}
		r.printf("  Health: %s\n", vaultReport.Summary.HealthScore)
	}
}

// printS3Details prints S3Spectre-specific details
func (r *TextReporter) printS3Details(toolReport models.ToolReport, issueCount int) {
	if s3Report, ok := toolReport.RawData.(*models.S3Report); ok {
		r.printf("  Total Buckets: %d\n", s3Report.Summary.TotalBuckets)
		r.printf("  OK Buckets: %d\n", s3Report.Summary.OKBuckets)
		if len(s3Report.Summary.MissingBuckets) > 0 {
			r.printf("  Missing: %d\n", len(s3Report.Summary.MissingBuckets))
		}
		if len(s3Report.Summary.UnusedBuckets) > 0 {
			r.printf("  Unused: %d\n", len(s3Report.Summary.UnusedBuckets))
		}
		if len(s3Report.Summary.MissingPrefixes) > 0 {
			r.printf("  Missing Prefixes: %d\n", len(s3Report.Summary.MissingPrefixes))
		}
		if len(s3Report.Summary.StalePrefixes) > 0 {
			r.printf("  Stale Prefixes: %d\n", len(s3Report.Summary.StalePrefixes))
		}
		if len(s3Report.Summary.VersionSprawl) > 0 {
			r.printf("  Version Sprawl: %d\n", len(s3Report.Summary.VersionSprawl))
		}
		if len(s3Report.Summary.LifecycleMisconfig) > 0 {
			r.printf("  Lifecycle Misconfig: %d\n", len(s3Report.Summary.LifecycleMisconfig))
		}
	}
}

// printKafkaDetails prints KafkaSpectre-specific details
func (r *TextReporter) printKafkaDetails(toolReport models.ToolReport, issueCount int) {
	if kafkaReport, ok := toolReport.RawData.(*models.KafkaReport); ok {
		if kafkaReport.Summary != nil {
			r.printf("  Cluster: %s\n", kafkaReport.Summary.ClusterName)
			r.printf("  Total Topics: %d\n", kafkaReport.Summary.TotalTopics)
			r.printf("  Active: %d\n", kafkaReport.Summary.ActiveTopics)
			r.printf("  Unused: %d\n", kafkaReport.Summary.UnusedTopics)
			if kafkaReport.Summary.HighRiskCount > 0 {
				r.printf("  High Risk: %d\n", kafkaReport.Summary.HighRiskCount)
			}
			if kafkaReport.Summary.MediumRiskCount > 0 {
				r.printf("  Medium Risk: %d\n", kafkaReport.Summary.MediumRiskCount)
			}
			r.printf("  Health: %s\n", kafkaReport.Summary.ClusterHealthScore)
			if kafkaReport.Summary.UnusedPercentage > 0 {
				r.printf("  Unused %%: %.1f%%\n", kafkaReport.Summary.UnusedPercentage)
			}
		}
	}
}

// printClickHouseDetails prints ClickSpectre-specific details
func (r *TextReporter) printClickHouseDetails(toolReport models.ToolReport, issueCount int) {
	if clickReport, ok := toolReport.RawData.(*models.ClickHouseReport); ok {
		totalTables := len(clickReport.Tables)
		zeroUsage := 0
		for _, table := range clickReport.Tables {
			if table.ZeroUsage {
				zeroUsage++
			}
		}

		r.printf("  Host: %s\n", clickReport.Metadata.ClickHouseHost)
		r.printf("  Tables Analyzed: %d\n", totalTables)
		r.printf("  Zero Usage: %d\n", zeroUsage)
		r.printf("  Active: %d\n", totalTables-zeroUsage)
		if len(clickReport.Anomalies) > 0 {
			r.printf("  Anomalies: %d\n", len(clickReport.Anomalies))
		}
		r.printf("  Lookback: %d days\n", clickReport.Metadata.LookbackDays)
	}
}

// printRecommendations prints the recommendations section
func (r *TextReporter) printRecommendations(recommendations []models.Recommendation) {
	r.printf("\n")
	r.printf("Recommended Actions:\n")
	r.printf("--------------------------------------------------\n")

	// Group by severity
	gen := aggregator.NewRecommendationGenerator()
	grouped := gen.GroupBySeverity(recommendations)

	// Print in severity order
	for _, severity := range []string{models.SeverityCritical, models.SeverityHigh, models.SeverityMedium, models.SeverityLow} {
		recs := grouped[severity]
		if len(recs) == 0 {
			continue
		}

		for i, rec := range recs {
			r.printf("  %d. [%s] %s\n", i+1, strings.ToUpper(rec.Severity), rec.Action)
			r.printf("     Impact: %s\n", rec.Impact)
		}
	}
}

// printTrendInfo prints trend information
func (r *TextReporter) printTrendInfo(trend *models.Trend) {
	r.printf("Trend Analysis:\n")
	r.printf("--------------------------------------------------\n")
	r.printf("  Direction: %s %s\n", trend.Direction, aggregator.GetTrendIndicator(trend.Direction))
	r.printf("  Change: %d → %d issues (%.1f%%)\n",
		trend.PreviousIssues,
		trend.CurrentIssues,
		trend.ChangePercent)

	if trend.NewIssues > 0 {
		r.printf("  New Issues: %d\n", trend.NewIssues)
	}
	if trend.ResolvedIssues > 0 {
		r.printf("  Resolved: %d\n", trend.ResolvedIssues)
	}

	r.printf("  Compared With: %s\n", formatTimestamp(trend.ComparedWith))
}

// printf is a helper to write formatted output
func (r *TextReporter) printf(format string, args ...interface{}) {
	fmt.Fprintf(r.writer, format, args...)
}

// formatTimestamp formats a timestamp for display
func formatTimestamp(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
