# Performance

## What To Measure

Benchmark protocol decode, validation, dispatch, response encode, batch
handling, and client correlation separately. Use realistic method payloads and
report Go version, hardware, request shape, batch size, allocations, and
benchmark duration.

## Required Checks

Run:

```sh
GOWORK=off go test ./... -run '^$' -bench . -benchtime=100ms -benchmem
GOWORK=off go test -race ./...
```

Select these explicit benchmark and full race routes for the affected
performance or concurrency risk. `make check-full` retains the configured
100 ms benchmark campaign; ordinary local and PR checks do not run that full
campaign.

Performance changes must preserve protocol semantics and error behavior.
Allocation reductions are not accepted when they weaken validation, duplicate
ID detection, cancellation, or response correlation.

## Capacity Planning

The package does not define server concurrency or admission control. Measure
the complete transport and handler stack under the expected method mix,
payload distribution, and deadline policy.
