# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

- Publish schema-v2 cohesion metadata and versioned ecosystem navigation for
  the JSON-RPC module and its interoperability harness.

- Adopt the checksum-verified `go-library-tools` v1.3.0 CLI, add the local
  cohesion gate, and pin reusable-workflow cohesion enforcement to its final
  immutable revision.

### Documentation

- Make every registered JSON-RPC decision reproducible against the pinned
  maintained peer and separate maintained-peer differential evidence from
  official-fixture interoperability evidence.

  - JSONRPC-DEC-001 sha256:e7c97164879dd80e913343dc0591270476c667d38b4c9840a8c87b3344cd7e7f
  - JSONRPC-DEC-002 sha256:884338cefdc0b14295f2f8bc824f8f1b4a7232202e009b6151e72995aa90985e
  - JSONRPC-DEC-003 sha256:cd43cb366bfec47a9df2e0809353c0b1f473abba2f48d8178b19977c6ea044d2
  - JSONRPC-DEC-004 sha256:b5e05c5f685fec356b12c5c3283d60e81af847df40a58a5c3f682d67636757ec
  - JSONRPC-DEC-005 sha256:5a54e26f2808a25744877eedd498a83b143b8098a7ef337c58143df3cf0677f5
  - JSONRPC-DEC-006 sha256:c118a37adfbf495cacbc9a8ca16ce9676ae63c69093dab0d2d964f9142156db4
  - JSONRPC-DEC-007 sha256:8cf601cd0af0e3a175f80fab9719d2e6dd97647773fa4603c55689f8153344fe
  - JSONRPC-DEC-008 sha256:3ea835ba8ade0e5fa30a75d0f879018a54382113686f65982d624221f3850821
  - JSONRPC-DEC-009 sha256:70e0d1ae7a3d2d9a8f1bf3c877eef01faf64642ae203019cd237cd6eafb1152a
  - JSONRPC-DEC-010 sha256:9ed4d9b0f4bb1d32428b82fdf0c57de959fe78881884da703a7133d2b8f11cb2

- Make the [specification decision register](docs/specification-decisions.md)
  machine-auditable with exact source and errata monitoring, attributable
  conformance evidence, and classified interoperability results. Pin a
  reproducible `creachadair/jrpc2` v1.3.5 differential harness for null IDs,
  numeric ID equivalence, and HTTP binding decisions without adding the peer
  dependency to the public module.

  - JSONRPC-DEC-001 sha256:56d72c967858ae0ed974a7175e3974e21645a8ee01cbfdcf402c3f1748412bed
  - JSONRPC-DEC-002 sha256:e2898b8880a5b946f6378b98acd691fc4b67a8d7c1adb543e67e61a73a17bf84
  - JSONRPC-DEC-003 sha256:f709c7085240ac271f821bc8be8f8a5cff3cdcd6427ef543a5f2ab5e103ac5bf
  - JSONRPC-DEC-004 sha256:1be90684db04140fabb420f35597ab36f8b5d81f229678d82d3b0a3d7777b4df
  - JSONRPC-DEC-005 sha256:b2bcee89e0d0dc65c2cdbf55fc18b1268a716479ae2ad1a0da3a41ddab100c61
  - JSONRPC-DEC-006 sha256:e3134a8d0891f445195ffa91638c46d126303b924edc4d56626f4a64e6fff6db
  - JSONRPC-DEC-007 sha256:67b17e6ee213dc6ee07374f041a6cbb43b7d0115b333588376fcc2600200c130
  - JSONRPC-DEC-008 sha256:6a8c8ca88fa6ae039aac7a65b11ae9f4f66a61b33e53f6d68dde62541c3c5695
  - JSONRPC-DEC-009 sha256:e8b69e6bdd78f017845c2f1a47c2e0f64674395a5532258d07a13a9fccfb848a
  - JSONRPC-DEC-010 sha256:c02698468fddfee892dad09d2eccd29829b15794379a7e91cd2aad650f7e93c6

- Use the released `go-library-tools` v1.2.0 CLI and immutable merged
  workflow at `1f9629e5f27418600460b55a50a5b2fc81697fab` while preserving
  JSON-RPC conformance and mutation evidence.

- Clarify how shared safety-policy updates are coordinated across standalone
  repositories.

- Replace archived monorepo and AI-generated documentation entry points with
  a standalone, human-oriented documentation structure.

## [1.0.0] - 2026-08-25

### Fixed

- Strengthen single-request boundary verification at the exact byte limit and
  for invalid UTF-8 payloads.

- Align the executable documentation contract with the stable v1 repository
  state and current standalone contributor and conduct policies.

### Changed

- Validate action pinning from the standalone repository root and leave
  repository-foundation policy to the authoritative repository contract.

- Exclude intentional nested modules from root local-proxy archives so local,
  bootstrap, CI, and public module checksums describe the same source
  boundary.

- Track the pinned documentation-tool lockfile so clean CI checkouts install
  the exact validated cspell dependency.

- Reconcile standalone dependency checksums against deterministic current
  module archives so CI, local verification, and release consumers resolve
  identical content.

- Harden standalone documentation validation with deterministic spelling and
  link checks, package-specific documentation gates, and repository-local
  contributor guidance.

### Documentation

- Replace obsolete standalone-repository links and workflow claims with
  monorepo-canonical targets and current release guidance.
- Document the package's initial stable `v1.0.0` scope and compatibility
  boundary.

### Compatibility

- Added a pinned module export baseline so incompatible public API changes
  fail the canonical repository gate.

### Changed

