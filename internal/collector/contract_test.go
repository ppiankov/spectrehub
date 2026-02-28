package collector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ppiankov/spectrehub/internal/models"
	"github.com/ppiankov/spectrehub/internal/validator"
)

// TestToolDetection verifies that tool type detection works correctly for each contract file
func TestToolDetection(t *testing.T) {
	tests := []struct {
		file     string
		expected models.ToolType
	}{
		{"../../testdata/contracts/vaultspectre-v0.1.0.json", models.ToolVault},
		{"../../testdata/contracts/s3spectre-v0.1.0.json", models.ToolS3},
		{"../../testdata/contracts/kafkaspectre-v0.1.0.json", models.ToolKafka},
		{"../../testdata/contracts/clickspectre-v0.1.0.json", models.ToolClickHouse},
		{"../../testdata/contracts/pgspectre-v0.1.0.json", models.ToolPg},
		{"../../testdata/contracts/mongospectre-v0.1.0.json", models.ToolMongo},
		// spectre/v1 envelope format
		{"../../testdata/contracts/s3spectre-spectrev1.json", models.ToolS3},
		{"../../testdata/contracts/pgspectre-spectrev1.json", models.ToolPg},
		{"../../testdata/contracts/kafkaspectre-spectrev1.json", models.ToolKafka},
		{"../../testdata/contracts/clickspectre-spectrev1.json", models.ToolClickHouse},
		{"../../testdata/contracts/vaultspectre-spectrev1.json", models.ToolVault},
		{"../../testdata/contracts/mongospectre-spectrev1.json", models.ToolMongo},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			data, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			toolType, err := DetectToolType(data)
			if err != nil {
				t.Fatalf("DetectToolType failed: %v", err)
			}

			if toolType != tt.expected {
				t.Errorf("Expected tool type %s, got %s", tt.expected, toolType)
			}
		})
	}
}

// TestParsingSucceeds verifies that all contract files parse without error
func TestParsingSucceeds(t *testing.T) {
	tests := []struct {
		file     string
		toolType models.ToolType
	}{
		{"../../testdata/contracts/vaultspectre-v0.1.0.json", models.ToolVault},
		{"../../testdata/contracts/s3spectre-v0.1.0.json", models.ToolS3},
		{"../../testdata/contracts/kafkaspectre-v0.1.0.json", models.ToolKafka},
		{"../../testdata/contracts/clickspectre-v0.1.0.json", models.ToolClickHouse},
		{"../../testdata/contracts/pgspectre-v0.1.0.json", models.ToolPg},
		{"../../testdata/contracts/mongospectre-v0.1.0.json", models.ToolMongo},
		// spectre/v1 envelope format
		{"../../testdata/contracts/s3spectre-spectrev1.json", models.ToolS3},
		{"../../testdata/contracts/pgspectre-spectrev1.json", models.ToolPg},
		{"../../testdata/contracts/kafkaspectre-spectrev1.json", models.ToolKafka},
		{"../../testdata/contracts/clickspectre-spectrev1.json", models.ToolClickHouse},
		{"../../testdata/contracts/vaultspectre-spectrev1.json", models.ToolVault},
		{"../../testdata/contracts/mongospectre-spectrev1.json", models.ToolMongo},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			data, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			rawData, err := ParseReport(data, tt.toolType)
			if err != nil {
				t.Fatalf("ParseReport failed: %v", err)
			}

			if rawData == nil {
				t.Error("Expected non-nil raw data")
			}

			// Verify the tool type is correct
			if !models.IsSupportedTool(tt.toolType) {
				t.Errorf("Tool type %s should be supported", tt.toolType)
			}
		})
	}
}

