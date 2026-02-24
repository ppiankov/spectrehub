package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
	"github.com/ppiankov/spectrehub/internal/storage"
	"github.com/spf13/cobra"
)

var (
	exportFormat string
	exportOutput string
	exportLastN  int
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export audit data for compliance reporting",
	Long: `Export audit history in formats suitable for SOC2, ISO 27001, and other
compliance frameworks. Generates evidence packages from stored runs.

Supported formats:
  csv    Tabular format for spreadsheets and compliance tools
  json   Structured JSON for programmatic consumption
  sarif  SARIF 2.1.0 for GitHub Advanced Security and code scanning

Example:
  spectrehub export --format csv -o audit-evidence.csv
  spectrehub export --format sarif -o results.sarif --last 1
  spectrehub export --format json --last 30 -o evidence.json`,
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "csv",
		"output format: csv, json, or sarif")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "",
		"write output to file (default: stdout)")
	exportCmd.Flags().IntVarP(&exportLastN, "last", "n", 1,
		"number of recent runs to include")
}

// ComplianceRecord is a single row in the compliance export.
type ComplianceRecord struct {
	RunTimestamp string `json:"run_timestamp"`
	Tool         string `json:"tool"`
	Category     string `json:"category"`
	Severity     string `json:"severity"`
	Resource     string `json:"resource"`
	Evidence     string `json:"evidence"`
	Status       string `json:"status"` // "open" or "resolved"
	HealthScore  string `json:"health_score"`
	ScorePercent string `json:"score_percent"`
}

// ComplianceExport is the full export payload.
type ComplianceExport struct {
	ExportedAt string             `json:"exported_at"`
	RunCount   int                `json:"run_count"`
	IssueCount int                `json:"issue_count"`
	Framework  string             `json:"framework"`
	Records    []ComplianceRecord `json:"records"`
}

func runExport(cmd *cobra.Command, args []string) error {
	storagePath, err := getStoragePath(cfg.StorageDir)
	if err != nil {
		logError("Failed to get storage path: %v", err)
		return err
	}

	store := storage.NewLocal(storagePath)

	reports, err := store.GetLastNRuns(exportLastN)
	if err != nil || len(reports) == 0 {
		fmt.Println("No stored runs found. Run 'spectrehub run --store' first.")
		return nil
	}

	logVerbose("Exporting %d runs", len(reports))

	export := buildComplianceExport(reports)

	var writer *os.File
	if exportOutput != "" {
		writer, err = os.Create(exportOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() { _ = writer.Close() }()
	} else {
		writer = os.Stdout
	}

	switch exportFormat {
	case "csv":
		return writeCSV(writer, export)
	case "json":
		return writeExportJSON(writer, export)
	case "sarif":
		return writeSARIF(writer, reports)
	default:
		return fmt.Errorf("unsupported format: %s (use csv, json, or sarif)", exportFormat)
	}
}

func buildComplianceExport(reports []*models.AggregatedReport) *ComplianceExport {
	var records []ComplianceRecord

	for _, report := range reports {
		ts := report.Timestamp.Format(time.RFC3339)
		health := report.Summary.HealthScore
		score := fmt.Sprintf("%.1f", report.Summary.ScorePercent)

		for _, issue := range report.Issues {
			records = append(records, ComplianceRecord{
				RunTimestamp: ts,
				Tool:         issue.Tool,
				Category:     issue.Category,
				Severity:     issue.Severity,
				Resource:     issue.Resource,
				Evidence:     issue.Evidence,
				Status:       "open",
				HealthScore:  health,
				ScorePercent: score,
			})
		}
	}

	// Sort by severity (critical first), then tool, then resource.
	sevOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
	sort.Slice(records, func(i, j int) bool {
		si, sj := sevOrder[records[i].Severity], sevOrder[records[j].Severity]
		if si != sj {
			return si < sj
		}
		if records[i].Tool != records[j].Tool {
			return records[i].Tool < records[j].Tool
		}
		return records[i].Resource < records[j].Resource
	})

	return &ComplianceExport{
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		RunCount:   len(reports),
		IssueCount: len(records),
		Framework:  "SOC2/ISO27001",
		Records:    records,
	}
}

func writeCSV(w *os.File, export *ComplianceExport) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	header := []string{
		"run_timestamp", "tool", "category", "severity",
		"resource", "evidence", "status", "health_score", "score_percent",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, r := range export.Records {
		row := []string{
			r.RunTimestamp, r.Tool, r.Category, r.Severity,
			r.Resource, r.Evidence, r.Status, r.HealthScore, r.ScorePercent,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func writeExportJSON(w *os.File, export *ComplianceExport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(export)
}

// SARIF 2.1.0 output for GitHub Advanced Security integration.
// Minimal structures — only what's needed for valid SARIF.

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name    string      `json:"name"`
	Version string      `json:"version"`
	Rules   []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string             `json:"id"`
	ShortDescription sarifMessage       `json:"shortDescription"`
	DefaultConfig    sarifDefaultConfig `json:"defaultConfiguration"`
}

type sarifDefaultConfig struct {
	Level string `json:"level"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysical `json:"physicalLocation"`
}

type sarifPhysical struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
}

type sarifArtifact struct {
	URI string `json:"uri"`
}

func writeSARIF(w *os.File, reports []*models.AggregatedReport) error {
	rulesMap := map[string]sarifRule{}
	var results []sarifResult

	for _, report := range reports {
		for _, issue := range report.Issues {
			ruleID := issue.Tool + "/" + issue.Category
			if _, exists := rulesMap[ruleID]; !exists {
				rulesMap[ruleID] = sarifRule{
					ID:               ruleID,
					ShortDescription: sarifMessage{Text: issue.Tool + " " + issue.Category},
					DefaultConfig:    sarifDefaultConfig{Level: sarifLevel(issue.Severity)},
				}
			}

			results = append(results, sarifResult{
				RuleID:  ruleID,
				Level:   sarifLevel(issue.Severity),
				Message: sarifMessage{Text: formatEvidence(issue)},
				Locations: []sarifLocation{{
					PhysicalLocation: sarifPhysical{
						ArtifactLocation: sarifArtifact{URI: issue.Resource},
					},
				}},
			})
		}
	}

	var rules []sarifRule
	for _, r := range rulesMap {
		rules = append(rules, r)
	}
	sort.Slice(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })

	log := sarifLog{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{
				Driver: sarifDriver{
					Name:    "spectrehub",
					Version: "0.2.0",
					Rules:   rules,
				},
			},
			Results: results,
		}},
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(log)
}

func sarifLevel(severity string) string {
	switch severity {
	case "critical", "high":
		return "error"
	case "medium":
		return "warning"
	default:
		return "note"
	}
}

func formatEvidence(issue models.NormalizedIssue) string {
	parts := []string{issue.Tool + ": " + issue.Category + " — " + issue.Resource}
	if issue.Evidence != "" {
		parts = append(parts, issue.Evidence)
	}
	return strings.Join(parts, ". ")
}
