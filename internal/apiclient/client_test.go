package apiclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

const testLicenseKey = "sh_test_0123456789abcdef0123456789abcdef"

func validReportPayload() ReportPayload {
	return ReportPayload{
		Repo:       "org/repo",
		TotalTools: 3,
		Issues:     10,
		Score:      85.0,
		Health:     "good",
		RawJSON:    `{"test":true}`,
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newMockClient(baseURL, licenseKey string, fn roundTripFunc) *Client {
	client := New(baseURL, licenseKey)
	if client == nil {
		return nil
	}

	client.httpClient = &http.Client{
		Transport: fn,
	}
	return client
}

func jsonResponse(t *testing.T, status int, payload interface{}) *http.Response {
	t.Helper()

	body := []byte{}
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal response payload: %v", err)
		}
	}

	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestNewNilWhenEmpty(t *testing.T) {
	c := New("https://api.spectrehub.dev", "")
	if c != nil {
		t.Error("expected nil client when license key is empty")
	}
}

func TestSubmitReportNilClient(t *testing.T) {
	var c *Client
	err := c.SubmitReport(ReportPayload{})
	if err != nil {
		t.Errorf("expected nil error for nil client, got %v", err)
	}
}

func TestSubmitReportSuccess(t *testing.T) {
	client := newMockClient("https://api.spectrehub.dev", testLicenseKey, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/reports" {
			t.Fatalf("path = %s, want /v1/reports", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+testLicenseKey {
			t.Fatalf("authorization header = %s", got)
		}

		var payload ReportPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload.TotalTools != 3 {
			t.Fatalf("payload.TotalTools = %d, want 3", payload.TotalTools)
		}

		return jsonResponse(t, http.StatusCreated, map[string]interface{}{"id": 1}), nil
	})

	if err := client.SubmitReport(validReportPayload()); err != nil {
		t.Fatalf("SubmitReport: %v", err)
	}
}

func TestSubmitReportUnauthorized(t *testing.T) {
	client := newMockClient("https://api.spectrehub.dev", testLicenseKey, func(r *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusUnauthorized, map[string]string{"error": "invalid license"}), nil
	})

	err := client.SubmitReport(validReportPayload())
	if err == nil {
		t.Fatal("expected error for unauthorized request")
	}
}

func TestSubmitReportValidationError(t *testing.T) {
	client := newMockClient("https://api.spectrehub.dev", "bad_key", func(r *http.Request) (*http.Response, error) {
		t.Fatal("transport should not be called when validation fails")
		return nil, nil
	})

	err := client.SubmitReport(validReportPayload())
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateLicenseSuccess(t *testing.T) {
	client := newMockClient("https://api.spectrehub.dev", testLicenseKey, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/license/validate" {
			t.Fatalf("path = %s, want /v1/license/validate", r.URL.Path)
		}
		return jsonResponse(t, http.StatusOK, map[string]interface{}{
			"valid":      true,
			"tier":       "team",
			"max_repos":  10,
			"email":      "user@example.com",
			"expires_at": "2027-02-01T00:00:00Z",
		}), nil
	})

	info, err := client.ValidateLicense()
	if err != nil {
		t.Fatalf("ValidateLicense: %v", err)
	}
	if info.Tier != "team" {
		t.Fatalf("Tier = %s, want team", info.Tier)
	}
}

func TestValidateLicenseInvalid(t *testing.T) {
	client := newMockClient("https://api.spectrehub.dev", testLicenseKey, func(r *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusUnauthorized, nil), nil
	})

	_, err := client.ValidateLicense()
	if err == nil {
		t.Fatal("expected error for invalid license")
	}
}

func TestValidateLicenseInvalidFormat(t *testing.T) {
	client := newMockClient("https://api.spectrehub.dev", "bad_key", func(r *http.Request) (*http.Response, error) {
		t.Fatal("transport should not be called when validation fails")
		return nil, nil
	})

	_, err := client.ValidateLicense()
	if err == nil {
		t.Fatal("expected license format error")
	}
}

func TestValidateLicenseNilClient(t *testing.T) {
	var c *Client
	info, err := c.ValidateLicense()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if info != nil {
		t.Fatalf("expected nil info, got %+v", info)
	}
}

func TestNewWithKey(t *testing.T) {
	c := New("https://api.spectrehub.dev", testLicenseKey)
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.baseURL != "https://api.spectrehub.dev" {
		t.Fatalf("baseURL = %s", c.baseURL)
	}
	if c.licenseKey != testLicenseKey {
		t.Fatalf("licenseKey = %s", c.licenseKey)
	}
}

func TestSubmitReportServerError(t *testing.T) {
	client := newMockClient("https://api.spectrehub.dev", testLicenseKey, func(r *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusInternalServerError, map[string]string{"error": "internal error"}), nil
	})

	if err := client.SubmitReport(validReportPayload()); err == nil {
		t.Fatal("expected error for server error")
	}
}

func TestValidateLicenseServerError(t *testing.T) {
	client := newMockClient("https://api.spectrehub.dev", testLicenseKey, func(r *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusInternalServerError, nil), nil
	})

	if _, err := client.ValidateLicense(); err == nil {
		t.Fatal("expected error for server error")
	}
}

