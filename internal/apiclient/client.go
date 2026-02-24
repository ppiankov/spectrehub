package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
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
