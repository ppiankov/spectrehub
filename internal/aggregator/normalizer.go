package aggregator

import (
	"fmt"

	"github.com/ppiankov/spectrehub/internal/models"
)

// Normalizer converts tool-specific reports into normalized issues
type Normalizer struct{}

// NewNormalizer creates a new normalizer
func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

// Normalize converts a tool report into normalized issues
func (n *Normalizer) Normalize(report *models.ToolReport) ([]models.NormalizedIssue, error) {
	if !report.IsSupported {
		// Don't normalize unsupported tools
		return nil, nil
	}

	switch models.ToolType(report.Tool) {
	case models.ToolVault:
		return n.NormalizeVault(report)
	case models.ToolS3:
		return n.NormalizeS3(report)
	case models.ToolKafka:
		return n.NormalizeKafka(report)
	case models.ToolClickHouse:
		return n.NormalizeClickHouse(report)
	default:
		return nil, fmt.Errorf("unknown tool type: %s", report.Tool)
	}
}

// NormalizeVault converts VaultSpectre report to normalized issues
func (n *Normalizer) NormalizeVault(report *models.ToolReport) ([]models.NormalizedIssue, error) {
	vaultReport, ok := report.RawData.(*models.VaultReport)
	if !ok {
		return nil, fmt.Errorf("invalid vault report data")
	}

	var issues []models.NormalizedIssue

	for path, secret := range vaultReport.Secrets {
		// Skip OK secrets
		if secret.Status == "ok" {
			continue
		}

		category := mapVaultStatus(secret.Status)
		severity := models.DetermineSeverity(category, models.ToolVault)

		issue := models.NormalizedIssue{
			Tool:      report.Tool,
			Category:  category,
			Severity:  severity,
			Resource:  path,
			Evidence:  buildVaultEvidence(secret),
			Count:     len(secret.References),
			FirstSeen: report.Timestamp,
			LastSeen:  report.Timestamp,
		}

		issues = append(issues, issue)
	}

	return issues, nil
}

// NormalizeS3 converts S3Spectre report to normalized issues
func (n *Normalizer) NormalizeS3(report *models.ToolReport) ([]models.NormalizedIssue, error) {
	s3Report, ok := report.RawData.(*models.S3Report)
	if !ok {
		return nil, fmt.Errorf("invalid S3 report data")
	}

	var issues []models.NormalizedIssue

	for name, bucket := range s3Report.Buckets {
		// Skip OK buckets
		if bucket.Status == "OK" {
			continue
		}

		category := mapS3Status(bucket.Status)
		severity := models.DetermineSeverity(category, models.ToolS3)

		resource := fmt.Sprintf("s3://%s", name)
		if len(bucket.Prefixes) > 0 {
			// If there are prefix issues, create separate issues for each
			for _, prefix := range bucket.Prefixes {
				if prefix.Status != "OK" {
					prefixCategory := mapS3Status(prefix.Status)
					prefixSeverity := models.DetermineSeverity(prefixCategory, models.ToolS3)

					issue := models.NormalizedIssue{
						Tool:      report.Tool,
						Category:  prefixCategory,
						Severity:  prefixSeverity,
						Resource:  fmt.Sprintf("s3://%s/%s", name, prefix.Prefix),
						Evidence:  prefix.Message,
						Count:     prefix.ObjectCount,
						FirstSeen: report.Timestamp,
						LastSeen:  report.Timestamp,
					}
					issues = append(issues, issue)
				}
			}
		} else {
			issue := models.NormalizedIssue{
				Tool:      report.Tool,
				Category:  category,
				Severity:  severity,
				Resource:  resource,
				Evidence:  bucket.Message,
				Count:     1,
				FirstSeen: report.Timestamp,
				LastSeen:  report.Timestamp,
			}
			issues = append(issues, issue)
		}
	}

	return issues, nil
}

