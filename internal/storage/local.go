package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

// LocalStorage implements Storage interface using local filesystem
type LocalStorage struct {
	baseDir string
}

// NewLocal creates a new local storage instance
func NewLocal(baseDir string) *LocalStorage {
	return &LocalStorage{
		baseDir: baseDir,
	}
}

// SaveAggregatedReport stores an aggregated report to disk
func (s *LocalStorage) SaveAggregatedReport(report *models.AggregatedReport) error {
	// Create runs directory
	runsDir := filepath.Join(s.baseDir, "runs")
	if err := os.MkdirAll(runsDir, 0755); err != nil {
		return fmt.Errorf("failed to create runs directory: %w", err)
	}

	// Generate filename with timestamp
	filename := s.formatTimestamp(report.Timestamp) + "-aggregated.json"
	path := filepath.Join(runsDir, filename)

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// LoadAggregatedReport loads a report from a specific timestamp
func (s *LocalStorage) LoadAggregatedReport(timestamp time.Time) (*models.AggregatedReport, error) {
	filename := s.formatTimestamp(timestamp) + "-aggregated.json"
	path := filepath.Join(s.baseDir, "runs", filename)

	return s.loadReportFromFile(path)
}

// GetLatestRun retrieves the most recent aggregated report
func (s *LocalStorage) GetLatestRun() (*models.AggregatedReport, error) {
	timestamps, err := s.ListRuns()
	if err != nil {
		return nil, err
	}

	if len(timestamps) == 0 {
		return nil, fmt.Errorf("no runs found")
	}

	// Get the latest timestamp
	latest := timestamps[len(timestamps)-1]
	return s.LoadAggregatedReport(latest)
}

// GetLastNRuns retrieves the last N aggregated reports
func (s *LocalStorage) GetLastNRuns(n int) ([]*models.AggregatedReport, error) {
	timestamps, err := s.ListRuns()
	if err != nil {
		return nil, err
	}

	if len(timestamps) == 0 {
		return nil, fmt.Errorf("no runs found")
	}

	// Get the last N timestamps
	start := len(timestamps) - n
	if start < 0 {
		start = 0
	}

	selectedTimestamps := timestamps[start:]
	reports := make([]*models.AggregatedReport, 0, len(selectedTimestamps))

	for _, timestamp := range selectedTimestamps {
		report, err := s.LoadAggregatedReport(timestamp)
		if err != nil {
			// Skip reports that fail to load but continue with others
			continue
		}
		reports = append(reports, report)
	}

	return reports, nil
}

// ListRuns returns all available run timestamps sorted chronologically
func (s *LocalStorage) ListRuns() ([]time.Time, error) {
	runsDir := filepath.Join(s.baseDir, "runs")

	// Check if directory exists
	if _, err := os.Stat(runsDir); os.IsNotExist(err) {
		return []time.Time{}, nil
	}

	// Read directory
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read runs directory: %w", err)
	}

	var timestamps []time.Time

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process aggregated report files
		if !strings.HasSuffix(entry.Name(), "-aggregated.json") {
			continue
		}

		// Parse timestamp from filename
		// Format: 2006-01-02T15-04-05-aggregated.json
		timestampStr := strings.TrimSuffix(entry.Name(), "-aggregated.json")
		timestamp, err := s.parseTimestamp(timestampStr)
		if err != nil {
			// Skip files with invalid timestamp format
			continue
		}

		timestamps = append(timestamps, timestamp)
	}

	// Sort chronologically
	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i].Before(timestamps[j])
	})

	return timestamps, nil
}

// loadReportFromFile loads a report from a file path
func (s *LocalStorage) loadReportFromFile(path string) (*models.AggregatedReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("report not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var report models.AggregatedReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal report: %w", err)
	}

	return &report, nil
}

// formatTimestamp converts a time.Time to filename-safe format
func (s *LocalStorage) formatTimestamp(t time.Time) string {
	return t.Format("2006-01-02T15-04-05")
}

// parseTimestamp converts filename format back to time.Time
func (s *LocalStorage) parseTimestamp(str string) (time.Time, error) {
	return time.Parse("2006-01-02T15-04-05", str)
}

// GetStoragePath returns the full path to the storage directory
func (s *LocalStorage) GetStoragePath() string {
	return s.baseDir
}

// EnsureDirectoryExists creates the storage directory if it doesn't exist
func (s *LocalStorage) EnsureDirectoryExists() error {
	return os.MkdirAll(filepath.Join(s.baseDir, "runs"), 0755)
}
