# Versioning and release guide

Releases are immutable semantic-version tags created from a clean, reviewed
`main` commit. The repository workflow runs the shared `golib` release contract
before publishing release notes.

## Prepare a release

1. Confirm the public API and documented protocol compatibility impact.
2. Move the relevant `CHANGELOG.md` entries into a dated release section.
3. Run `make ci` and resolve every required failure.
4. Run `golib release check` and review its release metadata result.
5. Merge the release preparation through normal review.
6. Create and push an annotated `vMAJOR.MINOR.PATCH` tag from the verified
   `main` commit.
7. Confirm the tag workflow completes and the module is available from the Go
   module proxy.

Published tags are never force-updated or reused. A broken release is fixed by
publishing a new patch version.

## Release rehearsal

Use the `release_dry_run` workflow dispatch input to exercise the shared release
checks without publishing a tag. The rehearsal must pass before a stable tag is
created.

## Clean-consumer verification

In a clean temporary module, install the released package and compile a
minimal client:

```sh
go get github.com/faustbrian/go-jsonrpc@vX.Y.Z
```

The package's public module identity, API baseline, conformance fixtures, and
documentation must all refer to the same release.
