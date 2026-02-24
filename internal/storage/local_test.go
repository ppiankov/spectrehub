package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

func sampleReport(ts time.Time) *models.AggregatedReport {
	return &models.AggregatedReport{
		Timestamp: ts,
		Issues:    []models.NormalizedIssue{},
		ToolReports: map[string]models.ToolReport{
			"vaultspectre": {Tool: "vaultspectre"},
		},
		Summary: models.CrossToolSummary{
			TotalIssues:      2,
			TotalTools:       1,
			IssuesByTool:     map[string]int{"vaultspectre": 2},
			IssuesByCategory: map[string]int{"missing": 2},
			IssuesBySeverity: map[string]int{"critical": 2},
		},
		Recommendations: []models.Recommendation{},
	}
}

func TestNewLocal(t *testing.T) {
	s := NewLocal("/tmp/test")
	if s.baseDir != "/tmp/test" {
		t.Errorf("expected baseDir=/tmp/test, got %s", s.baseDir)
	}
}

func TestGetStoragePath(t *testing.T) {
	s := NewLocal("/tmp/spectre")
	if s.GetStoragePath() != "/tmp/spectre" {
		t.Errorf("expected /tmp/spectre, got %s", s.GetStoragePath())
	}
}

func TestEnsureDirectoryExists(t *testing.T) {
	dir := t.TempDir()
	baseDir := filepath.Join(dir, "nested", "spectre")
	s := NewLocal(baseDir)

	if err := s.EnsureDirectoryExists(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	runsDir := filepath.Join(baseDir, "runs")
	if _, err := os.Stat(runsDir); err != nil {
		t.Fatalf("expected runs directory to exist: %v", err)
	}
}

func TestSaveAndLoadAggregatedReport(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir)

	ts := time.Date(2026, 2, 15, 10, 30, 0, 0, time.UTC)
	report := sampleReport(ts)

	// Save
	if err := s.SaveAggregatedReport(report); err != nil {
		t.Fatalf("SaveAggregatedReport: %v", err)
	}

	// Load
	loaded, err := s.LoadAggregatedReport(ts)
	if err != nil {
		t.Fatalf("LoadAggregatedReport: %v", err)
	}
	if loaded.Summary.TotalIssues != 2 {
		t.Errorf("expected 2 issues, got %d", loaded.Summary.TotalIssues)
	}
	if loaded.Summary.TotalTools != 1 {
		t.Errorf("expected 1 tool, got %d", loaded.Summary.TotalTools)
	}
}

func TestLoadAggregatedReportNotFound(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir)

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := s.LoadAggregatedReport(ts)
	if err == nil {
		t.Fatal("expected error for missing report")
	}
}

func TestListRunsEmpty(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir)

	runs, err := s.ListRuns()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 0 {
		t.Errorf("expected 0 runs, got %d", len(runs))
	}
}

func TestListRunsMultiple(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir)

	ts1 := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 2, 12, 10, 0, 0, 0, time.UTC)
	ts3 := time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC)

	for _, ts := range []time.Time{ts2, ts1, ts3} {
		if err := s.SaveAggregatedReport(sampleReport(ts)); err != nil {
			t.Fatalf("SaveAggregatedReport: %v", err)
		}
	}

	runs, err := s.ListRuns()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(runs))
	}

	// Should be sorted chronologically
	if !runs[0].Before(runs[1]) || !runs[1].Before(runs[2]) {
		t.Error("runs should be sorted chronologically")
	}
}

func TestGetLatestRun(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir)

	ts1 := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 2, 12, 10, 0, 0, 0, time.UTC)

	if err := s.SaveAggregatedReport(sampleReport(ts1)); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveAggregatedReport(sampleReport(ts2)); err != nil {
		t.Fatal(err)
	}

	latest, err := s.GetLatestRun()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !latest.Timestamp.Equal(ts2) {
		t.Errorf("expected latest run at %v, got %v", ts2, latest.Timestamp)
	}
}

func TestGetLatestRunEmpty(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir)

	_, err := s.GetLatestRun()
	if err == nil {
		t.Fatal("expected error for empty storage")
	}
}

func TestGetLastNRuns(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir)

	timestamps := []time.Time{
		time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 12, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC),
	}

	for _, ts := range timestamps {
		if err := s.SaveAggregatedReport(sampleReport(ts)); err != nil {
			t.Fatal(err)
		}
	}

	// Get last 3
	runs, err := s.GetLastNRuns(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(runs))
	}

	// Get more than available
	runs, err = s.GetLastNRuns(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 5 {
		t.Fatalf("expected 5 runs, got %d", len(runs))
	}
}

func TestGetLastNRunsEmpty(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir)

	_, err := s.GetLastNRuns(3)
	if err == nil {
		t.Fatal("expected error for empty storage")
	}
}

func TestListRunsIgnoresNonAggregatedFiles(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir)

	runsDir := filepath.Join(dir, "runs")
	if err := os.MkdirAll(runsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a non-aggregated file
	if err := os.WriteFile(filepath.Join(runsDir, "not-aggregated.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create a directory inside runs
	if err := os.MkdirAll(filepath.Join(runsDir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	// Create a file with invalid timestamp
	if err := os.WriteFile(filepath.Join(runsDir, "bad-time-aggregated.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	runs, err := s.ListRuns()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 0 {
		t.Errorf("expected 0 runs, got %d", len(runs))
	}
}

func TestFormatAndParseTimestamp(t *testing.T) {
	s := NewLocal("/tmp")
	ts := time.Date(2026, 2, 15, 10, 30, 45, 0, time.UTC)

	formatted := s.formatTimestamp(ts)
	if formatted != "2026-02-15T10-30-45" {
		t.Errorf("expected 2026-02-15T10-30-45, got %s", formatted)
	}

	parsed, err := s.parseTimestamp(formatted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !parsed.Equal(ts) {
		t.Errorf("expected %v, got %v", ts, parsed)
	}
}

func TestParseTimestampInvalid(t *testing.T) {
	s := NewLocal("/tmp")
	_, err := s.parseTimestamp("not-a-timestamp")
	if err == nil {
		t.Fatal("expected error for invalid timestamp")
	}
}
