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

// Normalize converts a tool report into normalized issues.
// spectre/v1 envelopes are normalized directly from their findings array.
func (n *Normalizer) Normalize(report *models.ToolReport) ([]models.NormalizedIssue, error) {
	if !report.IsSupported {
		// Don't normalize unsupported tools
		return nil, nil
	}

	// Check for spectre/v1 envelope first
	if v1, ok := report.RawData.(*models.SpectreV1Report); ok {
		return n.NormalizeSpectreV1(report, v1)
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
	case models.ToolPg:
		return n.NormalizePg(report)
	case models.ToolMongo:
		return n.NormalizeMongo(report)
	default:
		return nil, fmt.Errorf("unknown tool type: %s", report.Tool)
	}
}

// NormalizeSpectreV1 converts a spectre/v1 envelope into normalized issues.
// The findings already contain id, severity, location, and message — the mapping is direct.
func (n *Normalizer) NormalizeSpectreV1(report *models.ToolReport, v1 *models.SpectreV1Report) ([]models.NormalizedIssue, error) {
	var issues []models.NormalizedIssue

	for _, f := range v1.Findings {
		category := mapSpectreV1IDToCategory(f.ID)

		issue := models.NormalizedIssue{
			Tool:      report.Tool,
			Category:  category,
			Severity:  mapSpectreSeverity(f.Severity),
			Resource:  f.Location,
			Evidence:  f.Message,
			Count:     1,
			FirstSeen: report.Timestamp,
			LastSeen:  report.Timestamp,
		}
		issues = append(issues, issue)
	}

	return issues, nil
}