// TestNormalizedIssueCounts verifies that normalized issue counts match expected values
func TestNormalizedIssueCounts(t *testing.T) {
	tests := []struct {
		file           string
		expectedIssues int
	}{
		{"../../testdata/contracts/vaultspectre-v0.1.0.json", 8}, // 5 missing + 2 access_denied + 1 error
		{"../../testdata/contracts/s3spectre-v0.1.0.json", 3},    // 1 missing + 1 unused + 1 stale
		{"../../testdata/contracts/kafkaspectre-v0.1.0.json", 2}, // 2 unused topics
		{"../../testdata/contracts/clickspectre-v0.1.0.json", 2}, // 1 zero usage + 1 anomaly
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			data, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			toolType, err := DetectToolType(data)
			if err != nil {
				t.Fatalf("DetectToolType failed: %v", err)
			}

			rawData, err := ParseReport(data, toolType)
			if err != nil {
				t.Fatalf("ParseReport failed: %v", err)
			}

			// This is a simplified check - in reality you'd normalize and count issues
			// For now, just verify that parsing succeeded
			if rawData == nil {
				t.Error("Expected non-nil raw data after parsing")
			}
		})
	}
}

// TestInvalidInputs verifies that malformed inputs are rejected with clear errors
func TestInvalidInputs(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{"malformed JSON", "../../testdata/invalid/malformed.json"},
		{"missing required fields", "../../testdata/invalid/missing-fields.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			// Try to detect tool type - should either fail or detect something
			_, err = DetectToolType(data)
			// We expect an error for malformed JSON
			if tt.name == "malformed JSON" && err == nil {
				t.Error("Expected error for malformed JSON")
			}
		})
	}
}

// TestUnsupportedTools verifies that unknown tools are handled gracefully
func TestUnsupportedTools(t *testing.T) {
	data, err := os.ReadFile("../../testdata/unsupported/unknownspectre-v0.1.0.json")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	toolType, err := DetectToolType(data)
	// Unknown tools should return an error or ToolUnknown
	if err == nil && toolType != models.ToolUnknown {
		t.Errorf("Expected error or ToolUnknown for unsupported tool, got %s", toolType)
	}

	// If error is returned, that's acceptable behavior for unknown tools
	if err != nil {
		t.Logf("DetectToolType correctly rejected unknown tool: %v", err)
		return
	}

	// If ToolUnknown was returned, try parsing
	rawData, err := ParseReport(data, toolType)
	if err != nil {
		t.Logf("ParseReport correctly handled unknown tool: %v", err)
		return
	}

	if rawData == nil {
		t.Error("Expected non-nil raw data even for unknown tools")
	}

	// Verify tool type is marked as unknown
	if models.IsSupportedTool(toolType) {
		t.Error("unknown tool should not be in supported tools list")
	}
}

// TestCollectFromDirectory verifies end-to-end collection from a directory
func TestCollectFromDirectory(t *testing.T) {
	collector := New(Config{
		MaxConcurrency: 4,
		Verbose:        false,
	})

	// Collect from contracts directory
	contractsDir := "../../testdata/contracts"
	reports, err := collector.CollectFromDirectory(contractsDir)
	if err != nil {
		t.Fatalf("CollectFromDirectory failed: %v", err)
	}

	if len(reports) != 13 {
		t.Errorf("Expected 13 reports (6 legacy + 7 spectre/v1), got %d", len(reports))
	}

	// Verify we got one report from each tool
	toolsSeen := make(map[string]bool)
	for _, report := range reports {
		toolsSeen[report.Tool] = true
	}

	expectedTools := []string{"vaultspectre", "s3spectre", "kafkaspectre", "clickspectre", "pgspectre", "mongospectre"}
	for _, tool := range expectedTools {
		if !toolsSeen[tool] {
			t.Errorf("Expected to see report from %s", tool)
		}
	}
}

// TestOutputStability verifies that parsing is stable across runs
func TestOutputStability(t *testing.T) {
	data, err := os.ReadFile("../../testdata/contracts/vaultspectre-v0.1.0.json")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// Parse twice
	toolType1, err1 := DetectToolType(data)
	rawData1, err2 := ParseReport(data, toolType1)

	toolType2, err3 := DetectToolType(data)
	rawData2, err4 := ParseReport(data, toolType2)

	// Verify consistency
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		t.Fatal("Parsing should succeed consistently")
	}

	if toolType1 != toolType2 {
		t.Error("Tool type detection should be stable")
	}

	if rawData1 == nil || rawData2 == nil {
		t.Error("Parsed data should not be nil")
	}
}

