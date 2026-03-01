package aggregator

import (
	"testing"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

func issuesByResource(issues []models.NormalizedIssue) map[string]models.NormalizedIssue {
	byResource := make(map[string]models.NormalizedIssue, len(issues))
	for _, issue := range issues {
		byResource[issue.Resource] = issue
	}
	return byResource
}

func TestNormalizerNormalize(t *testing.T) {
	normalizer := NewNormalizer()
	ts := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		report  models.ToolReport
		wantErr bool
		wantNil bool
	}{
		{
			name:    "unsupported tool",
			report:  models.ToolReport{Tool: string(models.ToolVault), Timestamp: ts, IsSupported: false},
			wantErr: false,
			wantNil: true,
		},
		{
			name:    "unknown tool",
			report:  models.ToolReport{Tool: "unknownspectre", Timestamp: ts, IsSupported: true},
			wantErr: true,
		},
		{
			name:    "wrong raw data type",
			report:  models.ToolReport{Tool: string(models.ToolVault), Timestamp: ts, IsSupported: true, RawData: "not-a-report"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			issues, err := normalizer.Normalize(&tt.report)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil && issues != nil {
				t.Fatalf("expected nil issues, got %v", issues)
			}
		})
	}
}

func TestNormalizeVault(t *testing.T) {
	normalizer := NewNormalizer()
	ts := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)

	vaultReport := &models.VaultReport{
		Secrets: map[string]*models.SecretInfo{
			"secret/missing": {
				Status:     "missing",
				References: []models.VaultReference{{File: "a", Line: 1}},
			},
			"secret/access": {
				Status:     "access_denied",
				References: []models.VaultReference{{File: "b", Line: 2}},
			},
			"secret/invalid": {
				Status:     "invalid",
				References: []models.VaultReference{{File: "c", Line: 3}},
			},
			"secret/error": {
				Status:     "error",
				ErrorMsg:   "boom",
				References: []models.VaultReference{{File: "d", Line: 4}},
			},
			"secret/stale": {
				Status:       "stale",
				IsStale:      true,
				LastAccessed: "2026-02-01",
				References:   []models.VaultReference{{File: "e", Line: 5}},
			},
			"secret/unknown": {
				Status:     "unknown",
				References: []models.VaultReference{{File: "f", Line: 6}, {File: "g", Line: 7}, {File: "h", Line: 8}},
			},
			"secret/ok": {
				Status:     "ok",
				References: []models.VaultReference{{File: "i", Line: 9}},
			},
		},
	}

	report := models.ToolReport{
		Tool:        string(models.ToolVault),
		Timestamp:   ts,
		IsSupported: true,
		RawData:     vaultReport,
	}

	issues, err := normalizer.Normalize(&report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(issues) != 6 {
		t.Fatalf("expected 6 issues, got %d", len(issues))
	}

	issueMap := issuesByResource(issues)

	tests := []struct {
		resource string
		category string
		severity string
		evidence string
		count    int
	}{
		{"secret/missing", models.StatusMissing, models.SeverityCritical, "status: missing", 1},
		{"secret/access", models.StatusAccessDeny, models.SeverityHigh, "status: access_denied", 1},
		{"secret/invalid", models.StatusInvalid, models.SeverityHigh, "status: invalid", 1},
		{"secret/error", models.StatusError, models.SeverityCritical, "boom", 1},
		{"secret/stale", models.StatusStale, models.SeverityLow, "stale (last accessed: 2026-02-01)", 1},
		{"secret/unknown", models.StatusError, models.SeverityCritical, "status: unknown", 3},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.resource, func(t *testing.T) {
			issue, ok := issueMap[tt.resource]
			if !ok {
				t.Fatalf("missing issue for resource %s", tt.resource)
			}
			if issue.Tool != report.Tool {
				t.Fatalf("expected tool %s, got %s", report.Tool, issue.Tool)
			}
			if issue.Category != tt.category {
				t.Fatalf("expected category %s, got %s", tt.category, issue.Category)
			}
			if issue.Severity != tt.severity {
				t.Fatalf("expected severity %s, got %s", tt.severity, issue.Severity)
			}
			if issue.Evidence != tt.evidence {
				t.Fatalf("expected evidence %q, got %q", tt.evidence, issue.Evidence)
			}
			if issue.Count != tt.count {
				t.Fatalf("expected count %d, got %d", tt.count, issue.Count)
			}
			if !issue.FirstSeen.Equal(ts) || !issue.LastSeen.Equal(ts) {
				t.Fatalf("expected timestamps to match report timestamp")
			}
		})
	}
}

