# Specification provenance

`manifest.tsv` pins the local machine-readable transcription of every
JSON-RPC 2.0 example exercised by `TestSpecificationExamples`. The digest and
byte count apply to
`testdata/conformance/jsonrpc-2.0-specification.json` and detect accidental
fixture drift.

The [specification decision register](../docs/specification-decisions.md)
records ambiguities and transport policy that the official examples do not
fully determine. `TestSpecificationReferences`, the protocol matrix, peer
comparisons, hostile-input tests, and fuzz targets provide the remaining
conformance evidence.

The maintained differential harness pins `creachadair/jrpc2` v1.3.5 in the
non-releasable `interoperability` module. It exercises explicit null IDs,
numeric ID equivalence, notification-only HTTP responses, method and media
policies, and JSON-RPC error status mapping. Run it reproducibly with:

```bash
GOWORK=auto go -C interoperability test ./...
```

`interoperability/expected.tsv` records the peer version, fixture cases,
semantic and wire outcomes, and disagreement classification. The summary
bindings below remain module-owned specification evidence.

## Decision conformance matrix

| Decision | Normative sources | Executable evidence | Differential evidence |
| --- | --- | --- | --- |
| JSONRPC-DEC-001 | JSON-RPC 2.0 Request, Notification, and Batch | `TestDispatcherProtocolErrors`, `TestDispatcherBatch`, `TestDispatcherBatchEdgeCases`, `TestSpecificationExamples` | `specification/interoperability.tsv` |
| JSONRPC-DEC-002 | JSON-RPC 2.0 Request and Notification | `TestRequestDistinguishesNotificationFromNullID`, `TestIDRoundTripAndEquality` | `specification/interoperability.tsv` |
| JSONRPC-DEC-003 | JSON-RPC 2.0 Request and RFC 8259 Section 6 | `TestIDRoundTripAndEquality`, `TestIDNumberCanonicalizationIsBounded`, `TestIDCanonicalizationAllocationsDoNotScaleWithExponentDigits`, `TestClientMatchesEquivalentNumericID` | `specification/interoperability.tsv` |
| JSONRPC-DEC-004 | JSON-RPC 2.0 Batch | `TestDispatcherBatchEdgeCases`, `TestSpecificationExamples` | `specification/interoperability.tsv` |
| JSONRPC-DEC-005 | JSON-RPC 2.0 Batch and Examples | `TestDispatcherBatch`, `TestSpecificationExamples` | `specification/interoperability.tsv` |
| JSONRPC-DEC-006 | JSON-RPC 2.0 Batch | `TestDispatcherBatch`, `TestClientBatch`, `TestClientBatchValidation` | `specification/interoperability.tsv` |
| JSONRPC-DEC-007 | JSON-RPC 2.0 Error, Request, and Batch | `TestDispatcherProtocolErrors`, `TestProtocolRejectsInvalidUTF8`, `TestDispatcherClassifiesNestedParameterDuplicatesAsInvalidParams`, `FuzzDispatcher` | `specification/interoperability.tsv` |
| JSONRPC-DEC-008 | RFC 8259 Section 4 and JSON-RPC 2.0 objects | `TestProtocolDecodersRejectDuplicateMembers`, `TestDispatcherClassifiesNestedParameterDuplicatesAsInvalidParams` | `specification/interoperability.tsv` |
| JSONRPC-DEC-009 | JSON-RPC 2.0 Notification and RFC 9110 HTTP 204 | `TestHTTPHandlerRequestAndNotification`, `TestHTTPTransportNoContent`, `TestDispatcherBatchEdgeCases` | `specification/interoperability.tsv` |
| JSONRPC-DEC-010 | JSON-RPC 2.0, RFC 9110, and RFC 6839 | `TestHTTPHandlerTransportErrors`, `TestJSONContentTypes`, `TestHTTPHandlerRequestAndNotification`, `TestHTTPTransportRoundTrip` | `specification/interoperability.tsv` |

When the specification or an accepted erratum changes an example, retain the
old fixture in history, update the transcription from the authoritative page,
validate it with `jq -e .`, refresh the digest and byte count with
`shasum -a 256` and `wc -c`, and rerun the conformance and specification gates.
