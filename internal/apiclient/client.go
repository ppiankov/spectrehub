package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ppiankov/spectrehub/internal/api"
)

// Client submits reports to the SpectreHub API.
type Client struct {
	baseURL    string
	licenseKey string
	httpClient *http.Client
}

// New creates an API client. Returns nil if licenseKey is empty.
func New(baseURL, licenseKey string) *Client {
	if licenseKey == "" {
		return nil
	}
	return &Client{
		baseURL:    baseURL,
		licenseKey: licenseKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// ReportPayload is the body for POST /v1/reports.
type ReportPayload struct {
	Repo       string  `json:"repo"`
	TotalTools int     `json:"total_tools"`
	Issues     int     `json:"issues"`
	Score      float64 `json:"score"`
	Health     string  `json:"health"`
	RawJSON    string  `json:"raw_json"`
}

// SubmitReport sends an aggregated report to the API.
func (c *Client) SubmitReport(payload ReportPayload) error {
	if c == nil {
		return nil
	}
	if err := api.ValidateLicenseKey(c.licenseKey); err != nil {
		return fmt.Errorf("invalid license key: %w", err)
	}
	if err := api.ValidateReportInput(api.ReportInput{
		Repo:       payload.Repo,
		TotalTools: payload.TotalTools,
		Issues:     payload.Issues,
		Score:      payload.Score,
		Health:     payload.Health,
		RawJSON:    payload.RawJSON,
	}); err != nil {
		return fmt.Errorf("invalid report payload: %w", err)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/v1/reports", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.licenseKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("submit report: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		var errResp map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		msg := errResp["error"]
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("API error: %s", msg)
	}

	return nil
}

// LicenseInfo holds validated license details from the API.
type LicenseInfo struct {
	Valid     bool   `json:"valid"`
	Tier      string `json:"tier"`
	MaxRepos  int    `json:"max_repos"`
	Email     string `json:"email"`
	ExpiresAt string `json:"expires_at"`
}

// ValidateLicense checks if the license key is valid and returns plan info.
func (c *Client) ValidateLicense() (*LicenseInfo, error) {
	if c == nil {
		return nil, nil
	}
	if err := api.ValidateLicenseKey(c.licenseKey); err != nil {
		return nil, fmt.Errorf("invalid license key: %w", err)
	}

	req, err := http.NewRequest("GET", c.baseURL+"/v1/license/validate", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.licenseKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("validate license: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid or expired license key")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (HTTP %d)", resp.StatusCode)
	}

	var info LicenseInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if !info.Valid {
		return nil, fmt.Errorf("license key is not active")
	}

	return &info, nil
}

// UserActivityEntry is a single user record for the activity API.
type UserActivityEntry struct {
	Username     string `json:"username"`
	DatabaseName string `json:"database_name"`
	LastSeenAt   string `json:"last_seen_at"`
	RolesJSON    string `json:"roles_json"`
	IsPrivileged bool   `json:"is_privileged"`
}

// UserActivityPayload is the body for POST /v1/users/activity.
type UserActivityPayload struct {
	TargetHash string              `json:"target_hash"`
	Users      []UserActivityEntry `json:"users"`
}

// SubmitUserActivity sends user activity data to the API.
func (c *Client) SubmitUserActivity(payload UserActivityPayload) error {
	if c == nil {
		return nil
	}
	if err := api.ValidateLicenseKey(c.licenseKey); err != nil {
		return fmt.Errorf("invalid license key: %w", err)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal user activity: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/v1/users/activity", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.licenseKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("submit user activity: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		var errResp map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		msg := errResp["error"]
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("API error: %s", msg)
	}

	return nil
}

// ReposInfo holds the response from GET /v1/repos.
type ReposInfo struct {
	Repos    []string `json:"repos"`
	Count    int      `json:"count"`
	MaxRepos int      `json:"max_repos"`
}

// ListRepos returns the tracked repos for the license.
func (c *Client) ListRepos() (*ReposInfo, error) {
	if c == nil {
		return nil, nil
	}
	if err := api.ValidateLicenseKey(c.licenseKey); err != nil {
		return nil, fmt.Errorf("invalid license key: %w", err)
	}

	req, err := http.NewRequest("GET", c.baseURL+"/v1/repos", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.licenseKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (HTTP %d)", resp.StatusCode)
	}

	var info ReposInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &info, nil
}
