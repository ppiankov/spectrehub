package collector

import (
	"encoding/json"
	"testing"

	"github.com/ppiankov/spectrehub/internal/models"
)

func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	return data
}

func TestDetectToolTypeToolField(t *testing.T) {
	tests := []struct {
		name      string
		toolValue string
		expected  models.ToolType
		wantErr   bool
	}{
		{"vault", "vaultspectre", models.ToolVault, false},
		{"s3", "s3spectre", models.ToolS3, false},
		{"kafka", "kafkaspectre", models.ToolKafka, false},
		{"clickhouse", "clickspectre", models.ToolClickHouse, false},
		{"unknown", "unknownspectre", models.ToolUnknown, true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			data := mustJSON(t, map[string]interface{}{"tool": tt.toolValue})
			got, err := DetectToolType(data)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if got != models.ToolUnknown {
					t.Fatalf("expected ToolUnknown, got %s", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestDetectToolTypeByStructure(t *testing.T) {
	tests := []struct {
		name     string
		payload  map[string]interface{}
		expected models.ToolType
	}{
		{
			name: "kafka",
			payload: map[string]interface{}{
				"summary": map[string]interface{}{
					"cluster_name":  "c1",
					"total_brokers": 3,
				},
				"unused_topics": []interface{}{},
			},
			expected: models.ToolKafka,
		},
		{
			name: "clickhouse",
			payload: map[string]interface{}{
				"metadata": map[string]interface{}{
					"clickhouse_host": "ch1",
				},
				"tables":                  []interface{}{},
				"cleanup_recommendations": map[string]interface{}{},
			},
			expected: models.ToolClickHouse,
		},
		{
			name: "vault",
			payload: map[string]interface{}{
				"secrets": map[string]interface{}{},
				"summary": map[string]interface{}{
					"status_missing": 1,
					"status_ok":      2,
				},
			},
			expected: models.ToolVault,
		},
		{
			name: "s3",
			payload: map[string]interface{}{
				"buckets": map[string]interface{}{},
				"summary": map[string]interface{}{
					"total_buckets": 1,
				},
			},
			expected: models.ToolS3,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			data := mustJSON(t, tt.payload)
			got, err := DetectToolType(data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestDetectToolTypeInvalidJSON(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"invalid", []byte("{")},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if _, err := DetectToolType(tt.data); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestDetectToolTypeUnknownStructure(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"unknown", mustJSON(t, map[string]interface{}{"foo": "bar"})},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if _, err := DetectToolType(tt.data); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestHasKey(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		key      string
		expected bool
	}{
		{"present", map[string]interface{}{"a": 1}, "a", true},
		{"missing", map[string]interface{}{"a": 1}, "b", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := hasKey(tt.input, tt.key); got != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestValidateToolType(t *testing.T) {
	tests := []struct {
		name    string
		tool    models.ToolType
		wantErr bool
	}{
		{"unknown", models.ToolUnknown, true},
		{"unsupported", models.ToolType("customspectre"), true},
		{"supported", models.ToolVault, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToolType(tt.tool)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetToolName(t *testing.T) {
	tests := []struct {
		name     string
		tool     models.ToolType
		expected string
	}{
		{"supported", models.ToolVault, "vaultspectre"},
		{"unknown", models.ToolType("customspectre"), "customspectre"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := GetToolName(tt.tool); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
