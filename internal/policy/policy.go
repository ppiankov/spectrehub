package policy

import (
	"fmt"
	"os"
	"strings"

	"github.com/ppiankov/spectrehub/internal/apiclient"
	"github.com/ppiankov/spectrehub/internal/models"
	"gopkg.in/yaml.v3"
)

// Policy defines enforcement rules for audit results.
type Policy struct {
	Version string `yaml:"version"`
	Rules   Rules  `yaml:"rules"`
}

// Rules contains all configurable policy rules.
type Rules struct {
	MaxIssues           *int     `yaml:"max_issues,omitempty"`
	MaxCritical         *int     `yaml:"max_critical,omitempty"`
	MaxHigh             *int     `yaml:"max_high,omitempty"`
	MinScore            *float64 `yaml:"min_score,omitempty"`
	ForbidCategories    []string `yaml:"forbid_categories,omitempty"`
	RequireTools        []string `yaml:"require_tools,omitempty"`
	MaxOpenCriticalDays *int     `yaml:"max_open_critical_days,omitempty"`
	MaxOpenHighDays     *int     `yaml:"max_open_high_days,omitempty"`
}

// Violation is a single policy failure.
type Violation struct {
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// Result holds the outcome of a policy check.
type Result struct {
	Pass       bool        `json:"pass"`
	Violations []Violation `json:"violations"`
}

// LoadFromFile reads a policy file.
func LoadFromFile(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read policy: %w", err)
	}

	var p Policy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse policy: %w", err)
	}

	return &p, nil
}

// FindPolicyFile searches for a policy file in the current directory
// and parent directories up to the filesystem root.
func FindPolicyFile() string {
	names := []string{".spectrehub-policy.yaml", ".spectrehub-policy.yml"}

	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		for _, name := range names {
			path := dir + "/" + name
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
		parent := dir[:strings.LastIndex(dir, "/")]
		if parent == dir || parent == "" {
			break
		}
		dir = parent
	}

	return ""
}

// Evaluate checks an aggregated report against the policy rules.
func (p *Policy) Evaluate(report *models.AggregatedReport) *Result {
	if p == nil {
		return &Result{Pass: true}
	}

	var violations []Violation

	// max_issues
	if p.Rules.MaxIssues != nil {
		if report.Summary.TotalIssues > *p.Rules.MaxIssues {
			violations = append(violations, Violation{
				Rule:    "max_issues",
				Message: fmt.Sprintf("total issues %d exceeds limit %d", report.Summary.TotalIssues, *p.Rules.MaxIssues),
			})
		}
	}

	// max_critical
	if p.Rules.MaxCritical != nil {
		count := report.Summary.IssuesBySeverity[models.SeverityCritical]
		if count > *p.Rules.MaxCritical {
			violations = append(violations, Violation{
				Rule:    "max_critical",
				Message: fmt.Sprintf("critical issues %d exceeds limit %d", count, *p.Rules.MaxCritical),
			})
		}
	}

	// max_high
	if p.Rules.MaxHigh != nil {
		count := report.Summary.IssuesBySeverity[models.SeverityHigh]
		if count > *p.Rules.MaxHigh {
			violations = append(violations, Violation{
				Rule:    "max_high",
				Message: fmt.Sprintf("high issues %d exceeds limit %d", count, *p.Rules.MaxHigh),
			})
		}
	}

	// min_score
	if p.Rules.MinScore != nil {
		if report.Summary.ScorePercent < *p.Rules.MinScore {
			violations = append(violations, Violation{
				Rule:    "min_score",
				Message: fmt.Sprintf("score %.1f%% below minimum %.1f%%", report.Summary.ScorePercent, *p.Rules.MinScore),
			})
		}
	}

	// forbid_categories
	if len(p.Rules.ForbidCategories) > 0 {
		forbidden := make(map[string]bool, len(p.Rules.ForbidCategories))
		for _, c := range p.Rules.ForbidCategories {
			forbidden[c] = true
		}
		for cat, count := range report.Summary.IssuesByCategory {
			if forbidden[cat] && count > 0 {
				violations = append(violations, Violation{
					Rule:    "forbid_categories",
					Message: fmt.Sprintf("forbidden category %q has %d issues", cat, count),
				})
			}
		}
	}

	// require_tools
	if len(p.Rules.RequireTools) > 0 {
		for _, tool := range p.Rules.RequireTools {
			if _, found := report.ToolReports[tool]; !found {
				violations = append(violations, Violation{
					Rule:    "require_tools",
					Message: fmt.Sprintf("required tool %q not found in report", tool),
				})
			}
		}
	}

	return &Result{
		Pass:       len(violations) == 0,
		Violations: violations,
	}
}

// EvaluateSLA checks SLA summary data against the SLA-related policy rules
// (max_open_critical_days, max_open_high_days). Returns a passing result if
// no SLA rules are configured or if summary is nil.
func (p *Policy) EvaluateSLA(summary *apiclient.SLASummary) *Result {
	if p == nil || summary == nil {
		return &Result{Pass: true}
	}

	hasSLARules := p.Rules.MaxOpenCriticalDays != nil || p.Rules.MaxOpenHighDays != nil
	if !hasSLARules {
		return &Result{Pass: true}
	}

	var violations []Violation

	if p.Rules.MaxOpenCriticalDays != nil {
		if days, ok := summary.MedianRemediationDays["critical"]; ok && days != nil {
			if int(*days) > *p.Rules.MaxOpenCriticalDays {
				violations = append(violations, Violation{
					Rule:    "max_open_critical_days",
					Message: fmt.Sprintf("median critical remediation %.0f days exceeds limit %d", *days, *p.Rules.MaxOpenCriticalDays),
				})
			}
		}
		if summary.OldestOpen != nil && summary.OldestOpen.OpenDays > *p.Rules.MaxOpenCriticalDays {
			// Check if oldest open finding is in the count (it's always relevant).
			if count, ok := summary.OpenFindings["critical"]; ok && count > 0 {
				violations = append(violations, Violation{
					Rule:    "max_open_critical_days",
					Message: fmt.Sprintf("oldest open critical finding is %d days old (limit %d)", summary.OldestOpen.OpenDays, *p.Rules.MaxOpenCriticalDays),
				})
			}
		}
	}

	if p.Rules.MaxOpenHighDays != nil {
		if days, ok := summary.MedianRemediationDays["high"]; ok && days != nil {
			if int(*days) > *p.Rules.MaxOpenHighDays {
				violations = append(violations, Violation{
					Rule:    "max_open_high_days",
					Message: fmt.Sprintf("median high remediation %.0f days exceeds limit %d", *days, *p.Rules.MaxOpenHighDays),
				})
			}
		}
	}

	return &Result{
		Pass:       len(violations) == 0,
		Violations: violations,
	}
}
