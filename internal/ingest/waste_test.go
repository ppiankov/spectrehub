package ingest

import (
	"testing"

	"github.com/ppiankov/spectrehub/internal/models"
)

func ptr(f float64) *float64 { return &f }

func TestExtractWasteBasic(t *testing.T) {
	v1 := &models.SpectreV1Report{
		Tool: "awsspectre",
		Target: models.SpectreV1Target{
			Type:    "aws-account",
			URIHash: "abc123",
		},
		Findings: []models.SpectreV1Finding{
			{ID: "IDLE_EC2", Severity: "high", Location: "i-abc123", Message: "idle", EstimatedMonthlyWaste: ptr(45.50)},
			{ID: "DETACHED_EBS", Severity: "medium", Location: "vol-xyz", Message: "detached", EstimatedMonthlyWaste: ptr(12.00)},
		},
	}

	entries := ExtractWaste(v1)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].ResourceID != "i-abc123" {
		t.Errorf("ResourceID = %s, want i-abc123", entries[0].ResourceID)
	}
	if entries[0].ResourceType != "ec2" {
		t.Errorf("ResourceType = %s, want ec2", entries[0].ResourceType)
	}
	if entries[0].FindingID != "IDLE_EC2" {
		t.Errorf("FindingID = %s, want IDLE_EC2", entries[0].FindingID)
	}
	if entries[0].EstimatedMonthlyWaste != 45.50 {
		t.Errorf("EstimatedMonthlyWaste = %.2f, want 45.50", entries[0].EstimatedMonthlyWaste)
	}

	if entries[1].ResourceType != "ebs" {
		t.Errorf("ResourceType = %s, want ebs", entries[1].ResourceType)
	}
}

func TestExtractWasteNoWasteField(t *testing.T) {
	v1 := &models.SpectreV1Report{
		Tool: "awsspectre",
		Target: models.SpectreV1Target{
			Type: "aws-account",
		},
		Findings: []models.SpectreV1Finding{
			{ID: "IDLE_EC2", Severity: "high", Location: "i-abc123", Message: "idle"},
			{ID: "PUBLIC_BUCKET", Severity: "high", Location: "s3://bucket", Message: "public", EstimatedMonthlyWaste: ptr(0)},
		},
	}

	entries := ExtractWaste(v1)
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries (no waste or zero waste), got %d", len(entries))
	}
}

func TestExtractWasteNonWasteTool(t *testing.T) {
	v1 := &models.SpectreV1Report{
		Tool: "pgspectre",
		Target: models.SpectreV1Target{
			Type: "postgres",
		},
		Findings: []models.SpectreV1Finding{
			{ID: "WEAK_PASSWORD", Severity: "high", Location: "pg://db", Message: "weak", EstimatedMonthlyWaste: ptr(100.00)},
		},
	}

	entries := ExtractWaste(v1)
	if entries != nil {
		t.Fatalf("expected nil for non-waste tool, got %v", entries)
	}
}

func TestExtractWasteNilReport(t *testing.T) {
	entries := ExtractWaste(nil)
	if entries != nil {
		t.Fatalf("expected nil for nil report, got %v", entries)
	}
}

func TestExtractWasteResourceTypeMapping(t *testing.T) {
	cases := []struct {
		findingID    string
		expectedType string
	}{
		{"IDLE_EC2", "ec2"},
		{"STOPPED_EC2", "ec2"},
		{"DETACHED_EBS", "ebs"},
		{"IDLE_RDS", "rds"},
		{"IDLE_ALB", "elb"},
		{"IDLE_NLB", "elb"},
		{"IDLE_NAT_GATEWAY", "nat"},
		{"IDLE_LAMBDA", "lambda"},
		{"UNUSED_EIP", "eip"},
		{"IDLE_VM", "compute"},
		{"STOPPED_VM", "compute"},
		{"UNATTACHED_DISK", "disk"},
		{"IDLE_SQL", "cloudsql"},
		{"UNKNOWN_CHECK", "other"},
	}

	for _, tc := range cases {
		v1 := &models.SpectreV1Report{
			Tool: "awsspectre",
			Findings: []models.SpectreV1Finding{
				{ID: tc.findingID, Severity: "high", Location: "resource-1", Message: "test", EstimatedMonthlyWaste: ptr(10.00)},
			},
		}

		entries := ExtractWaste(v1)
		if len(entries) != 1 {
			t.Errorf("%s: expected 1 entry, got %d", tc.findingID, len(entries))
			continue
		}
		if entries[0].ResourceType != tc.expectedType {
			t.Errorf("%s: ResourceType = %s, want %s", tc.findingID, entries[0].ResourceType, tc.expectedType)
		}
	}
}

func TestExtractWasteGCPSpectre(t *testing.T) {
	v1 := &models.SpectreV1Report{
		Tool: "gcpspectre",
		Target: models.SpectreV1Target{
			Type: "gcp-projects",
		},
		Findings: []models.SpectreV1Finding{
			{ID: "IDLE_VM", Severity: "high", Location: "projects/my-proj/zones/us-central1-a/instances/vm-1", Message: "idle", EstimatedMonthlyWaste: ptr(30.00)},
		},
	}

	entries := ExtractWaste(v1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ResourceType != "compute" {
		t.Errorf("ResourceType = %s, want compute", entries[0].ResourceType)
	}
}

func TestExtractWasteAzureSpectre(t *testing.T) {
	v1 := &models.SpectreV1Report{
		Tool: "azurespectre",
		Target: models.SpectreV1Target{
			Type: "azure-subscription",
		},
		Findings: []models.SpectreV1Finding{
			{ID: "UNATTACHED_DISK", Severity: "medium", Location: "/subscriptions/sub1/disks/disk1", Message: "unattached", EstimatedMonthlyWaste: ptr(15.00)},
		},
	}

	entries := ExtractWaste(v1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ResourceType != "disk" {
		t.Errorf("ResourceType = %s, want disk", entries[0].ResourceType)
	}
}

func TestExtractWasteMixedFindings(t *testing.T) {
	v1 := &models.SpectreV1Report{
		Tool: "awsspectre",
		Findings: []models.SpectreV1Finding{
			{ID: "IDLE_EC2", Severity: "high", Location: "i-1", Message: "idle", EstimatedMonthlyWaste: ptr(50.00)},
			{ID: "PUBLIC_BUCKET", Severity: "high", Location: "s3://bucket", Message: "public"},
			{ID: "DETACHED_EBS", Severity: "medium", Location: "vol-1", Message: "detached", EstimatedMonthlyWaste: ptr(10.00)},
		},
	}

	entries := ExtractWaste(v1)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (skipping finding without waste), got %d", len(entries))
	}
}
