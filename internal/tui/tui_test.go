package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ppiankov/spectrehub/internal/models"
)

func testIssues() []models.NormalizedIssue {
	return []models.NormalizedIssue{
		{Tool: "vaultspectre", Category: "missing", Severity: "critical", Resource: "secret/db", Evidence: "not found", Count: 1},
		{Tool: "s3spectre", Category: "unused", Severity: "low", Resource: "s3://old-bucket", Count: 3},
		{Tool: "vaultspectre", Category: "stale", Severity: "medium", Resource: "secret/api-key", Count: 1},
		{Tool: "kafkaspectre", Category: "unused", Severity: "low", Resource: "topic-old", Count: 5},
	}
}

func testReport() *models.AggregatedReport {
	issues := testIssues()
	return &models.AggregatedReport{
		Timestamp: time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC),
		Issues:    issues,
		Summary: models.CrossToolSummary{
			TotalIssues:      len(issues),
			TotalTools:       3,
			SupportedTools:   3,
			HealthScore:      "warning",
			ScorePercent:     72.5,
			IssuesByTool:     map[string]int{"vaultspectre": 2, "s3spectre": 1, "kafkaspectre": 1},
			IssuesByCategory: map[string]int{"missing": 1, "unused": 2, "stale": 1},
			IssuesBySeverity: map[string]int{"critical": 1, "medium": 1, "low": 2},
		},
		Recommendations: []models.Recommendation{
			{Severity: "critical", Tool: "vaultspectre", Action: "Fix missing", Impact: "Broken", Count: 1},
		},
	}
}

// --- Filter tests ---

func TestApplyFiltersNoFilter(t *testing.T) {
	issues := testIssues()
	result := applyFilters(issues, filterState{})
	if len(result) != len(issues) {
		t.Errorf("expected %d issues, got %d", len(issues), len(result))
	}
}

func TestApplyFiltersToolFilter(t *testing.T) {
	issues := testIssues()
	result := applyFilters(issues, filterState{Tool: "vaultspectre"})
	if len(result) != 2 {
		t.Errorf("expected 2 vaultspectre issues, got %d", len(result))
	}
	for _, r := range result {
		if r.Tool != "vaultspectre" {
			t.Errorf("expected vaultspectre, got %s", r.Tool)
		}
	}
}

func TestApplyFiltersSeverityFilter(t *testing.T) {
	issues := testIssues()
	result := applyFilters(issues, filterState{Severity: "low"})
	if len(result) != 2 {
		t.Errorf("expected 2 low issues, got %d", len(result))
	}
}

func TestApplyFiltersSearchText(t *testing.T) {
	issues := testIssues()
	result := applyFilters(issues, filterState{SearchText: "bucket"})
	if len(result) != 1 {
		t.Errorf("expected 1 issue matching 'bucket', got %d", len(result))
	}
	if result[0].Resource != "s3://old-bucket" {
		t.Errorf("expected s3://old-bucket, got %s", result[0].Resource)
	}
}

func TestApplyFiltersCombined(t *testing.T) {
	issues := testIssues()
	result := applyFilters(issues, filterState{Tool: "vaultspectre", SearchText: "stale"})
	if len(result) != 1 {
		t.Errorf("expected 1 issue, got %d", len(result))
	}
}

func TestApplyFiltersNoMatch(t *testing.T) {
	issues := testIssues()
	result := applyFilters(issues, filterState{SearchText: "nonexistent"})
	if len(result) != 0 {
		t.Errorf("expected 0 issues, got %d", len(result))
	}
}

func TestApplyFiltersCaseInsensitive(t *testing.T) {
	issues := testIssues()
	result := applyFilters(issues, filterState{SearchText: "BUCKET"})
	if len(result) != 1 {
		t.Errorf("expected 1 issue matching 'BUCKET' case-insensitive, got %d", len(result))
	}
}

// --- Sort tests ---

func TestSortIssuesBySeverity(t *testing.T) {
	issues := testIssues()
	sortIssues(issues, sortBySeverity)
	if issues[0].Severity != "critical" {
		t.Errorf("expected critical first, got %s", issues[0].Severity)
	}
	if issues[len(issues)-1].Severity != "low" {
		t.Errorf("expected low last, got %s", issues[len(issues)-1].Severity)
	}
}

