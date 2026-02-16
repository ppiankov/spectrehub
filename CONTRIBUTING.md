# Contributing to SpectreHub

Thanks for contributing! This repo follows lightweight, test-first changes.

## Prerequisites
- Go 1.21+
- golangci-lint (for `make lint`)

## Adding a New Tool
When adding a new spectre tool, update all of the following:
- **Model struct**: add a report struct in `internal/models/<tool>.go` and register it in `internal/models/common.go` if applicable.
- **Parser**: teach `internal/collector/parser.go` how to unmarshal the report.
- **Normalizer**: map tool findings into the common issue format in `internal/aggregator/normalizer.go`.
- **Detector**: extend `internal/collector/detector.go` so the tool is recognized.
- **Testdata contract**: add a real output JSON file under `testdata/contracts/<tool>-vX.json`.

## Build / Test / Lint
- Build: `make build`
- Test: `make test`
- Contract tests: `make test-contract`
- Lint: `make lint`
- Format: `make fmt`

## PR Conventions
- Use **Conventional Commits** (e.g. `feat: ...`, `fix: ...`, `chore: ...`).
- Include or update tests for your changes; **test coverage is required**.
- Keep PRs focused and update docs when behavior or contracts change.
