# JSON-RPC interoperability harness

## Status and purpose

This is an internal, non-releasable engineering module. Do not install or
import it in an application. It compares the repository's documented JSON-RPC
decisions with pinned `github.com/creachadair/jrpc2` v1.3.5 behavior and
records deliberate differences rather than changing the public package
contract to match another implementation.

The harness depends on `github.com/faustbrian/go-jsonrpc` v1.0.0 and directly
on the external `github.com/creachadair/jrpc2` v1.3.5 module. The peer remains
outside the public root module: this harness does not make jrpc2 a runtime
dependency of applications that adopt `go-jsonrpc`.

## Run and evidence

Run the harness from the repository root:

```sh
make interoperability
```

The command executes both implementations in-process and verifies the checked-
in [`specification/interoperability.tsv`](../specification/interoperability.tsv)
matrix. It covers invalid notification-shaped requests, explicit null and
numeric IDs, empty and mixed batches, response ordering, malformed JSON,
duplicate members, notification-only HTTP responses, methods, media types, and
HTTP status mapping. Differences are attributable to the matching
[`JSONRPC-DEC-*`](../docs/specification-decisions.md) decision rather than
treated as peer failures.

The bridge and HTTP response bodies are closed by the harness. It owns no
external service, persistent state, background worker, credential, or
production lifecycle. Fixtures are synthetic and must not contain production
payloads or secrets.

## Compatibility and maintenance

The module requires Go 1.26.6. Repository CI verifies it on Ubuntu 24.04; other
platforms are not part of the published tested-platform claim. Update the peer
version, module checksums, recorded matrix, decision evidence, and changelog in
one reviewed change when the comparison target changes.

Use the public [`jsonrpc` module](../README.md) in applications. Shared
construction, ownership, lifecycle, and composition expectations are in the
versioned [Golib ecosystem index](https://github.com/faustbrian/go-library-tools/blob/v1.5.3/docs/ecosystem/README.md)
and its [Protocols and descriptions family](https://github.com/faustbrian/go-library-tools/blob/v1.5.3/docs/ecosystem/design-language.md#package-families-and-selection).
