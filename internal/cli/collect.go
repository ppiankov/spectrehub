package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ppiankov/spectrehub/internal/aggregator"
	"github.com/ppiankov/spectrehub/internal/collector"
	"github.com/ppiankov/spectrehub/internal/models"
	"github.com/ppiankov/spectrehub/internal/reporter"
	"github.com/ppiankov/spectrehub/internal/storage"
	"github.com/spf13/cobra"
)

var (
	// Collect command flags
	collectFormat     string
	collectOutput     string
	collectStore      bool
	collectStorageDir string
	collectThreshold  int
)

// collectCmd represents the collect command
var collectCmd = &cobra.Command{
	Use:   "collect <path>...",
	Short: "Collect and aggregate reports from Spectre tools",
	Long: `Collect JSON reports from VaultSpectre, S3Spectre, KafkaSpectre, ClickSpectre, PgSpectre, and MongoSpectre,
aggregate them into a unified view, and generate reports.

The command will:
1. Scan the path(s) for JSON report files
2. Detect and parse each tool's output format
3. Normalize issues into a common schema
4. Calculate health scores and trends
5. Generate actionable recommendations
6. Output results in the specified format

Example:
  spectrehub collect ./reports
  spectrehub collect ./reports/*.json
  spectrehub collect ./reports --format json --output summary.json
  spectrehub collect ./reports --fail-threshold 50 --store`,
	Args: cobra.MinimumNArgs(1),
	RunE: runCollect,
}

func init() {
	collectCmd.Flags().StringVarP(&collectFormat, "format", "f", "",
		"output format: text, json, or both (default from config)")
	collectCmd.Flags().StringVarP(&collectOutput, "output", "o", "",
		"output file path (default: stdout)")
	collectCmd.Flags().BoolVar(&collectStore, "store", true,
		"store aggregated report for trend analysis")
	collectCmd.Flags().StringVar(&collectStorageDir, "storage-dir", "",
		"storage directory (default from config)")
	collectCmd.Flags().IntVar(&collectThreshold, "fail-threshold", -1,
		"exit with code 1 if issues exceed this threshold (default from config)")
}

func runCollect(cmd *cobra.Command, args []string) error {
	reportPaths := args

	// Apply config defaults if flags not set
	if collectFormat == "" {
		collectFormat = cfg.Format
	}
	if collectStorageDir == "" {
		collectStorageDir = cfg.StorageDir
	}
	if collectThreshold == -1 {
		collectThreshold = cfg.FailThreshold
	}

	logVerbose("Collecting reports from: %s", strings.Join(reportPaths, ", "))
	logDebug("Config: format=%s, store=%v, threshold=%d", collectFormat, collectStore, collectThreshold)

	// Step 1: Collect tool reports
	c := collector.New(collector.Config{
		MaxConcurrency: 10,
		Verbose:        cfg.Verbose,
	})

	toolReports, err := c.CollectFromPaths(reportPaths)
	if err != nil {
		logError("Failed to collect reports: %v", err)
		return err
	}

	if len(toolReports) == 0 {
		logError("No valid reports found in provided paths")
		return &ValidationError{Message: "no valid reports found"}
	}

	logVerbose("Collected %d tool reports", len(toolReports))

	// Step 2: Aggregate reports
	agg := aggregator.New()
	aggregatedReport, err := agg.Aggregate(toolReports)
	if err != nil {
		logError("Failed to aggregate reports: %v", err)
		return err
	}

	logVerbose("Aggregated %d issues across %d tools", aggregatedReport.Summary.TotalIssues, aggregatedReport.Summary.TotalTools)

	// Step 3: Add trend analysis if storage is enabled and previous runs exist
	if collectStore {
		storagePath, err := getStoragePath(collectStorageDir)
		if err != nil {
			logError("Failed to get storage path: %v", err)
			return err
		}

		store := storage.NewLocal(storagePath)

		// Try to load previous run for trend comparison
		if previousReport, err := store.GetLatestRun(); err == nil {
			logVerbose("Found previous run from %s", previousReport.Timestamp)
			agg.AddTrend(aggregatedReport, previousReport)
		} else {
			logDebug("No previous run found: %v", err)
		}
	}

	// Step 4: Generate recommendations
	recGen := aggregator.NewRecommendationGenerator()
	aggregatedReport.Recommendations = recGen.GenerateRecommendations(aggregatedReport)

	logVerbose("Generated %d recommendations", len(aggregatedReport.Recommendations))

	// Step 5: Store if enabled
	if collectStore {
		storagePath, err := getStoragePath(collectStorageDir)
		if err != nil {
			logError("Failed to get storage path: %v", err)
			return err
		}

		store := storage.NewLocal(storagePath)

		// Ensure directory exists
		if err := store.EnsureDirectoryExists(); err != nil {
			logError("Failed to create storage directory: %v", err)
			return err
		}

		if err := store.SaveAggregatedReport(aggregatedReport); err != nil {
			logError("Failed to store report: %v", err)
			return err
		}

		logVerbose("Stored report in: %s", storagePath)
	}

	// Step 6: Generate output
	if err := generateOutput(aggregatedReport, collectFormat, collectOutput); err != nil {
		logError("Failed to generate output: %v", err)
		return err
	}

	// Step 7: Check threshold
	if collectThreshold > 0 && aggregatedReport.Summary.TotalIssues > collectThreshold {
		logError("Issue count (%d) exceeds threshold (%d)", aggregatedReport.Summary.TotalIssues, collectThreshold)
		return &ThresholdExceededError{
			IssueCount: aggregatedReport.Summary.TotalIssues,
			Threshold:  collectThreshold,
		}
	}

	return nil
}

// generateOutput generates the output in the specified format(s)
func generateOutput(report *models.AggregatedReport, format, outputPath string) error {
	// Determine output writer
	var writer *os.File
	if outputPath == "" {
		writer = os.Stdout
	} else {
		var err error
		writer, err = os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() { _ = writer.Close() }()
	}

	switch format {
	case "text":
		textReporter := reporter.NewTextReporter(writer)
		return textReporter.Generate(report)

	case "json":
		jsonReporter := reporter.NewJSONReporter(writer, true) // pretty print
		return jsonReporter.Generate(report)

	case "both":
		// If no output file specified, text goes to stdout, json to file
		if outputPath == "" {
			// Text to stdout
			textReporter := reporter.NewTextReporter(os.Stdout)
			if err := textReporter.Generate(report); err != nil {
				return err
			}

			// JSON to default file
			jsonFile, err := os.Create("spectrehub-report.json")
			if err != nil {
				return fmt.Errorf("failed to create JSON file: %w", err)
			}
			defer func() { _ = jsonFile.Close() }()

			jsonReporter := reporter.NewJSONReporter(jsonFile, true)
			return jsonReporter.Generate(report)
		} else {
			// Both to the same file (text first, then JSON)
			textReporter := reporter.NewTextReporter(writer)
			if err := textReporter.Generate(report); err != nil {
				return err
			}

			if _, err := fmt.Fprintf(writer, "\n=== JSON Output ===\n\n"); err != nil {
				return err
			}

			jsonReporter := reporter.NewJSONReporter(writer, true)
			return jsonReporter.Generate(report)
		}

	default:
		return fmt.Errorf("unsupported format: %s (use text, json, or both)", format)
	}
}

// getStoragePath resolves the storage path
func getStoragePath(storageDir string) (string, error) {
	// Expand ~ to home directory
	if len(storageDir) >= 2 && storageDir[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		storageDir = filepath.Join(home, storageDir[2:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(storageDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return absPath, nil
}
