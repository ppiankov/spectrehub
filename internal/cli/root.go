package cli

import (
	"fmt"
	"os"

	"github.com/ppiankov/spectrehub/internal/config"
	"github.com/spf13/cobra"
)

const (
	// Exit codes as specified in the plan
	ExitOK           = 0 // Success
	ExitPolicyFail   = 1 // Issues exceed threshold
	ExitInvalidInput = 2 // Schema validation or parse error
	ExitRuntimeError = 3 // I/O, permissions, or runtime error
)

var (
	// Global config instance
	cfg *config.Config

	// Global flags
	configFile string
	verbose    bool
	debug      bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "spectrehub",
	Short: "SpectreHub - Unified infrastructure audit aggregator",
	Long: `SpectreHub aggregates and analyzes security audit reports from the Spectre
tool family (VaultSpectre, S3Spectre, KafkaSpectre, ClickSpectre, PgSpectre, MongoSpectre).

It provides:
- Unified view across multiple infrastructure domains
- Trend analysis and historical tracking
- Actionable recommendations prioritized by severity
- CI/CD integration with exit codes

Quick start:
  spectrehub activate <license-key>
  spectrehub doctor
  spectrehub run --store --repo org/name
  spectrehub status

Other commands:
  spectrehub discover
  spectrehub diff --last 2
  spectrehub export --format sarif
  spectrehub collect ./reports --repo org/name`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		var err error
		cfg, err = config.LoadFromFile(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Override config with flags if provided
		if verbose {
			cfg.Verbose = true
		}
		if debug {
			cfg.Debug = true
		}

		return nil
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Cobra already prints the error
		os.Exit(ExitRuntimeError)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "",
		"config file (default: ~/.spectrehub.yaml or ./spectrehub.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"verbose output")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false,
		"debug mode (very verbose)")

	// Add subcommands
	rootCmd.AddCommand(collectCmd)
	rootCmd.AddCommand(summarizeCmd)
	rootCmd.AddCommand(discoverCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(activateCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(versionCmd)
}

// versionCmd shows version information
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("SpectreHub v0.2.0")
		fmt.Println("The unified infrastructure audit aggregator")
	},
}

// HandleError determines the appropriate exit code for an error
func HandleError(err error) int {
	if err == nil {
		return ExitOK
	}

	// Check error type to determine exit code
	switch err.(type) {
	case *ValidationError:
		return ExitInvalidInput
	case *ThresholdExceededError:
		return ExitPolicyFail
	default:
		// Check if it's an I/O or permission error
		if os.IsNotExist(err) || os.IsPermission(err) {
			return ExitRuntimeError
		}
		return ExitRuntimeError
	}
}

// ValidationError represents a validation failure
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ThresholdExceededError represents a threshold policy failure
type ThresholdExceededError struct {
	IssueCount int
	Threshold  int
}

func (e *ThresholdExceededError) Error() string {
	return fmt.Sprintf("issue count (%d) exceeds threshold (%d)", e.IssueCount, e.Threshold)
}

// logVerbose prints a message if verbose mode is enabled
func logVerbose(format string, args ...interface{}) {
	if cfg != nil && cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[INFO] "+format+"\n", args...)
	}
}

// logDebug prints a message if debug mode is enabled
func logDebug(format string, args ...interface{}) {
	if cfg != nil && cfg.Debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// logError prints an error message
func logError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
}
