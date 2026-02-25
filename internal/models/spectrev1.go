package models

import "time"

// SpectreV1Report represents a spectre/v1 envelope — the standardized output
// format that all spectre tools emit with --format spectrehub.
type SpectreV1Report struct {
	Schema    string             `json:"schema"`
	Tool      string             `json:"tool"`
	Version   string             `json:"version"`
	Timestamp time.Time          `json:"timestamp"`
	Target    SpectreV1Target    `json:"target"`
	Findings  []SpectreV1Finding `json:"findings"`
	Summary   SpectreV1Summary   `json:"summary"`
}

// SpectreV1Target describes what was scanned.
type SpectreV1Target struct {
	Type    string `json:"type"` // s3, postgres, kafka, clickhouse, vault, mongodb, aws-account, gcp-project, gcs
	URIHash string `json:"uri_hash,omitempty"`
}

// SpectreV1Finding is a single issue in the spectre/v1 envelope.
type SpectreV1Finding struct {
	ID       string `json:"id"`
	Severity string `json:"severity"` // high, medium, low, info
	Location string `json:"location"`
	Message  string `json:"message"`
}

// SpectreV1Summary counts findings by severity.
type SpectreV1Summary struct {
	Total  int `json:"total"`
	High   int `json:"high"`
	Medium int `json:"medium"`
	Low    int `json:"low"`
	Info   int `json:"info"`
}

// ValidSpectreV1Severities defines the allowed severity values in spectre/v1 findings.
var ValidSpectreV1Severities = map[string]bool{
	"high":   true,
	"medium": true,
	"low":    true,
	"info":   true,
}

// SpectreV1TargetTypes maps tool names to their expected target.type values.
// Tools not listed here (e.g., iamspectre which emits aws-account or gcp-project
// depending on the cloud) are accepted with any target type.
var SpectreV1TargetTypes = map[string]string{
	"s3spectre":    "s3",
	"pgspectre":    "postgres",
	"kafkaspectre": "kafka",
	"clickspectre": "clickhouse",
	"vaultspectre": "vault",
	"mongospectre": "mongodb",
	"awsspectre":   "aws-account",
	"gcsspectre":   "gcs",
}
