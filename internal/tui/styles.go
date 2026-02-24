package tui

import "github.com/charmbracelet/lipgloss"

// Severity colors
var (
	colorCritical = lipgloss.Color("#FF0000")
	colorHigh     = lipgloss.Color("#FF8800")
	colorMedium   = lipgloss.Color("#FFFF00")
	colorLow      = lipgloss.Color("#00FF00")
	colorMuted    = lipgloss.Color("#888888")
	colorAccent   = lipgloss.Color("#7B68EE")
	colorBorder   = lipgloss.Color("#444444")
)

// Panel styles
var (
	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	styleDetailPanel = lipgloss.NewStyle().
				Padding(0, 1).
				BorderStyle(lipgloss.NormalBorder()).
				BorderTop(true).
				BorderForeground(colorBorder)

	styleFooter = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	styleSearchPrompt = lipgloss.NewStyle().
				Foreground(colorAccent).Bold(true)
)

// severityStyle returns the lipgloss style for a severity level.
func severityStyle(severity string) lipgloss.Style {
	switch severity {
	case "critical":
		return lipgloss.NewStyle().Foreground(colorCritical).Bold(true)
	case "high":
		return lipgloss.NewStyle().Foreground(colorHigh).Bold(true)
	case "medium":
		return lipgloss.NewStyle().Foreground(colorMedium)
	case "low":
		return lipgloss.NewStyle().Foreground(colorLow)
	default:
		return lipgloss.NewStyle()
	}
}

// healthStyle returns the lipgloss style for a health score level.
func healthStyle(health string) lipgloss.Style {
	switch health {
	case "excellent":
		return lipgloss.NewStyle().Foreground(colorLow).Bold(true)
	case "good":
		return lipgloss.NewStyle().Foreground(colorLow)
	case "warning":
		return lipgloss.NewStyle().Foreground(colorMedium).Bold(true)
	case "critical":
		return lipgloss.NewStyle().Foreground(colorHigh).Bold(true)
	case "severe":
		return lipgloss.NewStyle().Foreground(colorCritical).Bold(true)
	default:
		return lipgloss.NewStyle()
	}
}
