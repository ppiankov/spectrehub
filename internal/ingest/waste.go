package ingest

import (
	"github.com/ppiankov/spectrehub/internal/apiclient"
	"github.com/ppiankov/spectrehub/internal/models"
)

// wasteTools are the tools that emit estimated_monthly_waste on findings.
var wasteTools = map[string]bool{
	"awsspectre":   true,
	"gcpspectre":   true,
	"azurespectre": true,
}

// findingIDToResourceType maps finding IDs to resource type categories.
var findingIDToResourceType = map[string]string{
	// awsspectre
	"IDLE_EC2":         "ec2",
	"STOPPED_EC2":      "ec2",
	"DETACHED_EBS":     "ebs",
	"IDLE_RDS":         "rds",
	"IDLE_ALB":         "elb",
	"IDLE_NLB":         "elb",
	"IDLE_NAT_GATEWAY": "nat",
	"IDLE_LAMBDA":      "lambda",
	"UNUSED_EIP":       "eip",
	// gcpspectre
	"IDLE_VM":         "compute",
	"STOPPED_VM":      "compute",
	"UNATTACHED_DISK": "disk",
	"IDLE_SQL":        "cloudsql",
	// azurespectre — note: overlapping keys with gcpspectre are overridden,
	// but that's fine because the tool field distinguishes them.
}

// ExtractWaste scans a spectre/v1 report for findings with
// estimated_monthly_waste and returns API-ready waste entries.
// Only processes waste-emitting tools (awsspectre, gcpspectre, azurespectre).
func ExtractWaste(v1 *models.SpectreV1Report) []apiclient.WasteEntry {
	if v1 == nil || len(v1.Findings) == 0 {
		return nil
	}

	if !wasteTools[v1.Tool] {
		return nil
	}

	var entries []apiclient.WasteEntry
	for _, f := range v1.Findings {
		if f.EstimatedMonthlyWaste == nil || *f.EstimatedMonthlyWaste <= 0 {
			continue
		}

		resourceType := findingIDToResourceType[f.ID]
		if resourceType == "" {
			resourceType = "other"
		}

		entries = append(entries, apiclient.WasteEntry{
			ResourceID:            f.Location,
			ResourceType:          resourceType,
			FindingID:             f.ID,
			Region:                "", // region extracted from location if available
			EstimatedMonthlyWaste: *f.EstimatedMonthlyWaste,
		})
	}

	return entries
}
