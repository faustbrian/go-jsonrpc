# Contributing

Contributions are welcome through issues and pull requests. By participating,
you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## Development setup

Install Go 1.24 or newer, clone the repository, then run:

```sh
go test ./...
scripts/check-coverage.sh
go test -race ./...
go vet ./...
```

Install the pinned analysis tools used by CI when needed:

```sh
go install honnef.co/go/tools/cmd/staticcheck@v0.6.1
go install golang.org/x/vuln/cmd/govulncheck@v1.6.0
staticcheck ./...
govulncheck ./...
```

Format all Go files with `gofmt`. Add or update tests before implementation for
behavior changes. Tests must prove outcomes and error semantics, not merely
execute lines to preserve the coverage percentage.

## Pull requests

Keep pull requests focused and explain:

- the user or protocol problem;
- why the chosen API and behavior solve it;
- wire and compatibility implications;
- tests, fixtures, fuzzing, or benchmarks added;
- documentation and migration impact.

All CI, fuzz-seed, security, formatting, analysis, race, documentation, and
coverage checks must pass. Update `CHANGELOG.md` for user-visible changes.

## Protocol changes

Protocol changes require exceptional care. Include the relevant JSON-RPC 2.0
rule, valid and invalid fixtures, notification and batch impact, ID behavior,
and client/server interoperability reasoning. An intentional specification
divergence will not be accepted. A compliance fix that changes observable
behavior must include a regression test and migration note.

## Public API changes

Prefer explicit types and narrow extension points. Avoid new global state,
transport assumptions in the dispatcher, hidden reflection, or dependencies on
application infrastructure. Update `docs/api.md`, examples, compatibility notes,
and benchmarks where relevant.

## Fuzzing and benchmarks

Run the existing targets locally:

```sh
go test -fuzz=FuzzDispatcher -fuzztime=30s .
go test -fuzz=FuzzRequestUnmarshal -fuzztime=30s .
go test -run='^$' -bench=. -benchmem ./...
```

Commit a minimized regression fixture for every real fuzz-discovered protocol
bug. Performance changes should report comparable before/after benchmark output
and must not weaken validation for speed.

## Reporting security issues

Do not open a public issue for a vulnerability. Follow
[`SECURITY.md`](SECURITY.md) so maintainers can coordinate a fix and disclosure.

