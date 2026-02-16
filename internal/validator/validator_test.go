package validator

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	return data
}

func containsError(errors []string, substr string) bool {
	for _, err := range errors {
		if strings.Contains(err, substr) {
			return true
		}
	}
	return false
}

func TestValidatorValidateReport(t *testing.T) {
	validator := New()
	now := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		toolType models.ToolType
		data     []byte
		wantErr  bool
	}{
		{
			name:     "vault",
			toolType: models.ToolVault,
			data: mustJSON(t, models.VaultReport{
				Tool:      string(models.ToolVault),
				Timestamp: now,
				Summary:   models.VaultSummary{TotalReferences: 1},
				Secrets: map[string]*models.SecretInfo{
					"secret/ok": {Status: "ok", References: []models.VaultReference{}},
				},
			}),
		},
		{
			name:     "s3",
			toolType: models.ToolS3,
			data: mustJSON(t, models.S3Report{
				Tool:      string(models.ToolS3),
				Timestamp: now,
				Summary:   models.S3Summary{TotalBuckets: 1},
				Buckets: map[string]*models.BucketAnalysis{
					"bucket-ok": {Status: "OK"},
				},
			}),
		},
		{
			name:     "kafka",
			toolType: models.ToolKafka,
			data: mustJSON(t, models.KafkaReport{
				Summary: &models.KafkaSummary{
					TotalTopics:  1,
					TotalBrokers: 1,
				},
				ClusterMetadata: &models.ClusterMetadata{},
				UnusedTopics: []*models.UnusedTopic{
					{Name: "t1", Partitions: 1, Risk: "low"},
				},
			}),
		},
		{
			name:     "clickhouse",
			toolType: models.ToolClickHouse,
			data: mustJSON(t, models.ClickHouseReport{
				Metadata: models.ClickMetadata{
					GeneratedAt:  now,
					LookbackDays: 1,
				},
				Tables: []models.ClickTable{
					{FullName: "db.t1", Category: "active"},
				},
			}),
		},
		{
			name:     "unknown tool",
			toolType: models.ToolUnknown,
			data:     []byte("{"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateReport(tt.toolType, tt.data)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidatorValidateVaultReport(t *testing.T) {
	validator := New()
	now := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		report         models.VaultReport
		raw            []byte
		wantErrContain string
	}{
		{
			name: "valid",
			report: models.VaultReport{
				Tool:      string(models.ToolVault),
				Timestamp: now,
				Summary:   models.VaultSummary{TotalReferences: 1},
				Secrets: map[string]*models.SecretInfo{
					"secret/ok": {Status: "ok", References: []models.VaultReference{}},
				},
			},
		},
		{
			name: "missing tool",
			report: models.VaultReport{
				Timestamp: now,
				Summary:   models.VaultSummary{TotalReferences: 1},
				Secrets: map[string]*models.SecretInfo{
					"secret/ok": {Status: "ok", References: []models.VaultReference{}},
				},
			},
			wantErrContain: "Missing required field: 'tool'",
		},
		{
			name: "missing timestamp",
			report: models.VaultReport{
				Tool:    string(models.ToolVault),
				Summary: models.VaultSummary{TotalReferences: 1},
				Secrets: map[string]*models.SecretInfo{
					"secret/ok": {Status: "ok", References: []models.VaultReference{}},
				},
			},
			wantErrContain: "Missing or invalid field: 'timestamp'",
		},
		{
			name: "negative references",
			report: models.VaultReport{
				Tool:      string(models.ToolVault),
				Timestamp: now,
				Summary:   models.VaultSummary{TotalReferences: -1},
				Secrets: map[string]*models.SecretInfo{
					"secret/ok": {Status: "ok", References: []models.VaultReference{}},
				},
			},
			wantErrContain: "summary.total_references",
		},
		{
			name: "invalid status",
			report: models.VaultReport{
				Tool:      string(models.ToolVault),
				Timestamp: now,
				Summary:   models.VaultSummary{TotalReferences: 1},
				Secrets: map[string]*models.SecretInfo{
					"secret/bad": {Status: "stale", References: []models.VaultReference{}},
				},
			},
			wantErrContain: "invalid status",
		},
		{
			name:           "invalid json",
			raw:            []byte("{"),
			wantErrContain: "Failed to parse JSON",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			data := tt.raw
			if data == nil {
				data = mustJSON(t, tt.report)
			}

			err := validator.ValidateVaultReport(data)
			if tt.wantErrContain == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			var vErr *ValidationError
			if !errors.As(err, &vErr) {
				t.Fatalf("expected ValidationError, got %T", err)
			}
			if !containsError(vErr.Errors, tt.wantErrContain) {
				t.Fatalf("expected error to contain %q, got %v", tt.wantErrContain, vErr.Errors)
			}
		})
	}
}

func TestValidatorValidateS3Report(t *testing.T) {
	validator := New()
	now := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		report         models.S3Report
		raw            []byte
		wantErrContain string
	}{
		{
			name: "valid",
			report: models.S3Report{
				Tool:      string(models.ToolS3),
				Timestamp: now,
				Summary:   models.S3Summary{TotalBuckets: 1},
				Buckets: map[string]*models.BucketAnalysis{
					"bucket-ok": {Status: "OK"},
				},
			},
		},
		{
			name: "missing tool",
			report: models.S3Report{
				Timestamp: now,
				Summary:   models.S3Summary{TotalBuckets: 1},
				Buckets: map[string]*models.BucketAnalysis{
					"bucket-ok": {Status: "OK"},
				},
			},
			wantErrContain: "Missing required field: 'tool'",
		},
		{
			name: "missing timestamp",
			report: models.S3Report{
				Tool:    string(models.ToolS3),
				Summary: models.S3Summary{TotalBuckets: 1},
				Buckets: map[string]*models.BucketAnalysis{
					"bucket-ok": {Status: "OK"},
				},
			},
			wantErrContain: "Missing or invalid field: 'timestamp'",
		},
		{
			name: "negative total buckets",
			report: models.S3Report{
				Tool:      string(models.ToolS3),
				Timestamp: now,
				Summary:   models.S3Summary{TotalBuckets: -1},
				Buckets: map[string]*models.BucketAnalysis{
					"bucket-ok": {Status: "OK"},
				},
			},
			wantErrContain: "summary.total_buckets",
		},
		{
			name: "invalid bucket status",
			report: models.S3Report{
				Tool:      string(models.ToolS3),
				Timestamp: now,
				Summary:   models.S3Summary{TotalBuckets: 1},
				Buckets: map[string]*models.BucketAnalysis{
					"bucket-bad": {Status: "BROKEN"},
				},
			},
			wantErrContain: "invalid status",
		},
		{
			name:           "invalid json",
			raw:            []byte("{"),
			wantErrContain: "Failed to parse JSON",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			data := tt.raw
			if data == nil {
				data = mustJSON(t, tt.report)
			}

			err := validator.ValidateS3Report(data)
			if tt.wantErrContain == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			var vErr *ValidationError
			if !errors.As(err, &vErr) {
				t.Fatalf("expected ValidationError, got %T", err)
			}
			if !containsError(vErr.Errors, tt.wantErrContain) {
				t.Fatalf("expected error to contain %q, got %v", tt.wantErrContain, vErr.Errors)
			}
		})
	}
}