func TestNormalizeS3(t *testing.T) {
	normalizer := NewNormalizer()
	ts := time.Date(2026, 2, 15, 11, 0, 0, 0, time.UTC)

	s3Report := &models.S3Report{
		Buckets: map[string]*models.BucketAnalysis{
			"missing-bucket": {
				Status:  "MISSING_BUCKET",
				Message: "missing",
			},
			"prefix-bucket": {
				Status: "UNUSED_BUCKET",
				Prefixes: []models.PrefixAnalysis{
					{Prefix: "stale", Status: "STALE_PREFIX", Message: "stale", ObjectCount: 5},
					{Prefix: "bad", Status: "UNKNOWN", Message: "bad", ObjectCount: 2},
					{Prefix: "ok", Status: "OK", ObjectCount: 1},
				},
			},
			"ok-bucket": {
				Status: "OK",
			},
		},
	}

	report := models.ToolReport{
		Tool:        string(models.ToolS3),
		Timestamp:   ts,
		IsSupported: true,
		RawData:     s3Report,
	}

	issues, err := normalizer.Normalize(&report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(issues))
	}

	issueMap := issuesByResource(issues)

	tests := []struct {
		resource string
		category string
		severity string
		evidence string
		count    int
	}{
		{"s3://missing-bucket", models.StatusMissing, models.SeverityCritical, "missing", 1},
		{"s3://prefix-bucket/stale", models.StatusStale, models.SeverityLow, "stale", 5},
		{"s3://prefix-bucket/bad", models.StatusError, models.SeverityCritical, "bad", 2},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.resource, func(t *testing.T) {
			issue, ok := issueMap[tt.resource]
			if !ok {
				t.Fatalf("missing issue for resource %s", tt.resource)
			}
			if issue.Category != tt.category {
				t.Fatalf("expected category %s, got %s", tt.category, issue.Category)
			}
			if issue.Severity != tt.severity {
				t.Fatalf("expected severity %s, got %s", tt.severity, issue.Severity)
			}
			if issue.Evidence != tt.evidence {
				t.Fatalf("expected evidence %q, got %q", tt.evidence, issue.Evidence)
			}
			if issue.Count != tt.count {
				t.Fatalf("expected count %d, got %d", tt.count, issue.Count)
			}
			if !issue.FirstSeen.Equal(ts) || !issue.LastSeen.Equal(ts) {
				t.Fatalf("expected timestamps to match report timestamp")
			}
		})
	}
}

