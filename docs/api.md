# Public API reference

This is the semantic reference for the exported surface. Exact Go signatures
are also available through `go doc github.com/shipit-dev/go-jsonrpc`.

## Protocol

- `Version` is the required `"2.0"` protocol marker.
- `Request` contains `JSONRPC`, `Method`, raw `Params`, and an `ID`. Its custom
  JSON implementation preserves ID presence. `Validate` checks the envelope;
  `IsNotification` is true only when the ID member is absent.
- `Response` contains raw `Result`, structured `Error`, and `ID`. `Validate`
  requires version 2.0, an ID, and exactly one of result or error.
- `ID` represents a string, JSON number, explicit null, or an internal missing
  state. Construct values with `StringID`, `NumberID`, and `NullID`; inspect with
  `Kind` and compare without coercion using `Equal`.
- `IDKind` values are `IDMissing`, `IDString`, `IDNumber`, and `IDNull`.
- `NewRequest` and `NewNotification` validate method names and require params to
  encode as an object or array when present.
- `DecodeParams[T]` strictly decodes params and maps malformed or unknown fields
  to `InvalidParams`.

### Errors

`Error` is both a JSON-RPC error object and a Go error. `Code`, `Message`, and
optional raw `Data` cross the wire. `WithCause` retains a local cause without
serializing it; `WithData` JSON-encodes safe public details.

`NewError` constructs application-defined errors. `ParseError`,
`InvalidRequest`, `MethodNotFound`, `InvalidParams`, and `InternalError` build
the standard errors. Their codes are also exported as `CodeParseError`,
`CodeInvalidRequest`, `CodeMethodNotFound`, `CodeInvalidParams`, and
`CodeInternalError`.

## Server

- `Handler` is `func(context.Context, json.RawMessage) (any, error)`.
- `Registry` is created by `NewRegistry`; `Register` adds one unique method and
  `Lookup` supports explicit inspection.
- `Dispatcher` is created with `NewDispatcher`. `Dispatch` processes a single
  request or batch and returns bytes plus a boolean indicating whether a reply
  exists.
- `Middleware` wraps a `Handler`. `WithMiddleware` installs middleware in the
  listed order, with the first item outermost.
- `ErrorMapper` converts ordinary application errors to safe RPC errors.
  `WithErrorMapper` replaces the default internal-error mapping.
- `RequestFromContext` retrieves the validated request during middleware or
  handler execution.
- Registration errors are detectable with `ErrInvalidMethodName`,
  `ErrMethodAlreadyRegistered`, and `ErrNilHandler`.

## Client

- `Transport` exchanges one complete JSON-RPC payload. `TransportFunc` adapts a
  function to that interface.
- `Client` is created with `NewClient`. `Call` decodes into a supplied pointer,
  `Notify` sends a notification, and `Batch` sends one or more `BatchCall`
  values.
- `Call[T]` is the typed result helper.
- `BatchCall` holds `Method`, `Params`, `Result`, and `Notification`. After a
  valid response, its `Error` holds any per-call RPC failure.
- `IDGenerator` supplies request IDs. `AtomicIDGenerator` creates monotonic
  numeric IDs; `WithIDGenerator` installs a custom strategy.
- Client validation sentinels are `ErrTransport`, `ErrInvalidResponse`,
  `ErrMismatchedID`, `ErrUnexpectedResponse`, `ErrMissingResponse`,
  `ErrDuplicateResponse`, and `ErrEmptyBatch`.

## HTTP

- `NewHTTPHandler` adapts a dispatcher to `http.Handler`.
  `WithMaxRequestBytes` changes its four-megabyte default request limit.
- `IsJSONContentType` recognizes `application/json`,
  `application/json-rpc`, and `application/*+json`, including parameters.
- `NewHTTPTransport` validates an HTTP(S) endpoint and returns a client
  transport. `WithHTTPClient`, `WithHTTPHeader`, and `WithMaxResponseBytes`
  configure it.
- `HTTPStatusError` exposes `StatusCode` and the bounded response `Body`.
  `errors.Is` recognizes `ErrHTTPStatus`.
- Other transport sentinels are `ErrHTTPContentType` and
  `ErrResponseTooLarge`.

## Compatibility notes

Wire behavior, standard error codes, ID semantics, middleware ordering, and
exported error identities are compatibility-sensitive. Before `v1.0.0`, minor
versions may refine exported Go APIs but must not introduce undocumented
protocol divergence.
