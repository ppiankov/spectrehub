package aggregator

import (
	"fmt"
	"sort"

	"github.com/ppiankov/spectrehub/internal/models"
)

// issueGroup represents a group of issues by tool, category, and severity
type issueGroup struct {
	tool     string
	category string
	severity string
	count    int
}

// RecommendationGenerator creates actionable recommendations from aggregated issues
type RecommendationGenerator struct{}

// NewRecommendationGenerator creates a new recommendation generator
func NewRecommendationGenerator() *RecommendationGenerator {
	return &RecommendationGenerator{}
}

// GenerateRecommendations analyzes the report and creates prioritized recommendations
func (r *RecommendationGenerator) GenerateRecommendations(report *models.AggregatedReport) []models.Recommendation {
	// Group issues by (Tool, Category, Severity)
	groups := make(map[string]*issueGroup)

	for _, issue := range report.Issues {
		key := fmt.Sprintf("%s:%s:%s", issue.Tool, issue.Category, issue.Severity)
		if g, exists := groups[key]; exists {
			g.count += issue.Count
		} else {
			groups[key] = &issueGroup{
				tool:     issue.Tool,
				category: issue.Category,
				severity: issue.Severity,
				count:    issue.Count,
			}
		}
	}

	// Generate recommendations from groups
	var recommendations []models.Recommendation

	for _, group := range groups {
		rec := models.Recommendation{
			Severity: group.severity,
			Tool:     group.tool,
			Action:   r.generateAction(group),
			Impact:   r.generateImpact(group),
			Count:    group.count,
		}
		recommendations = append(recommendations, rec)
	}

	// Sort by severity priority (critical > high > medium > low)
	sort.Slice(recommendations, func(i, j int) bool {
		return r.severityPriority(recommendations[i].Severity) > r.severityPriority(recommendations[j].Severity)
	})

	return recommendations
}

// generateAction creates actionable text based on category and count
func (r *RecommendationGenerator) generateAction(group *issueGroup) string {
	resourceName := r.getResourceName(group.tool)

	switch group.category {
	case models.StatusMissing:
		return fmt.Sprintf("Fix %d missing %s", group.count, resourceName)
	case models.StatusUnused:
		return fmt.Sprintf("Clean up %d unused %s", group.count, resourceName)
	case models.StatusStale:
		return fmt.Sprintf("Review %d stale %s", group.count, resourceName)
	case models.StatusError:
		return fmt.Sprintf("Investigate %d error(s) in %s", group.count, group.tool)
	case models.StatusMisconfig:
		return fmt.Sprintf("Fix %d misconfiguration(s) in %s", group.count, group.tool)
	case models.StatusAccessDeny:
		return fmt.Sprintf("Restore access to %d %s", group.count, resourceName)
	case models.StatusInvalid:
		return fmt.Sprintf("Fix %d invalid %s", group.count, resourceName)
	case models.StatusDrift:
		return fmt.Sprintf("Resolve %d drift issue(s) in %s", group.count, group.tool)
	default:
		return fmt.Sprintf("Address %d issue(s) in %s", group.count, group.tool)
	}
}

// generateImpact describes the potential impact based on severity and category
func (r *RecommendationGenerator) generateImpact(group *issueGroup) string {
	switch group.severity {
	case models.SeverityCritical:
		switch group.category {
		case models.StatusMissing:
			return "Services may fail to start or operate incorrectly"
		case models.StatusAccessDeny:
			return "Critical operations are blocked"
		case models.StatusError:
			return "System integrity is compromised"
		default:
			return "Immediate action required to prevent outages"
		}

	case models.SeverityHigh:
		switch group.category {
		case models.StatusMissing:
			return "Important features may not work as expected"
		case models.StatusUnused:
			return "Significant waste of resources and potential security risks"
		case models.StatusStale:
			return "Data may be outdated or invalid"
		case models.StatusMisconfig:
			return "System behavior may be unpredictable"
		default:
			return "Significant impact on system reliability"
		}

	case models.SeverityMedium:
		switch group.category {
		case models.StatusUnused:
			return "Resources are wasted but no immediate risk"
		case models.StatusStale:
			return "Data quality may degrade over time"
		case models.StatusMisconfig:
			return "Suboptimal performance or behavior"
		default:
			return "Moderate impact on system efficiency"
		}

	case models.SeverityLow:
		switch group.category {
		case models.StatusUnused:
			return "Minor cleanup to improve maintainability"
		case models.StatusStale:
			return "Consider updating or removing"
		default:
			return "Low priority cleanup or optimization"
		}

	default:
		return "Review and address as needed"
	}
}

// getResourceName returns human-readable resource name for the tool
func (r *RecommendationGenerator) getResourceName(tool string) string {
	switch models.ToolType(tool) {
	case models.ToolVault:
		return "Vault secrets"
	case models.ToolS3:
		return "S3 buckets/prefixes"
	case models.ToolKafka:
		return "Kafka topics"
	case models.ToolClickHouse:
		return "ClickHouse tables"
	case models.ToolPg:
		return "Postgres tables/indexes"
	case models.ToolMongo:
		return "MongoDB collections/indexes"
	default:
		return "resources"
	}
}

// severityPriority returns numeric priority for sorting (higher = more urgent)
func (r *RecommendationGenerator) severityPriority(severity string) int {
	switch severity {
	case models.SeverityCritical:
		return 4
	case models.SeverityHigh:
		return 3
	case models.SeverityMedium:
		return 2
	case models.SeverityLow:
		return 1
	default:
		return 0
	}
}

// GetTopRecommendations returns the top N most critical recommendations
func (r *RecommendationGenerator) GetTopRecommendations(recommendations []models.Recommendation, n int) []models.Recommendation {
	if n >= len(recommendations) {
		return recommendations
	}
	return recommendations[:n]
}

// GroupBySeverity groups recommendations by severity level
func (r *RecommendationGenerator) GroupBySeverity(recommendations []models.Recommendation) map[string][]models.Recommendation {
	grouped := make(map[string][]models.Recommendation)

	for _, rec := range recommendations {
		grouped[rec.Severity] = append(grouped[rec.Severity], rec)
	}

	return grouped
}
