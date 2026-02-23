package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/ppiankov/spectrehub/internal/collector"
	"github.com/ppiankov/spectrehub/internal/discovery"
	"github.com/ppiankov/spectrehub/internal/runner"
	"github.com/spf13/cobra"
)

var (
	runFormat     string
	runOutput     string
	runStore      bool
	runStorageDir string
	runThreshold  int
	runTimeout    time.Duration
	runDryRun     bool
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Discover, execute, and aggregate spectre tools in one step",
	Long: `Run performs a full audit cycle:

  1. Discover — find installed tools and configured targets
  2. Execute  — run each tool with --format json, capture output
  3. Aggregate — feed outputs through the standard pipeline
  4. Report   — print results (text, json, or both)

Use --dry-run to see the discovery plan without executing anything.
Use --timeout to set per-tool execution timeout (default: 5m).`,
	RunE: runRun,
}

func init() {
	runCmd.Flags().StringVar(&runFormat, "format", "text",
		"output format: text, json, or both")
	runCmd.Flags().StringVarP(&runOutput, "output", "o", "",
		"write output to file")
	runCmd.Flags().BoolVar(&runStore, "store", false,
		"persist results for trend analysis")
	runCmd.Flags().StringVar(&runStorageDir, "storage-dir", "",
		"storage directory (default: .spectre)")
	runCmd.Flags().IntVar(&runThreshold, "fail-threshold", 0,
		"exit 1 if issues exceed threshold (0 = disabled)")
	runCmd.Flags().DurationVar(&runTimeout, "timeout", runner.DefaultTimeout,
		"per-tool execution timeout")
	runCmd.Flags().BoolVar(&runDryRun, "dry-run", false,
		"show discovery plan without executing tools")
}

func runRun(cmd *cobra.Command, args []string) error {
	// Step 1: Discover
	logVerbose("discovering spectre tools...")
	d := discovery.New(exec.LookPath, os.Getenv)
	plan := d.Discover()

	logVerbose("found %d tools, %d runnable", plan.TotalFound, plan.TotalRunnable)

	if plan.TotalRunnable == 0 {
		fmt.Fprintln(os.Stderr, "No runnable tools found. Run 'spectrehub discover' for details.")
		return nil
	}

	// Dry-run: show plan and exit
	if runDryRun {
		fmt.Printf("Dry run — would execute %d tool(s):\n\n", plan.TotalRunnable)
		for _, td := range plan.RunnableTools() {
			info := discovery.Registry[td.Tool]
			fmt.Printf("  %s %s %s\n", td.BinaryPath, info.Subcommand, info.JSONFlag)
		}
		return nil
	}

	// Step 2: Execute
	configs := runner.ConfigsFromDiscovery(plan, runTimeout)

	execFn := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		c := exec.CommandContext(ctx, name, args...)
		return c.Output()
	}

	r := runner.New(execFn)
	defer func() { _ = r.Cleanup() }()

	logVerbose("executing %d tool(s) with timeout %s...", len(configs), runTimeout)
	results := r.Run(context.Background(), configs)

	// Report execution results
	successCount := 0
	for _, res := range results {
		if res.Success {
			logVerbose("  ✓ %s (%s)", res.Binary, res.Duration)
			successCount++
		} else {
			logError("  ✗ %s: %s", res.Binary, res.Error)
		}
	}

	if successCount == 0 {
		return fmt.Errorf("all tools failed — nothing to aggregate")
	}

	logVerbose("%d/%d tools succeeded", successCount, len(results))

	// Step 3: Aggregate — collect output files through standard pipeline
	outputFiles := runner.OutputFiles(results)

	coll := collector.New(collector.Config{
		MaxConcurrency: len(outputFiles),
		Verbose:        verbose,
	})
	toolReports, err := coll.CollectFromPaths(outputFiles)
	if err != nil {
		return fmt.Errorf("failed to collect tool outputs: %w", err)
	}

	if len(toolReports) == 0 {
		return fmt.Errorf("no valid tool reports produced")
	}

	// Step 4: Report through shared pipeline
	return RunPipeline(toolReports, PipelineConfig{
		Format:     runFormat,
		Output:     runOutput,
		Store:      runStore,
		StorageDir: runStorageDir,
		Threshold:  runThreshold,
		LicenseKey: cfg.LicenseKey,
		APIURL:     cfg.APIURL,
	})
}
