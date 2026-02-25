# SpectreHub

[![spectrehub.dev](https://img.shields.io/badge/docs-spectrehub.dev-blue)](https://spectrehub.dev)

**The unified infrastructure audit aggregator for the Spectre tool family**

SpectreHub is the central brain of the Spectre ecosystem. It discovers installed spectre tools, executes them against configured infrastructure, and aggregates JSON reports from VaultSpectre, S3Spectre, KafkaSpectre, ClickSpectre, PgSpectre, and MongoSpectre into a coherent, actionable view of your infrastructure drift.

## Overview

Modern infrastructure spans multiple domains: secrets management (Vault), object storage (S3), message queues (Kafka), and analytics databases (ClickHouse). Each system has its own audit tools, producing separate reports that are hard to synthesize.

**SpectreHub solves this** by:
- **Aggregating** - Collecting all Spectre tool outputs into one place
- **Normalizing** - Converting different formats to a unified schema
- **Analyzing** - Calculating cross-tool health scores and trends
- **Reporting** - Generating actionable insights prioritized by severity
- **Tracking** - Storing historical data for trend analysis

## Workflow Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Infrastructure Audit                     │
│                                                             │
│  ┌──────────┐  ┌──────────┐   ┌──────────┐  ┌──────────┐    │
│  │  Vault   │  │   S3     │   │  Kafka   │  │ ClickH.  │    │
│  │ Spectre  │  │ Spectre  │   │ Spectre  │  │ Spectre  │    │
│  └────┬─────┘  └────┬─────┘   └────┬─────┘  └────┬─────┘    │
│       │             │              │             │          │
│       └─────────────┴──────────────┴─────────────┘          │
│                          │                                  │
│                          ▼                                  │
│              ┌─────────────────────┐                        │
│              │   SpectreHub        │                        │
│              │   (Aggregator)      │                        │
│              └──────────┬──────────┘                        │
│                         │                                   │
│            ┌────────────┼────────────┐                      │
│            ▼            ▼            ▼                      │
│       ┌────────┐  ┌─────────┐  ┌─────────┐                  │
│       │ Report │  │ Trends  │  │ Storage │                  │
│       └────────┘  └─────────┘  └─────────┘                  │
│                                                             │
│       "One coherent view of infrastructure drift"           │
└─────────────────────────────────────────────────────────────┘
```

## Inputs

SpectreHub accepts JSON reports from:

- **VaultSpectre** - HashiCorp Vault secret audits
- **S3Spectre** - AWS S3 bucket analysis
- **KafkaSpectre** - Kafka cluster audits
- **ClickSpectre** - ClickHouse table usage analysis
- **PgSpectre** - PostgreSQL security audits
- **MongoSpectre** - MongoDB configuration audits

### Input Format Requirements

Each tool must produce JSON output. SpectreHub auto-detects tool type from structure.

### Example Directory Structure

```
reports/
├── vaultspectre-2026-01-27.json
├── s3spectre-2026-01-27.json
├── kafkaspectre-2026-01-27.json
└── clickspectre-2026-01-27.json
```

## Outputs

### Text Report (Human-Readable)

```
╔════════════════════════════════════════════╗
║       SpectreHub Aggregated Report         ║
╚════════════════════════════════════════════╝

Timestamp: 2026-01-27 15:30:45

Overall Summary:
--------------------------------------------------
  Total Tools: 4 (4 supported, 0 unsupported)
  Total Issues: 73
  Health Score: GOOD (91.0%) ↓ improved -9.0% from previous run

Issues by Category:
  Missing: 8
  Unused: 42
  Stale: 23

vaultspectre (v0.1.0)
--------------------------------------------------
  Total References: 145
  OK: 132
  Missing: 8
  Stale: 5
  Health: good

s3spectre (v0.1.0)
--------------------------------------------------
  Total Buckets: 15
  OK: 12
  Missing: 2
  Unused: 1

... [per-tool breakdowns]

Recommended Actions:
--------------------------------------------------
  1. [CRITICAL] Fix 8 missing Vault secrets
     Impact: Services may fail to start or operate incorrectly
  2. [HIGH] Clean up 7 unused Kafka topics
     Impact: Significant waste of resources and potential security risks
  3. [MEDIUM] Review 18 stale ClickHouse tables
     Impact: Moderate impact on system efficiency
```

### JSON Report (Machine-Readable)

```json
{
  "timestamp": "2026-01-27T15:30:45Z",
  "summary": {
    "total_issues": 73,
    "health_score": "good",
    "score_percent": 91.0,
    "issues_by_tool": {
      "vaultspectre": 13,
      "s3spectre": 3,
      "kafkaspectre": 10,
      "clickspectre": 18
    },
    "issues_by_category": {
      "missing": 8,
      "unused": 42,
      "stale": 23
    }
  },
  "trend": {
    "direction": "improving",
    "change_percent": -9.0,
    "previous_issues": 80,
    "current_issues": 73,
    "new_issues": 0,
    "resolved_issues": 7
  },
  "recommendations": [
    {
      "severity": "critical",
      "tool": "vaultspectre",
      "action": "Fix 8 missing Vault secrets",
      "impact": "Services may fail to start or operate incorrectly",
      "count": 8
    }
  ]
}
```

### Stored Data

Reports are stored in `.spectre/runs/` for historical tracking:

- `.spectre/runs/{timestamp}-aggregated.json` - Full aggregated report

## Quick Start

```bash
# Install
go install github.com/ppiankov/spectrehub/cmd/spectrehub@latest

# Discover what's available
spectrehub discover

# Run all detected tools and aggregate
spectrehub run

# Or collect pre-existing reports
spectrehub collect ./reports

# View trends
spectrehub summarize

# CI/CD integration
spectrehub run --fail-threshold 50
```

## Configuration

### Config File: `~/.spectrehub.yaml`

SpectreHub supports configuration files for team-wide defaults:

```yaml
# ~/.spectrehub.yaml
storage_dir: .spectre
fail_threshold: 50
format: text
last_runs: 7
verbose: false
```

### Precedence (lowest to highest)

1. Default values
2. Config file (`~/.spectrehub.yaml` or `./spectrehub.yaml`)
3. Environment variables (`SPECTREHUB_STORAGE_DIR`, etc.)
4. CLI flags

### Example

```bash
# Uses config file defaults
spectrehub collect ./reports

# Override config with flags
spectrehub collect ./reports --fail-threshold 100

# Use custom config file
spectrehub collect ./reports --config /path/to/config.yaml
```

## Exit Codes (CI/CD Integration)

SpectreHub uses standard exit codes for reliable CI/CD integration:

| Code | Meaning | Description |
|------|---------|-------------|
| 0 | Success | All checks passed |
| 1 | Policy Failure | Issues exceed `--fail-threshold` |
| 2 | Invalid Input | Malformed JSON or schema violation |
| 3 | Runtime Error | File not found, permission denied, I/O error |

### Example CI/CD Usage

```bash
# Fail build if more than 50 issues found
spectrehub collect ./reports --fail-threshold 50
if [ $? -eq 1 ]; then
    echo "Too many infrastructure issues detected!"
    exit 1
fi

# Or check specific exit codes
spectrehub collect ./reports --fail-threshold 50
EXIT_CODE=$?
case $EXIT_CODE in
    0) echo "✓ All checks passed" ;;
    1) echo "✗ Policy failed: too many issues" ;;
    2) echo "✗ Invalid report format" ;;
    3) echo "✗ Runtime error" ;;
