package models

// MongoReport represents the JSON output from MongoSpectre.
type MongoReport struct {
	Metadata    MongoMetadata  `json:"metadata"`
	Findings    []MongoFinding `json:"findings"`
	MaxSeverity string         `json:"maxSeverity"`
	Summary     MongoSummary   `json:"summary"`
}

// MongoMetadata holds report context for MongoSpectre.
type MongoMetadata struct {
	Version        string `json:"version"`
	Command        string `json:"command"`
	Timestamp      string `json:"timestamp"`
	Database       string `json:"database,omitempty"`
	MongoDBVersion string `json:"mongodbVersion,omitempty"`
	RepoPath       string `json:"repoPath,omitempty"`
	URIHash        string `json:"uriHash,omitempty"`
}

// MongoSummary counts findings by severity.
type MongoSummary struct {
	Total  int `json:"total"`
	High   int `json:"high"`
	Medium int `json:"medium"`
	Low    int `json:"low"`
	Info   int `json:"info"`
}

// MongoFinding represents a single MongoSpectre finding.
type MongoFinding struct {
	Type       string `json:"type"`
	Severity   string `json:"severity"`
	Database   string `json:"database"`
	Collection string `json:"collection"`
	Index      string `json:"index,omitempty"`
	Message    string `json:"message"`
}
