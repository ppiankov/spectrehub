package discovery

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ppiankov/spectrehub/internal/models"
)

// LookPathFunc matches the signature of exec.LookPath.
type LookPathFunc func(file string) (string, error)

// GetenvFunc matches the signature of os.Getenv.
type GetenvFunc func(key string) string

// Discoverer probes the local environment to find available spectre tools
// and infrastructure targets. Injectable deps make it fully testable.
type Discoverer struct {
	lookPath LookPathFunc
	getenv   GetenvFunc
}

// New creates a Discoverer with the given dependency functions.
func New(lookPath LookPathFunc, getenv GetenvFunc) *Discoverer {
	return &Discoverer{
		lookPath: lookPath,
		getenv:   getenv,
	}
}

// ToolDiscovery describes what was found for a single tool.
type ToolDiscovery struct {
	Tool       models.ToolType `json:"tool"`
	Binary     string          `json:"binary"`
	BinaryPath string          `json:"binary_path"`
	Available  bool            `json:"available"`
	EnvVars    []EnvVarStatus  `json:"env_vars,omitempty"`
	Configs    []ConfigStatus  `json:"configs,omitempty"`
	HasTarget  bool            `json:"has_target"`
	Runnable   bool            `json:"runnable"`
}

// EnvVarStatus tracks whether an environment variable is set.
type EnvVarStatus struct {
	Name string `json:"name"`
	Set  bool   `json:"set"`
}

// ConfigStatus tracks whether a config file exists.
type ConfigStatus struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

// DiscoveryPlan is the complete result of a discovery scan.
type DiscoveryPlan struct {
	Tools         []ToolDiscovery `json:"tools"`
	TotalFound    int             `json:"total_found"`
	TotalRunnable int             `json:"total_runnable"`
}

// Discover checks which spectre tools are installed, which env vars are set,
// and which config files exist. No network calls, no port scanning.
func (d *Discoverer) Discover() *DiscoveryPlan {
	plan := &DiscoveryPlan{}

	for toolType, info := range Registry {
		td := ToolDiscovery{
			Tool:   toolType,
			Binary: info.Binary,
		}

		// Check if binary exists in PATH
		if path, err := d.lookPath(info.Binary); err == nil {
			td.Available = true
			td.BinaryPath = path
		}

		// Check env vars
		anyEnvSet := false
		for _, envVar := range info.EnvVars {
			val := d.getenv(envVar)
			isSet := val != ""
			td.EnvVars = append(td.EnvVars, EnvVarStatus{
				Name: envVar,
				Set:  isSet,
			})
			if isSet {
				anyEnvSet = true
			}
		}

		// Check config files
		anyConfigExists := false
		for _, cfgPath := range info.ConfigFiles {
			expanded := expandHome(cfgPath)
			exists := fileExists(expanded)
			td.Configs = append(td.Configs, ConfigStatus{
				Path:   cfgPath,
				Exists: exists,
			})
			if exists {
				anyConfigExists = true
			}
		}

		// A tool has a target if env vars or configs indicate infrastructure
		td.HasTarget = anyEnvSet || anyConfigExists

		// Runnable = binary available AND has a target
		td.Runnable = td.Available && td.HasTarget

		plan.Tools = append(plan.Tools, td)

		if td.Available {
			plan.TotalFound++
		}
		if td.Runnable {
			plan.TotalRunnable++
		}
	}

	// Sort tools by tool type name for deterministic output
	sortTools(plan.Tools)

	return plan
}

// RunnableTools returns only the tools that can be executed.
func (p *DiscoveryPlan) RunnableTools() []ToolDiscovery {
	var runnable []ToolDiscovery
	for _, t := range p.Tools {
		if t.Runnable {
			runnable = append(runnable, t)
		}
	}
	return runnable
}

// sortTools sorts by tool type name for deterministic output.
func sortTools(tools []ToolDiscovery) {
	n := len(tools)
	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			if string(tools[i].Tool) > string(tools[j].Tool) {
				tools[i], tools[j] = tools[j], tools[i]
			}
		}
	}
}

// expandHome replaces a leading ~/ with the user's home directory.
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// fileExists checks if a file exists (not a directory).
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
