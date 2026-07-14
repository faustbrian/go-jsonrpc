# Architecture

The package separates the protocol from every delivery mechanism. The normal
server flow is:

```text
Transport -> Protocol parsing -> Dispatch -> Middleware -> Execution
          <- Response shaping  <-          <-            <-
```

## Transport

`Dispatcher.Dispatch` accepts a context and JSON bytes. It does not know about
HTTP status codes, headers, connections, or routers. `HTTPHandler` is a small
adapter that enforces POST, JSON media types, request-size limits, and `204 No
Content` for requests that contain only notifications.

On the client, `Transport` has one `RoundTrip` method. `HTTPTransport` supplies
the standard HTTP implementation; callers can replace it without changing
request construction or response validation.

## Protocol

`Request`, `Response`, `ID`, and `Error` own wire semantics. Custom JSON
decoding preserves whether an ID was omitted and ensures that ID strings,
numbers, and explicit `null` are not conflated. Envelope validation happens
before dispatch or result decoding.

Malformed JSON becomes a parse error. Valid JSON with an invalid JSON-RPC
shape becomes an invalid-request error. Invalid batch members are handled
independently, while an empty batch produces a single invalid-request object.

## Dispatch

`Registry` is concurrency-safe and rejects duplicates and the
specification-reserved `rpc.` prefix. A dispatcher looks up a method only after
the request envelope validates. It echoes an ID only for a valid request;
invalid requests use `null`.

Batches execute sequentially in input order. This keeps handler ordering
deterministic and avoids silently imposing concurrency requirements on
application code. The protocol permits batch responses in any order, so this
is an implementation choice rather than a wire guarantee.

## Execution

Handlers receive `context.Context` and `json.RawMessage`. `DecodeParams[T]` is
an optional strict decoder; handlers may perform their own decoding. Handler
panics and ordinary Go errors are contained as internal errors. An `*Error`
returned by a handler is sent as the protocol error. `WithErrorMapper` provides
one auditable place to map application errors without exposing internal text.

Middleware wraps handlers in registration-independent chains. The current
`Request` is placed in context before middleware and execution, so logging,
tracing, metrics, correlation, and authorization can inspect the method and ID.
Dispatcher `Hooks` surround the wider protocol lifecycle, including failures
that occur before a handler or middleware exists. Hook panics are contained so
observability cannot corrupt a protocol response.

## Client validation

The client owns ID generation and validates version, result/error exclusivity,
ID equality, batch completeness, duplicate responses, unexpected responses,
and typed result decoding. Batch response order is irrelevant. Transport errors
wrap `ErrTransport`; valid JSON-RPC error objects remain `*Error` values.

## Performance model

Envelopes retain raw params and results until an application type is actually
needed. Handlers are stored as compiled functions rather than reflected method
descriptors. Batch storage is proportional to the number of members. Benchmarks
track single and mixed-batch dispatch allocations in CI artifacts.
