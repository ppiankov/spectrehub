package tui

import (
	"sort"
	"strings"

	"github.com/ppiankov/spectrehub/internal/models"
)

// filterState holds current active filters.
type filterState struct {
	Tool       string
	Severity   string
	SearchText string
}

// sortField enumerates columns that can be sorted.
type sortField int

const (
	sortBySeverity sortField = iota
	sortByTool
	sortByCategory
	sortByResource
	sortByCount
)

// sortFieldCount is the total number of sortable columns.
const sortFieldCount = 5

var severityPriority = map[string]int{
	"critical": 0, "high": 1, "medium": 2, "low": 3,
}

// applyFilters returns issues matching all active filters.
func applyFilters(issues []models.NormalizedIssue, f filterState) []models.NormalizedIssue {
	result := make([]models.NormalizedIssue, 0, len(issues))
	searchLower := strings.ToLower(f.SearchText)

	for _, issue := range issues {
		if f.Tool != "" && issue.Tool != f.Tool {
			continue
		}
		if f.Severity != "" && issue.Severity != f.Severity {
			continue
		}
		if searchLower != "" && !matchesSearch(issue, searchLower) {
			continue
		}
		result = append(result, issue)
	}
	return result
}

func matchesSearch(issue models.NormalizedIssue, searchLower string) bool {
	return strings.Contains(strings.ToLower(issue.Tool), searchLower) ||
		strings.Contains(strings.ToLower(issue.Category), searchLower) ||
		strings.Contains(strings.ToLower(issue.Severity), searchLower) ||
		strings.Contains(strings.ToLower(issue.Resource), searchLower) ||
		strings.Contains(strings.ToLower(issue.Evidence), searchLower)
}

// sortIssues sorts a slice of issues in place by the given field.
func sortIssues(issues []models.NormalizedIssue, field sortField) {
	sort.SliceStable(issues, func(i, j int) bool {
		switch field {
		case sortBySeverity:
			return severityPriority[issues[i].Severity] < severityPriority[issues[j].Severity]
		case sortByTool:
			return issues[i].Tool < issues[j].Tool
		case sortByCategory:
			return issues[i].Category < issues[j].Category
		case sortByResource:
			return issues[i].Resource < issues[j].Resource
		case sortByCount:
			return issues[i].Count > issues[j].Count
		default:
			return false
		}
	})
}

// uniqueTools returns deduplicated, sorted tool names from issues.
func uniqueTools(issues []models.NormalizedIssue) []string {
	seen := make(map[string]bool)
	var tools []string
	for _, issue := range issues {
		if !seen[issue.Tool] {
			seen[issue.Tool] = true
			tools = append(tools, issue.Tool)
		}
	}
	sort.Strings(tools)
	return tools
}

// sortFieldName returns a human-readable name for the sort field.
func sortFieldName(f sortField) string {
	switch f {
	case sortBySeverity:
		return "severity"
	case sortByTool:
		return "tool"
	case sortByCategory:
		return "category"
	case sortByResource:
		return "resource"
	case sortByCount:
		return "count"
	default:
		return "unknown"
	}
}
