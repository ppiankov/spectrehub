package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/ppiankov/spectrehub/internal/api"
	"github.com/ppiankov/spectrehub/internal/apiclient"
	"github.com/ppiankov/spectrehub/internal/config"
	"github.com/ppiankov/spectrehub/internal/discovery"
	"github.com/spf13/cobra"
)

var doctorFormat string

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check environment readiness and diagnose common problems",
	Long: `Doctor validates your SpectreHub setup end-to-end:

  1. Config file — found and readable?
  2. License key — present and valid?
  3. API connectivity — reachable?
  4. Repo identifier — configured for upload?
  5. Spectre tools — installed and runnable?
  6. Storage — directory writable?

Fix the issues it reports, then run 'spectrehub run' with confidence.`,
	RunE: runDoctor,
}

func init() {
	doctorCmd.Flags().StringVar(&doctorFormat, "format", "text",
		"output format: text or json")
}

type doctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "ok", "warn", "fail"
	Detail string `json:"detail,omitempty"`
}

type doctorResult struct {
	Checks  []doctorCheck `json:"checks"`
	Summary string        `json:"summary"`
}

func runDoctor(cmd *cobra.Command, args []string) error {
	var checks []doctorCheck

	// 1. Config file
	checks = append(checks, checkConfig())

	// 2. License key
	checks = append(checks, checkLicense())

	// 3. API connectivity
	checks = append(checks, checkAPI())

	// 4. Repo identifier
	checks = append(checks, checkRepo())

	// 5. Spectre tools
	checks = append(checks, checkTools()...)

	// 6. Storage directory
	checks = append(checks, checkStorage())

	// Build summary
	fails, warns := 0, 0
	for _, c := range checks {
		switch c.Status {
		case "fail":
			fails++
		case "warn":
			warns++
		}
	}

	summary := "all checks passed"
	if fails > 0 {
		summary = fmt.Sprintf("%d issue(s) found", fails)
	} else if warns > 0 {
		summary = fmt.Sprintf("ok with %d warning(s)", warns)
	}

	result := doctorResult{Checks: checks, Summary: summary}

	if doctorFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	return writeDoctorText(result)
}

func writeDoctorText(result doctorResult) error {
	icons := map[string]string{
		"ok":   "✓",
		"warn": "△",
		"fail": "✗",
	}

	for _, c := range result.Checks {
		icon := icons[c.Status]
		if c.Detail != "" {
			fmt.Printf("  %s %-20s %s\n", icon, c.Name, c.Detail)
		} else {
			fmt.Printf("  %s %s\n", icon, c.Name)
		}
	}

	fmt.Printf("\n%s\n", result.Summary)
	return nil
}

func checkConfig() doctorCheck {
	path := config.ConfigPath()
	if configFile != "" {
		path = configFile
	}

	if _, err := os.Stat(path); err != nil {
		return doctorCheck{
			Name:   "config",
			Status: "warn",
			Detail: "no config file found (using defaults). Run: spectrehub activate <key>",
		}
	}

	return doctorCheck{
		Name:   "config",
		Status: "ok",
		Detail: path,
	}
}

func checkLicense() doctorCheck {
	if cfg.LicenseKey == "" {
		return doctorCheck{
			Name:   "license",
			Status: "warn",
			Detail: "not configured (free tier). Set SPECTREHUB_LICENSE_KEY or run: spectrehub activate <key>",
		}
	}

	// Validate against API
	apiURL := cfg.APIURL
	if apiURL == "" {
		apiURL = "https://api.spectrehub.dev"
	}

	client := apiclient.New(apiURL, cfg.LicenseKey)
	info, err := client.ValidateLicense()
	if err != nil {
		return doctorCheck{
			Name:   "license",
			Status: "fail",
			Detail: fmt.Sprintf("invalid (%v)", err),
		}
	}

	return doctorCheck{
		Name:   "license",
		Status: "ok",
		Detail: fmt.Sprintf("%s (up to %d repos, expires %s)", info.Tier, info.MaxRepos, info.ExpiresAt),
	}
}

