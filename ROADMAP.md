# Roadmap

The roadmap is ordered by protocol confidence, not feature count. New adapters
must reuse the same envelopes, validation, dispatch, and error semantics.

## v1.0.0

- Complete external API review and release-candidate adoption.
- Preserve 100% meaningful production-package coverage and all conformance
  fixtures.
- Publish the compatibility and security policies with the initial stable tag.
- Confirm installation through the public Go module proxy.

The stable release will not intentionally diverge from JSON-RPC 2.0. Any
compliance defect found during release-candidate use blocks v1.

## After core stabilization

- WebSocket transport helpers with explicit connection and correlation rules.
- Stream-friendly batch helpers that retain complete-message semantics.
- OpenRPC schema generation from opt-in method metadata.
- Optional idempotency-aware retry building blocks without business defaults.
- Router adapters that remain thin wrappers over `HTTPHandler`.
- More transport and interoperability fixtures from external implementations.

## Explicitly not planned

- Service discovery, load balancing, or service-mesh behavior.
- Application-specific authentication schemes.
- Queue orchestration or background job frameworks.
- Automatic retries that assume method idempotency.
- SOAP, XML-RPC, GraphQL, or JSON:API support in this module.
