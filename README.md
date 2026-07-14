# go-jsonrpc

`go-jsonrpc` is a transport-neutral JSON-RPC 2.0 server and client package for
Go. It is designed for production APIs: protocol behavior is explicit, errors
are auditable, middleware is composable, HTTP is optional, and malformed input
is covered by conformance and fuzz tests.

The module is currently preparing its first stable release. Until `v1.0.0`,
minor releases may refine the public API. Protocol behavior is treated as
compatibility-sensitive at every version.

## Install

```sh
go get github.com/shipit-dev/go-jsonrpc
```

The module requires Go 1.24 or newer and has no runtime dependencies outside
the standard library.

## Quick server

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
    log.Fatal(err)
}

handler := jsonrpc.NewHTTPHandler(jsonrpc.NewDispatcher(registry))
log.Fatal(http.ListenAndServe(":8080", handler))
```

## Quick client

```go
transport, err := jsonrpc.NewHTTPTransport("http://localhost:8080")
if err != nil {
    log.Fatal(err)
}
client := jsonrpc.NewClient(transport)

sum, err := jsonrpc.Call[int](context.Background(), client, "math.add", []int{2, 3})
if err != nil {
    log.Fatal(err)
}
fmt.Println(sum)
```

## Protocol guarantees

- Requests, notifications, and explicit `null` IDs remain distinct.
- String, number, and `null` IDs are echoed without coercion.
- Parse error, invalid request, method not found, invalid params, and internal
  error responses use the standard codes and shapes.
- Empty batches produce one invalid-request response; notification-only
  batches produce no response; mixed batches omit notification responses.
- Client responses are checked for version, shape, ID correlation, duplicates,
  missing batch members, and result decoding errors.
- The core dispatcher accepts bytes and returns bytes, with no HTTP assumption.

The conformance suite includes the normative JSON-RPC 2.0 examples. Production
package code is held at meaningful 100% statement coverage, with race, fuzz,
static-analysis, vulnerability, and benchmark automation.

## Documentation

- [Quickstart](docs/quickstart.md)
- [Architecture](docs/architecture.md)
- [Public API reference](docs/api.md)
- [Middleware and observability](docs/middleware.md)
- [Scenario cookbook](docs/cookbook.md)
- [Adoption guide](docs/adoption.md)
- [Troubleshooting](docs/troubleshooting.md)
- [FAQ](docs/faq.md)
- [Compatibility policy](docs/compatibility.md)
- [Versioning and release guide](docs/releasing.md)
- [Roadmap](ROADMAP.md)
- [Changelog](CHANGELOG.md)

Runnable programs live under [`examples`](examples).
See [`CONTRIBUTING.md`](CONTRIBUTING.md) for development instructions,
[`SECURITY.md`](SECURITY.md) for private vulnerability reporting, and
[`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md) for community expectations.

## License

MIT. See [`LICENSE`](LICENSE).