func TestNormalizeKafka(t *testing.T) {
	normalizer := NewNormalizer()
	ts := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)

	kafkaReport := &models.KafkaReport{
		UnusedTopics: []*models.UnusedTopic{
			{Name: "t-high", Partitions: 3, Risk: "high", Reason: "no consumers"},
			{Name: "t-med", Partitions: 2, Risk: "medium", Reason: "no consumers"},
			{Name: "t-low", Partitions: 1, Risk: "low", Reason: "no consumers"},
			{Name: "t-unk", Partitions: 4, Risk: "unknown", Reason: "no consumers"},
		},
	}

	report := models.ToolReport{
		Tool:        string(models.ToolKafka),
		Timestamp:   ts,
		IsSupported: true,
		RawData:     kafkaReport,
	}

	issues, err := normalizer.Normalize(&report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(issues) != 4 {
		t.Fatalf("expected 4 issues, got %d", len(issues))
	}

	issueMap := issuesByResource(issues)

	tests := []struct {
		resource string
		severity string
		count    int
		evidence string
	}{
		{"topic:t-high", models.SeverityHigh, 3, "no consumers (partitions: 3, risk: high)"},
		{"topic:t-med", models.SeverityMedium, 2, "no consumers (partitions: 2, risk: medium)"},
		{"topic:t-low", models.SeverityLow, 1, "no consumers (partitions: 1, risk: low)"},
		{"topic:t-unk", models.SeverityMedium, 4, "no consumers (partitions: 4, risk: unknown)"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.resource, func(t *testing.T) {
			issue, ok := issueMap[tt.resource]
			if !ok {
				t.Fatalf("missing issue for resource %s", tt.resource)
			}
			if issue.Category != models.StatusUnused {
				t.Fatalf("expected category %s, got %s", models.StatusUnused, issue.Category)
			}
			if issue.Severity != tt.severity {
				t.Fatalf("expected severity %s, got %s", tt.severity, issue.Severity)
			}
			if issue.Count != tt.count {
				t.Fatalf("expected count %d, got %d", tt.count, issue.Count)
			}
			if issue.Evidence != tt.evidence {
				t.Fatalf("expected evidence %q, got %q", tt.evidence, issue.Evidence)
			}
			if !issue.FirstSeen.Equal(ts) || !issue.LastSeen.Equal(ts) {
				t.Fatalf("expected timestamps to match report timestamp")
			}
		})
	}
}

func TestNormalizePg(t *testing.T) {
	normalizer := NewNormalizer()
	ts := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)

	pgReport := &models.PgReport{
		Findings: []models.PgFinding{
			{Type: "UNUSED_TABLE", Severity: "medium", Schema: "public", Table: "old_table", Message: "unused"},
			{Type: "UNUSED_INDEX", Severity: "low", Schema: "public", Table: "users", Index: "idx_old", Message: "unused index"},
			{Type: "MISSING_TABLE", Severity: "high", Schema: "app", Table: "missing_t", Message: "missing"},
			{Type: "MISSING_COLUMN", Severity: "high", Schema: "app", Table: "users", Column: "email", Message: "missing column"},
			{Type: "BLOATED_INDEX", Severity: "medium", Schema: "public", Table: "orders", Index: "idx_bloated", Message: "bloated"},
			{Type: "MISSING_VACUUM", Severity: "medium", Table: "big_table", Message: "needs vacuum"},
			{Type: "NO_PRIMARY_KEY", Severity: "high", Schema: "public", Table: "legacy", Message: "no pk"},
			{Type: "DUPLICATE_INDEX", Severity: "low", Schema: "public", Table: "users", Index: "idx_dup", Message: "dup"},
			{Type: "UNINDEXED_QUERY", Severity: "medium", Schema: "public", Table: "logs", Message: "slow"},
			{Type: "UNREFERENCED_TABLE", Severity: "low", Schema: "public", Table: "orphan", Message: "orphan"},
			{Type: "CODE_MATCH", Severity: "info", Schema: "public", Table: "t1", Message: "ok"},
			{Type: "OK", Severity: "info", Schema: "public", Table: "t2", Message: "ok"},
			{Type: "UNKNOWN_TYPE", Severity: "medium", Schema: "public", Table: "t3", Message: "unknown"},
		},
	}

	report := models.ToolReport{
		Tool:        string(models.ToolPg),
		Timestamp:   ts,
		IsSupported: true,
		RawData:     pgReport,
	}

	issues, err := normalizer.Normalize(&report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// CODE_MATCH and OK types should be skipped
	if len(issues) != 11 {
		t.Fatalf("expected 11 issues, got %d", len(issues))
	}

	// Check that all issues have correct tool name
	for _, issue := range issues {
		if issue.Tool != string(models.ToolPg) {
			t.Fatalf("expected tool %s, got %s", models.ToolPg, issue.Tool)
		}
	}
}

