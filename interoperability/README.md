# JSON-RPC interoperability harness

This non-releasable module compares the repository's documented JSON-RPC
decisions with pinned `creachadair/jrpc2` v1.3.5 behavior. It keeps the peer
dependency outside the public module and records deliberate differences rather
than changing the package contract to match another implementation.

Run the harness from the repository root:

```sh
make interoperability
```

Use the public [`jsonrpc` module](../README.md) in applications. Shared
construction, ownership, lifecycle, and composition expectations are in the
versioned [Golib ecosystem index](https://github.com/faustbrian/go-library-tools/blob/v1.3.0/docs/ecosystem/README.md).
