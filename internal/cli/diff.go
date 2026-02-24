package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ppiankov/spectrehub/internal/models"
	"github.com/ppiankov/spectrehub/internal/storage"
	"github.com/spf13/cobra"
)

var (
	diffFormat   string
	diffOutput   string
	diffBaseline string
	diffFailNew  bool
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show what changed between two audit runs",
	Long: `Compare the latest audit run against a baseline to show drift.

Shows new issues, resolved issues, and summary deltas between two runs.
Useful in CI/CD to catch regressions introduced by a pull request.

By default compares the two most recent stored runs. Use --baseline to
specify a report file as the comparison target.

Exit codes:
  0  No new issues (or --fail-new not set)
  1  New issues detected (with --fail-new)

Example:
  spectrehub diff
  spectrehub diff --fail-new
  spectrehub diff --baseline ./baseline.json --format json`,
	RunE: runDiff,
}

func init() {
	diffCmd.Flags().StringVarP(&diffFormat, "format", "f", "text",
		"output format: text or json")
	diffCmd.Flags().StringVarP(&diffOutput, "output", "o", "",
		"write output to file instead of stdout")
	diffCmd.Flags().StringVar(&diffBaseline, "baseline", "",
		"path to baseline report JSON (default: previous stored run)")
	diffCmd.Flags().BoolVar(&diffFailNew, "fail-new", false,
		"exit 1 if new issues are found (for CI gating)")
}

// DiffResult is the structured output of a diff operation.
type DiffResult struct {
	Baseline       string                   `json:"baseline"`
	Current        string                   `json:"current"`
	NewIssues      []models.NormalizedIssue `json:"new_issues"`
	ResolvedIssues []models.NormalizedIssue `json:"resolved_issues"`
	Summary        DiffSummary              `json:"summary"`
}

// DiffSummary holds aggregate counts for a diff.
type DiffSummary struct {
	BaselineTotal int            `json:"baseline_total"`
	CurrentTotal  int            `json:"current_total"`
	NewCount      int            `json:"new_count"`
	ResolvedCount int            `json:"resolved_count"`
	Delta         int            `json:"delta"` // positive = more issues
	NewBySeverity map[string]int `json:"new_by_severity"`
	NewByTool     map[string]int `json:"new_by_tool"`
	NewByCategory map[string]int `json:"new_by_category"`
}

func runDiff(cmd *cobra.Command, args []string) error {
	storagePath, err := getStoragePath(cfg.StorageDir)
	if err != nil {
		logError("Failed to get storage path: %v", err)
		return err
	}

	store := storage.NewLocal(storagePath)

	// Load current (latest) run.
	current, err := store.GetLatestRun()
	if err != nil {
		logError("No current run found: %v", err)
		fmt.Println("No stored runs found. Run 'spectrehub run --store' first.")
		return err
	}

	// Load baseline.
	var baseline *models.AggregatedReport
	if diffBaseline != "" {
		baseline, err = loadReportFromFile(diffBaseline)
		if err != nil {
			logError("Failed to load baseline: %v", err)
			return err
		}
	} else {
		reports, err := store.GetLastNRuns(2)
		if err != nil || len(reports) < 2 {
			fmt.Println("Need at least 2 stored runs for diff.")
			fmt.Println("Run 'spectrehub run --store' to generate more reports.")
			return nil
		}
		baseline = reports[0]
		// current may differ from reports[1] if new run was added; use latest.
	}

	logVerbose("Comparing %s (current) vs %s (baseline)",
		current.Timestamp.Format("2006-01-02 15:04"),
		baseline.Timestamp.Format("2006-01-02 15:04"))

	result := computeDiff(baseline, current)

	// Output.
	if err := outputDiff(result, diffFormat, diffOutput); err != nil {
		return err
	}

	// CI gate.
	if diffFailNew && result.Summary.NewCount > 0 {
		return &ThresholdExceededError{
			IssueCount: result.Summary.NewCount,
			Threshold:  0,
		}
	}

	return nil
}

// issueKey returns a string that uniquely identifies an issue for diff purposes.
func issueKey(issue models.NormalizedIssue) string {
	return issue.Tool + "|" + issue.Category + "|" + issue.Resource
}

