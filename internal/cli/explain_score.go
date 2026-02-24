package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ppiankov/spectrehub/internal/models"
	"github.com/ppiankov/spectrehub/internal/storage"
	"github.com/spf13/cobra"
)

var explainFormat string

var explainScoreCmd = &cobra.Command{
	Use:   "explain-score",
	Short: "Show the health score formula step by step",
	Long: `Explain-score loads the latest stored report and shows exactly how
the health score was calculated:

  1. Per-tool resource counts (total resources)
  2. Distinct affected resources (resources with at least one issue)
  3. The formula: score = (total - affected) / total * 100
  4. The health level thresholds

This command requires a previous run stored with --store.`,
	RunE: runExplainScore,
}

func init() {
	explainScoreCmd.Flags().StringVar(&explainFormat, "format", "text",
		"output format: text or json")
}

// explainResult holds the structured explanation.
type explainResult struct {
	PerTool          []toolContribution `json:"per_tool"`
	TotalResources   int                `json:"total_resources"`
	AffectedList     []string           `json:"affected_resources"`
	AffectedCount    int                `json:"affected_count"`
	Score            float64            `json:"score"`
	Health           string             `json:"health"`
	Formula          string             `json:"formula"`
	Thresholds       []threshold        `json:"thresholds"`
	IssuesBySeverity map[string]int     `json:"issues_by_severity"`
}

type toolContribution struct {
	Tool      string `json:"tool"`
	Resources int    `json:"resources"`
	Issues    int    `json:"issues"`
	Affected  int    `json:"affected"`
}

type threshold struct {
	Min   float64 `json:"min"`
	Label string  `json:"label"`
}

func runExplainScore(cmd *cobra.Command, args []string) error {
	storageDir := cfg.StorageDir
	if storageDir == "" {
		storageDir = ".spectre"
	}

	storagePath, err := getStoragePath(storageDir)
	if err != nil {
		return fmt.Errorf("failed to resolve storage path: %w", err)
	}

	store := storage.NewLocal(storagePath)
	report, err := store.GetLatestRun()
	if err != nil {
		return fmt.Errorf("no stored runs found. Run 'spectrehub run --store' first: %w", err)
	}

	result := buildExplanation(report)

	if explainFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	return writeExplainText(result)
}

func buildExplanation(report *models.AggregatedReport) explainResult {
	result := explainResult{
		Thresholds: []threshold{
			{Min: 95, Label: "excellent"},
			{Min: 85, Label: "good"},
			{Min: 70, Label: "warning"},
			{Min: 50, Label: "critical"},
			{Min: 0, Label: "severe"},
		},
		IssuesBySeverity: report.Summary.IssuesBySeverity,
	}

	// Count resources per tool (same logic as aggregator.calculateHealthScore)
	totalResources := 0
	toolIssues := make(map[string]int)
	for _, issue := range report.Issues {
		toolIssues[issue.Tool]++
	}

	// Count affected resources per tool
	affectedByTool := make(map[string]map[string]bool)
	for _, issue := range report.Issues {
		if issue.Resource == "" {
			continue
		}
		if affectedByTool[issue.Tool] == nil {
			affectedByTool[issue.Tool] = make(map[string]bool)
		}
		affectedByTool[issue.Tool][issue.Resource] = true
	}

	for _, toolReport := range report.ToolReports {
		if !toolReport.IsSupported {
			continue
		}

		resources := countToolResources(toolReport)
		totalResources += resources

		tc := toolContribution{
			Tool:      toolReport.Tool,
			Resources: resources,
			Issues:    toolIssues[toolReport.Tool],
		}
		if m, ok := affectedByTool[toolReport.Tool]; ok {
			tc.Affected = len(m)
		}
		result.PerTool = append(result.PerTool, tc)
	}

	// Sort by tool name for deterministic output
	sort.Slice(result.PerTool, func(i, j int) bool {
		return result.PerTool[i].Tool < result.PerTool[j].Tool
	})

	// Count total distinct affected resources
	allAffected := make(map[string]bool)
	for _, issue := range report.Issues {
		if issue.Resource != "" {
			allAffected[issue.Resource] = true
		}
	}

	affectedList := make([]string, 0, len(allAffected))
	for r := range allAffected {
		affectedList = append(affectedList, r)
	}
	sort.Strings(affectedList)

	result.TotalResources = totalResources
	result.AffectedCount = len(allAffected)
	result.AffectedList = affectedList
	result.Health = report.Summary.HealthScore
	result.Score = report.Summary.ScorePercent
	result.Formula = fmt.Sprintf("(%d - %d) / %d * 100 = %.1f",
		totalResources, len(allAffected), totalResources, report.Summary.ScorePercent)

	return result
}

