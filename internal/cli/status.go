package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ppiankov/spectrehub/internal/apiclient"
	"github.com/spf13/cobra"
)

var statusFormat string

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show license, repo usage, and configuration",
	Long: `Status displays current SpectreHub configuration and license details.

If a license key is configured, it validates against the API and shows
plan tier, repo usage, and expiration.

Example:
  spectrehub status
  spectrehub status --format json`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().StringVar(&statusFormat, "format", "text",
		"output format: text or json")
}

type statusResult struct {
	License    *statusLicense `json:"license,omitempty"`
	Config     statusConfig   `json:"config"`
	ConfigFile string         `json:"config_file"`
}

type statusLicense struct {
	Valid     bool     `json:"valid"`
	Tier      string   `json:"tier"`
	MaxRepos  int      `json:"max_repos"`
	UsedRepos int      `json:"used_repos"`
	Repos     []string `json:"repos,omitempty"`
	ExpiresAt string   `json:"expires_at"`
}

type statusConfig struct {
	StorageDir string `json:"storage_dir"`
	Format     string `json:"format"`
	Repo       string `json:"repo,omitempty"`
	HasKey     bool   `json:"has_license_key"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	result := statusResult{
		Config: statusConfig{
			StorageDir: cfg.StorageDir,
			Format:     cfg.Format,
			Repo:       cfg.Repo,
			HasKey:     cfg.LicenseKey != "",
		},
		ConfigFile: configFile,
	}

	// If license key is configured, validate and fetch repos.
	if cfg.LicenseKey != "" {
		apiURL := cfg.APIURL
		if apiURL == "" {
			apiURL = "https://api.spectrehub.dev"
		}

		client := apiclient.New(apiURL, cfg.LicenseKey)

		info, err := client.ValidateLicense()
		if err != nil {
			if statusFormat == "json" {
				result.License = &statusLicense{Valid: false}
				return writeStatusJSON(result)
			}
			fmt.Fprintf(os.Stderr, "License: invalid (%v)\n", err)
			return nil
		}

		lic := &statusLicense{
			Valid:     true,
			Tier:      info.Tier,
			MaxRepos:  info.MaxRepos,
			ExpiresAt: info.ExpiresAt,
		}

		// Fetch repo usage.
		if repos, err := client.ListRepos(); err == nil {
			lic.UsedRepos = repos.Count
			lic.Repos = repos.Repos
		}

		result.License = lic
	}

	if statusFormat == "json" {
		return writeStatusJSON(result)
	}

	return writeStatusText(result)
}

func writeStatusJSON(result statusResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func writeStatusText(result statusResult) error {
	if result.License != nil && result.License.Valid {
		fmt.Printf("License:  %s\n", result.License.Tier)
		if result.License.MaxRepos > 0 {
			fmt.Printf("Repos:    %d/%d used\n", result.License.UsedRepos, result.License.MaxRepos)
		} else {
			fmt.Printf("Repos:    %d (unlimited)\n", result.License.UsedRepos)
		}
		for _, r := range result.License.Repos {
			fmt.Printf("          - %s\n", r)
		}
		fmt.Printf("Expires:  %s\n", result.License.ExpiresAt)
	} else if result.Config.HasKey {
		fmt.Println("License:  invalid or expired")
	} else {
		fmt.Println("License:  not configured (free tier)")
	}

	fmt.Printf("Storage:  %s\n", result.Config.StorageDir)
	if result.Config.Repo != "" {
		fmt.Printf("Repo:     %s\n", result.Config.Repo)
	}

	return nil
}
