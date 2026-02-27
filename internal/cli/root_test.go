package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/ppiankov/spectrehub/internal/config"
)

// --- Test helpers ---

// captureStdout runs fn and returns whatever it printed to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// withTestConfig sets the global cfg for the duration of the test.
func withTestConfig(t *testing.T, c *config.Config) {
	t.Helper()
	old := cfg
	cfg = c
	t.Cleanup(func() { cfg = old })
}

// --- HandleError tests ---

func TestHandleErrorNil(t *testing.T) {
	if code := HandleError(nil); code != ExitOK {
		t.Errorf("HandleError(nil) = %d, want %d", code, ExitOK)
	}
}

func TestHandleErrorValidation(t *testing.T) {
	err := &ValidationError{Message: "bad input"}
	if code := HandleError(err); code != ExitInvalidInput {
		t.Errorf("HandleError(ValidationError) = %d, want %d", code, ExitInvalidInput)
	}
}

func TestHandleErrorThreshold(t *testing.T) {
	err := &ThresholdExceededError{IssueCount: 10, Threshold: 5}
	if code := HandleError(err); code != ExitPolicyFail {
		t.Errorf("HandleError(ThresholdExceededError) = %d, want %d", code, ExitPolicyFail)
	}
}

func TestHandleErrorNotExist(t *testing.T) {
	if code := HandleError(os.ErrNotExist); code != ExitRuntimeError {
		t.Errorf("HandleError(ErrNotExist) = %d, want %d", code, ExitRuntimeError)
	}
}

func TestHandleErrorPermission(t *testing.T) {
	if code := HandleError(os.ErrPermission); code != ExitRuntimeError {
		t.Errorf("HandleError(ErrPermission) = %d, want %d", code, ExitRuntimeError)
	}
}

func TestHandleErrorGeneric(t *testing.T) {
	if code := HandleError(errors.New("something went wrong")); code != ExitRuntimeError {
		t.Errorf("HandleError(generic) = %d, want %d", code, ExitRuntimeError)
	}
}

// --- Error type tests ---

func TestValidationErrorMessage(t *testing.T) {
	err := &ValidationError{Message: "invalid schema"}
	if err.Error() != "invalid schema" {
		t.Errorf("ValidationError.Error() = %q, want %q", err.Error(), "invalid schema")
	}
}

func TestThresholdExceededErrorMessage(t *testing.T) {
	err := &ThresholdExceededError{IssueCount: 15, Threshold: 10}
	want := "issue count (15) exceeds threshold (10)"
	if err.Error() != want {
		t.Errorf("ThresholdExceededError.Error() = %q, want %q", err.Error(), want)
	}
}

// --- SetVersion tests ---

func TestSetVersion(t *testing.T) {
	old := buildVersion
	t.Cleanup(func() { buildVersion = old })

	SetVersion("1.2.3")
	if buildVersion != "1.2.3" {
		t.Errorf("buildVersion = %q, want %q", buildVersion, "1.2.3")
	}
}

func TestSetVersionDev(t *testing.T) {
	// Default should be "dev"
	old := buildVersion
	t.Cleanup(func() { buildVersion = old })

	buildVersion = "dev"
	if buildVersion != "dev" {
		t.Errorf("default buildVersion = %q, want %q", buildVersion, "dev")
	}
}

// --- Logging tests ---

func TestLogVerboseEnabled(t *testing.T) {
	withTestConfig(t, &config.Config{Verbose: true})

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logVerbose("test %s", "message")

	_ = w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if !bytes.Contains(buf.Bytes(), []byte("[INFO] test message")) {
		t.Errorf("logVerbose output = %q, want to contain '[INFO] test message'", buf.String())
	}
}

func TestLogVerboseDisabled(t *testing.T) {
	withTestConfig(t, &config.Config{Verbose: false})

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logVerbose("should not appear")

	_ = w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if buf.Len() > 0 {
		t.Errorf("logVerbose with Verbose=false should produce no output, got %q", buf.String())
	}
}

func TestLogDebugEnabled(t *testing.T) {
	withTestConfig(t, &config.Config{Debug: true})

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logDebug("debug %d", 42)

	_ = w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if !bytes.Contains(buf.Bytes(), []byte("[DEBUG] debug 42")) {
		t.Errorf("logDebug output = %q, want to contain '[DEBUG] debug 42'", buf.String())
	}
}

func TestLogErrorAlwaysPrints(t *testing.T) {
	withTestConfig(t, &config.Config{})

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logError("fail %s", "now")

	_ = w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if !bytes.Contains(buf.Bytes(), []byte("[ERROR] fail now")) {
		t.Errorf("logError output = %q, want to contain '[ERROR] fail now'", buf.String())
	}
}