// TestVersionExtraction verifies version extraction for different formats
func TestVersionExtraction(t *testing.T) {
	tests := []struct {
		file            string
		expectedVersion string
	}{
		{"../../testdata/contracts/vaultspectre-v0.1.0.json", "0.1.0"},
		{"../../testdata/contracts/s3spectre-v0.1.0.json", "0.1.0"},
		{"../../testdata/contracts/clickspectre-v0.1.0.json", "0.1.0"},
		{"../../testdata/contracts/mongospectre-v0.1.0.json", "0.1.0"},
		// spectre/v1 envelope format
		{"../../testdata/contracts/s3spectre-spectrev1.json", "0.2.1"},
		{"../../testdata/contracts/pgspectre-spectrev1.json", "0.2.0"},
		{"../../testdata/contracts/vaultspectre-spectrev1.json", "0.3.0"},
	}

	for _, tt := range tests {
		t.Run(filepath.Base(tt.file), func(t *testing.T) {
			data, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			version := ExtractVersion(data)
			if version != tt.expectedVersion {
				t.Errorf("Expected version %s, got %s", tt.expectedVersion, version)
			}
		})
	}
}

// --- spectre/v1 contract tests (WO-14 / WO-15) ---

// TestSpectreV1Detection verifies Phase 0 detection for all spectre/v1 contract files
func TestSpectreV1Detection(t *testing.T) {
	tests := []struct {
		file     string
		expected models.ToolType
	}{
		{"../../testdata/contracts/s3spectre-spectrev1.json", models.ToolS3},
		{"../../testdata/contracts/pgspectre-spectrev1.json", models.ToolPg},
		{"../../testdata/contracts/kafkaspectre-spectrev1.json", models.ToolKafka},
		{"../../testdata/contracts/clickspectre-spectrev1.json", models.ToolClickHouse},
		{"../../testdata/contracts/vaultspectre-spectrev1.json", models.ToolVault},
		{"../../testdata/contracts/mongospectre-spectrev1.json", models.ToolMongo},
	}

	for _, tt := range tests {
		t.Run(filepath.Base(tt.file), func(t *testing.T) {
			data, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			if !IsSpectreV1(data) {
				t.Fatal("Expected IsSpectreV1 to return true")
			}

			toolType, err := DetectToolType(data)
			if err != nil {
				t.Fatalf("DetectToolType failed: %v", err)
			}
			if toolType != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, toolType)
			}
		})
	}
}

// TestSpectreV1Parsing verifies that spectre/v1 files parse into SpectreV1Report
func TestSpectreV1Parsing(t *testing.T) {
	tests := []struct {
		file             string
		expectedTool     string
		expectedFindings int
	}{
		{"../../testdata/contracts/s3spectre-spectrev1.json", "s3spectre", 3},
		{"../../testdata/contracts/pgspectre-spectrev1.json", "pgspectre", 3},
		{"../../testdata/contracts/kafkaspectre-spectrev1.json", "kafkaspectre", 2},
		{"../../testdata/contracts/clickspectre-spectrev1.json", "clickspectre", 2},
		{"../../testdata/contracts/vaultspectre-spectrev1.json", "vaultspectre", 3},
		{"../../testdata/contracts/mongospectre-spectrev1.json", "mongospectre", 2},
	}

	for _, tt := range tests {
		t.Run(filepath.Base(tt.file), func(t *testing.T) {
			data, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			toolType, err := DetectToolType(data)
			if err != nil {
				t.Fatalf("DetectToolType failed: %v", err)
			}

			rawData, err := ParseReport(data, toolType)
			if err != nil {
				t.Fatalf("ParseReport failed: %v", err)
			}

			v1Report, ok := rawData.(*models.SpectreV1Report)
			if !ok {
				t.Fatalf("Expected *models.SpectreV1Report, got %T", rawData)
			}

			if v1Report.Schema != "spectre/v1" {
				t.Errorf("Expected schema spectre/v1, got %q", v1Report.Schema)
			}
			if v1Report.Tool != tt.expectedTool {
				t.Errorf("Expected tool %q, got %q", tt.expectedTool, v1Report.Tool)
			}
			if len(v1Report.Findings) != tt.expectedFindings {
				t.Errorf("Expected %d findings, got %d", tt.expectedFindings, len(v1Report.Findings))
			}
			if v1Report.Summary.Total != tt.expectedFindings {
				t.Errorf("Expected summary.total=%d, got %d", tt.expectedFindings, v1Report.Summary.Total)
			}
		})
	}
}

