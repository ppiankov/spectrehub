package ingest

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/ppiankov/spectrehub/internal/models"
)

func TestExtractFindingsBasic(t *testing.T) {
	v1 := &models.SpectreV1Report{
		Tool: "awsspectre",
		Target: models.SpectreV1Target{
			Type:    "aws-account",
			URIHash: "abc123",
		},
		Findings: []models.SpectreV1Finding{
			{ID: "PUBLIC_BUCKET", Severity: "high", Location: "s3://my-bucket", Message: "publicly accessible"},
			{ID: "UNENCRYPTED", Severity: "medium", Location: "s3://data-bucket", Message: "no encryption"},
		},
	}

	entries := ExtractFindings(v1)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Verify deterministic hashes.
	expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte("awsspectre:s3://my-bucket:PUBLIC_BUCKET")))
	if entries[0].FindingHash != expectedHash {
		t.Errorf("hash = %s, want %s", entries[0].FindingHash, expectedHash)
	}
	if entries[0].FindingID != "PUBLIC_BUCKET" {
		t.Errorf("FindingID = %s, want PUBLIC_BUCKET", entries[0].FindingID)
	}
	if entries[0].ResourceID != "s3://my-bucket" {
		t.Errorf("ResourceID = %s, want s3://my-bucket", entries[0].ResourceID)
	}
	if entries[0].Severity != "high" {
		t.Errorf("Severity = %s, want high", entries[0].Severity)
	}

	// Non-IAM findings should have empty IAM fields.
	if entries[0].Identity != "" {
		t.Errorf("Identity = %s, want empty", entries[0].Identity)
	}
	if entries[0].CredentialType != "" {
		t.Errorf("CredentialType = %s, want empty", entries[0].CredentialType)
	}
	if entries[0].Cloud != "" {
		t.Errorf("Cloud = %s, want empty", entries[0].Cloud)
	}
}

func TestExtractFindingsIAMSpectre(t *testing.T) {
	v1 := &models.SpectreV1Report{
		Tool: "iamspectre",
		Target: models.SpectreV1Target{
			Type: "aws-account",
		},
		Findings: []models.SpectreV1Finding{
			{ID: "STALE_ACCESS_KEY", Severity: "high", Location: "arn:aws:iam::123:user/alice", Message: "stale key"},
			{ID: "NO_MFA", Severity: "critical", Location: "arn:aws:iam::123:user/bob", Message: "no MFA enabled"},
		},
	}

	entries := ExtractFindings(v1)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// First entry: STALE_ACCESS_KEY
	if entries[0].Identity != "arn:aws:iam::123:user/alice" {
		t.Errorf("Identity = %s, want arn:aws:iam::123:user/alice", entries[0].Identity)
	}
	if entries[0].CredentialType != "access_key" {
		t.Errorf("CredentialType = %s, want access_key", entries[0].CredentialType)
	}
	if entries[0].Cloud != "aws" {
		t.Errorf("Cloud = %s, want aws", entries[0].Cloud)
	}

	// Second entry: NO_MFA
	if entries[1].CredentialType != "mfa" {
		t.Errorf("CredentialType = %s, want mfa", entries[1].CredentialType)
	}
}

func TestExtractFindingsIAMSpectreGCP(t *testing.T) {
	v1 := &models.SpectreV1Report{
		Tool: "iamspectre",
		Target: models.SpectreV1Target{
			Type: "gcp-project",
		},
		Findings: []models.SpectreV1Finding{
			{ID: "ADMIN_ACCESS", Severity: "high", Location: "user:admin@example.com", Message: "admin access"},
		},
	}

	entries := ExtractFindings(v1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Cloud != "gcp" {
		t.Errorf("Cloud = %s, want gcp", entries[0].Cloud)
	}
	if entries[0].CredentialType != "admin" {
		t.Errorf("CredentialType = %s, want admin", entries[0].CredentialType)
	}
}

func TestExtractFindingsNilReport(t *testing.T) {
	entries := ExtractFindings(nil)
	if entries != nil {
		t.Fatalf("expected nil for nil report, got %v", entries)
	}
}

func TestExtractFindingsEmptyFindings(t *testing.T) {
	v1 := &models.SpectreV1Report{
		Tool:     "pgspectre",
		Findings: []models.SpectreV1Finding{},
	}

	entries := ExtractFindings(v1)
	if entries != nil {
		t.Fatalf("expected nil for empty findings, got %v", entries)
	}
}

func TestExtractFindingsUnknownIAMFindingID(t *testing.T) {
	v1 := &models.SpectreV1Report{
		Tool: "iamspectre",
		Target: models.SpectreV1Target{
			Type: "aws-account",
		},
		Findings: []models.SpectreV1Finding{
			{ID: "CUSTOM_CHECK", Severity: "low", Location: "arn:aws:iam::123:user/test", Message: "custom"},
		},
	}

	entries := ExtractFindings(v1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	// Unknown finding ID should leave credential_type empty.
	if entries[0].CredentialType != "" {
		t.Errorf("CredentialType = %s, want empty for unknown finding ID", entries[0].CredentialType)
	}
	// But identity and cloud should still be populated.
	if entries[0].Identity != "arn:aws:iam::123:user/test" {
		t.Errorf("Identity = %s, want arn:aws:iam::123:user/test", entries[0].Identity)
	}
	if entries[0].Cloud != "aws" {
		t.Errorf("Cloud = %s, want aws", entries[0].Cloud)
	}
}

func TestComputeFindingHashDeterministic(t *testing.T) {
	hash1 := computeFindingHash("awsspectre", "s3://bucket", "PUBLIC_BUCKET")
	hash2 := computeFindingHash("awsspectre", "s3://bucket", "PUBLIC_BUCKET")
	if hash1 != hash2 {
		t.Errorf("hashes should be deterministic: %s != %s", hash1, hash2)
	}

	hash3 := computeFindingHash("pgspectre", "s3://bucket", "PUBLIC_BUCKET")
	if hash1 == hash3 {
		t.Error("different tools should produce different hashes")
	}
}

func TestExtractFindingsMultipleTools(t *testing.T) {
	pgReport := &models.SpectreV1Report{
		Tool:   "pgspectre",
		Target: models.SpectreV1Target{Type: "postgres"},
		Findings: []models.SpectreV1Finding{
			{ID: "WEAK_PASSWORD", Severity: "high", Location: "pg://db/users", Message: "weak password"},
		},
	}

	mongoReport := &models.SpectreV1Report{
		Tool:   "mongospectre",
		Target: models.SpectreV1Target{Type: "mongodb"},
		Findings: []models.SpectreV1Finding{
			{ID: "NO_AUTH", Severity: "critical", Location: "mongodb://host/admin", Message: "no auth"},
		},
	}

	pgEntries := ExtractFindings(pgReport)
	mongoEntries := ExtractFindings(mongoReport)

	if len(pgEntries) != 1 {
		t.Fatalf("expected 1 pg entry, got %d", len(pgEntries))
	}
	if len(mongoEntries) != 1 {
		t.Fatalf("expected 1 mongo entry, got %d", len(mongoEntries))
	}

	// Hashes should differ because tool is different.
	if pgEntries[0].FindingHash == mongoEntries[0].FindingHash {
		t.Error("findings from different tools should have different hashes")
	}
}
