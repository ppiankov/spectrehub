package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/ppiankov/spectrehub/internal/models"
)

var tableColumns = []table.Column{
	{Title: "Severity", Width: 10},
	{Title: "Tool", Width: 14},
	{Title: "Category", Width: 12},
	{Title: "Resource", Width: 36},
	{Title: "Count", Width: 6},
}

// buildRows converts normalized issues to table rows.
func buildRows(issues []models.NormalizedIssue) []table.Row {
	rows := make([]table.Row, 0, len(issues))
	for _, issue := range issues {
		rows = append(rows, table.Row{
			severityLabel(issue.Severity),
			issue.Tool,
			issue.Category,
			truncate(issue.Resource, tableColumns[3].Width),
			fmt.Sprintf("%d", issue.Count),
		})
	}
	return rows
}

func severityLabel(s string) string {
	switch s {
	case "critical":
		return "CRITICAL"
	case "high":
		return "HIGH"
	case "medium":
		return "MEDIUM"
	case "low":
		return "LOW"
	default:
		return s
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	const ellipsis = "..."
	if maxLen <= len(ellipsis) {
		return s[:maxLen]
	}
	return s[:maxLen-len(ellipsis)] + ellipsis
}

// newTable creates a bubbles table with standard columns and styling.
func newTable(rows []table.Row, height int) table.Model {
	t := table.New(
		table.WithColumns(tableColumns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorBorder).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(colorAccent).
		Bold(false)
	t.SetStyles(s)

	return t
}