func checkAPI() doctorCheck {
	apiURL := cfg.APIURL
	if apiURL == "" {
		apiURL = "https://api.spectrehub.dev"
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiURL + "/v1/health")
	if err != nil {
		return doctorCheck{
			Name:   "api",
			Status: "fail",
			Detail: fmt.Sprintf("unreachable (%v)", err),
		}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return doctorCheck{
			Name:   "api",
			Status: "fail",
			Detail: fmt.Sprintf("unhealthy (HTTP %d)", resp.StatusCode),
		}
	}

	return doctorCheck{
		Name:   "api",
		Status: "ok",
		Detail: apiURL,
	}
}

func checkRepo() doctorCheck {
	repo := cfg.Repo
	if repo == "" {
		repo = os.Getenv("SPECTREHUB_REPO")
	}

	if repo == "" {
		if cfg.LicenseKey != "" {
			return doctorCheck{
				Name:   "repo",
				Status: "warn",
				Detail: "not set. Use --repo, config repo:, or SPECTREHUB_REPO env var",
			}
		}
		return doctorCheck{
			Name:   "repo",
			Status: "ok",
			Detail: "not needed (no license key)",
		}
	}
	if err := api.ValidateRepo(repo); err != nil {
		return doctorCheck{
			Name:   "repo",
			Status: "fail",
			Detail: fmt.Sprintf("invalid repo: %v", err),
		}
	}

	return doctorCheck{
		Name:   "repo",
		Status: "ok",
		Detail: repo,
	}
}

func checkTools() []doctorCheck {
	d := discovery.New(exec.LookPath, os.Getenv)
	plan := d.Discover()

	var checks []doctorCheck

	for _, td := range plan.Tools {
		c := doctorCheck{Name: td.Binary}

		if td.Runnable {
			c.Status = "ok"
			c.Detail = "ready"
		} else if td.Available && !td.HasTarget {
			var missing []string
			for _, ev := range td.EnvVars {
				if !ev.Set {
					missing = append(missing, ev.Name)
				}
			}
			c.Status = "warn"
			if len(missing) > 0 {
				c.Detail = fmt.Sprintf("installed but no target (missing: %s)", joinMax(missing, 3))
			} else {
				c.Detail = "installed but no target configured"
			}
		} else {
			info := discovery.Registry[td.Tool]
			c.Status = "warn"
			c.Detail = fmt.Sprintf("not installed. Run: %s", info.InstallHint)
		}

		checks = append(checks, c)
	}

	if plan.TotalRunnable == 0 {
		checks = append(checks, doctorCheck{
			Name:   "tools",
			Status: "fail",
			Detail: "no runnable tools found — nothing to audit",
		})
	}

	return checks
}

func checkStorage() doctorCheck {
	storagePath := cfg.StorageDir
	if storagePath == "" {
		storagePath = ".spectre"
	}

	// Check if directory exists and is writable
	info, err := os.Stat(storagePath)
	if err != nil {
		// Directory doesn't exist yet — that's fine, it will be created
		return doctorCheck{
			Name:   "storage",
			Status: "ok",
			Detail: fmt.Sprintf("%s (will be created on first --store)", storagePath),
		}
	}

	if !info.IsDir() {
		return doctorCheck{
			Name:   "storage",
			Status: "fail",
			Detail: fmt.Sprintf("%s exists but is not a directory", storagePath),
		}
	}

	// Try writing a temp file to check write access
	tmpFile := storagePath + "/.doctor-check"
	if err := os.WriteFile(tmpFile, []byte("ok"), 0600); err != nil {
		return doctorCheck{
			Name:   "storage",
			Status: "fail",
			Detail: fmt.Sprintf("%s not writable: %v", storagePath, err),
		}
	}
	_ = os.Remove(tmpFile)

	return doctorCheck{
		Name:   "storage",
		Status: "ok",
		Detail: storagePath,
	}
}

// joinMax joins up to n strings with ", ".
func joinMax(s []string, n int) string {
	if len(s) <= n {
		result := ""
		for i, v := range s {
			if i > 0 {
				result += ", "
			}
			result += v
		}
		return result
	}
	result := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			result += ", "
		}
		result += s[i]
	}
	return fmt.Sprintf("%s +%d more", result, len(s)-n)
}
