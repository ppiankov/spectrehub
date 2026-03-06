# CLI Reference

## Commands

### `spectrehub discover`

Detect available spectre tools and infrastructure targets. Read-only — no tools are executed.

```bash
spectrehub discover
spectrehub discover --format json
```

### `spectrehub run`

Discover, execute, and aggregate in one step.

```bash
spectrehub run
spectrehub run --dry-run
spectrehub run --format json --fail-threshold 50
spectrehub run --timeout 2m
```

**Flags:**
- `--dry-run` — show discovery plan without executing
- `--format` — output format (text, json, or both)
- `--output` / `-o` — output file path
- `--fail-threshold` — exit 1 if issues exceed threshold
- `--store` — persist results for trend analysis
- `--timeout` — per-tool execution timeout (default: 5m)

### `spectrehub collect <directory>`

Collect and aggregate reports from Spectre tools.

```bash
spectrehub collect ./reports
spectrehub collect ./reports --format json --output summary.json
spectrehub collect ./reports --fail-threshold 50 --store
```

**Flags:**
- `--config` — path to config file
- `--format` — output format (text, json, both)
- `--output` — output file path (default: stdout)
- `--storage-dir` — storage directory (default from config)
- `--fail-threshold` — exit with code 1 if issues exceed threshold
- `--store` — store aggregated report (default: true)
- `--verbose` / `-v` — verbose output
- `--debug` — debug mode

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
- `--last` / `-n` — number of runs to analyze (default from config)
- `--compare` / `-c` — compare latest run with previous
- `--format` / `-f` — output format (text or json)
- `--tui` — force interactive TUI (auto-enabled when stdout is a TTY)

### `spectrehub version`

Show version information.

## Configuration

### Config file: `~/.spectrehub.yaml`

```yaml
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

## Exit codes

| Code | Meaning | Description |
|------|---------|-------------|
| 0 | Success | All checks passed |
| 1 | Policy Failure | Issues exceed `--fail-threshold` |
| 2 | Invalid Input | Malformed JSON or schema violation |
| 3 | Runtime Error | File not found, permission denied, I/O error |

## Use cases

### CI/CD pipeline

```yaml
# .github/workflows/spectre-audit.yml
- name: Run SpectreHub Audit
  run: spectrehub run --fail-threshold 50 --format json --store
```

### Weekly infrastructure review

```bash
# Cron job (0 9 * * 1)
spectrehub collect /var/reports
spectrehub summarize --last 7 > weekly-report.txt
```

## Adding a new Spectre tool

1. Add models to `internal/models/{tool}.go`
2. Add detection logic to `internal/collector/detector.go`
3. Add parser to `internal/collector/parser.go`
4. Add normalizer to `internal/aggregator/normalizer.go`
5. Create contract test with real tool output