func TestNormalizePgWrongDataType(t *testing.T) {
	normalizer := NewNormalizer()
	report := models.ToolReport{
		Tool:        string(models.ToolPg),
		IsSupported: true,
		RawData:     "not-a-pg-report",
	}

	_, err := normalizer.Normalize(&report)
	if err == nil {
		t.Fatal("expected error for wrong data type")
	}
}

func TestNormalizeMongo(t *testing.T) {
	normalizer := NewNormalizer()
	ts := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)

	mongoReport := &models.MongoReport{
		Findings: []models.MongoFinding{
			{Type: "UNUSED_COLLECTION", Severity: "medium", Database: "mydb", Collection: "old_col", Message: "unused"},
			{Type: "UNUSED_INDEX", Severity: "low", Database: "mydb", Collection: "users", Index: "idx_old", Message: "unused index"},
			{Type: "ORPHANED_INDEX", Severity: "low", Database: "mydb", Collection: "users", Index: "idx_orphan", Message: "orphaned"},
			{Type: "MISSING_COLLECTION", Severity: "high", Database: "mydb", Collection: "missing_col", Message: "missing"},
			{Type: "MISSING_INDEX", Severity: "medium", Database: "mydb", Collection: "orders", Message: "needs index"},
			{Type: "DUPLICATE_INDEX", Severity: "low", Database: "mydb", Collection: "users", Index: "idx_dup", Message: "dup"},
			{Type: "MISSING_TTL", Severity: "medium", Database: "mydb", Collection: "sessions", Message: "no ttl"},
			{Type: "UNINDEXED_QUERY", Severity: "medium", Database: "mydb", Collection: "logs", Message: "slow"},
			{Type: "SUGGEST_INDEX", Severity: "low", Database: "mydb", Collection: "events", Message: "suggestion"},
			{Type: "OVERSIZED_COLLECTION", Severity: "high", Database: "mydb", Collection: "huge", Message: "oversized"},
			{Type: "DYNAMIC_COLLECTION", Severity: "medium", Database: "mydb", Collection: "dynamic", Message: "dynamic"},
			{Type: "ADMIN_IN_DATA_DB", Severity: "high", Database: "mydb", Message: "admin in data db"},
			{Type: "DUPLICATE_USER", Severity: "medium", Database: "mydb", Message: "dup user"},
			{Type: "OVERPRIVILEGED_USER", Severity: "high", Database: "mydb", Message: "overprivileged"},
			{Type: "MULTIPLE_ADMIN_USERS", Severity: "medium", Database: "mydb", Message: "many admins"},
			{Type: "OK", Severity: "info", Database: "mydb", Collection: "ok_col", Message: "ok"},
			{Type: "UNKNOWN_TYPE", Severity: "medium", Database: "mydb", Message: "unknown"},
			{Type: "INACTIVE_USER", Severity: "medium", Database: "admin", Message: `user "stale" has no authentication in the last 7 days`},
			{Type: "INACTIVE_PRIVILEGED_USER", Severity: "high", Database: "admin", Message: `privileged user "old_admin" has no authentication in the last 7 days`},
			{Type: "FAILED_AUTH_ONLY", Severity: "medium", Database: "admin", Message: `user "broken" has only failed authentication attempts in the last 7 days`},
		},
	}

	report := models.ToolReport{
		Tool:        string(models.ToolMongo),
		Timestamp:   ts,
		IsSupported: true,
		RawData:     mongoReport,
	}

	issues, err := normalizer.Normalize(&report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// OK type should be skipped; 16 original + 3 user findings = 19
	if len(issues) != 19 {
		t.Fatalf("expected 19 issues, got %d", len(issues))
	}

	// Verify user finding category mappings
	categoryByEvidence := make(map[string]string)
	for _, issue := range issues {
		categoryByEvidence[issue.Evidence] = issue.Category
	}
	if got := categoryByEvidence[`user "stale" has no authentication in the last 7 days`]; got != models.StatusStale {
		t.Fatalf("INACTIVE_USER category = %s, want %s", got, models.StatusStale)
	}
	if got := categoryByEvidence[`privileged user "old_admin" has no authentication in the last 7 days`]; got != models.StatusStale {
		t.Fatalf("INACTIVE_PRIVILEGED_USER category = %s, want %s", got, models.StatusStale)
	}
	if got := categoryByEvidence[`user "broken" has only failed authentication attempts in the last 7 days`]; got != models.StatusMisconfig {
		t.Fatalf("FAILED_AUTH_ONLY category = %s, want %s", got, models.StatusMisconfig)
	}
}

