package cli

import (
	"fmt"
	"os"

	"github.com/ppiankov/spectrehub/internal/aggregator"
	"github.com/ppiankov/spectrehub/internal/models"
	"github.com/ppiankov/spectrehub/internal/reporter"
	"github.com/ppiankov/spectrehub/internal/storage"
	"github.com/spf13/cobra"
)

var (
	// Summarize command flags
	summarizeLastN  int
	summarizeCompare bool
	summarizeFormat string
)

// summarizeCmd represents the summarize command
var summarizeCmd = &cobra.Command{
	Use:   "summarize",
	Short: "Show summary and trends from stored runs",
	Long: `Analyze historical data from stored runs and show trends over time.

This command displays:
- Latest run summary
- Trend analysis across last N runs
- Issue sparklines showing changes over time
- Per-tool trend comparison
- Improvement/degradation indicators

Example:
  spectrehub summarize
  spectrehub summarize --last 7
  spectrehub summarize --compare --format json`,
	RunE: runSummarize,
}

func init() {
	summarizeCmd.Flags().IntVarP(&summarizeLastN, "last", "n", 0,
		"number of runs to analyze (default from config)")
	summarizeCmd.Flags().BoolVarP(&summarizeCompare, "compare", "c", false,
		"compare latest run with previous")
	summarizeCmd.Flags().StringVarP(&summarizeFormat, "format", "f", "text",
		"output format: text or json")
}

func runSummarize(cmd *cobra.Command, args []string) error {
	// Apply config defaults if flags not set
	if summarizeLastN == 0 {
		summarizeLastN = cfg.LastRuns
	}

	storagePath, err := getStoragePath(cfg.StorageDir)
	if err != nil {
		logError("Failed to get storage path: %v", err)
		return err
	}

	store := storage.NewLocal(storagePath)

	logVerbose("Loading runs from: %s", storagePath)

	// Check if storage exists
	runs, err := store.ListRuns()
	if err != nil {
		logError("Failed to list runs: %v", err)
		return err
	}

	if len(runs) == 0 {
		fmt.Println("No stored runs found.")
		fmt.Printf("Run 'spectrehub collect <directory>' to generate your first report.\n")
		return nil
	}

	logVerbose("Found %d stored runs", len(runs))

	if summarizeCompare {
		// Compare mode: show comparison between latest and previous
		return runComparisonReport(store)
	} else {
		// Trend mode: show trends across last N runs
		return runTrendReport(store, summarizeLastN)
	}
}

// runComparisonReport generates a comparison report between latest and previous runs
func runComparisonReport(store *storage.LocalStorage) error {
	// Load last 2 runs
	reports, err := store.GetLastNRuns(2)
	if err != nil {
		logError("Failed to load runs: %v", err)
		return err
	}

	if len(reports) < 2 {
		fmt.Println("Need at least 2 runs for comparison.")
		fmt.Printf("Run 'spectrehub collect <directory>' to generate more reports.\n")
		return nil
	}

	previous := reports[0]
	current := reports[1]

	logVerbose("Comparing %s vs %s", current.Timestamp, previous.Timestamp)

	// Generate comparison using TrendAnalyzer
	analyzer := aggregator.NewTrendAnalyzer()
	comparisonText := analyzer.GenerateComparisonReport(current, previous)

	fmt.Print(comparisonText)

	return nil
}

// runTrendReport generates a trend report across last N runs
func runTrendReport(store *storage.LocalStorage, lastN int) error {
	// Load last N runs
	reports, err := store.GetLastNRuns(lastN)
	if err != nil {
		logError("Failed to load runs: %v", err)
		return err
	}

	if len(reports) == 0 {
		fmt.Println("No runs found.")
		return nil
	}

	logVerbose("Analyzing trends across %d runs", len(reports))

	// Generate trend summary
	analyzer := aggregator.NewTrendAnalyzer()
	trendSummary := analyzer.AnalyzeLastNRuns(reports)

	if trendSummary == nil {
		fmt.Println("Unable to generate trend summary.")
		return nil
	}

	// Output based on format
	switch summarizeFormat {
	case "text":
		printTrendSummaryText(trendSummary, reports)
	case "json":
		jsonReporter := reporter.NewJSONReporter(os.Stdout, true)
		return jsonReporter.Generate(reports[len(reports)-1]) // Latest report with trend
	default:
		return fmt.Errorf("unsupported format: %s", summarizeFormat)
	}

	return nil
}

