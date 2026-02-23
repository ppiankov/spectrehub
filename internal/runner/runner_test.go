package runner

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/ppiankov/spectrehub/internal/discovery"
	"github.com/ppiankov/spectrehub/internal/models"
)

// mockExec returns a function that produces canned output per binary.
func mockExec(outputs map[string][]byte, errs map[string]error) ExecFunc {
	return func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if err, ok := errs[name]; ok {
			return nil, err
		}
		if out, ok := outputs[name]; ok {
			// Respect context cancellation
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return out, nil
			}
		}
		return nil, errors.New("unknown binary: " + name)
	}
}

func TestRun_Success(t *testing.T) {
	jsonOutput := []byte(`{"tool":"vaultspectre","version":"0.3.0","issues":[]}`)
	exec := mockExec(
		map[string][]byte{"/usr/local/bin/vaultspectre": jsonOutput},
		nil,
	)

	r := New(exec)
	defer func() { _ = r.Cleanup() }()

	configs := []RunConfig{
		{
			Tool:       models.ToolVault,
			Binary:     "/usr/local/bin/vaultspectre",
			Subcommand: "scan",
			JSONFlag:   "--format json",
		},
	}

	results := r.Run(context.Background(), configs)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	res := results[0]
	if !res.Success {
		t.Fatalf("expected success, got error: %s", res.Error)
	}
	if res.OutputFile == "" {
		t.Fatal("expected output file path")
	}

	// Verify file contents
	data, err := os.ReadFile(res.OutputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(data) != string(jsonOutput) {
		t.Errorf("output mismatch: got %s", string(data))
	}
}

func TestRun_BinaryError(t *testing.T) {
	exec := mockExec(
		nil,
		map[string]error{"/usr/local/bin/vaultspectre": errors.New("exit status 1")},
	)

	r := New(exec)
	defer func() { _ = r.Cleanup() }()

	configs := []RunConfig{
		{
			Tool:   models.ToolVault,
			Binary: "/usr/local/bin/vaultspectre",
		},
	}

	results := r.Run(context.Background(), configs)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Success {
		t.Fatal("expected failure")
	}
	if results[0].Error == "" {
		t.Fatal("expected error message")
	}
}

func TestRun_Timeout(t *testing.T) {
	// Exec function that blocks until context is cancelled
	exec := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	r := New(exec)
	defer func() { _ = r.Cleanup() }()

	configs := []RunConfig{
		{
			Tool:    models.ToolVault,
			Binary:  "/usr/local/bin/vaultspectre",
			Timeout: 50 * time.Millisecond,
		},
	}

	results := r.Run(context.Background(), configs)
	if results[0].Success {
		t.Fatal("expected timeout failure")
	}
	if results[0].Duration < 50*time.Millisecond {
		t.Errorf("expected duration >= 50ms, got %v", results[0].Duration)
	}
}

func TestRun_PartialSuccess(t *testing.T) {
	exec := mockExec(
		map[string][]byte{
			"/usr/local/bin/vaultspectre": []byte(`{"tool":"vaultspectre"}`),
			"/usr/local/bin/pgspectre":    []byte(`{"tool":"pgspectre"}`),
		},
		map[string]error{
			"/usr/local/bin/s3spectre": errors.New("AWS credentials not found"),
		},
	)

	r := New(exec)
	defer func() { _ = r.Cleanup() }()

	configs := []RunConfig{
		{Tool: models.ToolVault, Binary: "/usr/local/bin/vaultspectre", Subcommand: "scan", JSONFlag: "--format json"},
		{Tool: models.ToolS3, Binary: "/usr/local/bin/s3spectre", Subcommand: "scan", JSONFlag: "--format json"},
		{Tool: models.ToolPg, Binary: "/usr/local/bin/pgspectre", Subcommand: "audit", JSONFlag: "--format json"},
	}

	results := r.Run(context.Background(), configs)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	successCount := 0
	for _, res := range results {
		if res.Success {
			successCount++
		}
	}
	if successCount != 2 {
		t.Errorf("expected 2 successes, got %d", successCount)
	}

	// OutputFiles should return only 2 paths
	paths := OutputFiles(results)
	if len(paths) != 2 {
		t.Errorf("expected 2 output files, got %d", len(paths))
	}
}

func TestRun_MissingBinary(t *testing.T) {
	exec := mockExec(nil, nil)

	r := New(exec)
	defer func() { _ = r.Cleanup() }()

	configs := []RunConfig{
		{Tool: models.ToolVault, Binary: "/nonexistent/vaultspectre"},
	}

	results := r.Run(context.Background(), configs)
	if results[0].Success {
		t.Fatal("expected failure for missing binary")
	}
}

func TestCleanup(t *testing.T) {
	exec := mockExec(
		map[string][]byte{"test": []byte(`{}`)},
		nil,
	)

	r := New(exec)
	results := r.Run(context.Background(), []RunConfig{
		{Tool: models.ToolVault, Binary: "test"},
	})

	if !results[0].Success {
		t.Fatalf("expected success: %s", results[0].Error)
	}

	tempDir := r.tempDir
	if _, err := os.Stat(tempDir); err != nil {
		t.Fatalf("temp dir should exist before cleanup: %v", err)
	}

	if err := r.Cleanup(); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Error("temp dir should not exist after cleanup")
	}
}

func TestCleanup_NoTempDir(t *testing.T) {
	r := New(nil)
	if err := r.Cleanup(); err != nil {
		t.Errorf("cleanup with no temp dir should not error: %v", err)
	}
}

func TestOutputFiles_Empty(t *testing.T) {
	paths := OutputFiles(nil)
	if len(paths) != 0 {
		t.Errorf("expected 0 paths, got %d", len(paths))
	}
}

func TestOutputFiles_MixedResults(t *testing.T) {
	results := []RunResult{
		{Success: true, OutputFile: "/tmp/a.json"},
		{Success: false, Error: "failed"},
		{Success: true, OutputFile: "/tmp/b.json"},
		{Success: true, OutputFile: ""},
	}

	paths := OutputFiles(results)
	if len(paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(paths))
	}
}

func TestConfigsFromDiscovery(t *testing.T) {
	plan := &discovery.DiscoveryPlan{
		Tools: []discovery.ToolDiscovery{
			{
				Tool:       models.ToolVault,
				Binary:     "vaultspectre",
				BinaryPath: "/usr/local/bin/vaultspectre",
				Available:  true,
				HasTarget:  true,
				Runnable:   true,
			},
			{
				Tool:       models.ToolS3,
				Binary:     "s3spectre",
				BinaryPath: "/usr/local/bin/s3spectre",
				Available:  true,
				HasTarget:  false,
				Runnable:   false,
			},
		},
		TotalFound:    2,
		TotalRunnable: 1,
	}

	configs := ConfigsFromDiscovery(plan, 2*time.Minute)
	if len(configs) != 1 {
		t.Fatalf("expected 1 config (only runnable), got %d", len(configs))
	}

	cfg := configs[0]
	if cfg.Tool != models.ToolVault {
		t.Errorf("expected vaultspectre, got %s", cfg.Tool)
	}
	if cfg.Binary != "/usr/local/bin/vaultspectre" {
		t.Errorf("expected binary path, got %s", cfg.Binary)
	}
	if cfg.Subcommand != "scan" {
		t.Errorf("expected 'scan', got %s", cfg.Subcommand)
	}
	if cfg.JSONFlag != "--format json" {
		t.Errorf("expected '--format json', got %s", cfg.JSONFlag)
	}
	if cfg.Timeout != 2*time.Minute {
		t.Errorf("expected 2m timeout, got %v", cfg.Timeout)
	}
}