func TestValidatorValidateKafkaReport(t *testing.T) {
	validator := New()

	tests := []struct {
		name           string
		report         models.KafkaReport
		raw            []byte
		wantErrContain string
	}{
		{
			name: "valid",
			report: models.KafkaReport{
				Summary: &models.KafkaSummary{
					TotalTopics:  1,
					TotalBrokers: 1,
				},
				ClusterMetadata: &models.ClusterMetadata{},
				UnusedTopics: []*models.UnusedTopic{
					{Name: "t1", Partitions: 1, Risk: "low"},
				},
			},
		},
		{
			name: "missing summary",
			report: models.KafkaReport{
				Summary:         nil,
				ClusterMetadata: &models.ClusterMetadata{},
			},
			wantErrContain: "Missing required field: 'summary'",
		},
		{
			name: "negative totals",
			report: models.KafkaReport{
				Summary: &models.KafkaSummary{
					TotalTopics:  -1,
					TotalBrokers: -1,
				},
				ClusterMetadata: &models.ClusterMetadata{},
			},
			wantErrContain: "total_topics_analyzed",
		},
		{
			name: "missing cluster metadata",
			report: models.KafkaReport{
				Summary: &models.KafkaSummary{
					TotalTopics:  1,
					TotalBrokers: 1,
				},
				ClusterMetadata: nil,
			},
			wantErrContain: "Missing required field: 'cluster_metadata'",
		},
		{
			name: "invalid risk",
			report: models.KafkaReport{
				Summary: &models.KafkaSummary{
					TotalTopics:  1,
					TotalBrokers: 1,
				},
				ClusterMetadata: &models.ClusterMetadata{},
				UnusedTopics: []*models.UnusedTopic{
					{Name: "t1", Partitions: 1, Risk: "unknown"},
				},
			},
			wantErrContain: "invalid risk",
		},
		{
			name: "invalid partitions",
			report: models.KafkaReport{
				Summary: &models.KafkaSummary{
					TotalTopics:  1,
					TotalBrokers: 1,
				},
				ClusterMetadata: &models.ClusterMetadata{},
				UnusedTopics: []*models.UnusedTopic{
					{Name: "t1", Partitions: -1, Risk: "low"},
				},
			},
			wantErrContain: "invalid partition count",
		},
		{
			name:           "invalid json",
			raw:            []byte("{"),
			wantErrContain: "Failed to parse JSON",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			data := tt.raw
			if data == nil {
				data = mustJSON(t, tt.report)
			}

			err := validator.ValidateKafkaReport(data)
			if tt.wantErrContain == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			var vErr *ValidationError
			if !errors.As(err, &vErr) {
				t.Fatalf("expected ValidationError, got %T", err)
			}
			if !containsError(vErr.Errors, tt.wantErrContain) {
				t.Fatalf("expected error to contain %q, got %v", tt.wantErrContain, vErr.Errors)
			}
		})
	}
}

