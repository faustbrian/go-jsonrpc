# Public API reference

This is the semantic reference for the exported surface. Exact Go signatures
are also available through `go doc github.com/faustbrian/go-jsonrpc`.

## Protocol

- `Version` is the required `"2.0"` protocol marker.
- `Request` contains `JSONRPC`, `Method`, raw `Params`, and an `ID`. Its custom
  JSON implementation preserves ID presence. `Validate` checks the envelope;
  `IsNotification` is true only when the ID member is absent. Duplicate
  envelope members and case variants of reserved members are rejected.
- `Response` contains raw `Result`, structured `Error`, and `ID`. `Validate`
  requires version 2.0, an ID, and exactly one of result or error. Duplicate
  response and error-object members and case variants of their reserved names
  are rejected.
- `ID` represents a string, JSON number, explicit null, or an internal missing
  state. Construct values with `StringID`, `NumberID`, and `NullID`; inspect with
  `Kind` and compare without coercion using `Equal`. Wire decoders reject
  invalid UTF-8 rather than replacing malformed bytes. Mathematically
  equivalent numeric spellings compare equal without expanding exponents;
  normalization work and storage stay proportional to input length.
  `StringID` applies Go's JSON replacement behavior to invalid UTF-8 in a Go
  string so the local correlation value exactly matches transmitted JSON.
- `IDKind` values are `IDMissing`, `IDString`, `IDNumber`, and `IDNull`.
- `NewRequest` and `NewNotification` validate method names and require params to
  encode as an object or array when present.
- `DecodeParams[T]` strictly decodes params and maps malformed, duplicate at
  any object depth, or unknown fields to `InvalidParams`.

### Errors

`Error` is both a JSON-RPC error object and a Go error. `Code`, `Message`, and
optional raw `Data` cross the wire. `WithCause` retains a local cause without
serializing it; `WithData` JSON-encodes safe public details.

`NewError` constructs application-defined errors. Application codes should be
outside the protocol-reserved `-32768` through `-32000` range; this package uses
`-32000` for bounded-dispatch rejection. `ParseError`,
`InvalidRequest`, `MethodNotFound`, `InvalidParams`, and `InternalError` build
the standard errors. `RequestLimitExceeded` builds the implementation-defined
server error used for whole-dispatch resource rejection. Their codes are also
exported as `CodeRequestLimitExceeded`, `CodeParseError`, `CodeInvalidRequest`,
`CodeMethodNotFound`, `CodeInvalidParams`, and `CodeInternalError`.

## Server

- `Handler` is `func(context.Context, json.RawMessage) (any, error)`.
- `Registry` has a usable, concurrency-safe zero value; `NewRegistry` is the
  explicit constructor. `Register` adds one unique application method,
  `RegisterSystem` adds an explicitly reserved `rpc.*` protocol method, and
  `Lookup` supports inspection. Do not copy a registry after first use. An
  empty string is a valid application method name because the protocol
  requires only a string; the `rpc.` prefix is reserved from applications.
- `Dispatcher` is created with `NewDispatcher`. `Dispatch` processes a single
  request or batch and returns bytes plus a boolean indicating whether a reply
  exists. `DispatchSingle` processes one non-batch message and returns the
  typed `Response` before wire encoding for compatible adapter-owned response
  envelopes. Direct payloads default to four MiB and batches to 1,024 members;
  `WithMaxDispatchBytes`, `WithMaxBatchItems`, and `WithMaxNestingDepth` can
  raise or lower the byte, batch-member, and JSON nesting bounds.
- `Middleware` wraps a `Handler`. `WithMiddleware` installs middleware in the
  listed order, with the first item outermost.
- `ErrorMapper` converts ordinary application errors to safe RPC errors.
  `WithErrorMapper` replaces the default internal-error mapping.
- `Hooks` observes the complete dispatcher lifecycle. `WithHooks` installs
  callbacks that see parse errors, invalid requests, method lookup failures,
  notifications, handler results, mapped errors, and recovered panics.
- `RequestFromContext` retrieves the validated request during middleware or
  handler execution.
- Registration errors are detectable with `ErrInvalidMethodName`,
  `ErrMethodAlreadyRegistered`, and `ErrNilHandler`.

## Client

