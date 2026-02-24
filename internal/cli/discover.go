package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ppiankov/spectrehub/internal/discovery"
	"github.com/spf13/cobra"
)

var (
	discoverFormat string
)

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Detect available spectre tools and infrastructure targets",
	Long: `Discover probes the local environment to find which spectre tools are
installed (in PATH), which infrastructure targets are configured (via
environment variables or config files), and reports which tools can be run.

This is a read-only operation — no tools are executed, no network calls are
made. Use 'spectrehub run' to execute the discovered tools.`,
	RunE: runDiscover,
}

func init() {
	discoverCmd.Flags().StringVar(&discoverFormat, "format", "text",
		"output format: text or json")
}

func runDiscover(cmd *cobra.Command, args []string) error {
	d := discovery.New(exec.LookPath, os.Getenv)
	plan := d.Discover()

	switch discoverFormat {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(plan)
	case "text":
		printDiscoveryText(plan)
		return nil
	default:
		return &ValidationError{Message: fmt.Sprintf("invalid format: %s (must be text or json)", discoverFormat)}
	}
}

func printDiscoveryText(plan *discovery.DiscoveryPlan) {
	fmt.Printf("Discovered %d tool(s), %d runnable\n\n", plan.TotalFound, plan.TotalRunnable)

	for _, td := range plan.Tools {
		status := "✗ not found"
		if td.Available && td.Runnable {
			status = "✓ ready"
		} else if td.Available {
			status = "○ installed (no target)"
		}

		fmt.Printf("  %-14s  %s\n", td.Binary, status)

		if td.Available {
			fmt.Printf("                  path: %s\n", td.BinaryPath)
		} else if info, ok := discovery.Registry[td.Tool]; ok && info.InstallHint != "" {
			fmt.Printf("                  install: %s\n", info.InstallHint)
		}

		// Show env var status
		if len(td.EnvVars) > 0 {
			var setVars, unsetVars []string
			for _, ev := range td.EnvVars {
				if ev.Set {
					setVars = append(setVars, ev.Name)
				} else {
					unsetVars = append(unsetVars, ev.Name)
				}
			}
			if len(setVars) > 0 {
				fmt.Printf("                  env:  %s\n", strings.Join(setVars, ", "))
			}
			if len(unsetVars) > 0 {
				fmt.Printf("                  missing: %s\n", strings.Join(unsetVars, ", "))
			}
		}

		// Show config file status
		if len(td.Configs) > 0 {
			for _, c := range td.Configs {
				if c.Exists {
					fmt.Printf("                  config: %s\n", c.Path)
				}
			}
		}

		fmt.Println()
	}

	if plan.TotalRunnable == 0 {
		fmt.Println("No runnable tools found. Install spectre tools and configure targets.")
		fmt.Println("See: https://spectrehub.dev")
	}
}
