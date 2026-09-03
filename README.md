# jsonrpc

[![CI](https://github.com/faustbrian/go-jsonrpc/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/faustbrian/go-jsonrpc/actions/workflows/ci.yml)
[![CodeQL](https://img.shields.io/badge/CodeQL-required-blue)](https://github.com/faustbrian/go-jsonrpc/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-100%25_required-blue)](CONTRIBUTING.md#verification)
[![Mutation](https://img.shields.io/badge/mutation-100%25_required-blue)](CONTRIBUTING.md#verification)
[![Documentation](https://img.shields.io/badge/docs-checked_in_CI-blue)](docs/)
[![Go Reference](https://pkg.go.dev/badge/github.com/faustbrian/go-jsonrpc.svg)](https://pkg.go.dev/github.com/faustbrian/go-jsonrpc)
[![Release](https://img.shields.io/github/v/release/faustbrian/go-jsonrpc?sort=semver)](https://github.com/faustbrian/go-jsonrpc/releases)
[![Go](https://img.shields.io/badge/go-1.26.6-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

`jsonrpc` is a transport-neutral, full JSON-RPC 2.0 server and client
package. Protocol behavior is explicit, errors are auditable, middleware is
composable, HTTP is optional, and malformed input is conformance- and
fuzz-tested.

## Status

The package has a stable v1 API and wire contract. Production
package code is held to meaningful 100% statement coverage.

## Requirements

- Go 1.26.6 or later
- no runtime dependencies outside the standard library

## Installation

```sh
go get github.com/faustbrian/go-jsonrpc
```

## Quickstart

```go
registry := jsonrpc.NewRegistry()
err := registry.Register("math.add", func(
    ctx context.Context,
    params json.RawMessage,
) (any, error) {
    values, rpcErr := jsonrpc.DecodeParams[[]int](params)
    if rpcErr != nil || len(values) != 2 {
        return nil, jsonrpc.InvalidParams()
    }

    return values[0] + values[1], nil
})
if err != nil {
    return err
}

handler := jsonrpc.NewHTTPHandler(jsonrpc.NewDispatcher(registry))
```

Trusted protocol adapters can register reserved `rpc.*` methods explicitly
with `Registry.RegisterSystem`; ordinary application registration continues to
reject that namespace.

Use `NewClient` with `NewHTTPTransport` for client calls. The
[quickstart](docs/quickstart.md) contains complete server, client,
notification, and batch examples.

## Package Guarantees

- requests, notifications, and explicit null IDs remain distinct
- string, number, and null IDs round-trip without coercion
- standard errors use the required codes and response shapes
- batch and notification-only behavior follows JSON-RPC 2.0
- clients validate response shape, ID correlation, duplicates, and missing
  batch members
- dispatcher and client parsing are independently resource-bounded
- protocol dispatch remains transport-neutral
- adapters can use `Dispatcher.DispatchSingle` to apply a compatible custom
  response envelope without decoding an already encoded dispatcher response

## Documentation

Start with the [documentation index](docs/README.md), [quickstart](docs/quickstart.md),
[guide for adopting the package](docs/adoption.md), and [API reference](docs/api.md). Use the
[conformance matrix](docs/conformance.md), [middleware guide](docs/middleware.md),
[security guide](docs/security.md), and [specification decision register](docs/specification-decisions.md)
for production review.

Release history is maintained in [CHANGELOG.md](CHANGELOG.md).
Runnable programs live under [examples](examples).
Shared construction, ownership, lifecycle, and composition expectations are in
the versioned [Golib ecosystem index](https://github.com/faustbrian/go-library-tools/blob/v1.4.0/docs/ecosystem/README.md)
and its [Protocols and descriptions family](https://github.com/faustbrian/go-library-tools/blob/v1.4.0/docs/ecosystem/design-language.md#package-families-and-selection).

## Development

Run `make cohesion` and `make check` before submitting a change. This enforces formatting, static
analysis, race tests, meaningful 100% coverage, fuzz smoke, benchmarks,
documentation, and vulnerability scanning.

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md) and follow the
[code of conduct](CODE_OF_CONDUCT.md). Protocol and public API changes require
explicit compatibility analysis.

## Security

Report vulnerabilities privately according to [SECURITY.md](SECURITY.md).
Review [docs/security.md](docs/security.md) before exposing a dispatcher to
untrusted clients.

## License

`jsonrpc` is available under the [MIT License](LICENSE). Attribution and
third-party policy are recorded in [NOTICE](NOTICE) and
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

## Related packages

- [OpenRPC](https://github.com/faustbrian/go-openrpc) models and validates
  OpenRPC descriptions and integrates discovery with this package.
- [service](https://github.com/faustbrian/go-service) provides application
  lifecycle and HTTP serving without changing JSON-RPC semantics.
