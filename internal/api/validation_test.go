package api

import (
	"strings"
	"testing"
)

func TestValidateRepo(t *testing.T) {
	tests := []struct {
		name    string
		repo    string
		wantErr bool
	}{
		{name: "valid", repo: "org/repo", wantErr: false},
		{name: "valid with punctuation", repo: "org-name/repo_name.v2", wantErr: false},
		{name: "missing slash", repo: "org", wantErr: true},
		{name: "empty", repo: "", wantErr: true},
		{name: "too long", repo: strings.Repeat("a", MaxRepoLength+1), wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepo(tt.repo)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateLicenseKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{name: "valid test key", key: "sh_test_0123456789abcdef0123456789abcdef", wantErr: false},
		{name: "valid live key", key: "sh_live_0123456789abcdef0123456789abcdef", wantErr: false},
		{name: "invalid prefix", key: "sh_dev_0123456789abcdef0123456789abcdef", wantErr: true},
		{name: "invalid length", key: "sh_test_deadbeef", wantErr: true},
		{name: "invalid chars", key: "sh_test_zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLicenseKey(tt.key)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateReportInput(t *testing.T) {
	valid := ReportInput{
		Repo:       "org/repo",
		TotalTools: 3,
		Issues:     4,
		Score:      97.5,
		Health:     "good",
		RawJSON:    `{"summary":{"total_issues":4}}`,
	}

	if err := ValidateReportInput(valid); err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}

	tests := []struct {
		name    string
		input   ReportInput
		wantErr bool
	}{
		{
			name: "invalid repo",
			input: ReportInput{
				Repo:       "bad_repo",
				TotalTools: 3,
				Issues:     4,
				Score:      97.5,
				Health:     "good",
				RawJSON:    `{"ok":true}`,
			},
			wantErr: true,
		},
		{
			name: "invalid score",
			input: ReportInput{
				Repo:       "org/repo",
				TotalTools: 3,
				Issues:     4,
				Score:      101,
				Health:     "good",
				RawJSON:    `{"ok":true}`,
			},
			wantErr: true,
		},
		{
			name: "invalid health",
			input: ReportInput{
				Repo:       "org/repo",
				TotalTools: 3,
				Issues:     4,
				Score:      50,
				Health:     "unknown",
				RawJSON:    `{"ok":true}`,
			},
			wantErr: true,
		},
		{
			name: "invalid json",
			input: ReportInput{
				Repo:       "org/repo",
				TotalTools: 3,
				Issues:     4,
				Score:      50,
				Health:     "warning",
				RawJSON:    `{"broken":`,
			},
			wantErr: true,
		},
		{
			name: "oversized payload",
			input: ReportInput{
				Repo:       "org/repo",
				TotalTools: 3,
				Issues:     4,
				Score:      50,
				Health:     "warning",
				RawJSON:    `{"data":"` + strings.Repeat("a", MaxRawReportPayloadBytes) + `"}`,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReportInput(tt.input)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}
