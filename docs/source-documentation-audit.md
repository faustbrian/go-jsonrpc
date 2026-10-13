# Source documentation audit

[Documentation index](README.md) · [Repository entry](../README.md)

## Scope and compact plan

Tier A source documentation at main source
`d9d43f8db6c82354ab288026d1207c4798749823`, in a detached task worktree.
The unrelated original checkout is preserved. The root module owns public
`jsonrpc`, release tooling and three command examples; the separate
`interoperability` module owns a non-releasable pinned-peer harness.

Acceptance: inspect every tracked Go file and public declaration, exported field,
interface method and canonical package comment; repair accurate contracts and
useful evidence-backed rationale without changing library/tool/harness behavior.
Only the three root executable documentation examples may change tokens to
handle public errors while keeping their demonstrated successful output.

Named risks: raw-message/ID aliasing, decoded member-presence semantics,
notification/null distinction, synchronous and concurrent borrowed callbacks,
partial batch-result mutation, caller-owned cancellation/network acquisition,
protocol versus transport errors and secret-safe diagnostic boundaries. Smallest
checks: all-file parse and non-comment token/AST equality excluding the explicitly
changed examples, name-led public docs, canonical package/rendered documentation,
changed-example execution, relevant links/markers/spec structure, pinned secret
scan and one ordinary complete-final-diff inspection. No unchanged runtime,
race/fuzz/benchmark/peer/consumer/release campaigns are required.

## Reproducible inventory

`git ls-files '*.go' 'go.mod' '*/go.mod'` inventories all 29 tracked Go files and
two module files. `modules.json`/`packages.json` identify these seven packages:

| Module | Package directory | Purpose |
| --- | --- | --- |
| `.` | `.` | Public JSON-RPC library |
| `.` | `cmd/semvercheck` | Release-tag CLI |
| `.` | `internal/semver` | Private release-tag helper |
| `.` | `examples/client` | Standalone client command |
| `.` | `examples/e2e` | In-process HTTP example command |
| `.` | `examples/server` | Standalone server command |
| `interoperability` | `.` | Engineering-only peer harness |

`rg -n '^func (Test|Fuzz|Benchmark|Example)' --glob '*.go'` inventories entry
points. Every tracked file was read; both internal and external test packages,
all fixtures/helpers, seven fuzz targets and their seeds, ten benchmarks,
three executable examples and all command variants were included. There are no
Go build tags, generated Go files, generators, deprecations or unfinished-work
markers at the selected source. Existing lint/security rationale is preserved.

