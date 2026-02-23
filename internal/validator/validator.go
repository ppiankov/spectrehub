package validator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

// ValidationError represents a validation failure
type ValidationError struct {
	Tool   string
	Errors []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("Invalid %s report:\n  - %s", e.Tool, strings.Join(e.Errors, "\n  - "))
}

// Validator validates JSON reports from Spectre tools
type Validator struct{}

// New creates a new validator
func New() *Validator {
	return &Validator{}
}

// ValidateReport validates a report based on tool type.
// spectre/v1 envelopes are validated using the envelope validator regardless of tool type.
func (v *Validator) ValidateReport(toolType models.ToolType, data []byte) error {
	if isSpectreV1(data) {
		return v.ValidateSpectreV1Report(data)
	}

	switch toolType {
	case models.ToolVault:
		return v.ValidateVaultReport(data)
	case models.ToolS3:
		return v.ValidateS3Report(data)
	case models.ToolKafka:
		return v.ValidateKafkaReport(data)
	case models.ToolClickHouse:
		return v.ValidateClickHouseReport(data)
	default:
		// Unknown tools are accepted but not validated
		return nil
	}
}

// isSpectreV1 checks if the data contains a spectre/v1 schema field.
func isSpectreV1(data []byte) bool {
	var s struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(data, &s); err == nil && s.Schema == "spectre/v1" {
		return true
	}
	return false
}

// ValidateSpectreV1Report validates a spectre/v1 envelope.
func (v *Validator) ValidateSpectreV1Report(data []byte) error {
	var report models.SpectreV1Report
	if err := json.Unmarshal(data, &report); err != nil {
		return &ValidationError{
			Tool:   "spectre/v1",
			Errors: []string{fmt.Sprintf("Failed to parse JSON: %v", err)},
		}
	}

	var errs []string

	if report.Schema != "spectre/v1" {
		errs = append(errs, fmt.Sprintf("Invalid schema: %q (expected spectre/v1)", report.Schema))
	}
	if report.Tool == "" {
		errs = append(errs, "Missing required field: 'tool'")
	}
	if report.Version == "" {
		errs = append(errs, "Missing required field: 'version'")
	}
	if report.Timestamp.IsZero() {
		errs = append(errs, "Missing or invalid field: 'timestamp'")
	}

	// Validate target
	if report.Target.Type == "" {
		errs = append(errs, "Missing required field: 'target.type'")
	} else if report.Tool != "" {
		if expected, ok := models.SpectreV1TargetTypes[report.Tool]; ok {
			if report.Target.Type != expected {
				errs = append(errs, fmt.Sprintf("target.type %q does not match tool %q (expected %q)", report.Target.Type, report.Tool, expected))
			}
		}
	}

	// Validate findings
	if report.Findings == nil {
		errs = append(errs, "Missing required field: 'findings' (must be array, not null)")
	}
	for i, f := range report.Findings {
		if f.ID == "" {
			errs = append(errs, fmt.Sprintf("findings[%d]: missing required field 'id'", i))
		}
		if !models.ValidSpectreV1Severities[f.Severity] {
			errs = append(errs, fmt.Sprintf("findings[%d]: invalid severity %q (allowed: high, medium, low, info)", i, f.Severity))
		}
		if f.Location == "" {
			errs = append(errs, fmt.Sprintf("findings[%d]: missing required field 'location'", i))
		}
		if f.Message == "" {
			errs = append(errs, fmt.Sprintf("findings[%d]: missing required field 'message'", i))
		}
	}

	// Validate summary totals match
	expectedTotal := len(report.Findings)
	if report.Summary.Total != expectedTotal {
		errs = append(errs, fmt.Sprintf("summary.total=%d does not match findings count=%d", report.Summary.Total, expectedTotal))
	}

	if len(errs) > 0 {
		return &ValidationError{Tool: "spectre/v1", Errors: errs}
	}

	return nil
}

// ValidateVaultReport validates VaultSpectre JSON output
func (v *Validator) ValidateVaultReport(data []byte) error {
	var report models.VaultReport
	if err := json.Unmarshal(data, &report); err != nil {
		return &ValidationError{
			Tool:   "VaultSpectre",
			Errors: []string{fmt.Sprintf("Failed to parse JSON: %v", err)},
		}
	}

	var errors []string

	// Check required fields
	if report.Tool == "" {
		errors = append(errors, "Missing required field: 'tool'")
	}
	if report.Timestamp.IsZero() {
		errors = append(errors, "Missing or invalid field: 'timestamp'")
	}

	// Validate summary
	if report.Summary.TotalReferences < 0 {
		errors = append(errors, "Field 'summary.total_references' must be non-negative")
	}

	// Validate secret statuses
	validStatuses := map[string]bool{
		"ok": true, "missing": true, "access_denied": true,
		"invalid": true, "dynamic": true, "error": true,
	}

	for path, secret := range report.Secrets {
		if !validStatuses[secret.Status] {
			errors = append(errors, fmt.Sprintf("Secret '%s' has invalid status: '%s'", path, secret.Status))
		}
	}

	if len(errors) > 0 {
		return &ValidationError{Tool: "VaultSpectre", Errors: errors}
	}

	return nil
}

