# Versioning and release guide

Releases are immutable semantic-version tags created from a clean, reviewed
`main` commit. The GitHub tag workflow reruns tests, race detection, coverage,
and vet before creating release notes.

## Choose a semantic version

- Patch: backward-compatible fixes and documentation corrections.
- Minor: backward-compatible features, new optional APIs, or a pre-v1 breaking
  API refinement.
- Major: post-v1 breaking exported API or documented behavior changes.

Use a prerelease such as `v1.0.0-rc.1` when external adopters need to validate a
candidate. Never move or replace a published tag.

## Release checklist

1. Confirm `CHANGELOG.md` moves relevant Unreleased entries under the version
   and date.
2. Confirm the compatibility impact and migration notes for every observable
   protocol or API change.
3. Run:

   ```sh
   test -z "$(gofmt -l .)"
   go vet ./...
   staticcheck ./...
   go test -race ./...
   scripts/check-coverage.sh
   go test -run='^$' -bench=. -benchmem ./...
   go test -fuzz=FuzzDispatcher -fuzztime=30s .
   go test -fuzz=FuzzRequestUnmarshal -fuzztime=30s .
   govulncheck ./...
   ```

4. Verify all required GitHub Actions checks are green on the release commit.
5. Create an annotated tag: `git tag -a vX.Y.Z -m "vX.Y.Z"`.
6. Push only that tag: `git push origin vX.Y.Z`.
7. Confirm the release workflow created the GitHub release and generated notes.
8. In a clean temporary module, run `go get
   github.com/shipit-dev/go-jsonrpc@vX.Y.Z` and compile a minimal client.

## Failure handling

If verification fails, fix forward with a normal commit and restart the
checklist. Do not bypass hooks, force-update the tag, or edit a published tag.
If a broken version was published, document it and release the next patch.

## Reproducibility

This repository publishes a Go library, so no binary artifact is required. The
tag identifies the source consumed by the Go module proxy. Workflows pin tool
versions where they install tools and use the module's declared Go version.

