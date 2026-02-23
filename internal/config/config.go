package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all configuration for SpectreHub
type Config struct {
	// Storage configuration
	StorageDir string `mapstructure:"storage_dir"`

	// Threshold for CI/CD failure
	FailThreshold int `mapstructure:"fail_threshold"`

	// Output format (text, json, both)
	Format string `mapstructure:"format"`

	// Number of last runs to analyze
	LastRuns int `mapstructure:"last_runs"`

	// Verbose output
	Verbose bool `mapstructure:"verbose"`

	// Debug mode
	Debug bool `mapstructure:"debug"`

	// License key for API access (from config or SPECTREHUB_LICENSE_KEY)
	LicenseKey string `mapstructure:"license_key"`

	// API URL (defaults to https://api.spectrehub.dev)
	APIURL string `mapstructure:"api_url"`
}

// DefaultConfig returns configuration with default values
func DefaultConfig() *Config {
	return &Config{
		StorageDir:    ".spectre",
		FailThreshold: 0, // 0 means no threshold check
		Format:        "text",
		LastRuns:      7,
		Verbose:       false,
		Debug:         false,
	}
}

// Load loads configuration with the following precedence (lowest to highest):
// 1. Default values
// 2. Config file (~/.spectrehub.yaml or ./spectrehub.yaml)
// 3. Environment variables (SPECTREHUB_*)
// 4. CLI flags (handled by caller)
func Load() (*Config, error) {
	return LoadFromFile("")
}

// LoadFromFile loads configuration from a specific file path
// If path is empty, it searches for config in standard locations
func LoadFromFile(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	defaults := DefaultConfig()
	v.SetDefault("storage_dir", defaults.StorageDir)
	v.SetDefault("fail_threshold", defaults.FailThreshold)
	v.SetDefault("format", defaults.Format)
	v.SetDefault("last_runs", defaults.LastRuns)
	v.SetDefault("verbose", defaults.Verbose)
	v.SetDefault("debug", defaults.Debug)
	v.SetDefault("license_key", "")
	v.SetDefault("api_url", "https://api.spectrehub.dev")

	// Set config file settings
	v.SetConfigName("spectrehub")
	v.SetConfigType("yaml")

	if configPath != "" {
		// Use explicit config file path
		v.SetConfigFile(configPath)
	} else {
		// Search for config in standard locations
		// 1. Current directory
		v.AddConfigPath(".")

		// 2. Home directory
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(home)
		}

		// 3. XDG config directory
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			v.AddConfigPath(filepath.Join(xdgConfig, "spectrehub"))
		}
	}

	// Enable environment variable support
	v.SetEnvPrefix("SPECTREHUB")
	v.AutomaticEnv()

	// Try to read config file (ignore error if not found)
	if err := v.ReadInConfig(); err != nil {
		// Only return error if it's not a "file not found" error
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is OK, we'll use defaults
	}

	// Unmarshal into config struct
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate format
	validFormats := map[string]bool{
		"text": true,
		"json": true,
		"both": true,
	}
	if !validFormats[c.Format] {
		return fmt.Errorf("invalid format: %s (must be text, json, or both)", c.Format)
	}

	// Validate threshold (can't be negative)
	if c.FailThreshold < 0 {
		return fmt.Errorf("fail_threshold cannot be negative")
	}

	// Validate last_runs (must be positive)
	if c.LastRuns <= 0 {
		return fmt.Errorf("last_runs must be positive")
	}

	// Validate storage_dir is not empty
	if c.StorageDir == "" {
		return fmt.Errorf("storage_dir cannot be empty")
	}

	return nil
}

// GetStoragePath returns the absolute path to the storage directory
func (c *Config) GetStoragePath() (string, error) {
	// Expand ~ to home directory
	if c.StorageDir[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(home, c.StorageDir[2:]), nil
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(c.StorageDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return absPath, nil
}

// ShouldFailOnThreshold checks if the issue count exceeds the threshold
func (c *Config) ShouldFailOnThreshold(issueCount int) bool {
	if c.FailThreshold == 0 {
		return false // No threshold check
	}
	return issueCount > c.FailThreshold
}

// GenerateSampleConfig generates a sample configuration file content
func GenerateSampleConfig() string {
	return `# SpectreHub Configuration
# Save this file as ~/.spectrehub.yaml or ./spectrehub.yaml

# Directory to store aggregated reports
storage_dir: .spectre

# Fail threshold for CI/CD (exit code 1 if issues exceed this number)
# Set to 0 to disable threshold checking
fail_threshold: 50

# Output format: text, json, or both
format: text

# Number of last runs to analyze in summarize command
last_runs: 7

# Enable verbose output
verbose: false

# Enable debug mode
debug: false

# License key for SpectreHub API (paid features)
# Get yours at https://spectrehub.dev
# Can also be set via SPECTREHUB_LICENSE_KEY env var
# license_key: sh_live_your_key_here

# API URL (change only for self-hosted or testing)
# api_url: https://api.spectrehub.dev
`
}
