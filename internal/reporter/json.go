package reporter

import (
	"encoding/json"
	"io"

	"github.com/ppiankov/spectrehub/internal/models"
)

// JSONReporter generates machine-readable JSON reports
type JSONReporter struct {
	writer io.Writer
	pretty bool
}

// NewJSONReporter creates a new JSON reporter
func NewJSONReporter(writer io.Writer, pretty bool) *JSONReporter {
	return &JSONReporter{
		writer: writer,
		pretty: pretty,
	}
}

// Generate creates a JSON report from the aggregated data
func (r *JSONReporter) Generate(report *models.AggregatedReport) error {
	var data []byte
	var err error

	if r.pretty {
		data, err = json.MarshalIndent(report, "", "  ")
	} else {
		data, err = json.Marshal(report)
	}

	if err != nil {
		return err
	}

	_, err = r.writer.Write(data)
	if err != nil {
		return err
	}

	// Add trailing newline for terminal output
	_, err = r.writer.Write([]byte("\n"))
	return err
}

// GenerateSummaryOnly creates a compact JSON summary without raw tool data
func (r *JSONReporter) GenerateSummaryOnly(report *models.AggregatedReport) error {
	// Create a summary-only structure
	summary := struct {
		Timestamp       string                    `json:"timestamp"`
		Summary         models.CrossToolSummary   `json:"summary"`
		Trend           *models.Trend             `json:"trend,omitempty"`
		Recommendations []models.Recommendation   `json:"recommendations"`
		IssuesByTool    map[string]int            `json:"issues_by_tool"`
	}{
		Timestamp:       report.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		Summary:         report.Summary,
		Trend:           report.Trend,
		Recommendations: report.Recommendations,
		IssuesByTool:    report.Summary.IssuesByTool,
	}

	var data []byte
	var err error

	if r.pretty {
		data, err = json.MarshalIndent(summary, "", "  ")
	} else {
		data, err = json.Marshal(summary)
	}

	if err != nil {
		return err
	}

	_, err = r.writer.Write(data)
	if err != nil {
		return err
	}

	// Add trailing newline
	_, err = r.writer.Write([]byte("\n"))
	return err
}