// mapSpectreV1IDToCategory maps spectre/v1 finding IDs to normalized categories.
func mapSpectreV1IDToCategory(id string) string {
	switch id {
	// --- missing ---
	case "MISSING_BUCKET", "MISSING_TABLE", "MISSING_COLUMN", "MISSING_COLLECTION", "MISSING_SECRET",
		// kubespectre
		"MISSING_NETWORK_POLICY", "MISSING_AUDIT_POLICY":
		return models.StatusMissing

	// --- unused ---
	case "UNUSED_BUCKET", "UNUSED_TABLE", "UNUSED_INDEX", "UNUSED_TOPIC", "UNUSED_COLLECTION",
		"UNREFERENCED_TABLE", "ORPHANED_INDEX",
		// kubespectre
		"UNUSED_SECRET_MOUNT",
		// redisspectre
		"CONNECTION_WASTE",
		// ecrspectre
		"UNUSED_REPO",
		// rdsspectre
		"IDLE_INSTANCE", "UNUSED_READ_REPLICA",
		// awsspectre
		"IDLE_EC2", "STOPPED_EC2", "IDLE_ALB", "IDLE_NLB", "IDLE_NAT_GATEWAY",
		"IDLE_RDS", "IDLE_LAMBDA", "DETACHED_EBS", "UNUSED_EIP", "UNUSED_SECURITY_GROUP",
		// iamspectre
		"UNUSED_ROLE", "UNATTACHED_POLICY":
		return models.StatusUnused

	// --- stale ---
	case "STALE_PREFIX", "BLOATED_INDEX", "MISSING_VACUUM", "OVERSIZED_COLLECTION", "STALE_SECRET",
		"INACTIVE_USER", "INACTIVE_PRIVILEGED_USER",
		// redisspectre
		"IDLE_KEY", "BIG_KEY", "SLOW_COMMAND",
		// ecrspectre
		"STALE_IMAGE", "LARGE_IMAGE", "MULTI_ARCH_BLOAT",
		// rdsspectre
		"STALE_SNAPSHOT", "OLD_ENGINE_VERSION",
		// awsspectre
		"LOW_TRAFFIC_NAT_GATEWAY",
		// iamspectre
		"STALE_USER", "STALE_ACCESS_KEY", "STALE_SA", "STALE_SA_KEY":
		return models.StatusStale

	// --- misconfig ---
	case "VERSION_SPRAWL", "LIFECYCLE_MISCONFIG", "NO_PRIMARY_KEY", "DUPLICATE_INDEX",
		"UNINDEXED_QUERY", "MISSING_INDEX", "MISSING_TTL", "SUGGEST_INDEX",
		"ADMIN_IN_DATA_DB", "DUPLICATE_USER", "OVERPRIVILEGED_USER", "MULTIPLE_ADMIN_USERS",
		"FAILED_AUTH_ONLY",
		// kubespectre
		"WILDCARD_RBAC", "CLUSTER_ADMIN_BINDING", "PRIVILEGED_CONTAINER",
		"HOST_NETWORK", "HOST_PID", "UNENCRYPTED_SECRETS",
		"DEFAULT_SERVICE_ACCOUNT", "AUTOMOUNT_TOKEN",
		"NO_IMAGE_DIGEST", "UNTRUSTED_REGISTRY",
		// redisspectre
		"HIGH_FRAGMENTATION", "EVICTION_RISK", "NO_PERSISTENCE",
		// ecrspectre
		"UNTAGGED_IMAGE", "NO_LIFECYCLE_POLICY", "VULNERABLE_IMAGE",
		// rdsspectre
		"OVERSIZED_INSTANCE", "UNENCRYPTED_STORAGE", "PUBLIC_ACCESS",
		"NO_AUTOMATED_BACKUPS", "NO_MULTI_AZ", "NO_DELETION_PROTECTION",
		// iamspectre
		"NO_MFA", "WILDCARD_POLICY", "OVERPRIVILEGED_SA", "CROSS_ACCOUNT_TRUST":
		return models.StatusMisconfig

	// --- access_denied ---
	case "RISKY", "ACCESS_DENIED":
		return models.StatusAccessDeny

	// --- drift ---
	case "DYNAMIC_COLLECTION",
		// rdsspectre
		"PARAMETER_GROUP_DRIFT":
		return models.StatusDrift

	default:
		return models.StatusError
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

// NormalizePg converts PgSpectre report to normalized issues
func (n *Normalizer) NormalizePg(report *models.ToolReport) ([]models.NormalizedIssue, error) {
	pgReport, ok := report.RawData.(*models.PgReport)
	if !ok {
		return nil, fmt.Errorf("invalid Pg report data")
	}

	var issues []models.NormalizedIssue

	for _, finding := range pgReport.Findings {
		category := mapPgFindingCategory(finding.Type)
		if category == "" {
			continue
		}

		issue := models.NormalizedIssue{
			Tool:      report.Tool,
			Category:  category,
			Severity:  mapSpectreSeverity(finding.Severity),
			Resource:  buildPgResource(finding),
			Evidence:  finding.Message,
			Count:     1,
			FirstSeen: report.Timestamp,
			LastSeen:  report.Timestamp,
		}

		issues = append(issues, issue)
	}

	return issues, nil
}

// NormalizeMongo converts MongoSpectre report to normalized issues
func (n *Normalizer) NormalizeMongo(report *models.ToolReport) ([]models.NormalizedIssue, error) {
	mongoReport, ok := report.RawData.(*models.MongoReport)
	if !ok {
		return nil, fmt.Errorf("invalid Mongo report data")
	}

	var issues []models.NormalizedIssue

	for _, finding := range mongoReport.Findings {
		category := mapMongoFindingCategory(finding.Type)
		if category == "" {
			continue
		}

		issue := models.NormalizedIssue{
			Tool:      report.Tool,
			Category:  category,
			Severity:  mapSpectreSeverity(finding.Severity),
			Resource:  buildMongoResource(finding),
			Evidence:  finding.Message,
			Count:     1,
			FirstSeen: report.Timestamp,
			LastSeen:  report.Timestamp,
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

func mapPgFindingCategory(findingType string) string {
	switch findingType {
	case "UNUSED_TABLE", "UNUSED_INDEX", "UNREFERENCED_TABLE":
		return models.StatusUnused
	case "MISSING_TABLE", "MISSING_COLUMN":
		return models.StatusMissing
	case "BLOATED_INDEX", "MISSING_VACUUM":
		return models.StatusStale
	case "NO_PRIMARY_KEY", "DUPLICATE_INDEX", "UNINDEXED_QUERY":
		return models.StatusMisconfig
	case "CODE_MATCH", "OK":
		return ""
	default:
		return models.StatusError
	}
}

func mapMongoFindingCategory(findingType string) string {
	switch findingType {
	case "UNUSED_COLLECTION", "UNUSED_INDEX", "ORPHANED_INDEX":
		return models.StatusUnused
	case "MISSING_COLLECTION":
		return models.StatusMissing
	case "MISSING_INDEX", "DUPLICATE_INDEX", "MISSING_TTL", "UNINDEXED_QUERY", "SUGGEST_INDEX":
		return models.StatusMisconfig
	case "OVERSIZED_COLLECTION":
		return models.StatusStale
	case "DYNAMIC_COLLECTION":
		return models.StatusDrift
	case "ADMIN_IN_DATA_DB", "DUPLICATE_USER", "OVERPRIVILEGED_USER", "MULTIPLE_ADMIN_USERS",
		"FAILED_AUTH_ONLY":
		return models.StatusMisconfig
	case "INACTIVE_USER", "INACTIVE_PRIVILEGED_USER":
		return models.StatusStale
	case "OK":
		return ""
	default:
		return models.StatusError
	}
}

func mapSpectreSeverity(severity string) string {
	switch severity {
	case "high":
		return models.SeverityHigh
	case "medium":
		return models.SeverityMedium
	case "low", "info":
		return models.SeverityLow
	default:
		return models.SeverityMedium
	}
}

func buildPgResource(finding models.PgFinding) string {
	base := finding.Table
	if finding.Schema != "" {
		base = finding.Schema + "." + finding.Table
	}
	if finding.Column != "" {
		return base + "." + finding.Column
	}
	if finding.Index != "" {
		return base + ".index:" + finding.Index
	}
	return base
}

func buildMongoResource(finding models.MongoFinding) string {
	base := finding.Database
	if base == "" {
		base = "unknown"
	}
	if finding.Collection != "" {
		base = base + "." + finding.Collection
	}
	if finding.Index != "" {
		base = base + "." + finding.Index
	}
	return base
}
