package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ppiankov/spectrehub/internal/discovery"
	"github.com/ppiankov/spectrehub/internal/models"
)

// DefaultTimeout is the per-tool execution timeout.
const DefaultTimeout = 5 * time.Minute

// ExecFunc is the signature for running a command and capturing stdout.
// It receives the context, binary path, and args. Returns stdout bytes and error.
type ExecFunc func(ctx context.Context, name string, args ...string) ([]byte, error)

// RunConfig describes a single tool invocation.
type RunConfig struct {
	Tool       models.ToolType
	Binary     string
	Subcommand string
	JSONFlag   string
	ExtraArgs  []string
	Timeout    time.Duration
}

// RunResult is the outcome of a single tool invocation.
type RunResult struct {
	Tool       models.ToolType `json:"tool"`
	Binary     string          `json:"binary"`
	OutputFile string          `json:"output_file,omitempty"`
	Duration   time.Duration   `json:"duration"`
	Success    bool            `json:"success"`
	Error      string          `json:"error,omitempty"`
}

// Runner executes spectre tools and captures their JSON output.
type Runner struct {
	execFn  ExecFunc
	tempDir string
}

// New creates a Runner with the given exec function.
// The temp directory is created lazily on first Run call.
func New(execFn ExecFunc) *Runner {
	return &Runner{
		execFn: execFn,
	}
}

// Run executes each tool sequentially, capturing output to temp JSON files.
// Partial success: returns all results even if some tools fail.
func (r *Runner) Run(ctx context.Context, configs []RunConfig) []RunResult {
	if r.tempDir == "" {
		dir, err := os.MkdirTemp("", "spectrehub-run-*")
		if err != nil {
			// If we can't create temp dir, fail all
			var results []RunResult
			for _, cfg := range configs {
				results = append(results, RunResult{
					Tool:    cfg.Tool,
					Binary:  cfg.Binary,
					Success: false,
					Error:   fmt.Sprintf("failed to create temp directory: %v", err),
				})
			}
			return results
		}
		r.tempDir = dir
	}

	var results []RunResult
	for _, cfg := range configs {
		result := r.runOne(ctx, cfg)
		results = append(results, result)
	}
	return results
}

// runOne executes a single tool.
func (r *Runner) runOne(ctx context.Context, cfg RunConfig) RunResult {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	toolCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build args: subcommand + JSON flag + extra args
	args := []string{cfg.Subcommand}
	args = append(args, strings.Fields(cfg.JSONFlag)...)
	args = append(args, cfg.ExtraArgs...)

	start := time.Now()
	stdout, err := r.execFn(toolCtx, cfg.Binary, args...)
	duration := time.Since(start)

	if err != nil {
		return RunResult{
			Tool:     cfg.Tool,
			Binary:   cfg.Binary,
			Duration: duration,
			Success:  false,
			Error:    err.Error(),
		}
	}

	// Write output to temp file
	outputFile := filepath.Join(r.tempDir, string(cfg.Tool)+".json")
	if err := os.WriteFile(outputFile, stdout, 0o600); err != nil {
		return RunResult{
			Tool:     cfg.Tool,
			Binary:   cfg.Binary,
			Duration: duration,
			Success:  false,
			Error:    fmt.Sprintf("failed to write output: %v", err),
		}
	}

	return RunResult{
		Tool:       cfg.Tool,
		Binary:     cfg.Binary,
		OutputFile: outputFile,
		Duration:   duration,
		Success:    true,
	}
}

// OutputFiles returns paths of successful run outputs only.
func OutputFiles(results []RunResult) []string {
	var paths []string
	for _, r := range results {
		if r.Success && r.OutputFile != "" {
			paths = append(paths, r.OutputFile)
		}
	}
	return paths
}

// Cleanup removes the temp directory and all output files.
func (r *Runner) Cleanup() error {
	if r.tempDir == "" {
		return nil
	}
	return os.RemoveAll(r.tempDir)
}

// ConfigsFromDiscovery converts runnable discovery results into RunConfigs.
func ConfigsFromDiscovery(plan *discovery.DiscoveryPlan, timeout time.Duration) []RunConfig {
	var configs []RunConfig
	for _, td := range plan.RunnableTools() {
		info := discovery.Registry[td.Tool]
		configs = append(configs, RunConfig{
			Tool:       td.Tool,
			Binary:     td.BinaryPath,
			Subcommand: info.Subcommand,
			JSONFlag:   info.JSONFlag,
			Timeout:    timeout,
		})
	}
	return configs
}