// countToolResources extracts the total resource count from a tool report.
// Mirrors aggregator.calculateHealthScore logic exactly.
func countToolResources(toolReport models.ToolReport) int {
	switch models.ToolType(toolReport.Tool) {
	case models.ToolVault:
		if vr, ok := toolReport.RawData.(*models.VaultReport); ok {
			return vr.Summary.TotalReferences
		}
	case models.ToolS3:
		if sr, ok := toolReport.RawData.(*models.S3Report); ok {
			return sr.Summary.TotalBuckets
		}
	case models.ToolKafka:
		if kr, ok := toolReport.RawData.(*models.KafkaReport); ok {
			if kr.Summary != nil {
				return kr.Summary.TotalTopics
			}
		}
	case models.ToolClickHouse:
		if cr, ok := toolReport.RawData.(*models.ClickHouseReport); ok {
			return len(cr.Tables)
		}
	case models.ToolPg:
		if pr, ok := toolReport.RawData.(*models.PgReport); ok {
			return pr.Scanned.Tables
		}
	}
	return 0
}

func writeExplainText(result explainResult) error {
	fmt.Println("Health Score Breakdown")
	fmt.Println("======================")
	fmt.Println()

	// Step 1: Per-tool resources
	fmt.Println("1. Resources per tool:")
	for _, tc := range result.PerTool {
		fmt.Printf("   %-14s  %d resources, %d issues, %d affected\n",
			tc.Tool, tc.Resources, tc.Issues, tc.Affected)
	}
	fmt.Printf("   %-14s  %d resources total\n", "", result.TotalResources)
	fmt.Println()

	// Step 2: Affected resources
	fmt.Printf("2. Affected resources: %d distinct\n", result.AffectedCount)
	if len(result.AffectedList) <= 20 {
		for _, r := range result.AffectedList {
			fmt.Printf("   - %s\n", r)
		}
	} else {
		for _, r := range result.AffectedList[:15] {
			fmt.Printf("   - %s\n", r)
		}
		fmt.Printf("   ... +%d more\n", len(result.AffectedList)-15)
	}
	fmt.Println()

	// Step 3: Formula
	fmt.Println("3. Formula:")
	fmt.Printf("   score = (total - affected) / total * 100\n")
	fmt.Printf("   score = %s\n", result.Formula)
	fmt.Println()

	// Step 4: Thresholds
	fmt.Println("4. Thresholds:")
	for _, t := range result.Thresholds {
		marker := "  "
		if strings.EqualFold(result.Health, t.Label) {
			marker = "→ "
		}
		fmt.Printf("   %s≥ %.0f%%  %s\n", marker, t.Min, t.Label)
	}
	fmt.Println()

	// Step 5: Issues by severity
	if len(result.IssuesBySeverity) > 0 {
		fmt.Println("5. Issues by severity:")
		for _, sev := range []string{"critical", "high", "medium", "low"} {
			if count, ok := result.IssuesBySeverity[sev]; ok {
				fmt.Printf("   %-10s  %d\n", sev, count)
			}
		}
		fmt.Println()
	}

	fmt.Printf("Result: %s (%.1f%%)\n", strings.ToUpper(result.Health), result.Score)
	return nil
}