| Tracked Go file | Investigated contract and evidence |
| --- | --- |
| [automation_test.go](../automation_test.go) | Tracked repository/docs/workflow structure checks; unchanged |
| [benchmark_test.go](../benchmark_test.go) | Ten benchmarks: dispatch, limits, hostile ID exponents and oversized client replies |
| [client.go](../client.go) | Transport and ID-generator seam, client correlation/batch partial state, request builders |
| [client_test.go](../client_test.go) | Correlation identity, batch membership/count/duplicate IDs and concurrent generator tests |
| [cmd/semvercheck/main.go](../cmd/semvercheck/main.go) | Release-tag CLI lifecycle and output/exit boundaries |
| [cmd/semvercheck/main_test.go](../cmd/semvercheck/main_test.go) | Focused stable-version CLI examples; unchanged |
| [conformance_test.go](../conformance_test.go) | Pinned transcription of official JSON-RPC 2.0 examples |
| [doc.go](../doc.go) | Canonical public package protocol and ownership entry point |
| [documentation_test.go](../documentation_test.go) | Existing exported-symbol, docs structure/version and spec reference checks |
| [error.go](../error.go) | Public error/code/data versus local cause; mutable fluent methods and decoder validation |
| [example_test.go](../example_test.go) | Three executable examples: HTTP call, batch correlation, middleware; error handling repair |
| [examples/client/main.go](../examples/client/main.go) | Standalone network client example; default bounded HTTP transport |
| [examples/e2e/main.go](../examples/e2e/main.go) | In-process HTTP e2e example with owned test server cleanup |
| [examples/server/main.go](../examples/server/main.go) | Standalone server example with caller-owned HTTP timeouts and lifecycle |
| [fuzz_test.go](../fuzz_test.go) | Seven native parser/dispatch/client-correlation targets and official fixture seeds |
| [hardening_test.go](../hardening_test.go) | Nil options, malformed envelopes, mapper/HTTP cleanup and cancellation propagation |
| [http.go](../http.go) | HTTP handler admission/body ownership/status and actual media-type predicate |
| [http_test.go](../http_test.go) | POST/media/body status, chunked closure and accepted media-type cases |
| [http_transport.go](../http_transport.go) | Borrowed network client, bounded replies, redirect/proxy defaults and diagnostic boundaries |
| [http_transport_test.go](../http_transport_test.go) | Exact limits, body cleanup, timeout/proxy/redirect and protected diagnostic regressions |
| [internal/semver/semver.go](../internal/semver/semver.go) | Private release-tool package, exported tag validation and bounded-machine arithmetic caveat |
| [internal/semver/semver_test.go](../internal/semver/semver_test.go) | Stable/prerelease/build tag syntax partitions; unchanged |
| [interoperability/runner.go](../interoperability/runner.go) | Separate non-releasable peer harness; workspace source and pinned peer comparison |
| [interoperability/runner_test.go](../interoperability/runner_test.go) | Recorded peer matrix exactness; unchanged, no new interoperability claim |
| [mutation_regression_test.go](../mutation_regression_test.go) | Existing semantic ID/presence/config/scanner/hook/name regressions; unchanged |
| [protocol.go](../protocol.go) | Raw ID ownership, exact-decimal normalization, presence versus validation, strict envelope policy |
| [protocol_test.go](../protocol_test.go) | Round trip/correlation, reuse, UTF-8, duplicate/case/presence and error partitions |
| [server.go](../server.go) | Registry lock owner, synchronous callbacks, hook borrowing, limits and response-overflow semantics |
| [server_test.go](../server_test.go) | Registry/order/resource/error/middleware/params/hook ownership and overflow tests |

## Missing versus investigated rationale

Missing/underspecified: individual sentinel comments, Transport/IDGenerator
methods, all private/tool/example canonical package comments, zero-value and
borrowed-resource requirements, ID/raw buffer ownership and reuse, validation
versus serialization/presence, integer generator wraparound, callback lifetime,
partial batch mutation and lack of automatic retry/preemption. Existing useful
source comments and specification references remain intact.

The public source protocol is JSON-RPC 2.0; JSON input policy cites RFC 8259,
and HTTP binding decisions cite RFC 9110 and RFC 6839. Existing source authority
IDs/digests are in [monitoring](../specification/monitoring.json); all eleven
stable JSONRPC-DEC IDs and executable/fixture bindings are in the
[decision register](specification-decisions.md). The checked-in official-example
transcription is version 2.0, 3,605 bytes, SHA256
`3fa49a5ec274ac7c53dab8d70431fd728bdbad22e167e2fdca4aa673e8496a03`.
The peer is `creachadair/jrpc2` v1.3.5; the nested module separately declares
published go-jsonrpc v1.1.1, while explicit `go.work` selects the candidate source.
No online source freshness, peer rerun or new conformance proof is claimed.

Two concrete source-contract contradictions received narrow reconciliation:
`OnResponse` borrows the actual Request and its ID backing storage, so blanket
hook-mutation isolation is false; borrowed requests/causes must remain read-only.
`IsJSONContentType` accepts any successfully parsed media type ending in `+json`,
not only an application subtype, so DEC-010/current guides must reflect the
actual predicate. Stable decision IDs/source pins/old digest history remain;
only DEC-010 current-record wording and an appended canonical digest changed.
New digest: `45257b23fa4ee86b3064baa0ff9b28c84b974717589427d72377d93e713791a5`.
All prior history/source pins remain intact; no archived decision or historical authority review was rewritten.
Any stronger hook isolation or media-type restriction is a separate behavioral
follow-up for maintainers, not a runtime repair in this pass. ID copies share raw
storage when re-decoded, and internal NextStable uses unchecked Go int arithmetic;
contracts must not invent copy isolation or overflow validation.

