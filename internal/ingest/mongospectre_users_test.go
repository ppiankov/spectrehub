package ingest

import (
	"testing"

	"github.com/ppiankov/spectrehub/internal/models"
)

func TestExtractUserActivityInactiveUser(t *testing.T) {
	v1 := &models.SpectreV1Report{
		Tool: "mongospectre",
		Findings: []models.SpectreV1Finding{
			{
				ID:       "INACTIVE_USER",
				Severity: "medium",
				Location: "admin.",
				Message:  `user "appuser" has no authentication in the last 7 days`,
			},
		},
	}

	entries := ExtractUserActivity(v1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Username != "appuser" {
		t.Fatalf("Username = %s, want appuser", entries[0].Username)
	}
	if entries[0].DatabaseName != "admin" {
		t.Fatalf("DatabaseName = %s, want admin", entries[0].DatabaseName)
	}
	if entries[0].IsPrivileged {
		t.Fatal("expected IsPrivileged = false")
	}
}

func TestExtractUserActivityPrivilegedUser(t *testing.T) {
	v1 := &models.SpectreV1Report{
		Tool: "mongospectre",
		Findings: []models.SpectreV1Finding{
			{
				ID:       "INACTIVE_PRIVILEGED_USER",
				Severity: "high",
				Location: "admin.",
				Message:  `privileged user "root" has no authentication in the last 7 days`,
			},
		},
	}

	entries := ExtractUserActivity(v1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Username != "root" {
		t.Fatalf("Username = %s, want root", entries[0].Username)
	}
	if !entries[0].IsPrivileged {
		t.Fatal("expected IsPrivileged = true")
	}
}

func TestExtractUserActivityFailedAuth(t *testing.T) {
	v1 := &models.SpectreV1Report{
		Tool: "mongospectre",
		Findings: []models.SpectreV1Finding{
			{
				ID:       "FAILED_AUTH_ONLY",
				Severity: "medium",
				Location: "admin.",
				Message:  `user "botuser" has only failed authentication attempts in the last 7 days`,
			},
		},
	}

	entries := ExtractUserActivity(v1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Username != "botuser" {
		t.Fatalf("Username = %s, want botuser", entries[0].Username)
	}
	if entries[0].IsPrivileged {
		t.Fatal("expected IsPrivileged = false for FAILED_AUTH_ONLY")
	}
}

func TestExtractUserActivityMixedFindings(t *testing.T) {
	v1 := &models.SpectreV1Report{
		Tool: "mongospectre",
		Findings: []models.SpectreV1Finding{
			{ID: "UNUSED_COLLECTION", Severity: "medium", Location: "mydb.legacy_logs", Message: "zero reads"},
			{ID: "INACTIVE_USER", Severity: "medium", Location: "admin.", Message: `user "stale" has no authentication in the last 7 days`},
			{ID: "MISSING_INDEX", Severity: "medium", Location: "mydb.orders", Message: "needs index"},
			{ID: "INACTIVE_PRIVILEGED_USER", Severity: "high", Location: "admin.", Message: `privileged user "old_admin" has no authentication in the last 7 days`},
		},
	}

	entries := ExtractUserActivity(v1)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (user findings only), got %d", len(entries))
	}
	if entries[0].Username != "stale" {
		t.Fatalf("first entry Username = %s, want stale", entries[0].Username)
	}
	if entries[1].Username != "old_admin" {
		t.Fatalf("second entry Username = %s, want old_admin", entries[1].Username)
	}
}

func TestExtractUserActivityNonMongoTool(t *testing.T) {
	v1 := &models.SpectreV1Report{
		Tool: "pgspectre",
		Findings: []models.SpectreV1Finding{
			{ID: "INACTIVE_USER", Severity: "medium", Location: "admin.", Message: `user "x" has no authentication in the last 7 days`},
		},
	}

	entries := ExtractUserActivity(v1)
	if entries != nil {
		t.Fatalf("expected nil for non-mongospectre tool, got %v", entries)
	}
}

func TestExtractUserActivityNilReport(t *testing.T) {
	entries := ExtractUserActivity(nil)
	if entries != nil {
		t.Fatalf("expected nil for nil report, got %v", entries)
	}
}

func TestExtractUserActivityNoUserFindings(t *testing.T) {
	v1 := &models.SpectreV1Report{
		Tool: "mongospectre",
		Findings: []models.SpectreV1Finding{
			{ID: "UNUSED_COLLECTION", Severity: "medium", Location: "mydb.old", Message: "unused"},
			{ID: "MISSING_INDEX", Severity: "medium", Location: "mydb.orders", Message: "needs index"},
		},
	}

	entries := ExtractUserActivity(v1)
	if entries != nil {
		t.Fatalf("expected nil when no user findings, got %d entries", len(entries))
	}
}

func TestExtractUsernameVariations(t *testing.T) {
	tests := []struct {
		message  string
		expected string
	}{
		{`user "simple" has no authentication in the last 7 days`, "simple"},
		{`privileged user "admin_root" has no authentication in the last 7 days`, "admin_root"},
		{`user "user-with-dashes" has only failed authentication attempts in the last 7 days`, "user-with-dashes"},
		{`user "user.with.dots" has no authentication in the last 7 days`, "user.with.dots"},
		{"no username here", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := extractUsername(tt.message)
		if got != tt.expected {
			t.Errorf("extractUsername(%q) = %q, want %q", tt.message, got, tt.expected)
		}
	}
}

func TestExtractDatabaseFormats(t *testing.T) {
	tests := []struct {
		location string
		expected string
	}{
		{"admin.", "admin"},
		{"mydb.users", "mydb"},
		{"", ""},
		{".", ""},
		{"standalone", "standalone"},
	}

	for _, tt := range tests {
		got := extractDatabase(tt.location)
		if got != tt.expected {
			t.Errorf("extractDatabase(%q) = %q, want %q", tt.location, got, tt.expected)
		}
	}
}
