package ingest

import (
	"crypto/sha256"
	"fmt"

	"github.com/ppiankov/spectrehub/internal/apiclient"
	"github.com/ppiankov/spectrehub/internal/models"
)

// iamCredentialTypes maps iamspectre finding IDs to credential_type values.
var iamCredentialTypes = map[string]string{
	"STALE_ACCESS_KEY":         "access_key",
	"INACTIVE_ACCESS_KEY":      "access_key",
	"UNUSED_ACCESS_KEY":        "access_key",
	"NO_MFA":                   "mfa",
	"MFA_NOT_ENABLED":          "mfa",
	"ADMIN_ACCESS":             "admin",
	"OVERPRIVILEGED_USER":      "admin",
	"UNUSED_ROLE":              "role",
	"INACTIVE_ROLE":            "role",
	"INACTIVE_USER":            "user",
	"INACTIVE_PRIVILEGED_USER": "user",
	"ROOT_ACCESS_KEY":          "root",
	"ROOT_MFA_DISABLED":        "root",
}

// targetTypeToCloud maps spectre/v1 target types to cloud providers.
var targetTypeToCloud = map[string]string{
	"aws-account":  "aws",
	"gcp-project":  "gcp",
	"gcp-projects": "gcp",
}

// ExtractFindings scans a spectre/v1 report and returns API-ready finding
// entries for the finding lifecycle API. Each finding gets a deterministic
// hash based on tool + location + finding ID.
// For iamspectre reports, IAM metadata (identity, credential_type, cloud)
// is populated automatically.
func ExtractFindings(v1 *models.SpectreV1Report) []apiclient.FindingEntry {
	if v1 == nil || len(v1.Findings) == 0 {
		return nil
	}

	entries := make([]apiclient.FindingEntry, 0, len(v1.Findings))
	for _, f := range v1.Findings {
		hash := computeFindingHash(v1.Tool, f.Location, f.ID)

		entry := apiclient.FindingEntry{
			FindingHash: hash,
			FindingID:   f.ID,
			ResourceID:  f.Location,
			Severity:    f.Severity,
		}

		// WO-97: populate IAM metadata for iamspectre findings.
		if v1.Tool == "iamspectre" {
			entry.Identity = f.Location
			if ct, ok := iamCredentialTypes[f.ID]; ok {
				entry.CredentialType = ct
			}
			if cloud, ok := targetTypeToCloud[v1.Target.Type]; ok {
				entry.Cloud = cloud
			}
		}

		entries = append(entries, entry)
	}

	return entries
}

// computeFindingHash produces a deterministic SHA-256 hash from the
// tool name, resource location, and finding ID. This ensures the same
// finding always maps to the same hash across runs.
func computeFindingHash(tool, location, findingID string) string {
	input := tool + ":" + location + ":" + findingID
	sum := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", sum)
}
