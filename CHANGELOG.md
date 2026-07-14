# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/).

## Unreleased

### Added

- Transport-neutral JSON-RPC 2.0 request, notification, response, and batch
  processing.
- Concurrency-safe server registry, middleware, request context, safe error
  mapping, and panic containment.
- Plain `net/http` handler with media-type and body-size enforcement.
- Typed client calls, notifications, mixed batches, strict response validation,
  custom ID generation, and custom transport support.
- Bounded HTTP client transport with headers and caller-provided HTTP clients.
- Official-spec conformance fixtures, meaningful full coverage, race tests,
  fuzz targets, and single/batch benchmarks.
- CI, static analysis, security scanning, dependency updates, benchmark/fuzz
  automation, and semantic-version tag releases.
- Quickstart, architecture, API, cookbook, adoption, middleware,
  troubleshooting, FAQ, compatibility, release, and community documentation.

[Keep a Changelog]: https://keepachangelog.com/en/1.1.0/
[Semantic Versioning]: https://semver.org/
