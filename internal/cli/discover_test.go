package cli

import (
	"strings"
	"testing"

	"github.com/ppiankov/spectrehub/internal/discovery"
	"github.com/ppiankov/spectrehub/internal/models"
)

func TestPrintDiscoveryTextReady(t *testing.T) {
	plan := &discovery.DiscoveryPlan{
		Tools: []discovery.ToolDiscovery{
			{
				Tool:       models.ToolVault,
				Binary:     "vaultspectre",
				BinaryPath: "/usr/local/bin/vaultspectre",
				Available:  true,
				HasTarget:  true,
				Runnable:   true,
				EnvVars: []discovery.EnvVarStatus{
					{Name: "VAULT_ADDR", Set: true},
				},
			},
		},
		TotalFound:    1,
		TotalRunnable: 1,
	}

	output := captureStdout(t, func() {
		printDiscoveryText(plan)
	})

	if !strings.Contains(output, "1 tool(s), 1 runnable") {
		t.Error("missing summary line")
	}
	if !strings.Contains(output, "✓ ready") {
		t.Error("missing ready status")
	}
	if !strings.Contains(output, "vaultspectre") {
		t.Error("missing tool name")
	}
	if !strings.Contains(output, "VAULT_ADDR") {
		t.Error("missing env var name")
	}
}

func TestPrintDiscoveryTextNoTarget(t *testing.T) {
	plan := &discovery.DiscoveryPlan{
		Tools: []discovery.ToolDiscovery{
			{
				Tool:       models.ToolS3,
				Binary:     "s3spectre",
				BinaryPath: "/usr/local/bin/s3spectre",
				Available:  true,
				HasTarget:  false,
				Runnable:   false,
				EnvVars: []discovery.EnvVarStatus{
					{Name: "AWS_PROFILE", Set: false},
				},
			},
		},
		TotalFound:    1,
		TotalRunnable: 0,
	}

	output := captureStdout(t, func() {
		printDiscoveryText(plan)
	})

	if !strings.Contains(output, "○ installed (no target)") {
		t.Error("missing no-target status")
	}
	if !strings.Contains(output, "AWS_PROFILE") {
		t.Error("missing env var in missing section")
	}
	if !strings.Contains(output, "No runnable tools found") {
		t.Error("missing no-runnable message")
	}
}

func TestPrintDiscoveryTextNotFound(t *testing.T) {
	plan := &discovery.DiscoveryPlan{
		Tools: []discovery.ToolDiscovery{
			{
				Tool:      models.ToolKafka,
				Binary:    "kafkaspectre",
				Available: false,
				Runnable:  false,
			},
		},
		TotalFound:    0,
		TotalRunnable: 0,
	}

	output := captureStdout(t, func() {
		printDiscoveryText(plan)
	})

	if !strings.Contains(output, "✗ not found") {
		t.Error("missing not-found status")
	}
}
