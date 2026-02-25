package collector

import (
	"encoding/json"
	"fmt"

	"github.com/ppiankov/spectrehub/internal/models"
)

// IsSpectreV1 returns true if the JSON data contains a spectre/v1 schema field.
func IsSpectreV1(data []byte) bool {
	var schemaField struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(data, &schemaField); err == nil && schemaField.Schema == "spectre/v1" {
		return true
	}
	return false
}

// DetectToolType identifies which Spectre tool produced the JSON data
// It uses a three-phase approach:
// 0. Check for spectre/v1 envelope (schema: "spectre/v1")
// 1. Check for explicit "tool" field
// 2. Fallback to structural analysis
func DetectToolType(data []byte) (models.ToolType, error) {
	// Phase 0: Check for spectre/v1 envelope
	if IsSpectreV1(data) {
		var envelope struct {
			Tool string `json:"tool"`
		}
		if err := json.Unmarshal(data, &envelope); err == nil && envelope.Tool != "" {
			return mapToolName(envelope.Tool)
		}
		return models.ToolUnknown, fmt.Errorf("spectre/v1 envelope missing tool field")
	}

	// Phase 1: Try to find explicit tool field
	var toolField struct {
		Tool string `json:"tool"`
	}

	if err := json.Unmarshal(data, &toolField); err == nil && toolField.Tool != "" {
		return mapToolName(toolField.Tool)
	}

	// Phase 1b: Try tool field inside metadata
	var metadataToolField struct {
		Metadata struct {
			Tool string `json:"tool"`
		} `json:"metadata"`
	}

	if err := json.Unmarshal(data, &metadataToolField); err == nil && metadataToolField.Metadata.Tool != "" {
		switch metadataToolField.Metadata.Tool {
		case "pgspectre":
			return models.ToolPg, nil
		}
	}

	// Phase 2: Structural analysis (fallback for tools without "tool" field)
	return detectByStructure(data)
}

// mapToolName converts a tool name string to the corresponding ToolType.
func mapToolName(name string) (models.ToolType, error) {
	switch name {
	case "vaultspectre":
		return models.ToolVault, nil
	case "s3spectre":
		return models.ToolS3, nil
	case "kafkaspectre":
		return models.ToolKafka, nil
	case "clickspectre":
		return models.ToolClickHouse, nil
	case "pgspectre":
		return models.ToolPg, nil
	case "mongospectre":
		return models.ToolMongo, nil
	case "awsspectre":
		return models.ToolAWS, nil
	case "iamspectre":
		return models.ToolIAM, nil
	case "gcsspectre":
		return models.ToolGCS, nil
	default:
		return models.ToolUnknown, fmt.Errorf("unknown tool: %s", name)
	}
}

// detectByStructure uses JSON structure to identify the tool
func detectByStructure(data []byte) (models.ToolType, error) {
	// Parse as generic map to inspect structure
	var structure map[string]interface{}
	if err := json.Unmarshal(data, &structure); err != nil {
		return models.ToolUnknown, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Check for KafkaSpectre indicators
	if hasKey(structure, "summary") && hasKey(structure, "unused_topics") {
		// Could be Kafka
		if summary, ok := structure["summary"].(map[string]interface{}); ok {
			if hasKey(summary, "cluster_name") && hasKey(summary, "total_brokers") {
				return models.ToolKafka, nil
			}
		}
	}

	// Check for ClickSpectre indicators
	if hasKey(structure, "metadata") && hasKey(structure, "tables") && hasKey(structure, "cleanup_recommendations") {
		// Likely ClickSpectre
		if metadata, ok := structure["metadata"].(map[string]interface{}); ok {
			if hasKey(metadata, "clickhouse_host") {
				return models.ToolClickHouse, nil
			}
		}
	}

	// Check for VaultSpectre indicators
	if hasKey(structure, "secrets") && hasKey(structure, "summary") {
		// Could be Vault
		if summary, ok := structure["summary"].(map[string]interface{}); ok {
			if hasKey(summary, "status_missing") && hasKey(summary, "status_ok") {
				return models.ToolVault, nil
			}
		}
	}

	// Check for S3Spectre indicators
	if hasKey(structure, "buckets") && hasKey(structure, "summary") {
		// Could be S3
		if summary, ok := structure["summary"].(map[string]interface{}); ok {
			if hasKey(summary, "total_buckets") || hasKey(summary, "missing_buckets") {
				return models.ToolS3, nil
			}
		}
	}

	// Check for PgSpectre indicators
	if hasKey(structure, "metadata") && hasKey(structure, "findings") && hasKey(structure, "summary") {
		if metadata, ok := structure["metadata"].(map[string]interface{}); ok {
			if tool, ok := metadata["tool"].(string); ok && tool == "pgspectre" {
				return models.ToolPg, nil
			}
		}
		if scanned, ok := structure["scanned"].(map[string]interface{}); ok {
			if hasKey(scanned, "tables") && hasKey(scanned, "indexes") {
				return models.ToolPg, nil
			}
		}
	}

	// Check for MongoSpectre indicators
	if hasKey(structure, "metadata") && hasKey(structure, "findings") && hasKey(structure, "summary") {
		if metadata, ok := structure["metadata"].(map[string]interface{}); ok {
			if hasKey(metadata, "mongodbVersion") || hasKey(metadata, "uriHash") || hasKey(metadata, "repoPath") {
				return models.ToolMongo, nil
			}
		}
		if findings, ok := structure["findings"].([]interface{}); ok && len(findings) > 0 {
			if first, ok := findings[0].(map[string]interface{}); ok {
				if hasKey(first, "database") && hasKey(first, "collection") {
					return models.ToolMongo, nil
				}
			}
		}
	}

	return models.ToolUnknown, fmt.Errorf("unable to detect tool type from structure")
}

// hasKey checks if a key exists in a map
func hasKey(m map[string]interface{}, key string) bool {
	_, exists := m[key]
	return exists
}

// ValidateToolType checks if detected tool type is supported
func ValidateToolType(toolType models.ToolType) error {
	if toolType == models.ToolUnknown {
		return fmt.Errorf("unknown tool type")
	}

	if !models.IsSupportedTool(toolType) {
		return fmt.Errorf("tool type '%s' is not supported", toolType)
	}

	return nil
}

// GetToolName returns the human-readable name for a tool type
func GetToolName(toolType models.ToolType) string {
	if info, ok := models.GetToolInfo(toolType); ok {
		return info.Name
	}
	return string(toolType)
}
