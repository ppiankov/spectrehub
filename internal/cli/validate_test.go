package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ppiankov/spectrehub/internal/models"
)

func TestRunValidateValid(t *testing.T) {
	report := models.SpectreV1Report{
		Schema:    "spectre/v1",
		Tool:      "s3spectre",
		Version:   "0.1.0",
		Timestamp: time.Now().Add(-1 * time.Hour),
		Target:    models.SpectreV1Target{Type: "s3"},
		Findings: []models.SpectreV1Finding{
			{ID: "UNUSED_BUCKET", Severity: "medium", Location: "s3://test", Message: "unused"},
		},
		Summary: models.SpectreV1Summary{Total: 1, Medium: 1},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}

	tmp := filepath.Join(t.TempDir(), "valid.json")
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		t.Fatal(err)
	}

	output := captureStdout(t, func() {
		err = runValidate(nil, []string{tmp})
	})

	if err != nil {
		t.Fatalf("runValidate(valid) = %v, want nil", err)
	}
	if output == "" {
		t.Error("expected VALID output")
	}
}

func TestRunValidateMissingFile(t *testing.T) {
	err := runValidate(nil, []string{"/nonexistent/report.json"})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
