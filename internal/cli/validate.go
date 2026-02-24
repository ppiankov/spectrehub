package cli

import (
	"fmt"
	"os"

	"github.com/ppiankov/spectrehub/internal/validator"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate a report against the spectre/v1 schema",
	Long: `Validate checks that a JSON report file conforms to the spectre/v1 schema.

Returns exit 0 if valid, exit 2 if invalid with details on stderr.

Example:
  spectrehub validate report.json
  vaultspectre scan --format spectrehub | spectrehub validate /dev/stdin`,
	Args: cobra.ExactArgs(1),
	RunE: runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	v := validator.New()
	if err := v.ValidateSpectreV1Report(data); err != nil {
		fmt.Fprintf(os.Stderr, "INVALID: %v\n", err)
		os.Exit(ExitInvalidInput)
		return nil
	}

	fmt.Println("VALID: conforms to spectre/v1")
	return nil
}
