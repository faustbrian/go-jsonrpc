# Security

## Trust Boundary

Treat request bodies, IDs, method names, parameters, batch members, remote
responses, HTTP status bodies, and endpoint configuration as untrusted.

The dispatcher defaults to four-MiB request and encoded-response limits and a
1,024-member request-batch limit. The client independently defaults to a
four-MiB reply limit and a 1,024-member call and response-batch limit. Tune
these with `WithMaxDispatchBytes`, `WithMaxDispatchResponseBytes`,
`WithMaxBatchItems`, `WithMaxClientResponseBytes`, and
`WithMaxClientBatchItems`.

`HTTPStatusError.Error` omits peer-controlled response text, and `Body` is empty
by default. `WithHTTPDiagnosticPreviewBytes` explicitly opts into a
control-sanitized preview capped at four KiB. That preview remains untrusted;
do not log it where it could expose upstream secrets.

## Application Responsibilities

- authenticate and authorize methods before business execution;
- apply request, batch, concurrency, and rate limits;
- set handler deadlines and context deadlines shorter than transport defaults
  where required;
- avoid returning internal error details in JSON-RPC error data;
- define safe retry behavior for each client method.

Notifications require observability because they cannot return protocol
errors. The default HTTP client has a 30-second whole-request timeout, does not
follow redirects, and ignores process proxy variables. `WithHTTPClient`
explicitly transfers timeout, redirect, proxy, DNS, and dial policy to the
caller. HTTP request-failure strings omit endpoint and transport diagnostics;
callers can use `errors.Is` or `errors.As` when programmatic cause inspection is
necessary.

`NewHTTPTransport` rejects URL user information but intentionally does not
block private, loopback, or link-local destinations because internal RPC
services are valid deployments. Never construct an endpoint directly from
untrusted input. Apply an origin allowlist and, when network egress matters, a
custom dialer that validates resolved addresses. See the [threat
model](threat-model.md).

## Reporting

Follow [SECURITY.md](../SECURITY.md) for private vulnerability reporting. The
[conformance contract](conformance.md) records the maintained protocol and
defensive-input behavior.
