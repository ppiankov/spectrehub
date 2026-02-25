package models

import "time"

// ToolType represents the type of Spectre tool
type ToolType string

const (
	ToolVault      ToolType = "vaultspectre"
	ToolS3         ToolType = "s3spectre"
	ToolKafka      ToolType = "kafkaspectre"
	ToolClickHouse ToolType = "clickspectre"
	ToolPg         ToolType = "pgspectre"
	ToolMongo      ToolType = "mongospectre"
	ToolAWS        ToolType = "awsspectre"
	ToolUnknown    ToolType = "unknown"
)

// ToolInfo contains metadata about a supported tool
type ToolInfo struct {
	Name          string
	MinVersion    string
	HasValidation bool
	HasNormalizer bool
}

// SupportedTools defines explicitly supported tools
var SupportedTools = map[ToolType]ToolInfo{
	ToolVault: {
		Name:          "vaultspectre",
		MinVersion:    "0.1.0",
		HasValidation: true,
		HasNormalizer: true,
	},
	ToolS3: {
		Name:          "s3spectre",
		MinVersion:    "0.1.0",
		HasValidation: true,
		HasNormalizer: true,
	},
	ToolKafka: {
		Name:          "kafkaspectre",
		MinVersion:    "0.1.0",
		HasValidation: true,
		HasNormalizer: true,
	},
	ToolClickHouse: {
		Name:          "clickspectre",
		MinVersion:    "0.1.0",
		HasValidation: true,
		HasNormalizer: true,
	},
	ToolPg: {
		Name:          "pgspectre",
		MinVersion:    "0.1.0",
		HasValidation: false,
		HasNormalizer: true,
	},
	ToolMongo: {
		Name:          "mongospectre",
		MinVersion:    "0.1.0",
		HasValidation: false,
		HasNormalizer: true,
	},
	ToolAWS: {
		Name:          "awsspectre",
		MinVersion:    "0.1.0",
		HasValidation: true,
		HasNormalizer: true,
	},
}

// IsSupportedTool checks if a tool is explicitly supported
func IsSupportedTool(tool ToolType) bool {
	_, ok := SupportedTools[tool]
	return ok
}

// GetToolInfo returns information about a tool
func GetToolInfo(tool ToolType) (ToolInfo, bool) {
	info, ok := SupportedTools[tool]
	return info, ok
}

// Status categories for normalized issues
const (
	StatusOK         = "ok"
	StatusMissing    = "missing"
	StatusUnused     = "unused"
	StatusStale      = "stale"
	StatusDrift      = "drift"
	StatusError      = "error"
	StatusMisconfig  = "misconfig"
	StatusAccessDeny = "access_denied"
	StatusInvalid    = "invalid"
)

// Severity levels for issues
const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityLow      = "low"
)

// NormalizedIssue is the atomic unit that every tool maps into
// This enables diff, query, and correlation without refactoring
type NormalizedIssue struct {
	Tool      string    `json:"tool"`               // vaultspectre, s3spectre, etc.
	Category  string    `json:"category"`           // missing, unused, stale, error, misconfig
	Severity  string    `json:"severity"`           // critical, high, medium, low
	Resource  string    `json:"resource"`           // vault path / s3://bucket / topic / db.table
	Evidence  string    `json:"evidence,omitempty"` // short explanation
	Count     int       `json:"count,omitempty"`    // how many instances
	FirstSeen time.Time `json:"first_seen,omitempty"`
	LastSeen  time.Time `json:"last_seen,omitempty"`
}

// AggregatedReport contains the complete aggregated output from all tools
type AggregatedReport struct {
	Timestamp       time.Time             `json:"timestamp"`
	Issues          []NormalizedIssue     `json:"issues"`          // Atomic issue list
	ToolReports     map[string]ToolReport `json:"tool_reports"`    // Per-tool raw data
	Summary         CrossToolSummary      `json:"summary"`         // Overall statistics
	Trend           *Trend                `json:"trend,omitempty"` // Comparison with previous run
	Recommendations []Recommendation      `json:"recommendations"` // Prioritized actions
}