esac
```

## Commands Reference

### `spectrehub discover`

Detect available spectre tools and infrastructure targets. Read-only — no tools are executed.

```bash
spectrehub discover
spectrehub discover --format json
```

**Flags:**
- `--format` - Output format (text or json)

### `spectrehub run`

Discover, execute, and aggregate in one step.

```bash
spectrehub run
spectrehub run --dry-run
spectrehub run --format json --fail-threshold 50
spectrehub run --timeout 2m
```

**Flags:**
- `--dry-run` - Show discovery plan without executing
- `--format` - Output format (text, json, or both)
- `--output` / `-o` - Output file path
- `--fail-threshold` - Exit 1 if issues exceed threshold
- `--store` - Persist results for trend analysis
- `--timeout` - Per-tool execution timeout (default: 5m)

### `spectrehub collect <directory>`

Collect and aggregate reports from Spectre tools.

```bash
spectrehub collect ./reports
spectrehub collect ./reports --format json --output summary.json
spectrehub collect ./reports --fail-threshold 50 --store
```

**Flags:**
- `--config` - Path to config file
- `--format` - Output format (text, json, both)
- `--output` - Output file path (default: stdout)
- `--storage-dir` - Storage directory (default from config)
- `--fail-threshold` - Exit with code 1 if issues exceed threshold
- `--store` - Store aggregated report (default: true)
- `--verbose` / `-v` - Verbose output
- `--debug` - Debug mode

### `spectrehub summarize`

Show summary and trends from stored runs.

```bash
spectrehub summarize
spectrehub summarize --last 7
spectrehub summarize --compare --format json
spectrehub summarize --tui
```

When running in a terminal, `summarize` automatically launches an interactive TUI (bubbletea) with filterable findings table, severity breakdown, and detail views. Use `--format json` or pipe output to bypass the TUI.

**Flags:**
- `--last` / `-n` - Number of runs to analyze (default from config)
- `--compare` / `-c` - Compare latest run with previous
- `--format` / `-f` - Output format (text or json)
- `--tui` - Force interactive TUI (auto-enabled when stdout is a TTY)

### `spectrehub version`

Show version information.

```bash
spectrehub version
```

## Use Cases

### Local Development

SpectreHub discovers and runs all available spectre tools automatically:

```bash
# One command — discover, execute, aggregate
spectrehub run

