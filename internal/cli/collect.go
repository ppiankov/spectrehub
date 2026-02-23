package cli

import (
	"strings"

	"github.com/ppiankov/spectrehub/internal/collector"
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

	// Step 1: Collect tool reports from files
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

	// Steps 2-7: Aggregate, trend, recommend, store, output, threshold
	return RunPipeline(toolReports, PipelineConfig{
		Format:     collectFormat,
		Output:     collectOutput,
		Store:      collectStore,
		StorageDir: collectStorageDir,
		Threshold:  collectThreshold,
		LicenseKey: cfg.LicenseKey,
		APIURL:     cfg.APIURL,
	})
}
