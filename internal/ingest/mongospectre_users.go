package ingest

import (
	"regexp"
	"strings"

	"github.com/ppiankov/spectrehub/internal/apiclient"
	"github.com/ppiankov/spectrehub/internal/models"
)

// userFindingIDs are the finding IDs that represent user activity data.
var userFindingIDs = map[string]bool{
	"INACTIVE_USER":            true,
	"INACTIVE_PRIVILEGED_USER": true,
	"FAILED_AUTH_ONLY":         true,
}

// usernameRe extracts a username from mongospectre message strings.
// Matches: user "USERNAME" or privileged user "USERNAME"
var usernameRe = regexp.MustCompile(`(?:privileged )?user "([^"]+)"`)

// ExtractUserActivity scans a spectre/v1 report for user activity findings
// and returns them as API-ready entries. Returns nil if tool is not mongospectre
// or no user findings exist.
func ExtractUserActivity(v1 *models.SpectreV1Report) []apiclient.UserActivityEntry {
	if v1 == nil || v1.Tool != "mongospectre" {
		return nil
	}

	var entries []apiclient.UserActivityEntry
	for _, f := range v1.Findings {
		if !userFindingIDs[f.ID] {
			continue
		}

		username := extractUsername(f.Message)
		if username == "" {
			continue
		}

		entries = append(entries, apiclient.UserActivityEntry{
			Username:     username,
			DatabaseName: extractDatabase(f.Location),
			IsPrivileged: f.ID == "INACTIVE_PRIVILEGED_USER",
		})
	}
	return entries
}

func extractUsername(message string) string {
	matches := usernameRe.FindStringSubmatch(message)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func extractDatabase(location string) string {
	parts := strings.SplitN(location, ".", 2)
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return ""
}
