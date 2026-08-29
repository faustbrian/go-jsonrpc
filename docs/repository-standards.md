# Repository Standards

This repository follows the shared maintenance baseline used by the
`faustbrian/go-*` OSS packages.

## Mandatory Root Files

Every repository contains `.gitattributes`, `.gitignore`,
`.golangci.yml`, `AGENTS.md`, `CHANGELOG.md`, `CLAUDE.md`,
`CODE_OF_CONDUCT.md`, `CONTRIBUTING.md`, `LICENSE`, `Makefile`, `NOTICE`,
`README.md`, `ROADMAP.md`, `SECURITY.md`, and `THIRD_PARTY_NOTICES.md`.

Completed implementation plans and verification snapshots belong in issue
tracking or Git history rather than the released source tree.

`NOTICE` identifies the project and its ownership. `THIRD_PARTY_NOTICES.md`
separately records attribution and provenance for copied, forked, generated,
or vendored third-party source. Both remain present even when no third-party
source requires attribution. A package may retain a different approved OSS
license when provenance requires it.

## Mandatory Documentation

The shared taxonomy is lowercase kebab-case and includes a documentation
index, quickstart, usage guidance, API reference, architecture, examples,
cookbook, FAQ, troubleshooting, migration, compatibility, performance,
security, Go safety and concurrency, and releasing guide.
Package-specific documents extend this taxonomy without renaming shared
concepts.

## Mandatory Automation

Every repository provides one pinned-SHA CI caller. The shared workflow owns
the benchmark, scheduled fuzzing, security, dependency, and release checks;
the repository keeps only its package-specific configuration and evidence. CI
uses the Go version declared by the repository manifest, and dependency review
runs on pull requests.

The local entry points are `make inventory`, `make check`, and `make ci`.
They invoke the same released `golib` contract used by CI.

The package family shares the `GO-SAFETY-1` baseline. It forbids `unsafe`,
cgo, and `go:linkname` in production code and standardizes ownership,
goroutine lifecycle, race, fuzz, resource-bound, leak, and benchmark evidence.

## Approved Package-Specific Differences

- `jsonapi` carries JSON:API feature, conformance, extension/profile,
  recommendation, and threat-model documentation.
- `jsonrpc` carries protocol conformance and middleware documentation.
- `queue` carries backend, delivery, lifecycle, failure, and integration
  documentation plus a live-backend integration workflow. Its fork provenance
  requires detailed third-party notices.
- `wire` carries format, dependency, and audit-evidence documentation.
- `tabular` carries format and ingest-limit documentation. It uses
  Apache-2.0 and retains XLS provenance notices.

Code, dependencies, fuzz targets, benchmark inputs, and domain-specific
security guidance are expected to differ. Shared policy wording and automation
structure must not drift without updating this contract across all affected
repositories.