// computeDiff calculates new and resolved issues between baseline and current.
func computeDiff(baseline, current *models.AggregatedReport) *DiffResult {
	baseSet := make(map[string]models.NormalizedIssue, len(baseline.Issues))
	for _, issue := range baseline.Issues {
		baseSet[issueKey(issue)] = issue
	}

	currSet := make(map[string]models.NormalizedIssue, len(current.Issues))
	for _, issue := range current.Issues {
		currSet[issueKey(issue)] = issue
	}

	var newIssues, resolvedIssues []models.NormalizedIssue

	for key, issue := range currSet {
		if _, found := baseSet[key]; !found {
			newIssues = append(newIssues, issue)
		}
	}

	for key, issue := range baseSet {
		if _, found := currSet[key]; !found {
			resolvedIssues = append(resolvedIssues, issue)
		}
	}

	// Build summary maps.
	newBySeverity := map[string]int{}
	newByTool := map[string]int{}
	newByCategory := map[string]int{}
	for _, issue := range newIssues {
		newBySeverity[issue.Severity]++
		newByTool[issue.Tool]++
		newByCategory[issue.Category]++
	}

	return &DiffResult{
		Baseline:       baseline.Timestamp.Format("2006-01-02 15:04:05"),
		Current:        current.Timestamp.Format("2006-01-02 15:04:05"),
		NewIssues:      newIssues,
		ResolvedIssues: resolvedIssues,
		Summary: DiffSummary{
			BaselineTotal: len(baseline.Issues),
			CurrentTotal:  len(current.Issues),
			NewCount:      len(newIssues),
			ResolvedCount: len(resolvedIssues),
			Delta:         len(current.Issues) - len(baseline.Issues),
			NewBySeverity: newBySeverity,
			NewByTool:     newByTool,
			NewByCategory: newByCategory,
		},
	}
}

// outputDiff renders the diff result to the chosen format.
func outputDiff(result *DiffResult, format, outputPath string) error {
	var writer *os.File
	if outputPath != "" {
		var err error
		writer, err = os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() { _ = writer.Close() }()
	} else {
		writer = os.Stdout
	}

	switch format {
	case "json":
		enc := json.NewEncoder(writer)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	case "text":
		return printDiffText(writer, result)
	default:
		return fmt.Errorf("unsupported format: %s (use text or json)", format)
	}
}

func printDiffText(w *os.File, r *DiffResult) error {
	p := func(format string, args ...interface{}) {
		_, _ = fmt.Fprintf(w, format, args...)
	}

	p("╔════════════════════════════════════════════╗\n")
	p("║         SpectreHub Drift Delta            ║\n")
	p("╚════════════════════════════════════════════╝\n\n")

	p("Baseline: %s\n", r.Baseline)
	p("Current:  %s\n\n", r.Current)

	// Summary line.
	deltaSign := "+"
	if r.Summary.Delta < 0 {
		deltaSign = ""
	}
	p("Issues: %d → %d (%s%d)\n", r.Summary.BaselineTotal, r.Summary.CurrentTotal, deltaSign, r.Summary.Delta)
	p("New: %d   Resolved: %d\n\n", r.Summary.NewCount, r.Summary.ResolvedCount)

	// New issues.
	if len(r.NewIssues) > 0 {
		p("New Issues:\n")
		p("--------------------------------------------------\n")
		for _, issue := range r.NewIssues {
			sev := strings.ToUpper(issue.Severity)
			p("  [%s] %s — %s: %s\n", sev, issue.Tool, issue.Category, issue.Resource)
			if issue.Evidence != "" {
				p("         %s\n", issue.Evidence)
			}
		}
		p("\n")
	}

	// Resolved issues.
	if len(r.ResolvedIssues) > 0 {
		p("Resolved Issues:\n")
		p("--------------------------------------------------\n")
		for _, issue := range r.ResolvedIssues {
			p("  ✓ %s — %s: %s\n", issue.Tool, issue.Category, issue.Resource)
		}
		p("\n")
	}

	// Breakdown tables.
	if len(r.Summary.NewBySeverity) > 0 {
		p("New by Severity:\n")
		for sev, count := range r.Summary.NewBySeverity {
			p("  %s: %d\n", strings.ToUpper(sev), count)
		}
		p("\n")
	}

	if len(r.Summary.NewByTool) > 0 {
		p("New by Tool:\n")
		for tool, count := range r.Summary.NewByTool {
			p("  %s: %d\n", tool, count)
		}
		p("\n")
	}

	if r.Summary.NewCount == 0 && r.Summary.ResolvedCount == 0 {
		p("No drift detected.\n")
	} else if r.Summary.NewCount == 0 {
		p("No new issues — only improvements.\n")
	}

	return nil
}

// loadReportFromFile loads an AggregatedReport from a JSON file path.
func loadReportFromFile(path string) (*models.AggregatedReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var report models.AggregatedReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to parse report: %w", err)
	}

	return &report, nil
}
