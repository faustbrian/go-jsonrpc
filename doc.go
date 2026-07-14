// Package jsonrpc implements transport-neutral JSON-RPC 2.0 clients and
// servers. It provides explicit protocol envelopes, request dispatch,
// middleware, structured errors, strict client response validation, and thin
// net/http adapters without imposing a router, logger, tracer, or validator.
//
// Requests and responses can be processed directly as bytes through Dispatcher
// and Transport, allowing custom transports to share the same protocol rules.
// See the repository documentation for compatibility guarantees and complete
// server, client, notification, and batch examples.
package jsonrpc
