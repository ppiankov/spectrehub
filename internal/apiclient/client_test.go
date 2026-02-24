package apiclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/reports" {
			t.Errorf("expected /v1/reports, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sh_test_key123" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}

		var payload ReportPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if payload.TotalTools != 3 {
			t.Errorf("expected 3 tools, got %d", payload.TotalTools)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": 1, "message": "report stored"})
	}))
	defer ts.Close()

	c := New(ts.URL, "sh_test_key123")
	err := c.SubmitReport(ReportPayload{
		Repo:       "org/repo",
		TotalTools: 3,
		Issues:     10,
		Score:      85.0,
		Health:     "good",
		RawJSON:    `{"test":true}`,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSubmitReportUnauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid license"})
	}))
	defer ts.Close()

	c := New(ts.URL, "bad_key")
	err := c.SubmitReport(ReportPayload{TotalTools: 1})
	if err == nil {
		t.Error("expected error for unauthorized request")
	}
}

func TestValidateLicenseSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/license/validate" {
			t.Errorf("expected /v1/license/validate, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":      true,
			"tier":       "team",
			"max_repos":  10,
			"email":      "user@example.com",
			"expires_at": "2027-02-01T00:00:00Z",
		})
	}))
	defer ts.Close()

	c := New(ts.URL, "sh_test_key123")
	info, err := c.ValidateLicense()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if info.Tier != "team" {
		t.Errorf("expected tier=team, got %s", info.Tier)
	}
	if info.MaxRepos != 10 {
		t.Errorf("expected max_repos=10, got %d", info.MaxRepos)
	}
}

func TestValidateLicenseInvalid(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	c := New(ts.URL, "bad_key")
	_, err := c.ValidateLicense()
	if err == nil {
		t.Error("expected error for invalid license")
	}
}

func TestValidateLicenseNilClient(t *testing.T) {
	var c *Client
	info, err := c.ValidateLicense()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if info != nil {
		t.Errorf("expected nil info, got %+v", info)
	}
}
