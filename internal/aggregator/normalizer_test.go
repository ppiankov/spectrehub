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