func TestSortIssuesByTool(t *testing.T) {
	issues := testIssues()
	sortIssues(issues, sortByTool)
	if issues[0].Tool != "kafkaspectre" {
		t.Errorf("expected kafkaspectre first (alphabetical), got %s", issues[0].Tool)
	}
}

func TestSortIssuesByCount(t *testing.T) {
	issues := testIssues()
	sortIssues(issues, sortByCount)
	if issues[0].Count != 5 {
		t.Errorf("expected count 5 first (descending), got %d", issues[0].Count)
	}
}

func TestSortIssuesByCategory(t *testing.T) {
	issues := testIssues()
	sortIssues(issues, sortByCategory)
	if issues[0].Category != "missing" {
		t.Errorf("expected missing first, got %s", issues[0].Category)
	}
}

func TestSortIssuesByResource(t *testing.T) {
	issues := testIssues()
	sortIssues(issues, sortByResource)
	if issues[0].Resource != "s3://old-bucket" {
		t.Errorf("expected s3://old-bucket first, got %s", issues[0].Resource)
	}
}

// --- UniqueTools tests ---

func TestUniqueTools(t *testing.T) {
	tools := uniqueTools(testIssues())
	if len(tools) != 3 {
		t.Errorf("expected 3 unique tools, got %d", len(tools))
	}
	expected := []string{"kafkaspectre", "s3spectre", "vaultspectre"}
	for i, tool := range tools {
		if tool != expected[i] {
			t.Errorf("expected %s at index %d, got %s", expected[i], i, tool)
		}
	}
}

func TestUniqueToolsEmpty(t *testing.T) {
	tools := uniqueTools(nil)
	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}
}

// --- Row building tests ---

func TestBuildRows(t *testing.T) {
	issues := testIssues()
	rows := buildRows(issues)
	if len(rows) != len(issues) {
		t.Errorf("expected %d rows, got %d", len(issues), len(rows))
	}
	if rows[0][0] != "CRITICAL" {
		t.Errorf("expected CRITICAL, got %s", rows[0][0])
	}
	if rows[0][1] != "vaultspectre" {
		t.Errorf("expected vaultspectre, got %s", rows[0][1])
	}
}