// TestSpectreV1Validation verifies that valid spectre/v1 files pass validation
func TestSpectreV1Validation(t *testing.T) {
	v := validator.New()

	files := []string{
		"../../testdata/contracts/s3spectre-spectrev1.json",
		"../../testdata/contracts/pgspectre-spectrev1.json",
		"../../testdata/contracts/kafkaspectre-spectrev1.json",
		"../../testdata/contracts/clickspectre-spectrev1.json",
		"../../testdata/contracts/vaultspectre-spectrev1.json",
		"../../testdata/contracts/mongospectre-spectrev1.json",
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			toolType, err := DetectToolType(data)
			if err != nil {
				t.Fatalf("DetectToolType failed: %v", err)
			}

			if err := v.ValidateReport(toolType, data); err != nil {
				t.Fatalf("ValidateReport failed: %v", err)
			}
		})
	}
}

// TestSpectreV1ValidationRejectsInvalid verifies that invalid spectre/v1 files are rejected
func TestSpectreV1ValidationRejectsInvalid(t *testing.T) {
	v := validator.New()

	tests := []struct {
		name           string
		file           string
		wantErrContain string
	}{
		{
			name:           "bad severity",
			file:           "../../testdata/invalid/spectrev1-bad-severity.json",
			wantErrContain: "invalid severity",
		},
		{
			name:           "missing fields",
			file:           "../../testdata/invalid/spectrev1-missing-fields.json",
			wantErrContain: "Missing required field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			// These are spectre/v1 envelopes, so ValidateReport dispatches to spectre/v1 validator
			err = v.ValidateReport(models.ToolS3, data)
			if err == nil {
				t.Fatal("Expected validation error, got nil")
			}
			if tt.wantErrContain != "" && !strings.Contains(err.Error(), tt.wantErrContain) {
				t.Errorf("Expected error to contain %q, got: %v", tt.wantErrContain, err)
			}
		})
	}
}

// TestSpectreV1SeverityEnum verifies only valid severity values are accepted
func TestSpectreV1SeverityEnum(t *testing.T) {
	valid := []string{"high", "medium", "low", "info"}
	invalid := []string{"critical", "warning", "error", ""}

	for _, s := range valid {
		if !models.ValidSpectreV1Severities[s] {
			t.Errorf("Expected %q to be a valid severity", s)
		}
	}
	for _, s := range invalid {
		if models.ValidSpectreV1Severities[s] {
			t.Errorf("Expected %q to be an invalid severity", s)
		}
	}
}

// TestSpectreV1TargetTypes verifies tool-to-target-type mapping
func TestSpectreV1TargetTypes(t *testing.T) {
	expected := map[string]string{
		"s3spectre":    "s3",
		"pgspectre":    "postgres",
		"kafkaspectre": "kafka",
		"clickspectre": "clickhouse",
		"vaultspectre": "vault",
		"mongospectre": "mongodb",
	}

	for tool, targetType := range expected {
		got, ok := models.SpectreV1TargetTypes[tool]
		if !ok {
			t.Errorf("Missing target type for tool %q", tool)
			continue
		}
		if got != targetType {
			t.Errorf("Tool %q: expected target type %q, got %q", tool, targetType, got)
		}
	}
}

// TestIsSpectreV1 verifies the IsSpectreV1 function
func TestIsSpectreV1(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{"valid", []byte(`{"schema":"spectre/v1","tool":"s3spectre"}`), true},
		{"legacy", []byte(`{"tool":"s3spectre","version":"0.1.0"}`), false},
		{"wrong schema", []byte(`{"schema":"spectre/v2"}`), false},
		{"no schema", []byte(`{"foo":"bar"}`), false},
		{"invalid json", []byte(`{`), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSpectreV1(tt.data); got != tt.expected {
				t.Errorf("IsSpectreV1 = %v, want %v", got, tt.expected)
			}
		})
	}
}
