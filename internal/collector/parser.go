package collector

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

// ParseReport parses JSON data based on detected tool type.
// If the data is a spectre/v1 envelope, it is parsed directly regardless of tool type.
func ParseReport(data []byte, toolType models.ToolType) (interface{}, error) {
	if IsSpectreV1(data) {
		return ParseSpectreV1Report(data)
	}

	switch toolType {
	case models.ToolVault:
		return ParseVaultReport(data)
	case models.ToolS3:
		return ParseS3Report(data)
	case models.ToolKafka:
		return ParseKafkaReport(data)
	case models.ToolClickHouse:
		return ParseClickHouseReport(data)
	case models.ToolPg:
		return ParsePgReport(data)
	case models.ToolMongo:
		return ParseMongoReport(data)
	default:
		return ParseUnsupportedReport(data)
	}
}

// ParseSpectreV1Report parses a spectre/v1 envelope JSON.
func ParseSpectreV1Report(data []byte) (*models.SpectreV1Report, error) {
	var report models.SpectreV1Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to parse spectre/v1 report: %w", err)
	}

	if report.Schema != "spectre/v1" {
		return nil, fmt.Errorf("expected schema spectre/v1, got %q", report.Schema)
	}
	if report.Tool == "" {
		return nil, fmt.Errorf("spectre/v1 envelope missing required field: tool")
	}
	if report.Findings == nil {
		report.Findings = []models.SpectreV1Finding{}
	}

	return &report, nil
}

// ParseVaultReport parses VaultSpectre JSON output
func ParseVaultReport(data []byte) (*models.VaultReport, error) {
	var report models.VaultReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to parse VaultSpectre report: %w", err)
	}

	// Validate required fields
	if report.Tool == "" {
		report.Tool = "vaultspectre"
	}
	if report.Timestamp.IsZero() {
		report.Timestamp = time.Now()
	}

	return &report, nil
}

// ParseS3Report parses S3Spectre JSON output
func ParseS3Report(data []byte) (*models.S3Report, error) {
	var report models.S3Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to parse S3Spectre report: %w", err)
	}

	// Validate required fields
	if report.Tool == "" {
		report.Tool = "s3spectre"
	}
	if report.Timestamp.IsZero() {
		report.Timestamp = time.Now()
	}

	return &report, nil
}

// ParseKafkaReport parses KafkaSpectre JSON output
func ParseKafkaReport(data []byte) (*models.KafkaReport, error) {
	var report models.KafkaReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to parse KafkaSpectre report: %w", err)
	}

	// KafkaSpectre doesn't have top-level tool/version/timestamp fields
	// We'll need to infer or add them during normalization

	return &report, nil
}

// ParseClickHouseReport parses ClickSpectre JSON output
func ParseClickHouseReport(data []byte) (*models.ClickHouseReport, error) {
	var report models.ClickHouseReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to parse ClickSpectre report: %w", err)
	}

	return &report, nil
}

// ParsePgReport parses PgSpectre JSON output
func ParsePgReport(data []byte) (*models.PgReport, error) {
	var report models.PgReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to parse PgSpectre report: %w", err)
	}

	if report.Metadata.Tool == "" {
		report.Metadata.Tool = "pgspectre"
	}
	if report.Metadata.Timestamp == "" {
		report.Metadata.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	if report.Findings == nil {
		report.Findings = []models.PgFinding{}
	}

	return &report, nil
}

// ParseMongoReport parses MongoSpectre JSON output
func ParseMongoReport(data []byte) (*models.MongoReport, error) {
	var report models.MongoReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to parse MongoSpectre report: %w", err)
	}

	if report.Metadata.Timestamp == "" {
		report.Metadata.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	if report.Findings == nil {
		report.Findings = []models.MongoFinding{}
	}

	return &report, nil
}

// ParseUnsupportedReport stores raw JSON for unsupported tools
func ParseUnsupportedReport(data []byte) (map[string]interface{}, error) {
	var rawData map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse unsupported report: %w", err)
	}

	return rawData, nil
}

// ExtractVersion attempts to extract version from any report
func ExtractVersion(data []byte) string {
	var versionField struct {
		Version string `json:"version"`
	}

	if err := json.Unmarshal(data, &versionField); err == nil && versionField.Version != "" {
		return versionField.Version
	}

	// Try metadata field (for ClickSpectre)
	var metadataVersion struct {
		Metadata struct {
			Version string `json:"version"`
		} `json:"metadata"`
	}

	if err := json.Unmarshal(data, &metadataVersion); err == nil && metadataVersion.Metadata.Version != "" {
		return metadataVersion.Metadata.Version
	}

	return "unknown"
}

// ExtractTimestamp attempts to extract timestamp from any report
func ExtractTimestamp(data []byte) time.Time {
	// Try top-level timestamp
	var timestampField struct {
		Timestamp time.Time `json:"timestamp"`
	}

	if err := json.Unmarshal(data, &timestampField); err == nil && !timestampField.Timestamp.IsZero() {
		return timestampField.Timestamp
	}

	// Try metadata generated_at (for ClickSpectre)
	var metadataTimestamp struct {
		Metadata struct {
			GeneratedAt time.Time `json:"generated_at"`
		} `json:"metadata"`
	}

	if err := json.Unmarshal(data, &metadataTimestamp); err == nil && !metadataTimestamp.Metadata.GeneratedAt.IsZero() {
		return metadataTimestamp.Metadata.GeneratedAt
	}

	// Try metadata timestamp string (PgSpectre, MongoSpectre)
	var metadataTimeString struct {
		Metadata struct {
			Timestamp string `json:"timestamp"`
		} `json:"metadata"`
	}

	if err := json.Unmarshal(data, &metadataTimeString); err == nil && metadataTimeString.Metadata.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, metadataTimeString.Metadata.Timestamp); err == nil {
			return t
		}
	}

	// Try cluster_metadata fetched_at (for KafkaSpectre)
	var kafkaTimestamp struct {
		ClusterMetadata struct {
			FetchedAt string `json:"fetched_at"`
		} `json:"cluster_metadata"`
	}

	if err := json.Unmarshal(data, &kafkaTimestamp); err == nil && kafkaTimestamp.ClusterMetadata.FetchedAt != "" {
		// Try parsing the timestamp string
		if t, err := time.Parse("2006-01-02 15:04:05 MST", kafkaTimestamp.ClusterMetadata.FetchedAt); err == nil {
			return t
		}
	}

	// Default to current time if we can't find timestamp
	return time.Now()
}
