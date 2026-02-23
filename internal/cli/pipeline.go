package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ppiankov/spectrehub/internal/aggregator"
	"github.com/ppiankov/spectrehub/internal/models"
	"github.com/ppiankov/spectrehub/internal/reporter"
	"github.com/ppiankov/spectrehub/internal/storage"
)

// PipelineConfig holds options for the shared aggregation pipeline.
type PipelineConfig struct {
	Format     string
	Output     string
	Store      bool
	StorageDir string
	Threshold  int
}

// RunPipeline executes the aggregation pipeline on a set of tool reports.
// This is the shared logic between collect and run commands:
// aggregate → trend → recommendations → store → output → threshold check.
func RunPipeline(toolReports []models.ToolReport, pcfg PipelineConfig) error {
	// Step 1: Aggregate reports
	agg := aggregator.New()
	aggregatedReport, err := agg.Aggregate(toolReports)
	if err != nil {
		logError("Failed to aggregate reports: %v", err)
		return err
	}

	logVerbose("Aggregated %d issues across %d tools", aggregatedReport.Summary.TotalIssues, aggregatedReport.Summary.TotalTools)

	// Step 2: Add trend analysis if storage is enabled and previous runs exist
	if pcfg.Store {
		storagePath, err := getStoragePath(pcfg.StorageDir)
		if err != nil {
			logError("Failed to get storage path: %v", err)
			return err
		}

		store := storage.NewLocal(storagePath)

		if previousReport, err := store.GetLatestRun(); err == nil {
			logVerbose("Found previous run from %s", previousReport.Timestamp)
			agg.AddTrend(aggregatedReport, previousReport)
		} else {
			logDebug("No previous run found: %v", err)
		}
	}

	// Step 3: Generate recommendations
	recGen := aggregator.NewRecommendationGenerator()
	aggregatedReport.Recommendations = recGen.GenerateRecommendations(aggregatedReport)

	logVerbose("Generated %d recommendations", len(aggregatedReport.Recommendations))

	// Step 4: Store if enabled
	if pcfg.Store {
		storagePath, err := getStoragePath(pcfg.StorageDir)
		if err != nil {
			logError("Failed to get storage path: %v", err)
			return err
		}

		store := storage.NewLocal(storagePath)

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

	// Step 5: Generate output
	if err := generateOutput(aggregatedReport, pcfg.Format, pcfg.Output); err != nil {
		logError("Failed to generate output: %v", err)
		return err
	}

	// Step 6: Check threshold
	if pcfg.Threshold > 0 && aggregatedReport.Summary.TotalIssues > pcfg.Threshold {
		logError("Issue count (%d) exceeds threshold (%d)", aggregatedReport.Summary.TotalIssues, pcfg.Threshold)
		return &ThresholdExceededError{
			IssueCount: aggregatedReport.Summary.TotalIssues,
			Threshold:  pcfg.Threshold,
		}
	}

	return nil
}

// generateOutput generates the output in the specified format(s).
func generateOutput(report *models.AggregatedReport, format, outputPath string) error {
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
		jsonReporter := reporter.NewJSONReporter(writer, true)
		return jsonReporter.Generate(report)

	case "both":
		if outputPath == "" {
			textReporter := reporter.NewTextReporter(os.Stdout)
			if err := textReporter.Generate(report); err != nil {
				return err
			}

			jsonFile, err := os.Create("spectrehub-report.json")
			if err != nil {
				return fmt.Errorf("failed to create JSON file: %w", err)
			}
			defer func() { _ = jsonFile.Close() }()

			jsonReporter := reporter.NewJSONReporter(jsonFile, true)
			return jsonReporter.Generate(report)
		}

		textReporter := reporter.NewTextReporter(writer)
		if err := textReporter.Generate(report); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(writer, "\n=== JSON Output ===\n\n"); err != nil {
			return err
		}

		jsonReporter := reporter.NewJSONReporter(writer, true)
		return jsonReporter.Generate(report)

	default:
		return fmt.Errorf("unsupported format: %s (use text, json, or both)", format)
	}
}

// getStoragePath resolves the storage path, expanding ~ and converting to absolute.
func getStoragePath(storageDir string) (string, error) {
	if len(storageDir) >= 2 && storageDir[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		storageDir = filepath.Join(home, storageDir[2:])
	}

	absPath, err := filepath.Abs(storageDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return absPath, nil
}
