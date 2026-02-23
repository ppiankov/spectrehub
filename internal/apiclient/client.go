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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		msg := errResp["error"]
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("API error: %s", msg)
	}

	return nil
}

// ValidateLicense checks if the license key is valid.
func (c *Client) ValidateLicense() (string, error) {
	if c == nil {
		return "", nil
	}

	req, err := http.NewRequest("GET", c.baseURL+"/v1/license/validate", nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.licenseKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("validate license: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("invalid license key (HTTP %d)", resp.StatusCode)
	}

	var result struct {
		Valid bool   `json:"valid"`
		Tier  string `json:"tier"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if !result.Valid {
		return "", fmt.Errorf("license key is not active")
	}

	return result.Tier, nil
}
