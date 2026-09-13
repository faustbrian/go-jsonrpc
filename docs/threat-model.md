# Threat Model

## Scope and assets

This model covers the public JSON-RPC protocol types, dispatcher, client,
`HTTPHandler`, and `HTTPTransport`. Protected assets are service availability,
request and response confidentiality, authorization decisions made by the
application, credentials attached to HTTP requests, and the integrity of RPC
correlation and error handling.

## Trust boundaries

1. **Remote caller to server adapter.** HTTP metadata and JSON bytes cross into
   `HTTPHandler` and `Dispatcher`; all are attacker-controlled.
2. **Dispatcher to application callbacks.** Handlers, middleware, error
   mappers, and hooks are caller-owned trusted code, but their returned values,
   blocking behavior, panics, and error data can affect availability or
   confidentiality.
3. **Client to transport.** The client supplies complete request bytes to a
   package or caller-owned transport and accepts untrusted reply bytes.
4. **HTTP transport to network.** Endpoint selection, DNS, proxies, redirects,
   TLS, and dialing determine which systems can receive payloads and headers.
5. **Diagnostics to observability.** Errors may enter logs, traces, metrics, or
   user-facing messages where peer-controlled text and secrets must not cross.

## Threats and controls

| Threat | Package control | Required application control |
| --- | --- | --- |
| Oversized or deeply nested requests | Four-MiB dispatcher default, nesting limit, 1,024-member batch pre-count, and HTTP body limit | Set lower deployment-specific limits, rate limits, and concurrency admission |
| Oversized server results or error data | Four-MiB encoded-response default, per-item containment, and incremental bounded batch assembly | Keep handler return values bounded and avoid secrets in public error data |
| Oversized or high-cardinality client batches and replies | 1,024-call preflight, four-MiB reply default, and 1,024-member response pre-count before response-slice materialization | Custom transports must bound acquisition before returning bytes |
| Log injection or secret disclosure from HTTP failures | `HTTPStatusError.Error` omits the body; `Body` is empty unless a control-sanitized preview of at most four KiB is explicitly enabled; endpoint and request-failure errors do not echo URLs | Treat an opted-in `Body` as untrusted and avoid logging it; put credentials in protected headers, not URLs |
| Credential forwarding across redirects | Default client refuses redirects | A supplied `http.Client` must enforce an origin-aware redirect policy |
| Indefinite HTTP waits | Default client has a 30-second whole-request timeout and accepts caller contexts | Use shorter context deadlines where the operation requires them; configure supplied clients explicitly |
| SSRF, DNS rebinding, or proxy-based egress | Endpoint is fixed at construction, limited to HTTP(S), rejects URL user information, never follows redirects, and ignores process proxy variables by default | Never derive endpoints directly from untrusted input; allowlist origins and validate resolved addresses in a custom dialer; control proxy settings |
| Unauthorized method execution | Protocol validation occurs before dispatch | Authenticate, authorize each method, and validate business inputs |
| Callback blocking or excessive callback allocation | Context is propagated; panics are contained | Handlers and other callbacks must honor cancellation and enforce their own CPU, memory, I/O, and concurrency budgets |

## Accepted residual risks and non-goals

| Risk | Owner | Rationale | Mitigation | Review condition |
| --- | --- | --- | --- | --- |
| Authentication, authorization, TLS termination, rate limits, firewall policy, and application-secret classification remain outside the protocol package. | Application integrator | These decisions depend on deployment identity, policy, and data semantics that the library cannot infer. | Authenticate and authorize before business execution, terminate TLS, apply admission controls, and keep secrets out of public error data. | Revisit when the package owns a deployment adapter or identity contract. |
| Private and link-local destinations and ordinary DNS resolution remain allowed. | Application integrator | Internal JSON-RPC services are a supported use case, so a universal destination block would reject valid deployments. | Never derive endpoints from untrusted input; use origin allowlists and a validating custom dialer where egress matters. | Revisit if a public-network-only transport profile is introduced. |
| Trusted callbacks can ignore context cancellation or perform unbounded work. | Application integrator | Go cannot safely preempt arbitrary callback code, and forcing callbacks into detached goroutines would leak work. | Handlers and hooks receive the request context; bound their concurrency, I/O, CPU, and memory at the application boundary. | Revisit if callback contracts gain mandatory cooperative budget interfaces. |
| Go's JSON encoder can allocate while serializing a handler-owned value before the encoded response limit can reject it. | Application integrator | The standard encoder does not expose a hard allocation budget for arbitrary object graphs or custom JSON encoders. | Bound returned value complexity and custom marshaling work; the dispatcher limits retained and adapter-written bytes. | Revisit when the standard library exposes bounded encoding or the public result contract changes. |
| Aggregate batch-response overflow hides individual outcomes after handlers may have committed side effects. | Application integrator | The dispatcher cannot return the full correlated response without exceeding its configured hard limit. | Size the response limit for expected batches and make retryable methods idempotent; treat the null-ID overflow error as ambiguous completion. | Revisit if streaming responses or another bounded per-member failure contract is introduced. |
| An explicitly enabled HTTP diagnostic preview can contain upstream secrets despite control-character sanitization. | Caller enabling the preview | The package cannot identify secrets in arbitrary peer text, while bounded diagnostics can aid controlled troubleshooting. | Leave previews disabled by default; restrict access and never send preview text to ordinary logs, traces, or user-facing errors. | Revisit when a structured caller-supplied redaction policy is available. |
| Supplying `WithHTTPClient` or a custom `Transport` transfers network acquisition, proxy, redirect, timeout, TLS, DNS, and cancellation enforcement. | Caller supplying the component | Custom components are explicit extension points whose acquisition behavior is outside the client byte-slice boundary. | Configure finite deadlines and response acquisition bounds; audit redirect and proxy behavior before installation. | Revisit whenever the extension interfaces or transport ownership model changes. |
| Released v1 consumers do not receive the pending v2 hardening in this source tree. | Maintainers and application integrators | Changing default diagnostics, proxy policy, request admission, and overflow responses is intentionally isolated behind the v2 module boundary. | Keep consumers on released v1 until v2 is published; apply deployment-level limits, redaction, egress, and deadline controls in the interim, then migrate each owned consumer deliberately. | Revisit when v2 is published and every owned consumer has recorded migration evidence. |