// ValidateS3Report validates S3Spectre JSON output
func (v *Validator) ValidateS3Report(data []byte) error {
	var report models.S3Report
	if err := json.Unmarshal(data, &report); err != nil {
		return &ValidationError{
			Tool:   "S3Spectre",
			Errors: []string{fmt.Sprintf("Failed to parse JSON: %v", err)},
		}
	}

	var errors []string

	// Check required fields
	if report.Tool == "" {
		errors = append(errors, "Missing required field: 'tool'")
	}
	if report.Timestamp.IsZero() {
		errors = append(errors, "Missing or invalid field: 'timestamp'")
	}

	// Validate summary
	if report.Summary.TotalBuckets < 0 {
		errors = append(errors, "Field 'summary.total_buckets' must be non-negative")
	}

	// Validate bucket statuses
	validStatuses := map[string]bool{
		"OK": true, "MISSING_BUCKET": true, "UNUSED_BUCKET": true,
		"MISSING_PREFIX": true, "STALE_PREFIX": true,
		"VERSION_SPRAWL": true, "LIFECYCLE_MISCONFIG": true,
	}

	for name, bucket := range report.Buckets {
		if !validStatuses[bucket.Status] {
			errors = append(errors, fmt.Sprintf("Bucket '%s' has invalid status: '%s'", name, bucket.Status))
		}
	}

	if len(errors) > 0 {
		return &ValidationError{Tool: "S3Spectre", Errors: errors}
	}

	return nil
}

// ValidateKafkaReport validates KafkaSpectre JSON output
func (v *Validator) ValidateKafkaReport(data []byte) error {
	var report models.KafkaReport
	if err := json.Unmarshal(data, &report); err != nil {
		return &ValidationError{
			Tool:   "KafkaSpectre",
			Errors: []string{fmt.Sprintf("Failed to parse JSON: %v", err)},
		}
	}

	var errors []string

	// Check required fields
	if report.Summary == nil {
		errors = append(errors, "Missing required field: 'summary'")
	} else {
		// Validate summary fields
		if report.Summary.TotalTopics < 0 {
			errors = append(errors, "Field 'summary.total_topics_analyzed' must be non-negative")
		}
		if report.Summary.TotalBrokers < 0 {
			errors = append(errors, "Field 'summary.total_brokers' must be non-negative")
		}
	}

	if report.ClusterMetadata == nil {
		errors = append(errors, "Missing required field: 'cluster_metadata'")
	}

	// Validate unused topics
	validRisks := map[string]bool{
		"low": true, "medium": true, "high": true,
	}

	for _, topic := range report.UnusedTopics {
		if !validRisks[topic.Risk] {
			errors = append(errors, fmt.Sprintf("Topic '%s' has invalid risk: '%s'", topic.Name, topic.Risk))
		}
		if topic.Partitions < 0 {
			errors = append(errors, fmt.Sprintf("Topic '%s' has invalid partition count: %d", topic.Name, topic.Partitions))
		}
	}

	if len(errors) > 0 {
		return &ValidationError{Tool: "KafkaSpectre", Errors: errors}
	}

	return nil
}

// ValidateClickHouseReport validates ClickSpectre JSON output
func (v *Validator) ValidateClickHouseReport(data []byte) error {
	var report models.ClickHouseReport
	if err := json.Unmarshal(data, &report); err != nil {
		return &ValidationError{
			Tool:   "ClickSpectre",
			Errors: []string{fmt.Sprintf("Failed to parse JSON: %v", err)},
		}
	}

	var errors []string

	// Check required fields
	if report.Metadata.GeneratedAt.IsZero() {
		errors = append(errors, "Missing or invalid field: 'metadata.generated_at'")
	}

	// Validate metadata
	if report.Metadata.LookbackDays < 0 {
		errors = append(errors, "Field 'metadata.lookback_days' must be non-negative")
	}

	// Validate tables
	validCategories := map[string]bool{
		"active": true, "unused": true, "suspect": true,
	}

	for _, table := range report.Tables {
		if !validCategories[table.Category] {
			errors = append(errors, fmt.Sprintf("Table '%s' has invalid category: '%s'", table.FullName, table.Category))
		}
		// Note: table.Reads is uint64, so it cannot be negative
	}

	if len(errors) > 0 {
		return &ValidationError{Tool: "ClickSpectre", Errors: errors}
	}

	return nil
}

// ValidateTimestamp checks if a timestamp is reasonable (not in future, not too old)
func ValidateTimestamp(t time.Time) error {
	now := time.Now()

	// Not in future
	if t.After(now.Add(1 * time.Hour)) {
		return fmt.Errorf("timestamp is in the future: %v", t)
	}

	// Not older than 1 year
	oneYearAgo := now.AddDate(-1, 0, 0)
	if t.Before(oneYearAgo) {
		return fmt.Errorf("timestamp is too old (> 1 year): %v", t)
	}

	return nil
}
