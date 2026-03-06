# Architecture

## Workflow

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

## Design principles

- **Minimal** — no databases, no dependencies, no cloud required
- **Fast** — concurrent processing with Go goroutines
- **Extensible** — easy to add new Spectre tools
- **Portable** — single binary, runs anywhere

## Concurrency model

- Worker pool with configurable concurrency (default: 10 goroutines)
- Parallel file reading and parsing
- Non-blocking result aggregation

## Storage

- Local filesystem only
- JSON files in `.spectre/runs/`
- Future: S3, PostgreSQL, SQLite backends

## Normalized issue model

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

This enables diff, query, and correlation across tools.

## Project structure

```
spectrehub/
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
└── Makefile
```