func TestValidateLicenseNotActive(t *testing.T) {
	client := newMockClient("https://api.spectrehub.dev", testLicenseKey, func(r *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusOK, map[string]interface{}{
			"valid": false,
			"tier":  "expired",
		}), nil
	})

	if _, err := client.ValidateLicense(); err == nil {
		t.Fatal("expected error for inactive license")
	}
}

func TestListReposSuccess(t *testing.T) {
	client := newMockClient("https://api.spectrehub.dev", testLicenseKey, func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/repos" {
			t.Fatalf("path = %s, want /v1/repos", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+testLicenseKey {
			t.Fatalf("authorization header = %s", got)
		}

		return jsonResponse(t, http.StatusOK, ReposInfo{
			Repos:    []string{"org/repo1", "org/repo2"},
			Count:    2,
			MaxRepos: 10,
		}), nil
	})

	info, err := client.ListRepos()
	if err != nil {
		t.Fatalf("ListRepos: %v", err)
	}
	if info.Count != 2 {
		t.Fatalf("Count = %d, want 2", info.Count)
	}
}

func TestListReposNilClient(t *testing.T) {
	var c *Client
	info, err := c.ListRepos()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if info != nil {
		t.Fatalf("expected nil info, got %+v", info)
	}
}

func TestListReposServerError(t *testing.T) {
	client := newMockClient("https://api.spectrehub.dev", testLicenseKey, func(r *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusInternalServerError, nil), nil
	})

	if _, err := client.ListRepos(); err == nil {
		t.Fatal("expected error for server error")
	}
}

func validUserActivityPayload() UserActivityPayload {
	return UserActivityPayload{
		TargetHash: "sha256:abc123",
		Users: []UserActivityEntry{
			{
				Username:     "appuser",
				DatabaseName: "admin",
				IsPrivileged: false,
			},
		},
	}
}

func TestSubmitUserActivityNilClient(t *testing.T) {
	var c *Client
	err := c.SubmitUserActivity(UserActivityPayload{})
	if err != nil {
		t.Errorf("expected nil error for nil client, got %v", err)
	}
}

func TestSubmitUserActivitySuccess(t *testing.T) {
	client := newMockClient("https://api.spectrehub.dev", testLicenseKey, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/users/activity" {
			t.Fatalf("path = %s, want /v1/users/activity", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+testLicenseKey {
			t.Fatalf("authorization header = %s", got)
		}

		var payload UserActivityPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload.TargetHash != "sha256:abc123" {
			t.Fatalf("payload.TargetHash = %s, want sha256:abc123", payload.TargetHash)
		}
		if len(payload.Users) != 1 {
			t.Fatalf("payload.Users len = %d, want 1", len(payload.Users))
		}

		return jsonResponse(t, http.StatusCreated, map[string]interface{}{"count": 1}), nil
	})

	if err := client.SubmitUserActivity(validUserActivityPayload()); err != nil {
		t.Fatalf("SubmitUserActivity: %v", err)
	}
}

func TestSubmitUserActivityUnauthorized(t *testing.T) {
	client := newMockClient("https://api.spectrehub.dev", testLicenseKey, func(r *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusUnauthorized, map[string]string{"error": "invalid license"}), nil
	})

	err := client.SubmitUserActivity(validUserActivityPayload())
	if err == nil {
		t.Fatal("expected error for unauthorized request")
	}
}

func TestSubmitUserActivityForbidden(t *testing.T) {
	client := newMockClient("https://api.spectrehub.dev", testLicenseKey, func(r *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusForbidden, map[string]string{"error": "user activity requires Team tier or higher"}), nil
	})

	err := client.SubmitUserActivity(validUserActivityPayload())
	if err == nil {
		t.Fatal("expected error for forbidden request")
	}
}

func TestSubmitUserActivityValidationError(t *testing.T) {
	client := newMockClient("https://api.spectrehub.dev", "bad_key", func(r *http.Request) (*http.Response, error) {
		t.Fatal("transport should not be called when validation fails")
		return nil, nil
	})

	err := client.SubmitUserActivity(validUserActivityPayload())
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSubmitUserActivityServerError(t *testing.T) {
	client := newMockClient("https://api.spectrehub.dev", testLicenseKey, func(r *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusInternalServerError, map[string]string{"error": "internal error"}), nil
	})

	if err := client.SubmitUserActivity(validUserActivityPayload()); err == nil {
		t.Fatal("expected error for server error")
	}
}

func TestConnectionErrors(t *testing.T) {
	transportErr := errors.New("dial error")
	client := newMockClient("https://api.spectrehub.dev", testLicenseKey, func(r *http.Request) (*http.Response, error) {
		return nil, transportErr
	})

	if err := client.SubmitReport(validReportPayload()); err == nil {
		t.Fatal("expected connection error for submit")
	}
	if _, err := client.ValidateLicense(); err == nil {
		t.Fatal("expected connection error for validate")
	}
	if _, err := client.ListRepos(); err == nil {
		t.Fatal("expected connection error for list")
	}
	if err := client.SubmitUserActivity(validUserActivityPayload()); err == nil {
		t.Fatal("expected connection error for user activity")
	}
}