func TestNormalizeMongoWrongDataType(t *testing.T) {
	normalizer := NewNormalizer()
	report := models.ToolReport{
		Tool:        string(models.ToolMongo),
		IsSupported: true,
		RawData:     "not-a-mongo-report",
	}

	_, err := normalizer.Normalize(&report)
	if err == nil {
		t.Fatal("expected error for wrong data type")
	}
}

func TestNormalizeMongoEmptyDatabase(t *testing.T) {
	normalizer := NewNormalizer()
	ts := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)

	mongoReport := &models.MongoReport{
		Findings: []models.MongoFinding{
			{Type: "MISSING_COLLECTION", Severity: "high", Collection: "missing_col", Message: "missing"},
		},
	}

	report := models.ToolReport{
		Tool:        string(models.ToolMongo),
		Timestamp:   ts,
		IsSupported: true,
		RawData:     mongoReport,
	}

	issues, err := normalizer.Normalize(&report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	// Should use "unknown" as database prefix
	if issues[0].Resource != "unknown.missing_col" {
		t.Fatalf("expected resource 'unknown.missing_col', got %q", issues[0].Resource)
	}
}

func TestNormalizeSpectreV1(t *testing.T) {
	normalizer := NewNormalizer()
	ts := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)

	v1Report := &models.SpectreV1Report{
		Schema:    "spectre/v1",
		Tool:      "s3spectre",
		Version:   "0.2.1",
		Timestamp: ts,
		Target:    models.SpectreV1Target{Type: "s3"},
		Findings: []models.SpectreV1Finding{
			{ID: "UNUSED_BUCKET", Severity: "medium", Location: "s3://old-bucket", Message: "no recent access"},
			{ID: "MISSING_BUCKET", Severity: "high", Location: "s3://missing-bucket", Message: "bucket not found"},
			{ID: "STALE_PREFIX", Severity: "low", Location: "s3://data/old/", Message: "stale prefix"},
			{ID: "VERSION_SPRAWL", Severity: "medium", Location: "s3://versioned-bucket", Message: "too many versions"},
			{ID: "RISKY", Severity: "high", Location: "s3://risky", Message: "risky access"},
			{ID: "DYNAMIC_COLLECTION", Severity: "info", Location: "db.dynamic", Message: "dynamic"},
			{ID: "UNKNOWN_ID", Severity: "low", Location: "somewhere", Message: "unknown finding"},
			{ID: "INACTIVE_USER", Severity: "medium", Location: "admin.", Message: `user "stale" has no authentication`},
			{ID: "INACTIVE_PRIVILEGED_USER", Severity: "high", Location: "admin.", Message: `privileged user "root" inactive`},
			{ID: "FAILED_AUTH_ONLY", Severity: "medium", Location: "admin.", Message: `user "broken" failed auth only`},
		},
		Summary: models.SpectreV1Summary{Total: 10},
	}

	report := models.ToolReport{
		Tool:        "s3spectre",
		Timestamp:   ts,
		IsSupported: true,
		RawData:     v1Report,
	}

	issues, err := normalizer.Normalize(&report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(issues) != 10 {
		t.Fatalf("expected 10 issues, got %d", len(issues))
	}

	// Verify category mappings
	issueMap := make(map[string]models.NormalizedIssue)
	for _, issue := range issues {
		issueMap[issue.Resource] = issue
	}

	expectedCategories := map[string]string{
		"s3://old-bucket":       models.StatusUnused,
		"s3://missing-bucket":   models.StatusMissing,
		"s3://data/old/":        models.StatusStale,
		"s3://versioned-bucket": models.StatusMisconfig,
		"s3://risky":            models.StatusAccessDeny,
		"db.dynamic":            models.StatusDrift,
		"somewhere":             models.StatusError,
	}

	// User activity findings share "admin." location — verify by message
	for _, issue := range issues {
		switch issue.Evidence {
		case `user "stale" has no authentication`:
			if issue.Category != models.StatusStale {
				t.Fatalf("INACTIVE_USER category = %s, want %s", issue.Category, models.StatusStale)
			}
		case `privileged user "root" inactive`:
			if issue.Category != models.StatusStale {
				t.Fatalf("INACTIVE_PRIVILEGED_USER category = %s, want %s", issue.Category, models.StatusStale)
			}
		case `user "broken" failed auth only`:
			if issue.Category != models.StatusMisconfig {
				t.Fatalf("FAILED_AUTH_ONLY category = %s, want %s", issue.Category, models.StatusMisconfig)
			}
		}
	}

	for resource, expectedCat := range expectedCategories {
		issue, ok := issueMap[resource]
		if !ok {
			t.Fatalf("missing issue for resource %s", resource)
		}
		if issue.Category != expectedCat {
			t.Fatalf("resource %s: expected category %s, got %s", resource, expectedCat, issue.Category)
		}
	}
}