## API, generation and verification boundary

`.golib.yaml` selects structural `apidiff` at `api/baseline.txt`, not a prose
Go-doc snapshot. There is no scripts directory or comment-snapshot generator;
Makefile/cmd/internal sources contain no separate prose snapshot operation.
The existing API baseline, module files, fixture bytes and interoperability
matrix remain unchanged. Go 1.27.2 is the catalog producer. Rendering covered
all seven packages using their own module boundary, with three owned caches in a coordinator-serialized bounded run.

## Verification and final disposition

- Catalog Go 1.27.2 formatted all 14 changed Go files. A task-owned standard-library
  checker parsed every tracked Go file, validated 142 name-led exported
  declarations/fields/interface methods and seven canonical package comments,
  and proved exact non-comment tokens and ASTs unchanged outside
  `example_test.go`. That file alone intentionally changes executable examples.
- `go test -mod=readonly . -run '^Example' -count=1 -timeout=2m` passed all three
  changed examples, preserving outputs `5`, `2 4` and `ping`. No other runtime
  tests or unchanged command examples were executed.
- `go doc -all` rendered the root, internal/semver, semvercheck and three example
  packages, plus the nested harness from its own module with GOWORK=off. Complete
  rendered contracts were inspected. This does not execute/qualify the harness
  or alter its published-dependency/candidate-workspace distinction.
- Coordinator-supplied native `golib source-docs check` passed for both module
  boundaries/all 29 Go files; `golib docs check --local` and offline
  `golib specification check` passed (one specification-bearing module,
  eleven decisions). No candidate-tool release or online-source freshness is
  implied by these local results.
- Reused the native typed Decision declaration and decisionDigest body in a
  disposable standard-library helper: all eleven current digests match history.
  The initial raw-map reproduction exposed absent `interoperability_evidence`
  needing typed nil-to-null encoding; native omitempty defaults and HTML escaping
  were then honored. All original current digests were reproduced before edits.
- Only five DEC-010 prose fields changed. Every old history prefix, decision ID,
  authority/conformance/fixture/matrix byte, API baseline and module file is
  unchanged; all eleven typed/rendered current records agree. No new optional
  digest prose was added to Unreleased, and historical release prose is retained.
- Relevant local links, inventory rows, marker/generated/tag/deprecation absence,
  example-output/error-handling structure, changed-content secret patterns and
  pinned Gitleaks v8.30.1 scan were checked. `git diff --check` passed.
- Ordinary complete-final-diff Tier A inspection found no unresolved source-doc
  omission or invented protocol/ownership guarantee. Borrowed hook-request ID
  mutation and broader +json admission remain source-traced behavioral follow-ups;
  no enforcement fix or runtime reproduction is claimed. Internal release helper
  arithmetic remains caller-bounded, not newly overflow-validated.
- Broad runtime, race, active fuzz, benchmark, peer, live-service, consumer and
  release campaigns were not run: their library/harness executable inputs are
  unchanged and this is not new conformance or security-hardening qualification.
- All owned GOCACHE/GOMODCACHE/GOTMPDIR, checker/digest helper and new execution
  output were removed on success/failure; historical evidence was preserved.
  Parent-owned verifier/scanner binaries remain untouched. The committed
  worktree is retained for coordinator delivery.

This audit does not claim remote-main delivery, hosted CI, protocol hardening,
consumer adoption or release qualification.

## Objective Enforcement

The qualified public Tools v2.4.0 binary and immutable reusable CI source
provide the strict tracked-source check through `make source-docs-check`,
the existing local check/CI routes and Required aggregation. Release and
selected security/assurance routes retain their existing controls. This
structural status supplements the rendered semantic audit; it does not
certify prose meaning, generator execution or runtime conformance.

Choose the next guide in the [documentation index](README.md)
or return to the [repository entry](../README.md) for installation and ownership.