- Publish the module from its standalone `github.com/faustbrian/go-jsonrpc` identity while preserving its documented API and behavior.
- Pin the official JSON-RPC 2.0 example corpus and make security, resource,
  compatibility, and wire consequences explicit for every protocol decision.
- Reject duplicate members at every nested object depth during strict
  parameter decoding instead of accepting last-member-wins ambiguity.
- Expose JSON-RPC specification verification as an explicit conformance gate.
- Added the `GO-SAFETY-1` ownership, concurrency, race, fuzz, resource, and
  benchmark standard with an executable `make safety` gate.
- Moved AI planning and hardening briefs into `.ai/` and clarified the
  separate purposes of project and third-party notice files.

### Added

- `Dispatcher.DispatchSingle` for adapters that need a validated typed response
  before applying their own wire-compatible response encoding.
- Add an auditable specification-decision register for JSON-RPC ambiguities,
  defensive JSON policy, and the package's HTTP binding choices.
- A configurable dispatcher JSON nesting-depth limit that rejects excessive
  arrays and objects before protocol decoding or handler execution.
- `Registry.RegisterSystem` for trusted protocol integrations such as
  OpenRPC's reserved `rpc.discover` method.
- A standardized OSS repository skeleton covering policy, documentation,
  legal notices, Go tooling, pinned CI, security, and release automation.

### Fixed

- Keep module-archive tests scoped to files shipped with the JSON-RPC module;
  repository-root workflow policy remains owned by the root verification gate.
- Strengthen protocol, transport, dispatcher-hook, and named-parameter boundary
  verification so every viable JSON-RPC mutation is detected without timeout
  or equivalent boundary predicates.

### v1.0.0 scope

The following initial scope is included in `v1.0.0`.

#### Added

- Evidence-driven audit and hardening goal covering JSON-RPC conformance,
  hostile inputs, transports, concurrency, compatibility, and release readiness.
- Living hardening report, threat model, and normative JSON-RPC conformance
  matrix with explicit evidence and open-risk tracking.
- Dispatcher payload and batch-member options with safe defaults of four MiB
  and 1,024 members, plus an inspectable request-limit protocol error.
- A transport-neutral four-MiB client reply parsing limit with an additive
  option and inspectable oversized-response sentinel.
- Transport-neutral JSON-RPC 2.0 request, notification, response, and batch
  processing.
- Canonical public module path at `github.com/faustbrian/go-jsonrpc`.
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
- Guarded patch, minor, and major Makefile release commands that create local
  annotated tags without pushing them.
- Quickstart, architecture, API, cookbook, adoption, middleware,
  troubleshooting, FAQ, compatibility, release, and community documentation.
- Shared repository instructions for Claude Code through the canonical
  `AGENTS.md` rules.
- Generated `llms.txt` index and `llms-full.txt` bundle sourced from the
  canonical Markdown documentation.
- Canonical JSON-RPC 2.0 section links on protocol types, dispatch behavior,
  error objects, and conformance fixtures.

#### Fixed

- Bound fuzz-smoke concurrency to avoid deadline flakes on high-core hosts.
- Reject duplicate members in request, response, and error envelopes instead
  of inheriting `encoding/json`'s last-member-wins behavior. This defensive
  interoperability policy prevents ambiguous peers from interpreting the same
  protocol object differently.
- Reject duplicate generated IDs within a client batch before transport I/O so
  every non-notification response remains unambiguously correlatable.
- Reject case variants of reserved request, response, and error-object members
  instead of inheriting `encoding/json`'s case-insensitive struct matching.
- Reject invalid UTF-8 throughout protocol envelopes and classify invalid
  server input as a parse error instead of silently replacing malformed bytes.
- Reject duplicate top-level names in `DecodeParams` named-parameter objects
  before Go's JSON decoder can collapse them.
- Reject oversized transport-neutral payloads and batches before parsing or
  handler execution can amplify their CPU, allocation, or downstream cost.
- Normalize arbitrarily long numeric-ID exponents with linear decimal-string
  arithmetic instead of allocation-heavy arbitrary-precision integers.
- Reject oversized replies from every client transport before JSON parsing,
  not only when the built-in HTTP transport enforces its body limit.
- Make the exported `Registry` zero value safe for concurrent registration and
  lookup instead of panicking on its first registration.
- Ignore nil functional options consistently and return HTTP request
  construction errors, including nil-context misuse, without network I/O.
- Expand continuous fuzzing across response and error decoding, ID round
  trips, and single and batch client correlation.
- Store the complete official JSON-RPC example corpus as stable conformance
  fixtures consumed by automated tests.
- Add hostile-boundary benchmarks for maximum dispatcher payloads and
  oversized generic client replies.
- Document every exported Go declaration and clarify safe custom error-code
  selection outside the JSON-RPC reserved range.
- Stop the default HTTP transport from following redirects and potentially
  forwarding caller-configured credentials to another origin.
- Reject trailing JSON values passed directly to `ID.UnmarshalJSON`.
- Keep `StringID` correlation equal to its actual JSON encoding when a Go
  string contains invalid UTF-8.
- Seed fuzzing with the checked-in specification corpus, deep JSON, and large
  batches in addition to malformed and boundary values.
- Add race, cancellation, chunked-reader, and response-body cleanup regression
  coverage for runtime ownership contracts.
- Preserve the nil-context transport regression test under both direct
  Staticcheck and golangci-lint without weakening either analyzer.

[Keep a Changelog]: https://keepachangelog.com/en/1.1.0/
[Semantic Versioning]: https://semver.org/
[Unreleased]: https://github.com/faustbrian/go-jsonrpc/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/faustbrian/go-jsonrpc/releases/tag/v1.0.0