func TestMapSpectreV1IDToCategory_NewTools(t *testing.T) {
	tests := []struct {
		id       string
		expected string
	}{
		// kubespectre
		{"WILDCARD_RBAC", models.StatusMisconfig},
		{"CLUSTER_ADMIN_BINDING", models.StatusMisconfig},
		{"PRIVILEGED_CONTAINER", models.StatusMisconfig},
		{"HOST_NETWORK", models.StatusMisconfig},
		{"HOST_PID", models.StatusMisconfig},
		{"MISSING_NETWORK_POLICY", models.StatusMissing},
		{"UNENCRYPTED_SECRETS", models.StatusMisconfig},
		{"UNUSED_SECRET_MOUNT", models.StatusUnused},
		{"STALE_SECRET", models.StatusStale},
		{"DEFAULT_SERVICE_ACCOUNT", models.StatusMisconfig},
		{"AUTOMOUNT_TOKEN", models.StatusMisconfig},
		{"NO_IMAGE_DIGEST", models.StatusMisconfig},
		{"UNTRUSTED_REGISTRY", models.StatusMisconfig},
		{"MISSING_AUDIT_POLICY", models.StatusMissing},
		// redisspectre
		{"HIGH_FRAGMENTATION", models.StatusMisconfig},
		{"IDLE_KEY", models.StatusStale},
		{"BIG_KEY", models.StatusStale},
		{"CONNECTION_WASTE", models.StatusUnused},
		{"EVICTION_RISK", models.StatusMisconfig},
		{"NO_PERSISTENCE", models.StatusMisconfig},
		{"SLOW_COMMAND", models.StatusStale},
		// ecrspectre
		{"UNTAGGED_IMAGE", models.StatusMisconfig},
		{"STALE_IMAGE", models.StatusStale},
		{"LARGE_IMAGE", models.StatusStale},
		{"NO_LIFECYCLE_POLICY", models.StatusMisconfig},
		{"VULNERABLE_IMAGE", models.StatusMisconfig},
		{"UNUSED_REPO", models.StatusUnused},
		{"MULTI_ARCH_BLOAT", models.StatusStale},
		// rdsspectre
		{"IDLE_INSTANCE", models.StatusUnused},
		{"OVERSIZED_INSTANCE", models.StatusMisconfig},
		{"UNENCRYPTED_STORAGE", models.StatusMisconfig},
		{"PUBLIC_ACCESS", models.StatusMisconfig},
		{"NO_AUTOMATED_BACKUPS", models.StatusMisconfig},
		{"STALE_SNAPSHOT", models.StatusStale},
		{"UNUSED_READ_REPLICA", models.StatusUnused},
		{"NO_MULTI_AZ", models.StatusMisconfig},
		{"OLD_ENGINE_VERSION", models.StatusStale},
		{"NO_DELETION_PROTECTION", models.StatusMisconfig},
		{"PARAMETER_GROUP_DRIFT", models.StatusDrift},
		// awsspectre
		{"IDLE_EC2", models.StatusUnused},
		{"STOPPED_EC2", models.StatusUnused},
		{"DETACHED_EBS", models.StatusUnused},
		{"UNUSED_EIP", models.StatusUnused},
		{"IDLE_ALB", models.StatusUnused},
		{"IDLE_NLB", models.StatusUnused},
		{"IDLE_NAT_GATEWAY", models.StatusUnused},
		{"LOW_TRAFFIC_NAT_GATEWAY", models.StatusStale},
		{"IDLE_RDS", models.StatusUnused},
		{"IDLE_LAMBDA", models.StatusUnused},
		{"UNUSED_SECURITY_GROUP", models.StatusUnused},
		// iamspectre
		{"STALE_USER", models.StatusStale},
		{"STALE_ACCESS_KEY", models.StatusStale},
		{"NO_MFA", models.StatusMisconfig},
		{"UNUSED_ROLE", models.StatusUnused},
		{"UNATTACHED_POLICY", models.StatusUnused},
		{"WILDCARD_POLICY", models.StatusMisconfig},
		{"CROSS_ACCOUNT_TRUST", models.StatusMisconfig},
		{"STALE_SA", models.StatusStale},
		{"STALE_SA_KEY", models.StatusStale},
		{"OVERPRIVILEGED_SA", models.StatusMisconfig},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.id, func(t *testing.T) {
			got := mapSpectreV1IDToCategory(tt.id)
			if got != tt.expected {
				t.Fatalf("mapSpectreV1IDToCategory(%q) = %q, want %q", tt.id, got, tt.expected)
			}
		})
	}
}

func TestMapSpectreSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"high", models.SeverityHigh},
		{"medium", models.SeverityMedium},
		{"low", models.SeverityLow},
		{"info", models.SeverityLow},
		{"unknown", models.SeverityMedium},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			if got := mapSpectreSeverity(tt.input); got != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestBuildPgResource(t *testing.T) {
	tests := []struct {
		name     string
		finding  models.PgFinding
		expected string
	}{
		{"schema and table", models.PgFinding{Schema: "public", Table: "users"}, "public.users"},
		{"table only", models.PgFinding{Table: "users"}, "users"},
		{"with column", models.PgFinding{Schema: "public", Table: "users", Column: "email"}, "public.users.email"},
		{"with index", models.PgFinding{Schema: "public", Table: "users", Index: "idx_email"}, "public.users.index:idx_email"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := buildPgResource(tt.finding); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestBuildMongoResource(t *testing.T) {
	tests := []struct {
		name     string
		finding  models.MongoFinding
		expected string
	}{
		{"db and collection", models.MongoFinding{Database: "mydb", Collection: "users"}, "mydb.users"},
		{"db only", models.MongoFinding{Database: "mydb"}, "mydb"},
		{"empty db", models.MongoFinding{Collection: "users"}, "unknown.users"},
		{"with index", models.MongoFinding{Database: "mydb", Collection: "users", Index: "idx_email"}, "mydb.users.idx_email"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := buildMongoResource(tt.finding); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestMapVaultStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"missing", models.StatusMissing},
		{"access_denied", models.StatusAccessDeny},
		{"invalid", models.StatusInvalid},
		{"error", models.StatusError},
		{"stale", models.StatusStale},
		{"unknown", models.StatusError},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			if got := mapVaultStatus(tt.input); got != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestMapS3Status(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"MISSING_BUCKET", models.StatusMissing},
		{"MISSING_PREFIX", models.StatusMissing},
		{"UNUSED_BUCKET", models.StatusUnused},
		{"STALE_PREFIX", models.StatusStale},
		{"VERSION_SPRAWL", models.StatusMisconfig},
		{"LIFECYCLE_MISCONFIG", models.StatusMisconfig},
		{"UNKNOWN", models.StatusError},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			if got := mapS3Status(tt.input); got != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestNormalizeClickHouse(t *testing.T) {
	normalizer := NewNormalizer()
	firstSeen := time.Date(2026, 2, 10, 8, 0, 0, 0, time.UTC)
	lastSeen := time.Date(2026, 2, 14, 9, 0, 0, 0, time.UTC)
	anomalyTime := time.Date(2026, 2, 15, 9, 30, 0, 0, time.UTC)

	clickReport := &models.ClickHouseReport{
		Tables: []models.ClickTable{
			{
				FullName:     "db.t1",
				ZeroUsage:    true,
				IsReplicated: false,
				Reads:        0,
				Writes:       1,
				FirstSeen:    firstSeen,
				LastAccess:   lastSeen,
			},
			{
				FullName:     "db.t2",
				ZeroUsage:    true,
				IsReplicated: true,
				Reads:        5,
				Writes:       0,
				FirstSeen:    firstSeen,
				LastAccess:   lastSeen,
			},
			{
				FullName:  "db.t3",
				ZeroUsage: false,
			},
		},
		Anomalies: []models.ClickAnomaly{
			{
				Type:          "configuration",
				Description:   "bad config",
				Severity:      models.SeverityHigh,
				AffectedTable: "db.t1",
				DetectedAt:    anomalyTime,
			},
			{
				Type:          "latency",
				Description:   "slow",
				Severity:      models.SeverityMedium,
				AffectedTable: "db.t2",
				DetectedAt:    anomalyTime.Add(1 * time.Hour),
			},
		},
	}

	report := models.ToolReport{
		Tool:        string(models.ToolClickHouse),
		Timestamp:   anomalyTime,
		IsSupported: true,
		RawData:     clickReport,
	}

	issues, err := normalizer.Normalize(&report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(issues) != 4 {
		t.Fatalf("expected 4 issues, got %d", len(issues))
	}

	tests := []struct {
		resource string
		category string
		severity string
		evidence string
		count    int
		first    time.Time
		last     time.Time
	}{
		{"db.t1", models.StatusUnused, models.SeverityLow, "zero usage (reads: 0, writes: 1)", 1, firstSeen, lastSeen},
		{"db.t2", models.StatusUnused, models.SeverityMedium, "zero usage (reads: 5, writes: 0)", 1, firstSeen, lastSeen},
		{"db.t1", models.StatusMisconfig, models.SeverityHigh, "bad config", 1, anomalyTime, anomalyTime},
		{"db.t2", models.StatusDrift, models.SeverityMedium, "slow", 1, anomalyTime.Add(1 * time.Hour), anomalyTime.Add(1 * time.Hour)},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.resource+"-"+tt.category, func(t *testing.T) {
			var issue *models.NormalizedIssue
			for i := range issues {
				if issues[i].Resource == tt.resource && issues[i].Category == tt.category {
					issue = &issues[i]
					break
				}
			}
			if issue == nil {
				t.Fatalf("missing issue for resource %s and category %s", tt.resource, tt.category)
			}
			if issue.Category != tt.category {
				t.Fatalf("expected category %s, got %s", tt.category, issue.Category)
			}
			if issue.Severity != tt.severity {
				t.Fatalf("expected severity %s, got %s", tt.severity, issue.Severity)
			}
			if issue.Evidence != tt.evidence {
				t.Fatalf("expected evidence %q, got %q", tt.evidence, issue.Evidence)
			}
			if issue.Count != tt.count {
				t.Fatalf("expected count %d, got %d", tt.count, issue.Count)
			}
			if !issue.FirstSeen.Equal(tt.first) || !issue.LastSeen.Equal(tt.last) {
				t.Fatalf("expected timestamps to match")
			}
		})
	}
}