- `Transport` exchanges one complete JSON-RPC payload. `TransportFunc` adapts a
  function to that interface. Custom transports remain responsible for
  bounding acquisition before returning the reply bytes.
- `Client` is created with `NewClient`. `Call` decodes into a supplied pointer,
  `Notify` sends a notification, and `Batch` sends one or more `BatchCall`
  values. Reply parsing defaults to four MiB and can be changed with
  `WithMaxClientResponseBytes`.
- `Call[T]` is the typed result helper.
- `BatchCall` holds `Method`, `Params`, `Result`, and `Notification`. After a
  valid response, its `Error` holds any per-call RPC failure.
- `IDGenerator` supplies request IDs. `AtomicIDGenerator` creates monotonic
  numeric IDs; `WithIDGenerator` installs a custom strategy.
- Client validation sentinels are `ErrTransport`, `ErrInvalidResponse`,
  `ErrMismatchedID`, `ErrUnexpectedResponse`, `ErrMissingResponse`,
  `ErrDuplicateResponse`, `ErrDuplicateRequestID`,
  `ErrClientResponseTooLarge`, and `ErrEmptyBatch`.

## HTTP

- `NewHTTPHandler` adapts a dispatcher to `http.Handler`.
  `WithMaxRequestBytes` changes its four-megabyte default request limit.
- `IsJSONContentType` recognizes `application/json`,
  `application/json-rpc`, and `application/*+json`, including parameters.
- `NewHTTPTransport` validates an HTTP(S) endpoint and returns a client
  transport. `WithHTTPClient`, `WithHTTPHeader`, and `WithMaxResponseBytes`
  configure it. Direct `RoundTrip` calls return request-construction errors,
  including a nil context, before network I/O. The default client does not
  follow redirects, preventing configured headers from crossing origins;
  `WithHTTPClient` explicitly opts into that client's redirect policy.
- `HTTPStatusError` exposes `StatusCode` and the bounded response `Body`.
  `errors.Is` recognizes `ErrHTTPStatus`.
- Other transport sentinels are `ErrHTTPContentType` and
  `ErrResponseTooLarge`.

## Options and ownership

All option functions run synchronously, in caller-supplied order, during
construction, and nil option functions are ignored. Repeated positive limits,
non-nil ID generators, and non-nil HTTP clients use the last supplied value;
non-positive limits and nil values for those two collaborators leave the
current value unchanged. Repeated `WithHooks` and `WithErrorMapper` options
replace the previous value, including a nil error mapper; invoking a nil mapper
is panic-contained as an internal error. Repeated header names use the last
value, and repeated `WithMiddleware` options append middleware in order.
Constructors copy the middleware function values, hook set, and HTTP header
strings that they retain.

`Dispatcher` retains the supplied `Registry`. `Client` retains its `Transport`
and `IDGenerator`, and `HTTPTransport` retains a supplied `*http.Client`; these
collaborators remain caller-owned and must satisfy their own concurrency and
lifecycle contracts. The package does not close a supplied HTTP client or its
idle connections. `HTTPHandler` closes the inbound request body, and
`HTTPTransport` closes every received response body. Request payloads and
transport reply buffers are used only for the owning synchronous call.

Handlers, middleware-produced handlers, error mappers, and hooks execute
synchronously in the goroutine calling `Dispatch`; the package creates no
goroutine or callback timeout. Concurrent dispatch can therefore invoke them
concurrently. They must honor the supplied context, avoid unbounded blocking,
and provide their own synchronization. Middleware constructors are applied for
each method execution. Hook panics are contained. `OnRequest` may return a
derived context; nil falls back to the original context. Hook request and
response arguments are observation values whose mutation cannot change the
protocol result and should be copied before retention.

`Batch` writes `Result`, `Error`, and internal correlation state on each
`BatchCall`; callers must not share one `BatchCall` across concurrent calls.

## Compatibility notes

Wire behavior, standard error codes, ID semantics, middleware ordering, and
exported error identities are compatibility-sensitive. Starting with `v1.0.0`,
changes follow the stable-release guarantees in the compatibility policy.
All package constructors ignore nil functional options. The public module is
pure Go, has no build tags or cgo, and requires Go 1.26.6 or later. Repository
CI verifies exactly Go 1.26.6 on Ubuntu 24.04; other operating systems and
architectures are source-portability expectations, not published tested
platform claims.