// NormalizeKafka converts KafkaSpectre report to normalized issues
func (n *Normalizer) NormalizeKafka(report *models.ToolReport) ([]models.NormalizedIssue, error) {
	kafkaReport, ok := report.RawData.(*models.KafkaReport)
	if !ok {
		return nil, fmt.Errorf("invalid Kafka report data")
	}

	var issues []models.NormalizedIssue

	// Process unused topics
	for _, topic := range kafkaReport.UnusedTopics {
		severity := mapKafkaRiskToSeverity(topic.Risk)

		issue := models.NormalizedIssue{
			Tool:      report.Tool,
			Category:  models.StatusUnused,
			Severity:  severity,
			Resource:  fmt.Sprintf("topic:%s", topic.Name),
			Evidence:  fmt.Sprintf("%s (partitions: %d, risk: %s)", topic.Reason, topic.Partitions, topic.Risk),
			Count:     topic.Partitions,
			FirstSeen: report.Timestamp,
			LastSeen:  report.Timestamp,
		}

		issues = append(issues, issue)
	}

	return issues, nil
}

// NormalizeClickHouse converts ClickSpectre report to normalized issues
func (n *Normalizer) NormalizeClickHouse(report *models.ToolReport) ([]models.NormalizedIssue, error) {
	clickReport, ok := report.RawData.(*models.ClickHouseReport)
	if !ok {
		return nil, fmt.Errorf("invalid ClickHouse report data")
	}

	var issues []models.NormalizedIssue

	// Process zero usage tables
	for _, table := range clickReport.Tables {
		if !table.ZeroUsage {
			continue
		}

		// Determine severity based on replication
		severity := models.SeverityMedium
		if !table.IsReplicated {
			severity = models.SeverityLow
		}

		issue := models.NormalizedIssue{
			Tool:      report.Tool,
			Category:  models.StatusUnused,
			Severity:  severity,
			Resource:  table.FullName,
			Evidence:  fmt.Sprintf("zero usage (reads: %d, writes: %d)", table.Reads, table.Writes),
			Count:     1,
			FirstSeen: table.FirstSeen,
			LastSeen:  table.LastAccess,
		}

		issues = append(issues, issue)
	}

	// Process anomalies as drift/misconfig issues
	for _, anomaly := range clickReport.Anomalies {
		category := models.StatusDrift
		if anomaly.Type == "configuration" {
			category = models.StatusMisconfig
		}

		issue := models.NormalizedIssue{
			Tool:      report.Tool,
			Category:  category,
			Severity:  anomaly.Severity,
			Resource:  anomaly.AffectedTable,
			Evidence:  anomaly.Description,
			Count:     1,
			FirstSeen: anomaly.DetectedAt,
			LastSeen:  anomaly.DetectedAt,
		}

		issues = append(issues, issue)
	}

	return issues, nil
}

// Helper functions to map tool-specific statuses to normalized categories

func mapVaultStatus(status string) string {
	switch status {
	case "missing":
		return models.StatusMissing
	case "access_denied":
		return models.StatusAccessDeny
	case "invalid":
		return models.StatusInvalid
	case "error":
		return models.StatusError
	case "stale":
		return models.StatusStale
	default:
		return models.StatusError
	}
}

func mapS3Status(status string) string {
	switch status {
	case "MISSING_BUCKET", "MISSING_PREFIX":
		return models.StatusMissing
	case "UNUSED_BUCKET":
		return models.StatusUnused
	case "STALE_PREFIX":
		return models.StatusStale
	case "VERSION_SPRAWL", "LIFECYCLE_MISCONFIG":
		return models.StatusMisconfig
	default:
		return models.StatusError
	}
}

func mapKafkaRiskToSeverity(risk string) string {
	switch risk {
	case "high":
		return models.SeverityHigh
	case "medium":
		return models.SeverityMedium
	case "low":
		return models.SeverityLow
	default:
		return models.SeverityMedium
	}
}

func buildVaultEvidence(secret *models.SecretInfo) string {
	if secret.ErrorMsg != "" {
		return secret.ErrorMsg
	}
	if secret.IsStale && secret.LastAccessed != "" {
		return fmt.Sprintf("stale (last accessed: %s)", secret.LastAccessed)
	}
	return fmt.Sprintf("status: %s", secret.Status)
}
