package models

// PgReport represents the JSON output from PgSpectre.
type PgReport struct {
	Metadata    PgMetadata    `json:"metadata"`
	Findings    []PgFinding   `json:"findings"`
	MaxSeverity string        `json:"maxSeverity"`
	Summary     PgSummary     `json:"summary"`
	Scanned     PgScanContext `json:"scanned,omitempty"`
}

// PgMetadata holds report context for PgSpectre.
type PgMetadata struct {
	Tool      string `json:"tool"`
	Version   string `json:"version"`
	Command   string `json:"command"`
	Timestamp string `json:"timestamp"`
}

// PgSummary counts findings by severity.
type PgSummary struct {
	Total  int `json:"total"`
	High   int `json:"high"`
	Medium int `json:"medium"`
	Low    int `json:"low"`
	Info   int `json:"info"`
}

// PgScanContext describes scanned objects.
type PgScanContext struct {
	Tables  int `json:"tables"`
	Indexes int `json:"indexes"`
	Schemas int `json:"schemas"`
}

// PgFinding represents a single PgSpectre finding.
type PgFinding struct {
	Type     string            `json:"type"`
	Severity string            `json:"severity"`
	Schema   string            `json:"schema"`
	Table    string            `json:"table"`
	Column   string            `json:"column,omitempty"`
	Index    string            `json:"index,omitempty"`
	Message  string            `json:"message"`
	Detail   map[string]string `json:"detail,omitempty"`
}
