package api

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const (
	// MaxRepoLength prevents pathological or abusive repo identifiers.
	MaxRepoLength = 255

	// MaxRawReportPayloadBytes bounds uploaded report JSON payload size.
	MaxRawReportPayloadBytes = 2 * 1024 * 1024 // 2 MiB

	maxReportTools  = 10_000
	maxReportIssues = 1_000_000
)

var (
	repoPattern       = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	licenseKeyPattern = regexp.MustCompile(`^sh_(live|test)_[0-9a-f]{32}$`)
	validHealthScores = map[string]struct{}{
		"excellent": {},
		"good":      {},
		"warning":   {},
		"critical":  {},
		"severe":    {},
	}
)

// ReportInput captures the API payload fields that must be validated.
type ReportInput struct {
	Repo       string
	TotalTools int
	Issues     int
	Score      float64
	Health     string
	RawJSON    string
}

// ValidateRepo verifies repository identifier format (org/repo).
func ValidateRepo(repo string) error {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return fmt.Errorf("repo is required")
	}
	if len(repo) > MaxRepoLength {
		return fmt.Errorf("repo exceeds %d characters", MaxRepoLength)
	}
	if !repoPattern.MatchString(repo) {
		return fmt.Errorf("repo must match org/repo format")
	}
	return nil
}

// ValidateLicenseKey enforces canonical SpectreHub license key format.
func ValidateLicenseKey(licenseKey string) error {
	licenseKey = strings.TrimSpace(licenseKey)
	if licenseKey == "" {
		return fmt.Errorf("license key is required")
	}
	if !licenseKeyPattern.MatchString(licenseKey) {
		return fmt.Errorf("license key must match sh_live_<32 hex> or sh_test_<32 hex>")
	}
	return nil
}

// ValidateReportInput checks report fields before submission.
func ValidateReportInput(input ReportInput) error {
	if err := ValidateRepo(input.Repo); err != nil {
		return err
	}
	if input.TotalTools < 0 || input.TotalTools > maxReportTools {
		return fmt.Errorf("total_tools must be between 0 and %d", maxReportTools)
	}
	if input.Issues < 0 || input.Issues > maxReportIssues {
		return fmt.Errorf("issues must be between 0 and %d", maxReportIssues)
	}
	if input.Score < 0 || input.Score > 100 {
		return fmt.Errorf("score must be between 0 and 100")
	}

	health := strings.TrimSpace(input.Health)
	if health == "" {
		return fmt.Errorf("health is required")
	}
	if _, ok := validHealthScores[health]; !ok {
		return fmt.Errorf("unsupported health score %q", health)
	}

	rawJSON := strings.TrimSpace(input.RawJSON)
	if rawJSON == "" {
		return fmt.Errorf("raw_json is required")
	}
	if len(rawJSON) > MaxRawReportPayloadBytes {
		return fmt.Errorf("raw_json exceeds %d bytes", MaxRawReportPayloadBytes)
	}
	if !json.Valid([]byte(rawJSON)) {
		return fmt.Errorf("raw_json must be valid JSON")
	}

	return nil
}
