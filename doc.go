// Package jsonrpc implements transport-neutral JSON-RPC 2.0 clients and
// servers. It provides explicit protocol envelopes, request dispatch,
// middleware, structured errors, strict client response validation, and thin
// net/http adapters without imposing a router, logger, tracer, or validator.
//
// Requests and responses can be processed directly as bytes through Dispatcher
// and Transport, allowing custom transports to share the same protocol rules.
// The protocol is defined by the JSON-RPC 2.0 specification:
// https://www.jsonrpc.org/specification.
//
// Construct dispatchers, clients and HTTP adapters with their New functions;
// Registry and AtomicIDGenerator also have usable zero values. Dispatch is
// synchronous and propagates context but cannot preempt callbacks or infer retry
// safety. Concurrent use requires caller-owned callbacks, transports and result
// targets to satisfy their own synchronization and lifetime contracts.
//
// Standalone envelope decoding and validation are separate steps. Dispatcher
// and Client provide byte/item bounds; direct decoders need caller-owned input
// bounds. Raw messages and copied IDs can share mutable storage. Public error
// messages/data cross the wire; retained local causes are not a redaction service.
// Specification decisions and pinned example provenance live under specification/
// and docs/specification-decisions.md; no HTTP binding is implied by JSON-RPC 2.0.
//
// See the repository documentation for compatibility guarantees and complete
// server, client, notification, and batch examples.
package jsonrpc