func TestValidatorValidateClickHouseReport(t *testing.T) {
	validator := New()
	now := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		report         models.ClickHouseReport
		raw            []byte
		wantErrContain string
	}{
		{
			name: "valid",
			report: models.ClickHouseReport{
				Metadata: models.ClickMetadata{
					GeneratedAt:  now,
					LookbackDays: 1,
				},
				Tables: []models.ClickTable{
					{FullName: "db.t1", Category: "active"},
				},
			},
		},
		{
			name: "missing generated_at",
			report: models.ClickHouseReport{
				Metadata: models.ClickMetadata{
					LookbackDays: 1,
				},
			},
			wantErrContain: "metadata.generated_at",
		},
		{
			name: "negative lookback",
			report: models.ClickHouseReport{
				Metadata: models.ClickMetadata{
					GeneratedAt:  now,
					LookbackDays: -1,
				},
			},
			wantErrContain: "metadata.lookback_days",
		},
		{
			name: "invalid category",
			report: models.ClickHouseReport{
				Metadata: models.ClickMetadata{
					GeneratedAt:  now,
					LookbackDays: 1,
				},
				Tables: []models.ClickTable{
					{FullName: "db.t1", Category: "unknown"},
				},
			},
			wantErrContain: "invalid category",
		},
		{
			name:           "invalid json",
			raw:            []byte("{"),
			wantErrContain: "Failed to parse JSON",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			data := tt.raw
			if data == nil {
				data = mustJSON(t, tt.report)
			}

			err := validator.ValidateClickHouseReport(data)
			if tt.wantErrContain == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			var vErr *ValidationError
			if !errors.As(err, &vErr) {
				t.Fatalf("expected ValidationError, got %T", err)
			}
			if !containsError(vErr.Errors, tt.wantErrContain) {
				t.Fatalf("expected error to contain %q, got %v", tt.wantErrContain, vErr.Errors)
			}
		})
	}
}

func TestValidateTimestamp(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		ts      time.Time
		wantErr bool
	}{
		{"valid", now.Add(-24 * time.Hour), false},
		{"future", now.Add(2 * time.Hour), true},
		{"too old", now.AddDate(-2, 0, 0), true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimestamp(tt.ts)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
