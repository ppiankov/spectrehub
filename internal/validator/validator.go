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

// ValidateReport validates a report based on tool type
func (v *Validator) ValidateReport(toolType models.ToolType, data []byte) error {
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
