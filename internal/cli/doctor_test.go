package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ppiankov/spectrehub/internal/config"
)

// --- joinMax tests ---

func TestJoinMaxUnderLimit(t *testing.T) {
	got := joinMax([]string{"a", "b"}, 3)
	if got != "a, b" {
		t.Errorf("joinMax(2 items, 3) = %q, want %q", got, "a, b")
	}
}

func TestJoinMaxExactLimit(t *testing.T) {
	got := joinMax([]string{"a", "b", "c"}, 3)
	if got != "a, b, c" {
		t.Errorf("joinMax(3 items, 3) = %q, want %q", got, "a, b, c")
	}
}

func TestJoinMaxOverLimit(t *testing.T) {
	got := joinMax([]string{"a", "b", "c", "d", "e"}, 2)
	want := "a, b +3 more"
	if got != want {
		t.Errorf("joinMax(5 items, 2) = %q, want %q", got, want)
	}
}

func TestJoinMaxEmpty(t *testing.T) {
	got := joinMax([]string{}, 3)
	if got != "" {
		t.Errorf("joinMax(empty, 3) = %q, want %q", got, "")
	}
}

func TestJoinMaxSingle(t *testing.T) {
	got := joinMax([]string{"only"}, 3)
	if got != "only" {
		t.Errorf("joinMax(1 item, 3) = %q, want %q", got, "only")
	}
}

// --- writeDoctorText tests ---

func TestWriteDoctorTextOK(t *testing.T) {
	result := doctorResult{
		Checks: []doctorCheck{
			{Name: "config", Status: "ok", Detail: "/home/.spectrehub.yaml"},
			{Name: "storage", Status: "ok"},
		},
		Summary: "all checks passed",
	}

	output := captureStdout(t, func() {
		_ = writeDoctorText(result)
	})

	if !strings.Contains(output, "✓") {
		t.Error("missing ok icon ✓")
	}
	if !strings.Contains(output, "config") {
		t.Error("missing check name")
	}
	if !strings.Contains(output, "all checks passed") {
		t.Error("missing summary")
	}
}

func TestWriteDoctorTextMixed(t *testing.T) {
	result := doctorResult{
		Checks: []doctorCheck{
			{Name: "config", Status: "ok", Detail: "found"},
			{Name: "license", Status: "warn", Detail: "not configured"},
			{Name: "api", Status: "fail", Detail: "unreachable"},
		},
		Summary: "1 issue(s) found",
	}

	output := captureStdout(t, func() {
		_ = writeDoctorText(result)
	})

	if !strings.Contains(output, "✓") {
		t.Error("missing ok icon")
	}
	if !strings.Contains(output, "△") {
		t.Error("missing warn icon")
	}
	if !strings.Contains(output, "✗") {
		t.Error("missing fail icon")
	}
	if !strings.Contains(output, "1 issue(s) found") {
		t.Error("missing summary")
	}
}

// --- checkConfig tests ---

func TestCheckConfigExists(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "spectrehub.yaml")
	if err := os.WriteFile(cfgPath, []byte("format: text\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Set the global configFile to our temp config
	old := configFile
	configFile = cfgPath
	t.Cleanup(func() { configFile = old })

	check := checkConfig()
	if check.Status != "ok" {
		t.Errorf("checkConfig() status = %q, want ok", check.Status)
	}
	if check.Detail != cfgPath {
		t.Errorf("checkConfig() detail = %q, want %q", check.Detail, cfgPath)
	}
}

func TestCheckConfigMissing(t *testing.T) {
	old := configFile
	configFile = "/nonexistent/path/spectrehub.yaml"
	t.Cleanup(func() { configFile = old })

	check := checkConfig()
	if check.Status != "warn" {
		t.Errorf("checkConfig() status = %q, want warn", check.Status)
	}
}

// --- checkRepo tests ---

func TestCheckRepoSet(t *testing.T) {
	withTestConfig(t, &config.Config{Repo: "org/myrepo"})

	check := checkRepo()
	if check.Status != "ok" {
		t.Errorf("checkRepo() status = %q, want ok", check.Status)
	}
	if check.Detail != "org/myrepo" {
		t.Errorf("checkRepo() detail = %q, want %q", check.Detail, "org/myrepo")
	}
}

func TestCheckRepoEmptyNoLicense(t *testing.T) {
	withTestConfig(t, &config.Config{})

	// Clear SPECTREHUB_REPO env
	old := os.Getenv("SPECTREHUB_REPO")
	_ = os.Unsetenv("SPECTREHUB_REPO")
	t.Cleanup(func() {
		if old != "" {
			_ = os.Setenv("SPECTREHUB_REPO", old)
		}
	})

	check := checkRepo()
	if check.Status != "ok" {
		t.Errorf("checkRepo() status = %q, want ok (no license = not needed)", check.Status)
	}
}

func TestCheckRepoEmptyWithLicense(t *testing.T) {
	withTestConfig(t, &config.Config{LicenseKey: "sh_test_00000000000000000000000000000000"})

	old := os.Getenv("SPECTREHUB_REPO")
	_ = os.Unsetenv("SPECTREHUB_REPO")
	t.Cleanup(func() {
		if old != "" {
			_ = os.Setenv("SPECTREHUB_REPO", old)
		}
	})

	check := checkRepo()
	if check.Status != "warn" {
		t.Errorf("checkRepo() status = %q, want warn (license set but no repo)", check.Status)
	}
}

// --- checkStorage tests ---

func TestCheckStorageWritable(t *testing.T) {
	tmpDir := t.TempDir()
	withTestConfig(t, &config.Config{StorageDir: tmpDir})

	check := checkStorage()
	if check.Status != "ok" {
		t.Errorf("checkStorage() status = %q, want ok", check.Status)
	}
}

func TestCheckStorageNotExist(t *testing.T) {
	withTestConfig(t, &config.Config{StorageDir: filepath.Join(t.TempDir(), "nonexistent")})

	check := checkStorage()
	if check.Status != "ok" {
		t.Errorf("checkStorage() status = %q, want ok (will be created)", check.Status)
	}
	if !strings.Contains(check.Detail, "will be created") {
		t.Errorf("checkStorage() detail = %q, want 'will be created' message", check.Detail)
	}
}

func TestCheckStorageIsFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "notadir")
	if err := os.WriteFile(tmpFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	withTestConfig(t, &config.Config{StorageDir: tmpFile})

	check := checkStorage()
	if check.Status != "fail" {
		t.Errorf("checkStorage() status = %q, want fail (path is a file)", check.Status)
	}
}
