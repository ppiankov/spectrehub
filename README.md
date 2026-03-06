<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo-dark.png">
    <source media="(prefers-color-scheme: light)" srcset="assets/logo-light.png">
    <img alt="spectrehub" src="assets/logo-light.png" width="128">
  </picture>
</p>

# SpectreHub

[![CI](https://github.com/ppiankov/spectrehub/actions/workflows/ci.yml/badge.svg)](https://github.com/ppiankov/spectrehub/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ppiankov/spectrehub)](https://goreportcard.com/report/github.com/ppiankov/spectrehub)
[![ANCC](https://img.shields.io/badge/ANCC-compliant-brightgreen)](https://ancc.dev)
[![spectrehub.dev](https://img.shields.io/badge/docs-spectrehub.dev-blue)](https://spectrehub.dev)

The unified infrastructure audit aggregator for the Spectre tool family.

SpectreHub discovers installed Spectre tools, executes them against configured infrastructure, and aggregates JSON reports into a coherent, actionable view of your infrastructure drift.

## What it is

- Discovers installed Spectre tools and executes them against configured infrastructure
- Aggregates JSON reports from all Spectre scanners into a unified view
- Normalizes different output formats to a common schema
- Calculates cross-tool health scores, trends, and severity breakdowns
- Stores historical data for trend analysis

## What it is NOT

- Not a scanner itself — orchestrates and aggregates other Spectre tools
- Not a monitoring dashboard — produces reports, not real-time views
- Not a remediation tool — presents findings, never modifies infrastructure
- Not a replacement for individual tools — each tool runs independently

## Quick start

```bash
# Install
brew install ppiankov/tap/spectrehub

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

## CLI commands

| Command | Description |
|---------|-------------|
| `spectrehub discover` | Detect available tools and targets |
| `spectrehub run` | Discover, execute, and aggregate in one step |
| `spectrehub collect` | Aggregate pre-existing reports |
| `spectrehub summarize` | Show trends from stored runs (interactive TUI) |
| `spectrehub diff` | Compare runs with `--fail-new` for CI gating |
| `spectrehub doctor` | Validate environment |
| `spectrehub version` | Print version |

See [CLI Reference](docs/cli-reference.md) for all flags, configuration, and exit codes.

## Supported tools

awsspectre, azurespectre, cispectre, clickspectre, dnsspectre, ecrspectre, elasticspectre, gcpspectre, gcsspectre, iamspectre, kafkaspectre, kubespectre, logspectre, logtap, mongospectre, pgspectre, rdsspectre, redisspectre, s3spectre, snowspectre, tote, vaultspectre

## Safety

spectrehub operates in **read-only mode**. It orchestrates read-only scanners and aggregates their output — never modifies your infrastructure.

## Documentation

| Document | Contents |
|----------|----------|
| [Architecture](docs/architecture.md) | System design, normalized issue model, project structure |
| [CLI Reference](docs/cli-reference.md) | All commands, flags, config, exit codes, CI/CD examples |

## License

[Business Source License 1.1](LICENSE) — source available, free for non-production use. Converts to Apache 2.0 after 4 years per version.

---

Built by [Obsta Labs](https://obstalabs.dev)
