# spectrehub

[![CI](https://github.com/ppiankov/spectrehub/actions/workflows/ci.yml/badge.svg)](https://github.com/ppiankov/spectrehub/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ppiankov/spectrehub)](https://goreportcard.com/report/github.com/ppiankov/spectrehub)

**spectrehub** — Unified infrastructure audit aggregator for the Spectre tool family. Part of [SpectreHub](https://spectrehub.dev).

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

### Homebrew

```sh
brew tap ppiankov/tap
brew install spectrehub
```

### From source

```sh
git clone https://github.com/ppiankov/spectrehub.git
cd spectrehub
make build
```

### Usage

```sh
spectrehub collect --all --format json
```

## CLI commands

| Command | Description |
|---------|-------------|
| `spectrehub collect` | Run Spectre tools and aggregate reports |
| `spectrehub report` | Generate unified report from collected data |
| `spectrehub status` | Show installed tools and last run status |
| `spectrehub version` | Print version |

## Supported tools

awsspectre, azurespectre, cispectre, clickspectre, dnsspectre, ecrspectre, elasticspectre, gcpspectre, gcsspectre, iamspectre, kafkaspectre, kubespectre, logspectre, logtap, mongospectre, pgspectre, rdsspectre, redisspectre, s3spectre, snowspectre, tote, vaultspectre

## Safety

spectrehub operates in **read-only mode**. It orchestrates read-only scanners and aggregates their output — never modifies your infrastructure.

## License

MIT — see [LICENSE](LICENSE).

---

Built by [Obsta Labs](https://github.com/ppiankov)
