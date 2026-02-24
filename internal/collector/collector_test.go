package collector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDefaults(t *testing.T) {
	c := New(Config{})
	if c.config.MaxConcurrency != 10 {
		t.Errorf("expected default MaxConcurrency=10, got %d", c.config.MaxConcurrency)
	}
	if c.config.Timeout <= 0 {
		t.Error("expected positive default timeout")
	}
}

func TestNewCustom(t *testing.T) {
	c := New(Config{MaxConcurrency: 2})
	if c.config.MaxConcurrency != 2 {
		t.Errorf("expected MaxConcurrency=2, got %d", c.config.MaxConcurrency)
	}
}

func TestCollectFromDirectoryEmpty(t *testing.T) {
	dir := t.TempDir()
	c := New(Config{})
	_, err := c.CollectFromDirectory(dir)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
}

func TestCollectFromDirectoryNonexistent(t *testing.T) {
	c := New(Config{})
	_, err := c.CollectFromDirectory("/nonexistent/dir")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestCollectFromPathsEmpty(t *testing.T) {
	c := New(Config{})
	_, err := c.CollectFromPaths(nil)
	if err == nil {
		t.Fatal("expected error for empty paths")
	}
}

func TestCollectFromPathsNonexistent(t *testing.T) {
	c := New(Config{})
	_, err := c.CollectFromPaths([]string{"/nonexistent/file.json"})
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestCollectFromPathsNonJSON(t *testing.T) {
	dir := t.TempDir()
	txtFile := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(txtFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	c := New(Config{})
	_, err := c.CollectFromPaths([]string{txtFile})
	if err == nil {
		t.Fatal("expected error for non-JSON file")
	}
}

func TestCollectFromPathsSingleFile(t *testing.T) {
	c := New(Config{MaxConcurrency: 1})
	reports, err := c.CollectFromPaths([]string{"../../testdata/contracts/vaultspectre-v0.1.0.json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
	if reports[0].Tool != "vaultspectre" {
		t.Errorf("expected tool=vaultspectre, got %s", reports[0].Tool)
	}
}

func TestCollectFromPathsDirectory(t *testing.T) {
	c := New(Config{MaxConcurrency: 2})
	reports, err := c.CollectFromPaths([]string{"../../testdata/contracts"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reports) < 6 {
		t.Fatalf("expected at least 6 reports, got %d", len(reports))
	}
}

func TestCollectFromPathsDeduplicate(t *testing.T) {
	c := New(Config{MaxConcurrency: 2})
	path := "../../testdata/contracts/vaultspectre-v0.1.0.json"
	reports, err := c.CollectFromPaths([]string{path, path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should deduplicate
	if len(reports) != 1 {
		t.Fatalf("expected 1 report (deduplicated), got %d", len(reports))
	}
}

func TestCollectFromPathsMixed(t *testing.T) {
	c := New(Config{MaxConcurrency: 2})
	reports, err := c.CollectFromPaths([]string{
		"../../testdata/contracts/vaultspectre-v0.1.0.json",
		"../../testdata/contracts",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reports) < 6 {
		t.Fatalf("expected at least 6 reports, got %d", len(reports))
	}
}

func TestCollectFromPathsAllInvalid(t *testing.T) {
	dir := t.TempDir()
	invalidFile := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(invalidFile, []byte("{"), 0644); err != nil {
		t.Fatal(err)
	}

	c := New(Config{MaxConcurrency: 1})
	_, err := c.CollectFromPaths([]string{invalidFile})
	if err == nil {
		t.Fatal("expected error when all files fail")
	}
}

func TestCollectFromPathsVerbose(t *testing.T) {
	c := New(Config{MaxConcurrency: 1, Verbose: true})
	reports, err := c.CollectFromPaths([]string{"../../testdata/contracts/vaultspectre-v0.1.0.json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
}
