package collector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

// Config holds configuration for the collector
type Config struct {
	MaxConcurrency int
	Verbose        bool
	Timeout        time.Duration
}

// Collector orchestrates the collection of reports from multiple files
type Collector struct {
	config Config
}

// New creates a new collector with the given configuration
func New(config Config) *Collector {
	// Set defaults
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = 10
	}
	if config.Timeout <= 0 {
		config.Timeout = 5 * time.Minute
	}

	return &Collector{
		config: config,
	}
}

// CollectFromDirectory reads all JSON files from a directory and parses them
func (c *Collector) CollectFromDirectory(dir string) ([]models.ToolReport, error) {
	// Find all JSON files in directory
	files, err := c.findJSONFiles(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to find JSON files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no JSON files found in directory: %s", dir)
	}

	if c.config.Verbose {
		fmt.Printf("Found %d JSON file(s) to process\n", len(files))
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	// Collect reports concurrently
	return c.collectFiles(ctx, files)
}

// findJSONFiles recursively finds all JSON files in a directory
func (c *Collector) findJSONFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-JSON files
		if info.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

// collectFiles processes files concurrently using a worker pool
func (c *Collector) collectFiles(ctx context.Context, files []string) ([]models.ToolReport, error) {
	// Channels for work distribution and results
	fileCh := make(chan string, len(files))
	resultCh := make(chan *collectResult, len(files))

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < c.config.MaxConcurrency; i++ {
		wg.Add(1)
		go c.worker(ctx, &wg, fileCh, resultCh)
	}

	// Send files to workers
	go func() {
		for _, file := range files {
			select {
			case fileCh <- file:
			case <-ctx.Done():
				break
			}
		}
		close(fileCh)
	}()

	// Wait for workers to finish
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	var reports []models.ToolReport
	var errors []error

	for result := range resultCh {
		if result.err != nil {
			errors = append(errors, result.err)
			if c.config.Verbose {
				fmt.Printf("Error processing %s: %v\n", result.file, result.err)
			}
		} else {
			reports = append(reports, *result.report)
			if c.config.Verbose {
				fmt.Printf("✓ Collected: %s (%s)\n", result.report.Tool, filepath.Base(result.file))
			}
		}
	}

	// Return partial results even if some files failed
	if len(errors) > 0 && len(reports) == 0 {
		return nil, fmt.Errorf("all files failed to process (%d errors)", len(errors))
	}

	if len(errors) > 0 && c.config.Verbose {
		fmt.Printf("Warning: %d file(s) failed to process\n", len(errors))
	}

	return reports, nil
}

// collectResult holds the result of processing a single file
type collectResult struct {
	file   string
	report *models.ToolReport
	err    error
}

// worker processes files from the work channel
func (c *Collector) worker(ctx context.Context, wg *sync.WaitGroup, fileCh <-chan string, resultCh chan<- *collectResult) {
	defer wg.Done()

	for {
		select {
		case file, ok := <-fileCh:
			if !ok {
				return
			}

			report, err := c.processFile(file)
			resultCh <- &collectResult{
				file:   file,
				report: report,
				err:    err,
			}

		case <-ctx.Done():
			return
		}
	}
}

// processFile reads and processes a single JSON file
func (c *Collector) processFile(filePath string) (*models.ToolReport, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Detect tool type
	toolType, err := DetectToolType(data)
	if err != nil {
		return nil, fmt.Errorf("failed to detect tool type: %w", err)
	}

	// Parse report
	parsedReport, err := ParseReport(data, toolType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse report: %w", err)
	}

	// Extract metadata
	version := ExtractVersion(data)
	timestamp := ExtractTimestamp(data)

	// Check if tool is supported
	isSupported := models.IsSupportedTool(toolType)
	status := "supported"
	if !isSupported {
		status = "unsupported"
	}

	// Create ToolReport
	report := &models.ToolReport{
		Tool:        string(toolType),
		Version:     version,
		Timestamp:   timestamp,
		RawData:     parsedReport,
		Status:      status,
		IsSupported: isSupported,
	}

	return report, nil
}
