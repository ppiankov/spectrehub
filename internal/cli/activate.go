package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/ppiankov/spectrehub/internal/apiclient"
	"github.com/ppiankov/spectrehub/internal/config"
	"github.com/spf13/cobra"
)

var activateFormat string

var activateCmd = &cobra.Command{
	Use:   "activate <license-key>",
	Short: "Activate a license key for API reporting",
	Long: `Activate writes the license key to your config file and validates it
against the SpectreHub API.

After activation, spectrehub run and spectrehub collect will automatically
submit reports to the API for historical regression tracking.

Example:
  spectrehub activate sh_live_abc123def456...`,
	Args: cobra.ExactArgs(1),
	RunE: runActivate,
}

func init() {
	activateCmd.Flags().StringVar(&activateFormat, "format", "text",
		"output format: text or json")
}

// keyPattern validates license key format: sh_live_ or sh_test_ followed by 32 hex chars.
var keyPattern = regexp.MustCompile(`^sh_(live|test)_[0-9a-f]{32}$`)

// maskKey masks a license key for safe display: sh_live_14...70fb
func maskKey(key string) string {
	if len(key) < 16 {
		return "****"
	}
	return key[:10] + "..." + key[len(key)-4:]
}

func runActivate(cmd *cobra.Command, args []string) error {
	key := args[0]

	// Validate key format.
	if !keyPattern.MatchString(key) {
		if activateFormat == "json" {
			writeActivateJSON(os.Stdout, "error", "invalid key format", "", 0, "")
		} else {
			fmt.Fprintln(os.Stderr, "Invalid license key format.")
			fmt.Fprintln(os.Stderr, "Expected: sh_live_<32 hex chars> or sh_test_<32 hex chars>")
		}
		os.Exit(1)
		return nil
	}

	// Validate against API.
	apiURL := cfg.APIURL
	if apiURL == "" {
		apiURL = "https://api.spectrehub.dev"
	}

	client := apiclient.New(apiURL, key)
	info, err := client.ValidateLicense()
	if err != nil {
		if activateFormat == "json" {
			writeActivateJSON(os.Stdout, "error", err.Error(), "", 0, "")
		} else {
			fmt.Fprintf(os.Stderr, "License validation failed: %v\n", err)
		}
		os.Exit(1)
		return nil
	}

	// Write to config.
	configPath := config.ConfigPath()
	if configFile != "" {
		configPath = configFile
	}

	if err := config.WriteActivation(key, apiURL, configPath); err != nil {
		if activateFormat == "json" {
			writeActivateJSON(os.Stdout, "error", err.Error(), "", 0, "")
		} else {
			fmt.Fprintf(os.Stderr, "Failed to write config: %v\n", err)
		}
		os.Exit(ExitRuntimeError)
		return nil
	}

	// Success output.
	if activateFormat == "json" {
		writeActivateJSON(os.Stdout, "activated", "", info.Tier, info.MaxRepos, configPath)
	} else {
		fmt.Printf("License activated. Plan: %s (up to %d repos).\n", info.Tier, info.MaxRepos)
		fmt.Printf("Config written to %s\n", configPath)
	}

	return nil
}

type activateResult struct {
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
	Plan       string `json:"plan,omitempty"`
	MaxRepos   int    `json:"max_repos,omitempty"`
	ConfigPath string `json:"config_path,omitempty"`
}

func writeActivateJSON(w *os.File, status, errMsg, plan string, maxRepos int, configPath string) {
	result := activateResult{
		Status:     status,
		Error:      errMsg,
		Plan:       plan,
		MaxRepos:   maxRepos,
		ConfigPath: configPath,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(result)
}
