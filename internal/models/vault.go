package models

import "time"

// VaultReport represents the complete output from VaultSpectre
type VaultReport struct {
	Tool       string              `json:"tool"`
	Version    string              `json:"version"`
	Timestamp  time.Time           `json:"timestamp"`
	Config     VaultConfig         `json:"config"`
	Summary    VaultSummary        `json:"summary"`
	Secrets    map[string]*SecretInfo `json:"secrets"`
	References []VaultReference    `json:"references,omitempty"`
}

// VaultConfig contains scan configuration for VaultSpectre
type VaultConfig struct {
	VaultAddr          string `json:"vault_addr"`
	RepoPath           string `json:"repo_path"`
	StaleThresholdDays int    `json:"stale_threshold_days"`
}

// VaultSummary contains summary statistics from VaultSpectre
type VaultSummary struct {
	TotalReferences    int    `json:"total_references"`
	StatusOK           int    `json:"status_ok"`
	StatusMissing      int    `json:"status_missing"`
	StatusAccessDenied int    `json:"status_access_denied"`
	StatusInvalid      int    `json:"status_invalid"`
	StatusDynamic      int    `json:"status_dynamic"`
	StatusError        int    `json:"status_error"`
	StaleSecrets       int    `json:"stale_secrets"`
	HealthScore        string `json:"health_score"`
}

// SecretInfo contains information about a specific secret
type SecretInfo struct {
	Path         string           `json:"path"`
	Status       string           `json:"status"`
	IsStale      bool             `json:"is_stale,omitempty"`
	LastAccessed string           `json:"last_accessed,omitempty"`
	ErrorMsg     string           `json:"error_msg,omitempty"`
	References   []VaultReference `json:"references"`
}

// VaultReference represents a reference to a secret in code
type VaultReference struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Type   string `json:"type"`   // lookup, env_var, template, etc.
	Path   string `json:"path"`
	Status string `json:"status"`
	IsStale bool  `json:"is_stale,omitempty"`
	LastAccessed string `json:"last_accessed,omitempty"`
	ErrorMsg string `json:"error_msg,omitempty"`
}
