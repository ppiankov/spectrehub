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

func TestLoadFromFileInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("{{{{not yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFromFile(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadFromFileUnreadable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noperm.yaml")
	if err := os.WriteFile(path, []byte("version: 1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0644) })

	_, err := LoadFromFile(path)
	if err == nil {
		t.Error("expected error for unreadable file")
	}
}

func TestMaxHighFail(t *testing.T) {
	report := baseReport()
	report.Summary.IssuesBySeverity["high"] = 5
	p := &Policy{Rules: Rules{MaxHigh: intPtr(2)}}
	result := p.Evaluate(report)
	if result.Pass {
		t.Error("expected fail: 5 high exceeds limit 2")
	}
	if result.Violations[0].Rule != "max_high" {
		t.Errorf("expected max_high, got %s", result.Violations[0].Rule)
	}
}

func TestForbidCategoriesZeroCountNotReported(t *testing.T) {
	report := baseReport()
	report.Summary.IssuesByCategory["drift"] = 0
	p := &Policy{Rules: Rules{ForbidCategories: []string{"drift"}}}
	result := p.Evaluate(report)
	if !result.Pass {
		t.Errorf("expected pass (drift count=0), got violations: %v", result.Violations)
	}
}

func TestRequireToolsMultiple(t *testing.T) {
	p := &Policy{Rules: Rules{RequireTools: []string{"vaultspectre", "kafkaspectre"}}}
	result := p.Evaluate(baseReport())
	if result.Pass {
		t.Error("expected fail: kafkaspectre not in report")
	}
	foundKafka := false
	for _, v := range result.Violations {
		if v.Rule == "require_tools" && v.Message == `required tool "kafkaspectre" not found in report` {
			foundKafka = true
		}
	}
	if !foundKafka {
		t.Error("expected violation for kafkaspectre not found")
	}
}

func TestFindPolicyFile(t *testing.T) {
	// Create a temp directory with a policy file
	dir := t.TempDir()
	policyPath := filepath.Join(dir, ".spectrehub-policy.yaml")
	if err := os.WriteFile(policyPath, []byte("version: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Save and restore cwd
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	path := FindPolicyFile()
	if path == "" {
		t.Fatal("expected to find policy file")
	}
	if filepath.Base(path) != ".spectrehub-policy.yaml" {
		t.Errorf("unexpected path: %s", path)
	}
}

func TestFindPolicyFileYML(t *testing.T) {
	dir := t.TempDir()
	policyPath := filepath.Join(dir, ".spectrehub-policy.yml")
	if err := os.WriteFile(policyPath, []byte("version: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	path := FindPolicyFile()
	if path == "" {
		t.Fatal("expected to find policy file (.yml)")
	}
}

func TestFindPolicyFileNone(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	path := FindPolicyFile()
	if path != "" {
		t.Errorf("expected empty path, got %q", path)
	}
}

func TestLoadFromFileAllRules(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".spectrehub-policy.yaml")
	content := `version: "1"
rules:
  max_issues: 10
  max_critical: 0
  max_high: 5
  min_score: 80.0
  forbid_categories:
    - missing
    - drift
  require_tools:
    - vaultspectre
    - s3spectre
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	p, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if p.Rules.MaxHigh == nil || *p.Rules.MaxHigh != 5 {
		t.Errorf("expected max_high=5, got %v", p.Rules.MaxHigh)
	}
	if p.Rules.MaxCritical == nil || *p.Rules.MaxCritical != 0 {
		t.Errorf("expected max_critical=0, got %v", p.Rules.MaxCritical)
	}
	if len(p.Rules.ForbidCategories) != 2 {
		t.Errorf("expected 2 forbid_categories, got %d", len(p.Rules.ForbidCategories))
	}
	if len(p.Rules.RequireTools) != 2 {
		t.Errorf("expected 2 require_tools, got %d", len(p.Rules.RequireTools))
	}
}

func TestEvaluateNoRules(t *testing.T) {
	p := &Policy{Version: "1"}
	result := p.Evaluate(baseReport())
	if !result.Pass {
		t.Error("expected pass with no rules")
	}
	if len(result.Violations) != 0 {
		t.Errorf("expected no violations, got %d", len(result.Violations))
	}
}