// ToolReport contains data for a single tool
type ToolReport struct {
	Tool        string      `json:"tool"`
	Version     string      `json:"version"`
	Timestamp   time.Time   `json:"timestamp"`
	RawData     interface{} `json:"raw_data"`     // Original tool output
	Summary     interface{} `json:"summary"`      // Tool-specific summary
	Score       float64     `json:"score"`        // Health score (0-100)
	Status      string      `json:"status"`       // "supported" or "unsupported"
	IssueCount  int         `json:"issue_count"`  // Total issues from this tool
	IsSupported bool        `json:"is_supported"` // Whether tool is explicitly supported
}

// CrossToolSummary provides aggregate statistics across all tools
type CrossToolSummary struct {
	TotalIssues      int            `json:"total_issues"`
	IssuesByTool     map[string]int `json:"issues_by_tool"`
	IssuesByCategory map[string]int `json:"issues_by_category"`
	IssuesBySeverity map[string]int `json:"issues_by_severity"`
	HealthScore      string         `json:"health_score"`  // excellent, good, warning, critical, severe
	ScorePercent     float64        `json:"score_percent"` // 0-100
	TotalTools       int            `json:"total_tools"`
	SupportedTools   int            `json:"supported_tools"`
	UnsupportedTools int            `json:"unsupported_tools"`
}

// Trend represents change between current and previous run
type Trend struct {
	Direction      string    `json:"direction"`      // "improving", "degrading", "stable"
	ChangePercent  float64   `json:"change_percent"` // negative = improvement
	PreviousIssues int       `json:"previous_issues"`
	CurrentIssues  int       `json:"current_issues"`
	ComparedWith   time.Time `json:"compared_with"`   // When previous run was
	NewIssues      int       `json:"new_issues"`      // Issues that appeared
	ResolvedIssues int       `json:"resolved_issues"` // Issues that disappeared
}

// Recommendation represents an actionable item to fix
type Recommendation struct {
	Severity string `json:"severity"` // critical, high, medium, low
	Tool     string `json:"tool"`
	Action   string `json:"action"` // What to do
	Impact   string `json:"impact"` // Why it matters
	Count    int    `json:"count"`  // How many items
}

// TrendSummary provides historical trend analysis
type TrendSummary struct {
	TimeRange      string                `json:"time_range"` // e.g., "Last 7 days"
	RunsAnalyzed   int                   `json:"runs_analyzed"`
	IssueSparkline []int                 `json:"issue_sparkline"` // Issue counts over time
	ByTool         map[string]*ToolTrend `json:"by_tool"`
}

// ToolTrend represents trend for a single tool
type ToolTrend struct {
	Name           string  `json:"name"`
	CurrentIssues  int     `json:"current_issues"`
	PreviousIssues int     `json:"previous_issues"`
	Change         int     `json:"change"`         // Positive = more issues
	ChangePercent  float64 `json:"change_percent"` // Positive = more issues
}

// CalculateHealthScore determines overall health from affected vs total resources.
// affectedResources is the count of unique resources with at least one issue.
// score = (totalResources - affectedResources) / totalResources * 100, clamped 0-100.
func CalculateHealthScore(affectedResources, totalResources int) (string, float64) {
	if totalResources == 0 {
		return "unknown", 0.0
	}

	// Score = percentage of clean (unaffected) resources.
	score := float64(totalResources-affectedResources) / float64(totalResources) * 100.0

	// Clamp to 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	// Determine health level
	var health string
	switch {
	case score >= 95:
		health = "excellent"
	case score >= 85:
		health = "good"
	case score >= 70:
		health = "warning"
	case score >= 50:
		health = "critical"
	default:
		health = "severe"
	}

	return health, score
}

// DetermineSeverity maps issue category to severity level
func DetermineSeverity(category string, toolType ToolType) string {
	// Critical issues
	if category == StatusMissing || category == StatusError {
		return SeverityCritical
	}

	// High severity
	if category == StatusAccessDeny || category == StatusInvalid {
		return SeverityHigh
	}

	// Medium severity
	if category == StatusDrift || category == StatusMisconfig {
		return SeverityMedium
	}

	// Low severity (stale, unused)
	return SeverityLow
}
