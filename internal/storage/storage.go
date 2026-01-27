package storage

import (
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

// Storage defines the interface for persisting reports
type Storage interface {
	// SaveAggregatedReport stores a complete aggregated report
	SaveAggregatedReport(report *models.AggregatedReport) error

	// LoadAggregatedReport loads a report from a specific timestamp
	LoadAggregatedReport(timestamp time.Time) (*models.AggregatedReport, error)

	// GetLatestRun retrieves the most recent aggregated report
	GetLatestRun() (*models.AggregatedReport, error)

	// GetLastNRuns retrieves the last N aggregated reports
	GetLastNRuns(n int) ([]*models.AggregatedReport, error)

	// ListRuns returns all available run timestamps
	ListRuns() ([]time.Time, error)
}
