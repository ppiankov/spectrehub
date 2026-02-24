package collector

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

func TestParseVaultReport(t *testing.T) {
	now := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	data, _ := json.Marshal(models.VaultReport{
		Tool:      "vaultspectre",
		Timestamp: now,
		Summary:   models.VaultSummary{TotalReferences: 1},
		Secrets: map[string]*models.SecretInfo{
			"secret/ok": {Status: "ok"},
		},
	})

	report, err := ParseVaultReport(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Tool != "vaultspectre" {
		t.Errorf("expected tool=vaultspectre, got %s", report.Tool)
	}
}

func TestParseVaultReportDefaultTool(t *testing.T) {
	data := []byte(`{"summary":{"total_references":0},"secrets":{}}`)
	report, err := ParseVaultReport(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Tool != "vaultspectre" {
		t.Errorf("expected default tool=vaultspectre, got %s", report.Tool)
	}
}

func TestParseVaultReportInvalidJSON(t *testing.T) {
	_, err := ParseVaultReport([]byte("{"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseS3Report(t *testing.T) {
	now := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	data, _ := json.Marshal(models.S3Report{
		Tool:      "s3spectre",
		Timestamp: now,
		Summary:   models.S3Summary{TotalBuckets: 1},
	})

	report, err := ParseS3Report(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Tool != "s3spectre" {
		t.Errorf("expected tool=s3spectre, got %s", report.Tool)
	}
}

func TestParseS3ReportDefaultTool(t *testing.T) {
	data := []byte(`{"summary":{"total_buckets":0},"buckets":{}}`)
	report, err := ParseS3Report(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Tool != "s3spectre" {
		t.Errorf("expected default tool=s3spectre, got %s", report.Tool)
	}
}

func TestParseS3ReportInvalidJSON(t *testing.T) {
	_, err := ParseS3Report([]byte("{"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseKafkaReport(t *testing.T) {
	data, _ := json.Marshal(models.KafkaReport{
		Summary:         &models.KafkaSummary{TotalTopics: 1},
		ClusterMetadata: &models.ClusterMetadata{},
	})

	report, err := ParseKafkaReport(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Summary == nil {
		t.Error("expected non-nil summary")
	}
}

func TestParseKafkaReportInvalidJSON(t *testing.T) {
	_, err := ParseKafkaReport([]byte("{"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseClickHouseReport(t *testing.T) {
	now := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	data, _ := json.Marshal(models.ClickHouseReport{
		Metadata: models.ClickMetadata{GeneratedAt: now, LookbackDays: 30},
	})

	report, err := ParseClickHouseReport(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Metadata.LookbackDays != 30 {
		t.Errorf("expected lookback=30, got %d", report.Metadata.LookbackDays)
	}
}

func TestParseClickHouseReportInvalidJSON(t *testing.T) {
	_, err := ParseClickHouseReport([]byte("{"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParsePgReport(t *testing.T) {
	data := []byte(`{"metadata":{"tool":"pgspectre","version":"0.1.0"},"findings":[],"summary":{}}`)
	report, err := ParsePgReport(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Metadata.Tool != "pgspectre" {
		t.Errorf("expected tool=pgspectre, got %s", report.Metadata.Tool)
	}
}

func TestParsePgReportDefaults(t *testing.T) {
	data := []byte(`{"metadata":{},"summary":{}}`)
	report, err := ParsePgReport(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Metadata.Tool != "pgspectre" {
		t.Errorf("expected default tool=pgspectre, got %s", report.Metadata.Tool)
	}
	if report.Metadata.Timestamp == "" {
		t.Error("expected non-empty default timestamp")
	}
	if report.Findings == nil {
		t.Error("expected non-nil findings")
	}
}

func TestParsePgReportInvalidJSON(t *testing.T) {
	_, err := ParsePgReport([]byte("{"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseMongoReport(t *testing.T) {
	data := []byte(`{"metadata":{"version":"0.1.0","timestamp":"2026-02-15T00:00:00Z"},"findings":[],"summary":{}}`)
	report, err := ParseMongoReport(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Findings == nil {
		t.Error("expected non-nil findings")
	}
}

func TestParseMongoReportDefaults(t *testing.T) {
	data := []byte(`{"metadata":{},"summary":{}}`)
	report, err := ParseMongoReport(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Metadata.Timestamp == "" {
		t.Error("expected non-empty default timestamp")
	}
	if report.Findings == nil {
		t.Error("expected non-nil findings")
	}
}

func TestParseMongoReportInvalidJSON(t *testing.T) {
	_, err := ParseMongoReport([]byte("{"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseUnsupportedReport(t *testing.T) {
	data := []byte(`{"tool":"customtool","version":"1.0"}`)
	result, err := ParseUnsupportedReport(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["tool"] != "customtool" {
		t.Errorf("expected tool=customtool, got %v", result["tool"])
	}
}

func TestParseUnsupportedReportInvalidJSON(t *testing.T) {
	_, err := ParseUnsupportedReport([]byte("{"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseSpectreV1Report(t *testing.T) {
	now := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	v1 := models.SpectreV1Report{
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
	data, _ := json.Marshal(v1)

	report, err := ParseSpectreV1Report(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Schema != "spectre/v1" {
		t.Errorf("expected schema=spectre/v1, got %s", report.Schema)
	}
	if len(report.Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(report.Findings))
	}
}

func TestParseSpectreV1ReportInvalidJSON(t *testing.T) {
	_, err := ParseSpectreV1Report([]byte("{"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseSpectreV1ReportWrongSchema(t *testing.T) {
	data := []byte(`{"schema":"spectre/v2","tool":"s3spectre"}`)
	_, err := ParseSpectreV1Report(data)
	if err == nil {
		t.Fatal("expected error for wrong schema")
	}
}

func TestParseSpectreV1ReportMissingTool(t *testing.T) {
	data := []byte(`{"schema":"spectre/v1"}`)
	_, err := ParseSpectreV1Report(data)
	if err == nil {
		t.Fatal("expected error for missing tool")
	}
}

func TestParseSpectreV1ReportNilFindings(t *testing.T) {
	data := []byte(`{"schema":"spectre/v1","tool":"s3spectre","version":"0.2.1","timestamp":"2026-02-15T00:00:00Z","target":{"type":"s3"},"summary":{"total":0}}`)
	report, err := ParseSpectreV1Report(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Findings == nil {
		t.Error("expected non-nil findings")
	}
}

func TestParseReportSpectreV1Dispatch(t *testing.T) {
	now := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	v1 := models.SpectreV1Report{
		Schema:    "spectre/v1",
		Tool:      "s3spectre",
		Version:   "0.2.1",
		Timestamp: now,
		Target:    models.SpectreV1Target{Type: "s3"},
		Findings:  []models.SpectreV1Finding{},
		Summary:   models.SpectreV1Summary{Total: 0},
	}
	data, _ := json.Marshal(v1)

	// Even if we pass ToolVault, spectre/v1 envelope takes precedence
	result, err := ParseReport(data, models.ToolVault)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.(*models.SpectreV1Report); !ok {
		t.Fatalf("expected *models.SpectreV1Report, got %T", result)
	}
}

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"top level", []byte(`{"version":"1.2.3"}`), "1.2.3"},
		{"metadata", []byte(`{"metadata":{"version":"2.0.0"}}`), "2.0.0"},
		{"none", []byte(`{"tool":"test"}`), "unknown"},
		{"invalid json", []byte(`{`), "unknown"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractVersion(tt.data); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestExtractTimestamp(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		expectZero bool
	}{
		{
			name: "top level",
			data: []byte(`{"timestamp":"2026-02-15T00:00:00Z"}`),
		},
		{
			name: "metadata generated_at",
			data: []byte(`{"metadata":{"generated_at":"2026-02-15T00:00:00Z"}}`),
		},
		{
			name: "metadata timestamp string",
			data: []byte(`{"metadata":{"timestamp":"2026-02-15T00:00:00Z"}}`),
		},
		{
			name: "kafka cluster_metadata",
			data: []byte(`{"cluster_metadata":{"fetched_at":"2026-02-15 10:30:00 UTC"}}`),
		},
		{
			name: "no timestamp returns now",
			data: []byte(`{"tool":"test"}`),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ts := ExtractTimestamp(tt.data)
			if ts.IsZero() && !tt.expectZero {
				t.Fatal("expected non-zero timestamp")
			}
		})
	}
}
