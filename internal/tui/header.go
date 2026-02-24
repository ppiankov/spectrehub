package tui

import (
	"fmt"
	"strings"

	"github.com/ppiankov/spectrehub/internal/models"
)

// headerHeight is the number of terminal lines the header occupies.
const headerHeight = 5

// renderHeader produces the header string from report summary data.
func renderHeader(summary models.CrossToolSummary, trend *models.Trend, sparkline []int, width int) string {
	var b strings.Builder

	// Line 1: title and health
	healthText := healthStyle(summary.HealthScore).Render(
		fmt.Sprintf("%s (%.0f%%)", strings.ToUpper(summary.HealthScore), summary.ScorePercent),
	)
	b.WriteString(fmt.Sprintf("SpectreHub  Health: %s", healthText))

	if trend != nil {
		indicator := trendIndicator(trend.Direction)
		b.WriteString(fmt.Sprintf("  %s %.1f%%", indicator, trend.ChangePercent))
	}
	b.WriteString("\n")

	// Line 2: tools and total issues
	b.WriteString(fmt.Sprintf("Tools: %d/%d  Issues: %d",
		summary.SupportedTools, summary.TotalTools, summary.TotalIssues))
	b.WriteString("\n")

	// Line 3: severity breakdown
	sevParts := make([]string, 0, 4)
	for _, sev := range []string{"critical", "high", "medium", "low"} {
		if count, ok := summary.IssuesBySeverity[sev]; ok && count > 0 {
			label := fmt.Sprintf("%s:%d", strings.ToUpper(sev[:1]), count)
			sevParts = append(sevParts, severityStyle(sev).Render(label))
		}
	}
	if len(sevParts) > 0 {
		b.WriteString(strings.Join(sevParts, "  "))
	}
	b.WriteString("\n")

	// Line 4: sparkline
	if len(sparkline) > 0 {
		b.WriteString("Trend: ")
		b.WriteString(renderSparkline(sparkline))
	}

	return styleHeader.Width(width).Render(b.String())
}

func trendIndicator(direction string) string {
	switch direction {
	case "improving":
		return "↓"
	case "degrading":
		return "↑"
	default:
		return "→"
	}
}

// renderSparkline converts an int slice to a unicode sparkline string.
func renderSparkline(values []int) string {
	if len(values) == 0 {
		return ""
	}

	bars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	var b strings.Builder
	for _, v := range values {
		if max == min {
			b.WriteRune(bars[len(bars)/2])
		} else {
			normalized := float64(v-min) / float64(max-min)
			idx := int(normalized * float64(len(bars)-1))
			b.WriteRune(bars[idx])
		}
	}

	b.WriteString(fmt.Sprintf(" [%d→%d]", values[0], values[len(values)-1]))
	return b.String()
}
