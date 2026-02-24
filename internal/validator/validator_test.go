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
		{"just now", now, false},
		{"borderline future within 1h", now.Add(30 * time.Minute), false},
		{"exactly 1 year ago", now.AddDate(-1, 0, 0).Add(1 * time.Hour), false},
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

func TestValidationErrorError(t *testing.T) {
	tests := []struct {
		name     string
		err      *ValidationError
		contains []string
	}{
		{
			name: "single error",
			err:  &ValidationError{Tool: "VaultSpectre", Errors: []string{"missing field"}},
			contains: []string{
				"Invalid VaultSpectre report",
				"missing field",
			},
		},
		{
			name: "multiple errors",
			err: &ValidationError{
				Tool:   "S3Spectre",
				Errors: []string{"error one", "error two"},
			},
			contains: []string{
				"Invalid S3Spectre report",
				"error one",
				"error two",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			for _, c := range tt.contains {
				if !strings.Contains(msg, c) {
					t.Fatalf("expected error message to contain %q, got %q", c, msg)
				}
			}
		})
	}
}

func TestIsSpectreV1(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{"valid spectre/v1", []byte(`{"schema":"spectre/v1","tool":"s3spectre"}`), true},
		{"different schema", []byte(`{"schema":"spectre/v2"}`), false},
		{"no schema", []byte(`{"tool":"s3spectre"}`), false},
		{"invalid json", []byte(`{`), false},
		{"empty object", []byte(`{}`), false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := isSpectreV1(tt.data); got != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestValidateReportSpectreV1Dispatch(t *testing.T) {
	validator := New()
	now := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)

	// Valid spectre/v1 report should be dispatched to spectre/v1 validator
	v1Report := models.SpectreV1Report{
		Schema:    "spectre/v1",
		Tool:      "s3spectre",
		Version:   "0.2.1",
		Timestamp: now,
		Target:    models.SpectreV1Target{Type: "s3"},
		Findings: []models.SpectreV1Finding{
			{ID: "UNUSED_BUCKET", Severity: "medium", Location: "s3://test", Message: "unused"},
		},
		Summary: models.SpectreV1Summary{Total: 1, Medium: 1},
	}

	data := mustJSON(t, v1Report)

	// Even if we pass a different tool type, spectre/v1 envelope takes precedence
	err := validator.ValidateReport(models.ToolVault, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSpectreV1Report(t *testing.T) {
	validator := New()
	now := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		report         models.SpectreV1Report
		raw            []byte
		wantErrContain string
	}{
		{
			name: "valid report",
			report: models.SpectreV1Report{
				Schema:    "spectre/v1",
				Tool:      "s3spectre",
				Version:   "0.2.1",
				Timestamp: now,
				Target:    models.SpectreV1Target{Type: "s3"},
				Findings: []models.SpectreV1Finding{
					{ID: "UNUSED_BUCKET", Severity: "medium", Location: "s3://test", Message: "unused"},
				},
				Summary: models.SpectreV1Summary{Total: 1, Medium: 1},
			},
		},
		{
			name:           "invalid json",
			raw:            []byte("{"),
			wantErrContain: "Failed to parse JSON",
		},
		{
			name: "missing tool",
			report: models.SpectreV1Report{
				Schema:    "spectre/v1",
				Version:   "0.2.1",
				Timestamp: now,
				Target:    models.SpectreV1Target{Type: "s3"},
				Findings:  []models.SpectreV1Finding{},
				Summary:   models.SpectreV1Summary{Total: 0},
			},
			wantErrContain: "Missing required field: 'tool'",
		},
		{
			name: "missing version",
			report: models.SpectreV1Report{
				Schema:    "spectre/v1",
				Tool:      "s3spectre",
				Timestamp: now,
				Target:    models.SpectreV1Target{Type: "s3"},
				Findings:  []models.SpectreV1Finding{},
				Summary:   models.SpectreV1Summary{Total: 0},
			},
			wantErrContain: "Missing required field: 'version'",
		},
		{
			name: "missing timestamp",
			report: models.SpectreV1Report{
				Schema:   "spectre/v1",
				Tool:     "s3spectre",
				Version:  "0.2.1",
				Target:   models.SpectreV1Target{Type: "s3"},
				Findings: []models.SpectreV1Finding{},
				Summary:  models.SpectreV1Summary{Total: 0},
			},
			wantErrContain: "Missing or invalid field: 'timestamp'",
		},
		{
			name: "missing target type",
			report: models.SpectreV1Report{
				Schema:    "spectre/v1",
				Tool:      "s3spectre",
				Version:   "0.2.1",
				Timestamp: now,
				Findings:  []models.SpectreV1Finding{},
				Summary:   models.SpectreV1Summary{Total: 0},
			},
			wantErrContain: "Missing required field: 'target.type'",
		},
		{
			name: "wrong target type for tool",
			report: models.SpectreV1Report{
				Schema:    "spectre/v1",
				Tool:      "s3spectre",
				Version:   "0.2.1",
				Timestamp: now,
				Target:    models.SpectreV1Target{Type: "vault"},
				Findings:  []models.SpectreV1Finding{},
				Summary:   models.SpectreV1Summary{Total: 0},
			},
			wantErrContain: "target.type",
		},
		{
			name: "nil findings",
			report: models.SpectreV1Report{
				Schema:    "spectre/v1",
				Tool:      "s3spectre",
				Version:   "0.2.1",
				Timestamp: now,
				Target:    models.SpectreV1Target{Type: "s3"},
				Findings:  nil,
				Summary:   models.SpectreV1Summary{Total: 0},
			},
			wantErrContain: "findings",
		},
		{
			name: "finding missing id",
			report: models.SpectreV1Report{
				Schema:    "spectre/v1",
				Tool:      "s3spectre",
				Version:   "0.2.1",
				Timestamp: now,
				Target:    models.SpectreV1Target{Type: "s3"},
				Findings: []models.SpectreV1Finding{
					{Severity: "medium", Location: "s3://test", Message: "unused"},
				},
				Summary: models.SpectreV1Summary{Total: 1, Medium: 1},
			},
			wantErrContain: "missing required field 'id'",
		},
		{
			name: "finding invalid severity",
			report: models.SpectreV1Report{
				Schema:    "spectre/v1",
				Tool:      "s3spectre",
				Version:   "0.2.1",
				Timestamp: now,
				Target:    models.SpectreV1Target{Type: "s3"},
				Findings: []models.SpectreV1Finding{
					{ID: "TEST", Severity: "critical", Location: "s3://test", Message: "test"},
				},
				Summary: models.SpectreV1Summary{Total: 1},
			},
			wantErrContain: "invalid severity",
		},
		{
			name: "finding missing location",
			report: models.SpectreV1Report{
				Schema:    "spectre/v1",
				Tool:      "s3spectre",
				Version:   "0.2.1",
				Timestamp: now,
				Target:    models.SpectreV1Target{Type: "s3"},
				Findings: []models.SpectreV1Finding{
					{ID: "TEST", Severity: "medium", Message: "test"},
				},
				Summary: models.SpectreV1Summary{Total: 1, Medium: 1},
			},
			wantErrContain: "missing required field 'location'",
		},
		{
			name: "finding missing message",
			report: models.SpectreV1Report{
				Schema:    "spectre/v1",
				Tool:      "s3spectre",
				Version:   "0.2.1",
				Timestamp: now,
				Target:    models.SpectreV1Target{Type: "s3"},
				Findings: []models.SpectreV1Finding{
					{ID: "TEST", Severity: "medium", Location: "s3://test"},
				},
				Summary: models.SpectreV1Summary{Total: 1, Medium: 1},
			},
			wantErrContain: "missing required field 'message'",
		},
		{
			name: "summary total mismatch",
			report: models.SpectreV1Report{
				Schema:    "spectre/v1",
				Tool:      "s3spectre",
				Version:   "0.2.1",
				Timestamp: now,
				Target:    models.SpectreV1Target{Type: "s3"},
				Findings: []models.SpectreV1Finding{
					{ID: "TEST", Severity: "medium", Location: "s3://test", Message: "test"},
				},
				Summary: models.SpectreV1Summary{Total: 5, Medium: 1},
			},
			wantErrContain: "summary.total=5 does not match findings count=1",
		},
		{
			name: "unknown tool target type accepted",
			report: models.SpectreV1Report{
				Schema:    "spectre/v1",
				Tool:      "customtool",
				Version:   "0.1.0",
				Timestamp: now,
				Target:    models.SpectreV1Target{Type: "custom"},
				Findings:  []models.SpectreV1Finding{},
				Summary:   models.SpectreV1Summary{Total: 0},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			data := tt.raw
			if data == nil {
				data = mustJSON(t, tt.report)
			}

			err := validator.ValidateSpectreV1Report(data)
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

func TestValidateReportAllToolTypes(t *testing.T) {
	validator := New()
	now := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)

	// Test that each tool type routes correctly (even with invalid data to hit error paths)
	tests := []struct {
		name     string
		toolType models.ToolType
		data     []byte
		wantErr  bool
	}{
		{
			name:     "vault invalid json",
			toolType: models.ToolVault,
			data:     []byte("{"),
			wantErr:  true,
		},
		{
			name:     "s3 invalid json",
			toolType: models.ToolS3,
			data:     []byte("{"),
			wantErr:  true,
		},
		{
			name:     "kafka invalid json",
			toolType: models.ToolKafka,
			data:     []byte("{"),
			wantErr:  true,
		},
		{
			name:     "clickhouse invalid json",
			toolType: models.ToolClickHouse,
			data:     []byte("{"),
			wantErr:  true,
		},
		{
			name:     "valid vault with all statuses",
			toolType: models.ToolVault,
			data: mustJSON(t, models.VaultReport{
				Tool:      "vaultspectre",
				Timestamp: now,
				Summary:   models.VaultSummary{TotalReferences: 3},
				Secrets: map[string]*models.SecretInfo{
					"secret/ok":      {Status: "ok"},
					"secret/missing": {Status: "missing"},
					"secret/dynamic": {Status: "dynamic"},
					"secret/error":   {Status: "error"},
					"secret/invalid": {Status: "invalid"},
					"secret/access":  {Status: "access_denied"},
				},
			}),
		},
		{
			name:     "kafka with negative brokers",
			toolType: models.ToolKafka,
			data: mustJSON(t, models.KafkaReport{
				Summary: &models.KafkaSummary{
					TotalTopics:  1,
					TotalBrokers: -1,
				},
				ClusterMetadata: &models.ClusterMetadata{},
			}),
			wantErr: true,
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