# Or run individual tools manually and aggregate
vaultspectre scan --format json > vault.json
kafkaspectre audit --format json > kafka.json
spectrehub collect .
```

### CI/CD Pipeline

```yaml
# .github/workflows/spectre-audit.yml
- name: Run SpectreHub Audit
  run: spectrehub run --fail-threshold 50 --format json --store
```

### Weekly Infrastructure Review

```bash
# Cron job (0 9 * * 1)
spectrehub collect /var/reports
spectrehub summarize --last 7 > weekly-report.txt
```

## The Platform Story

### Before SpectreHub

```
VaultSpectre → 12 issues
S3Spectre → 3 issues
KafkaSpectre → 7 issues
ClickSpectre → 18 issues

(You have to manually add: 12+3+7+18 = 40 issues)
```

### With SpectreHub

```
$ spectrehub collect ./reports
Total Issues: 40 (↓ 9% from last week)
Health Score: GOOD (improved from WARNING)
Critical: 8 missing Vault secrets
[Full actionable report with prioritized recommendations]
```

SpectreHub is the **central brain** of the Spectre family. It:

1. **Aggregates** - Collects all tool outputs into one place
2. **Normalizes** - Converts different formats to unified schema
3. **Analyzes** - Calculates cross-tool health scores and trends
4. **Reports** - Generates actionable insights
5. **Tracks** - Stores historical data for trend analysis

## Architecture

### Design Principles

- **Minimal** - No databases, no dependencies, no cloud required
- **Fast** - Concurrent processing with Go goroutines
- **Extensible** - Easy to add new Spectre tools
- **Portable** - Single binary, runs anywhere

### Concurrency Model

- Worker pool with configurable concurrency (default: 10 goroutines)
- Parallel file reading and parsing
- Non-blocking result aggregation

### Storage

- Local filesystem only (for MVP)
- JSON files in `.spectre/runs/`
- Future: S3, PostgreSQL, SQLite backends

### Normalized Issue Model

Every tool maps to atomic `NormalizedIssue` structure:

```go
type NormalizedIssue struct {
    Tool      string    // vaultspectre, s3spectre, etc.
    Category  string    // missing, unused, stale, error, misconfig
    Severity  string    // critical, high, medium, low
    Resource  string    // vault path / s3://bucket / topic / db.table
    Evidence  string    // short explanation
    Count     int       // how many instances
    FirstSeen time.Time
    LastSeen  time.Time
}
```

This enables:
- **Diff** - Compare any two runs
- **Query** - Filter issues across history
- **Correlation** - Find patterns across tools

## CI/CD and Releases

SpectreHub uses GitHub Actions for automated testing and releases.

### Continuous Integration

On every push to `main` or pull request:
- ✅ Build binary for Linux amd64
- ✅ Run all tests with coverage
- ✅ Run linter (golangci-lint)

### Release Process

To create a new release:

1. Tag a commit with a version:
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```

