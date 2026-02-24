package tui

import (
	"fmt"
	"strings"

	"github.com/ppiankov/spectrehub/internal/models"
)

// detailHeight is the fixed number of lines for the detail panel.
const detailHeight = 5

// renderDetail produces the detail view for a selected issue.
func renderDetail(issue *models.NormalizedIssue, width int) string {
	if issue == nil {
		return styleDetailPanel.Width(width).Render("No issue selected")
	}

	var b strings.Builder

	sevStyled := severityStyle(issue.Severity).Render(strings.ToUpper(issue.Severity))
	b.WriteString(fmt.Sprintf("%s  %s / %s\n", sevStyled, issue.Tool, issue.Category))
	b.WriteString(fmt.Sprintf("Resource: %s\n", issue.Resource))

	if issue.Evidence != "" {
		b.WriteString(fmt.Sprintf("Evidence: %s\n", issue.Evidence))
	}

	parts := make([]string, 0, 3)
	if issue.Count > 0 {
		parts = append(parts, fmt.Sprintf("Count: %d", issue.Count))
	}
	if !issue.FirstSeen.IsZero() {
		parts = append(parts, fmt.Sprintf("First: %s", issue.FirstSeen.Format("2006-01-02")))
	}
	if !issue.LastSeen.IsZero() {
		parts = append(parts, fmt.Sprintf("Last: %s", issue.LastSeen.Format("2006-01-02")))
	}
	if len(parts) > 0 {
		b.WriteString(strings.Join(parts, "  "))
	}

	return styleDetailPanel.Width(width).Render(b.String())
}
