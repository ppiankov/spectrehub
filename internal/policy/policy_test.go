package policy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ppiankov/spectrehub/internal/models"
)

func intPtr(v int) *int           { return &v }
func floatPtr(v float64) *float64 { return &v }

func baseReport() *models.AggregatedReport {
	return &models.AggregatedReport{
		Issues: []models.NormalizedIssue{
			{Tool: "vaultspectre", Category: "missing", Severity: "critical", Resource: "secret/db"},
			{Tool: "s3spectre", Category: "unused", Severity: "low", Resource: "s3://bucket"},
		},
		ToolReports: map[string]models.ToolReport{
			"vaultspectre": {Tool: "vaultspectre"},
			"s3spectre":    {Tool: "s3spectre"},
		},
		Summary: models.CrossToolSummary{
			TotalIssues:      2,
			IssuesBySeverity: map[string]int{"critical": 1, "low": 1},
			IssuesByCategory: map[string]int{"missing": 1, "unused": 1},
			ScorePercent:     75.0,
			HealthScore:      "warning",
			TotalTools:       2,
		},
	}
}

func TestEvaluateNilPolicy(t *testing.T) {
	var p *Policy
	result := p.Evaluate(baseReport())
	if !result.Pass {
		t.Error("nil policy should pass")
	}
}

func TestMaxIssuesPass(t *testing.T) {
	p := &Policy{Rules: Rules{MaxIssues: intPtr(5)}}
	result := p.Evaluate(baseReport())
	if !result.Pass {
		t.Errorf("expected pass, got violations: %v", result.Violations)
	}
}

func TestMaxIssuesFail(t *testing.T) {
	p := &Policy{Rules: Rules{MaxIssues: intPtr(1)}}
	result := p.Evaluate(baseReport())
	if result.Pass {
		t.Error("expected fail: 2 issues exceeds limit 1")
	}
	if len(result.Violations) != 1 || result.Violations[0].Rule != "max_issues" {
		t.Errorf("expected max_issues violation, got %v", result.Violations)
	}
}

func TestMaxCriticalPass(t *testing.T) {
	p := &Policy{Rules: Rules{MaxCritical: intPtr(1)}}
	result := p.Evaluate(baseReport())
	if !result.Pass {
		t.Errorf("expected pass, got violations: %v", result.Violations)
	}
}

func TestMaxCriticalFail(t *testing.T) {
	p := &Policy{Rules: Rules{MaxCritical: intPtr(0)}}
	result := p.Evaluate(baseReport())
	if result.Pass {
		t.Error("expected fail: 1 critical exceeds limit 0")
	}
	if result.Violations[0].Rule != "max_critical" {
		t.Errorf("expected max_critical, got %s", result.Violations[0].Rule)
	}
}

func TestMaxHighPass(t *testing.T) {
	p := &Policy{Rules: Rules{MaxHigh: intPtr(0)}}
	result := p.Evaluate(baseReport())
	if !result.Pass {
		t.Errorf("expected pass (0 high issues), got violations: %v", result.Violations)
	}
}

func TestMinScorePass(t *testing.T) {
	p := &Policy{Rules: Rules{MinScore: floatPtr(70.0)}}
	result := p.Evaluate(baseReport())
	if !result.Pass {
		t.Errorf("expected pass (75 >= 70), got violations: %v", result.Violations)
	}
}

func TestMinScoreFail(t *testing.T) {
	p := &Policy{Rules: Rules{MinScore: floatPtr(80.0)}}
	result := p.Evaluate(baseReport())
	if result.Pass {
		t.Error("expected fail: 75 < 80")
	}
	if result.Violations[0].Rule != "min_score" {
		t.Errorf("expected min_score, got %s", result.Violations[0].Rule)
	}
}

func TestForbidCategoriesFail(t *testing.T) {
	p := &Policy{Rules: Rules{ForbidCategories: []string{"missing"}}}
	result := p.Evaluate(baseReport())
	if result.Pass {
		t.Error("expected fail: missing category is forbidden")
	}
}

func TestForbidCategoriesPass(t *testing.T) {
	p := &Policy{Rules: Rules{ForbidCategories: []string{"drift"}}}
	result := p.Evaluate(baseReport())
	if !result.Pass {
		t.Errorf("expected pass (no drift issues), got violations: %v", result.Violations)
	}
}

func TestRequireToolsPass(t *testing.T) {
	p := &Policy{Rules: Rules{RequireTools: []string{"vaultspectre"}}}
	result := p.Evaluate(baseReport())
	if !result.Pass {
		t.Errorf("expected pass, got violations: %v", result.Violations)
	}
}

func TestRequireToolsFail(t *testing.T) {
	p := &Policy{Rules: Rules{RequireTools: []string{"kafkaspectre"}}}
	result := p.Evaluate(baseReport())
	if result.Pass {
		t.Error("expected fail: kafkaspectre not in report")
	}
}

func TestMultipleViolations(t *testing.T) {
	p := &Policy{
		Rules: Rules{
			MaxIssues:   intPtr(0),
			MaxCritical: intPtr(0),
			MinScore:    floatPtr(90.0),
		},
	}
	result := p.Evaluate(baseReport())
	if result.Pass {
		t.Error("expected fail")
	}
	if len(result.Violations) != 3 {
		t.Errorf("expected 3 violations, got %d: %v", len(result.Violations), result.Violations)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".spectrehub-policy.yaml")

	content := `version: "1"
rules:
  max_issues: 10
  max_critical: 0
  min_score: 80.0
  forbid_categories:
    - missing
  require_tools:
    - vaultspectre
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	p, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if p == nil {
		t.Fatal("expected policy, got nil")
	}
	if p.Version != "1" {
		t.Errorf("expected version 1, got %s", p.Version)
	}
	if p.Rules.MaxIssues == nil || *p.Rules.MaxIssues != 10 {
		t.Errorf("expected max_issues 10, got %v", p.Rules.MaxIssues)
	}
	if p.Rules.MinScore == nil || *p.Rules.MinScore != 80.0 {
		t.Errorf("expected min_score 80, got %v", p.Rules.MinScore)
	}
	if len(p.Rules.ForbidCategories) != 1 || p.Rules.ForbidCategories[0] != "missing" {
		t.Errorf("expected forbid missing, got %v", p.Rules.ForbidCategories)
	}
}

func TestLoadFromFileNotFound(t *testing.T) {
	p, err := LoadFromFile("/nonexistent/path")
	if err != nil {
		t.Errorf("expected nil error for missing file, got %v", err)
	}
	if p != nil {
		t.Error("expected nil policy for missing file")
	}
}