2. GitHub Actions automatically:
   - Runs all tests
   - Builds binaries for:
     - Linux (amd64, arm64)
     - macOS (amd64, arm64)
     - Windows (amd64)
   - Generates SHA256 checksums
   - Creates a GitHub release with all binaries attached
   - Auto-generates release notes from commits

### Local Multi-Platform Build

To build release binaries locally:

```bash
make release
```

This creates binaries in `bin/` for all supported platforms.

## Contributing

### Adding a New Spectre Tool

1. Add models to `internal/models/{tool}.go`
2. Add detection logic to `internal/collector/detector.go`
3. Add parser to `internal/collector/parser.go`
4. Add normalizer to `internal/aggregator/normalizer.go`
5. Create contract test with real tool output
6. Update README

### Running Tests

```bash
make test                  # Run all tests
make test-contract         # Run contract tests only
make lint                  # Run linter
make fmt                   # Format code
```

### Project Structure

```
spectrehub/
├── .github/
│   └── workflows/         # GitHub Actions (CI and release)
├── cmd/spectrehub/        # CLI entry point
├── internal/
│   ├── models/            # Data models for all tools
│   ├── collector/         # File collection and parsing
│   ├── aggregator/        # Aggregation and normalization
│   ├── storage/           # Storage layer (local filesystem)
│   ├── reporter/          # Text and JSON reporters
│   ├── config/            # Configuration management
│   ├── discovery/         # Tool and target detection
│   ├── runner/            # Tool execution engine
│   └── cli/               # Cobra CLI commands
├── testdata/
│   ├── contracts/         # Real tool outputs for testing
│   ├── invalid/           # Malformed inputs for error handling
│   └── unsupported/       # Unknown tools
├── Makefile
└── README.md
```

## Roadmap

### v0.1 (Current - MVP)

✅ Core aggregation engine
✅ Support for Vault, S3, Kafka, ClickHouse
✅ Local storage and trend analysis
✅ CLI with text/JSON output
✅ Config file support
✅ Exit code contract for CI/CD

### v0.2 (Current)

✅ `discover` command - Detect available tools and targets
✅ `run` command - Discover + execute + aggregate in one step
✅ PgSpectre and MongoSpectre support

### v0.3 (Planned)

- [ ] GitHub Action for automated audits
- [ ] `diff` command - Compare any two runs explicitly
- [ ] `query` command - Filter stored reports

### v0.3+ (User-Driven)

Features to be prioritized based on user feedback:
- Remote storage backends (S3, PostgreSQL)
- Notification webhooks (Slack, Teams, Email)
- Policy as code (custom thresholds per tool)
- Cross-tool correlation and dependency graphs
- Cost estimation calculators

## License

MIT License - see [LICENSE](LICENSE) file for details

## Support

- **Issues**: [GitHub Issues](https://github.com/ppiankov/spectrehub/issues)
- **Discussions**: [GitHub Discussions](https://github.com/ppiankov/spectrehub/discussions)

---

**Built as part of the Spectre family of infrastructure tools.**
