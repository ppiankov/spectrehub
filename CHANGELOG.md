# Changelog

All notable changes to SpectreHub will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-02-23

### Added

- spectre/v1 envelope detection, parsing, and validation (WO-14)
- Contract tests with golden files for all 6 tools in spectre/v1 format (WO-15)
- PgSpectre and MongoSpectre support: models, detection, parsing, normalization
- Severity enum validation for spectre/v1 findings (high, medium, low, info)
- Target type mapping for all 6 tools (s3, postgres, kafka, clickhouse, vault, mongodb)
- Invalid input testdata for spectre/v1 (bad severity, missing fields)

### Changed

- Detector uses three-phase approach: Phase 0 (spectre/v1), Phase 1 (tool field), Phase 2 (structural)
- Tool name mapping extracted into reusable `mapToolName()` helper
- Validator auto-dispatches spectre/v1 envelopes regardless of detected tool type
- Normalizer handles spectre/v1 findings directly via `NormalizeSpectreV1()`
- Contract test suite expanded from 6 to 12 golden files

### Fixed

- Resolved errcheck and staticcheck lint warnings

## [0.1.0] - 2026-02-15

### Added

- Initial release: collect, aggregate, and summarize reports from Spectre tools
- Support for VaultSpectre, S3Spectre, KafkaSpectre, and ClickSpectre
- Tool detection via explicit `tool` field and structural analysis fallback
- Report validation for all four supported tools
- Normalization pipeline converting tool-specific findings to `NormalizedIssue`
- Health scoring, trend analysis, and prioritized recommendations
- JSON and text output formats
- CI/CD exit codes: 0 (OK), 1 (policy fail), 2 (validation error), 3 (runtime error)
- Historical storage with SQLite backend
- Config file support (`.spectrehub.yaml`)

[0.2.0]: https://github.com/ppiankov/spectrehub/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/ppiankov/spectrehub/releases/tag/v0.1.0