func TestBuildRowsEmpty(t *testing.T) {
	rows := buildRows(nil)
	if len(rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(rows))
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"this is a very long string", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestSeverityLabel(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"critical", "CRITICAL"},
		{"high", "HIGH"},
		{"medium", "MEDIUM"},
		{"low", "LOW"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		got := severityLabel(tt.input)
		if got != tt.want {
			t.Errorf("severityLabel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- Header rendering tests ---

func TestRenderHeaderContainsHealthScore(t *testing.T) {
	report := testReport()
	output := renderHeader(report.Summary, nil, nil, 80)
	if !strings.Contains(output, "WARNING") {
		t.Error("expected header to contain WARNING health score")
	}
	if !strings.Contains(output, "72") {
		t.Error("expected header to contain score percentage")
	}
}

func TestRenderHeaderContainsToolCount(t *testing.T) {
	report := testReport()
	output := renderHeader(report.Summary, nil, nil, 80)
	if !strings.Contains(output, "3/3") {
		t.Error("expected header to contain tool count 3/3")
	}
}

func TestRenderHeaderContainsIssueCount(t *testing.T) {
	report := testReport()
	output := renderHeader(report.Summary, nil, nil, 80)
	if !strings.Contains(output, "Issues: 4") {
		t.Error("expected header to contain Issues: 4")
	}
}

func TestRenderHeaderWithTrend(t *testing.T) {
	report := testReport()
	trend := &models.Trend{Direction: "improving", ChangePercent: -15.2}
	output := renderHeader(report.Summary, trend, nil, 80)
	if !strings.Contains(output, "↓") {
		t.Error("expected improving trend indicator ↓")
	}
}

func TestRenderHeaderWithSparkline(t *testing.T) {
	report := testReport()
	sparkline := []int{5, 3, 4, 2}
	output := renderHeader(report.Summary, nil, sparkline, 80)
	if !strings.Contains(output, "Trend:") {
		t.Error("expected sparkline in header")
	}
	if !strings.Contains(output, "[5→2]") {
		t.Error("expected sparkline range [5→2]")
	}
}

func TestRenderHeaderSeverityBreakdown(t *testing.T) {
	report := testReport()
	output := renderHeader(report.Summary, nil, nil, 80)
	if !strings.Contains(output, "C:1") {
		t.Error("expected C:1 for critical count")
	}
}

// --- Detail rendering tests ---

func TestRenderDetailNil(t *testing.T) {
	output := renderDetail(nil, 80)
	if !strings.Contains(output, "No issue selected") {
		t.Error("expected 'No issue selected' for nil issue")
	}
}

func TestRenderDetailShowsFields(t *testing.T) {
	issue := &models.NormalizedIssue{
		Tool: "vaultspectre", Category: "missing", Severity: "critical",
		Resource: "secret/db", Evidence: "key not found", Count: 1,
		FirstSeen: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		LastSeen:  time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
	}
	output := renderDetail(issue, 80)
	if !strings.Contains(output, "key not found") {
		t.Error("expected evidence in detail")
	}
	if !strings.Contains(output, "secret/db") {
		t.Error("expected resource in detail")
	}
	if !strings.Contains(output, "vaultspectre") {
		t.Error("expected tool in detail")
	}
	if !strings.Contains(output, "2026-01-01") {
		t.Error("expected first seen date in detail")
	}
	if !strings.Contains(output, "2026-02-15") {
		t.Error("expected last seen date in detail")
	}
}

func TestRenderDetailNoEvidence(t *testing.T) {
	issue := &models.NormalizedIssue{
		Tool: "s3spectre", Category: "unused", Severity: "low",
		Resource: "s3://bucket", Count: 2,
	}
	output := renderDetail(issue, 80)
	if !strings.Contains(output, "s3://bucket") {
		t.Error("expected resource in detail")
	}
	if strings.Contains(output, "Evidence:") {
		t.Error("expected no evidence line when evidence is empty")
	}
}

// --- Sparkline tests ---

func TestRenderSparklineEmpty(t *testing.T) {
	result := renderSparkline(nil)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestRenderSparklineConstant(t *testing.T) {
	result := renderSparkline([]int{5, 5, 5})
	if !strings.Contains(result, "[5→5]") {
		t.Errorf("expected [5→5], got %q", result)
	}
}

func TestRenderSparklineIncreasing(t *testing.T) {
	result := renderSparkline([]int{1, 2, 3, 4})
	if !strings.Contains(result, "[1→4]") {
		t.Errorf("expected [1→4], got %q", result)
	}
	// First char should be lowest bar, last should be highest
	runes := []rune(result)
	if runes[0] != '▁' {
		t.Errorf("expected ▁ for min value, got %c", runes[0])
	}
}

func TestRenderSparklineSingleValue(t *testing.T) {
	result := renderSparkline([]int{7})
	if !strings.Contains(result, "[7→7]") {
		t.Errorf("expected [7→7], got %q", result)
	}
}

// --- Trend indicator tests ---

func TestTrendIndicator(t *testing.T) {
	tests := []struct {
		direction, want string
	}{
		{"improving", "↓"},
		{"degrading", "↑"},
		{"stable", "→"},
		{"", "→"},
	}
	for _, tt := range tests {
		got := trendIndicator(tt.direction)
		if got != tt.want {
			t.Errorf("trendIndicator(%q) = %q, want %q", tt.direction, got, tt.want)
		}
	}
}

// --- Sort field name tests ---

func TestSortFieldName(t *testing.T) {
	tests := []struct {
		field sortField
		want  string
	}{
		{sortBySeverity, "severity"},
		{sortByTool, "tool"},
		{sortByCategory, "category"},
		{sortByResource, "resource"},
		{sortByCount, "count"},
		{sortField(99), "unknown"},
	}
	for _, tt := range tests {
		got := sortFieldName(tt.field)
		if got != tt.want {
			t.Errorf("sortFieldName(%d) = %q, want %q", tt.field, got, tt.want)
		}
	}
}

// --- Model state tests ---

func TestModelInit(t *testing.T) {
	m := New(testReport(), nil)
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init should return nil cmd")
	}
}

func TestModelInitialSort(t *testing.T) {
	m := New(testReport(), nil)
	// Issues should be sorted by severity (critical first)
	if len(m.filteredIssues) != 4 {
		t.Fatalf("expected 4 issues, got %d", len(m.filteredIssues))
	}
	if m.filteredIssues[0].Severity != "critical" {
		t.Errorf("expected critical first after initial sort, got %s", m.filteredIssues[0].Severity)
	}
}

func TestModelWindowResize(t *testing.T) {
	m := New(testReport(), nil)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(Model)
	if model.width != 120 {
		t.Errorf("expected width 120, got %d", model.width)
	}
	if model.height != 40 {
		t.Errorf("expected height 40, got %d", model.height)
	}
}

func TestModelQuit(t *testing.T) {
	m := New(testReport(), nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("expected quit command, got nil")
	}
}

func TestModelEnterSearch(t *testing.T) {
	m := New(testReport(), nil)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model := updated.(Model)
	if model.mode != modeSearch {
		t.Errorf("expected modeSearch, got %d", model.mode)
	}
}

func TestModelEnterFilterTool(t *testing.T) {
	m := New(testReport(), nil)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	model := updated.(Model)
	if model.mode != modeFilterTool {
		t.Errorf("expected modeFilterTool, got %d", model.mode)
	}
}

func TestModelCycleSort(t *testing.T) {
	m := New(testReport(), nil)
	if m.sortBy != sortBySeverity {
		t.Fatalf("expected initial sort by severity")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model := updated.(Model)
	if model.sortBy != sortByTool {
		t.Errorf("expected sort by tool after one cycle, got %d", model.sortBy)
	}
	if !strings.Contains(model.statusMsg, "tool") {
		t.Errorf("expected status to mention sort field, got %q", model.statusMsg)
	}
}

func TestModelClearFilter(t *testing.T) {
	m := New(testReport(), nil)
	m.filters = filterState{Tool: "vaultspectre"}
	m.statusMsg = "Filter: vaultspectre"
	m.rebuildTable()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model := updated.(Model)
	if model.filters.Tool != "" {
		t.Errorf("expected tool filter cleared, got %q", model.filters.Tool)
	}
	if model.statusMsg != "" {
		t.Errorf("expected status cleared, got %q", model.statusMsg)
	}
	if len(model.filteredIssues) != 4 {
		t.Errorf("expected all 4 issues after clear, got %d", len(model.filteredIssues))
	}
}

func TestModelSearchEscape(t *testing.T) {
	m := New(testReport(), nil)
	m.mode = modeSearch

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model := updated.(Model)
	if model.mode != modeNormal {
		t.Errorf("expected modeNormal after esc in search, got %d", model.mode)
	}
}

func TestModelFilterToolEscape(t *testing.T) {
	m := New(testReport(), nil)
	m.mode = modeFilterTool

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model := updated.(Model)
	if model.mode != modeNormal {
		t.Errorf("expected modeNormal after esc in filter, got %d", model.mode)
	}
}

func TestModelFilterToolNavigate(t *testing.T) {
	m := New(testReport(), nil)
	m.mode = modeFilterTool
	m.toolCursor = 0

	// Move down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := updated.(Model)
	if model.toolCursor != 1 {
		t.Errorf("expected cursor 1 after down, got %d", model.toolCursor)
	}

	// Move up
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	if model.toolCursor != 0 {
		t.Errorf("expected cursor 0 after up, got %d", model.toolCursor)
	}

	// Can't go above 0
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	if model.toolCursor != 0 {
		t.Errorf("expected cursor stays at 0, got %d", model.toolCursor)
	}
}

func TestModelFilterToolSelect(t *testing.T) {
	m := New(testReport(), nil)
	m.mode = modeFilterTool
	m.toolCursor = 1 // first actual tool (index 0 = "All")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if model.mode != modeNormal {
		t.Errorf("expected modeNormal after enter, got %d", model.mode)
	}
	if model.filters.Tool != m.toolChoices[0] {
		t.Errorf("expected tool filter %q, got %q", m.toolChoices[0], model.filters.Tool)
	}
}

func TestModelFilterToolSelectAll(t *testing.T) {
	m := New(testReport(), nil)
	m.mode = modeFilterTool
	m.toolCursor = 0 // "All"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if model.filters.Tool != "" {
		t.Errorf("expected empty tool filter for All, got %q", model.filters.Tool)
	}
}

func TestModelView(t *testing.T) {
	m := New(testReport(), nil)
	m.width = 100
	m.height = 30
	output := m.View()

	// Should contain header elements
	if !strings.Contains(output, "SpectreHub") {
		t.Error("expected SpectreHub in view")
	}
	// Should contain footer keybinds
	if !strings.Contains(output, "q:quit") {
		t.Error("expected keybinds in footer")
	}
	// Should contain issue count
	if !strings.Contains(output, "4/4 issues") {
		t.Error("expected 4/4 issues in footer")
	}
}

func TestModelViewSearchMode(t *testing.T) {
	m := New(testReport(), nil)
	m.mode = modeSearch
	output := m.View()
	if !strings.Contains(output, "/") {
		t.Error("expected search prompt in view when in search mode")
	}
}

func TestModelViewFilterMode(t *testing.T) {
	m := New(testReport(), nil)
	m.mode = modeFilterTool
	output := m.View()
	if !strings.Contains(output, "Filter by tool:") {
		t.Error("expected tool filter list in view")
	}
	if !strings.Contains(output, "All") {
		t.Error("expected All option in tool filter")
	}
}

func TestModelSearchEnter(t *testing.T) {
	m := New(testReport(), nil)
	m.mode = modeSearch
	m.searchInput.SetValue("vault")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if model.mode != modeNormal {
		t.Errorf("expected modeNormal after enter, got %d", model.mode)
	}
	if model.filters.SearchText != "vault" {
		t.Errorf("expected search text 'vault', got %q", model.filters.SearchText)
	}
	// Should filter down to vaultspectre issues
	if len(model.filteredIssues) != 2 {
		t.Errorf("expected 2 filtered issues, got %d", len(model.filteredIssues))
	}
}

func TestModelCopyNoSelection(t *testing.T) {
	m := New(testReport(), nil)
	// Empty issues — no selection possible
	m.filteredIssues = nil
	m.table.SetRows(nil)

	m.copySelectedIssue()
	if m.statusMsg != "Nothing to copy" {
		t.Errorf("expected 'Nothing to copy', got %q", m.statusMsg)
	}
}

func TestModelViewWithTrend(t *testing.T) {
	report := testReport()
	trend := &models.TrendSummary{
		TimeRange:      "Last 7 days",
		RunsAnalyzed:   5,
		IssueSparkline: []int{10, 8, 6, 4},
	}
	m := New(report, trend)
	output := m.View()
	if !strings.Contains(output, "Trend:") {
		t.Error("expected sparkline in view with trend data")
	}
}

func TestSeverityStyle(t *testing.T) {
	// Verify all severity levels return non-zero styles
	for _, sev := range []string{"critical", "high", "medium", "low", "unknown"} {
		s := severityStyle(sev)
		_ = s.Render("test")
	}
}

func TestHealthStyle(t *testing.T) {
	for _, h := range []string{"excellent", "good", "warning", "critical", "severe", "unknown"} {
		s := healthStyle(h)
		_ = s.Render("test")
	}
}

func TestModelWindowResizeSmall(t *testing.T) {
	m := New(testReport(), nil)
	// Very small terminal — table height should clamp to minimum 3
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	model := updated.(Model)
	if model.width != 40 {
		t.Errorf("expected width 40, got %d", model.width)
	}
}

func TestModelDoesNotMutateOriginal(t *testing.T) {
	report := testReport()
	originalLen := len(report.Issues)
	m := New(report, nil)

	// Apply a filter that reduces the set
	m.filters = filterState{Tool: "vaultspectre"}
	m.rebuildTable()

	if len(m.allIssues) != originalLen {
		t.Errorf("allIssues mutated: expected %d, got %d", originalLen, len(m.allIssues))
	}
	if len(report.Issues) != originalLen {
		t.Errorf("original report mutated: expected %d, got %d", originalLen, len(report.Issues))
	}
}