// printTrendSummaryText prints trend summary in human-readable format
func printTrendSummaryText(summary *models.TrendSummary, reports []*models.AggregatedReport) {
	fmt.Println("╔════════════════════════════════════════════╗")
	fmt.Println("║       SpectreHub Trend Summary            ║")
	fmt.Println("╚════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Printf("Time Range: %s\n", summary.TimeRange)
	fmt.Printf("Runs Analyzed: %d\n", summary.RunsAnalyzed)
	fmt.Println()

	// Latest run info
	latestReport := reports[len(reports)-1]
	fmt.Printf("Latest Run: %s\n", latestReport.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("Total Issues: %d\n", latestReport.Summary.TotalIssues)
	fmt.Printf("Health Score: %s", latestReport.Summary.HealthScore)

	// Show trend if we have comparison data
	if len(reports) >= 2 {
		previous := reports[len(reports)-2]
		current := latestReport

		change := current.Summary.TotalIssues - previous.Summary.TotalIssues
		var direction string
		var indicator string

		if change < 0 {
			direction = "improved"
			indicator = "↓"
		} else if change > 0 {
			direction = "degraded"
			indicator = "↑"
		} else {
			direction = "stable"
			indicator = "→"
		}

		changePercent := 0.0
		if previous.Summary.TotalIssues > 0 {
			changePercent = float64(change) / float64(previous.Summary.TotalIssues) * 100.0
		}

		fmt.Printf(" (%s %s %.1f%%)\n", indicator, direction, changePercent)
	} else {
		fmt.Println()
	}

	fmt.Println()

	// Issue sparkline
	if len(summary.IssueSparkline) > 0 {
		fmt.Println("Issue Trend (over time):")
		fmt.Print("  ")
		printSparkline(summary.IssueSparkline)
		fmt.Println()
	}

	// Per-tool trends
	if len(summary.ByTool) > 0 {
		fmt.Println()
		fmt.Println("By Tool:")
		fmt.Println("--------------------------------------------------")

		for toolName, toolTrend := range summary.ByTool {
			indicator := "→"
			if toolTrend.Change < 0 {
				indicator = "↓"
			} else if toolTrend.Change > 0 {
				indicator = "↑"
			}

			fmt.Printf("  %s: %d issues (%s %+d, %.1f%%)\n",
				toolName,
				toolTrend.CurrentIssues,
				indicator,
				toolTrend.Change,
				toolTrend.ChangePercent)
		}
	}

	// Recommendations from latest run
	if len(latestReport.Recommendations) > 0 {
		fmt.Println()
		fmt.Println("Top Recommendations:")
		fmt.Println("--------------------------------------------------")

		// Show top 5
		recGen := aggregator.NewRecommendationGenerator()
		topRecs := recGen.GetTopRecommendations(latestReport.Recommendations, 5)

		for i, rec := range topRecs {
			fmt.Printf("  %d. [%s] %s\n", i+1, rec.Severity, rec.Action)
		}
	}

	fmt.Println()
	fmt.Println("Run 'spectrehub collect' to update data")
}

// printSparkline prints a simple ASCII sparkline
func printSparkline(values []int) {
	if len(values) == 0 {
		return
	}

	// Find min and max
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Normalize and print
	chars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	for _, v := range values {
		if max == min {
			fmt.Print(string(chars[len(chars)/2]))
		} else {
			normalized := float64(v-min) / float64(max-min)
			idx := int(normalized * float64(len(chars)-1))
			fmt.Print(string(chars[idx]))
		}
	}

	fmt.Printf(" [%d → %d]\n", values[0], values[len(values)-1])
}
