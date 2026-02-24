package tui

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ppiankov/spectrehub/internal/models"
)

// mode represents the current UI interaction mode.
type mode int

const (
	modeNormal mode = iota
	modeSearch
	modeFilterTool
)

const defaultTableHeight = 15

// Model is the top-level Bubble Tea model for the summarize TUI.
type Model struct {
	// Data (immutable after init)
	report    *models.AggregatedReport
	trend     *models.TrendSummary
	allIssues []models.NormalizedIssue

	// UI state
	table          table.Model
	searchInput    textinput.Model
	filteredIssues []models.NormalizedIssue
	filters        filterState
	sortBy         sortField
	mode           mode
	toolChoices    []string
	toolCursor     int
	width          int
	height         int
	statusMsg      string
	// clipboard is captured here for testing instead of writing to stdout
	clipboard string
}

// New creates a new TUI model from report data.
func New(report *models.AggregatedReport, trend *models.TrendSummary) Model {
	issues := make([]models.NormalizedIssue, len(report.Issues))
	copy(issues, report.Issues)

	sortIssues(issues, sortBySeverity)
	rows := buildRows(issues)
	t := newTable(rows, defaultTableHeight)

	ti := textinput.New()
	ti.Placeholder = "search..."
	ti.CharLimit = 64

	return Model{
		report:         report,
		trend:          trend,
		allIssues:      issues,
		filteredIssues: issues,
		table:          t,
		searchInput:    ti,
		sortBy:         sortBySeverity,
		mode:           modeNormal,
		toolChoices:    uniqueTools(issues),
		width:          80,
		height:         24,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(msg.Width)
		tableH := msg.Height - headerHeight - detailHeight - 3
		if tableH < 3 {
			tableH = 3
		}
		m.table.SetHeight(tableH)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	var cmd tea.Cmd
	switch m.mode {
	case modeSearch:
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	default:
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeSearch:
		return m.handleSearchKey(msg)
	case modeFilterTool:
		return m.handleFilterToolKey(msg)
	default:
		return m.handleNormalKey(msg)
	}
}

func (m Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, keys.Search):
		m.mode = modeSearch
		m.searchInput.Focus()
		return m, textinput.Blink
	case key.Matches(msg, keys.FilterTool):
		m.mode = modeFilterTool
		m.toolCursor = 0
		return m, nil
	case key.Matches(msg, keys.Sort):
		m.sortBy = (m.sortBy + 1) % sortField(sortFieldCount)
		m.rebuildTable()
		m.statusMsg = fmt.Sprintf("Sort: %s", sortFieldName(m.sortBy))
		return m, nil
	case key.Matches(msg, keys.Copy):
		m.copySelectedIssue()
		return m, nil
	case key.Matches(msg, keys.ClearFilter):
		m.filters = filterState{}
		m.statusMsg = ""
		m.rebuildTable()
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.filters.SearchText = m.searchInput.Value()
		m.mode = modeNormal
		m.searchInput.Blur()
		m.rebuildTable()
		return m, nil
	case "esc":
		m.mode = modeNormal
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m Model) handleFilterToolKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.toolCursor > 0 {
			m.toolCursor--
		}
	case "down", "j":
		if m.toolCursor < len(m.toolChoices) {
			m.toolCursor++
		}
	case "enter":
		if m.toolCursor == 0 {
			m.filters.Tool = ""
		} else if m.toolCursor <= len(m.toolChoices) {
			m.filters.Tool = m.toolChoices[m.toolCursor-1]
		}
		m.mode = modeNormal
		m.rebuildTable()
		if m.filters.Tool != "" {
			m.statusMsg = fmt.Sprintf("Filter: %s", m.filters.Tool)
		} else {
			m.statusMsg = ""
		}
	case "esc":
		m.mode = modeNormal
	}
	return m, nil
}

func (m *Model) rebuildTable() {
	filtered := applyFilters(m.allIssues, m.filters)
	sortIssues(filtered, m.sortBy)
	m.filteredIssues = filtered
	m.table.SetRows(buildRows(filtered))
}

func (m *Model) selectedIssue() *models.NormalizedIssue {
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(m.filteredIssues) {
		return nil
	}
	return &m.filteredIssues[cursor]
}

// copySelectedIssue writes the selected issue to clipboard via OSC 52.
func (m *Model) copySelectedIssue() {
	issue := m.selectedIssue()
	if issue == nil {
		m.statusMsg = "Nothing to copy"
		return
	}
	text := fmt.Sprintf("[%s] %s %s: %s", issue.Severity, issue.Tool, issue.Category, issue.Resource)
	if issue.Evidence != "" {
		text += " -- " + issue.Evidence
	}
	m.clipboard = text
	m.statusMsg = "Copied!"
	// OSC 52 clipboard escape: works in most modern terminals
	fmt.Printf("\033]52;c;%s\a", base64.StdEncoding.EncodeToString([]byte(text)))
}

// View implements tea.Model.
func (m Model) View() string {
	var b strings.Builder

	// Header
	var sparkline []int
	if m.trend != nil {
		sparkline = m.trend.IssueSparkline
	}
	b.WriteString(renderHeader(m.report.Summary, m.report.Trend, sparkline, m.width))
	b.WriteString("\n")

	// Search bar overlay
	if m.mode == modeSearch {
		b.WriteString(styleSearchPrompt.Render("/ "))
		b.WriteString(m.searchInput.View())
		b.WriteString("\n")
	}

	// Tool filter overlay
	if m.mode == modeFilterTool {
		b.WriteString(m.renderToolFilter())
		b.WriteString("\n")
	}

	// Table
	b.WriteString(m.table.View())
	b.WriteString("\n")

	// Detail panel
	b.WriteString(renderDetail(m.selectedIssue(), m.width))
	b.WriteString("\n")

	// Footer
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m *Model) renderToolFilter() string {
	var b strings.Builder
	b.WriteString("Filter by tool:\n")

	options := append([]string{"All"}, m.toolChoices...)
	for i, opt := range options {
		cursor := "  "
		if i == m.toolCursor {
			cursor = "> "
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, opt))
	}
	return b.String()
}

func (m *Model) renderFooter() string {
	left := "q:quit  /:search  t:tool  s:sort  c:copy  esc:clear"
	right := fmt.Sprintf("%d/%d issues", len(m.filteredIssues), len(m.allIssues))

	if m.statusMsg != "" {
		right = m.statusMsg + "  " + right
	}

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	return styleFooter.Render(left + strings.Repeat(" ", gap) + right)
}

// Run starts the Bubble Tea program. Called from the summarize command.
func Run(report *models.AggregatedReport, trend *models.TrendSummary) error {
	m := New(report, trend)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
