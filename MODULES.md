# Modules, development and releases

The repository contains 17 independent Go 1.24+ library modules. Imports retain
`github.com/tr1v3r/pkg/<name>`. The root module contains no library packages: it
exists as a migration boundary from the former single-module layout.

## Boundaries

| Module directory | Production dependencies within this repository | Test-only dependencies within this repository |
| --- | --- | --- |
| alfred | — | — |
| brute | — | — |
| cache | — | — |
| calendar | — | — |
| circuitbreaker | — | — |
| config | fetch | guard |
| fetch | — | circuitbreaker |
| guard | — | — |
| hash | — | — |
| log | — | — |
| netool | — | — |
| notion | log | — |
| pools | — | — |
| rss | — | — |
| sort | — | — |
| thread | — | — |
| websocket | log | — |

Each module owns its `go.mod` and, when needed, `go.sum`. Third-party dependency
versions are inherited from the former root manifest. No business API is changed
by the split. The root LICENSE applies to all modules; Go includes that file in
subdirectory module archives when no module-local LICENSE exists.

## Development

`go.work` selects the root and all library modules. Its version-specific local
replacements also prevent older, unpruned dependency graphs from looking up
unreleased internal versions. Published `go.mod` files contain no local replacements.

```sh
make modules
make test-fast
make test-race
make lint
make MODULE=cache test-race
make test-package PKG=./fetch
make test-cover                 # coverage.out and coverage.html inside each module
./scripts/modules.sh go vet ./...
python3 scripts/check-workspace.py
python3 scripts/test-tooling.py
```

`./scripts/modules.sh` discovers library `go.mod` files, excludes the root migration
module, and stops on the first failed command. `MODULE=<directory>` selects one
module. Root `go test ./...` does not recurse into nested modules; use the Makefile
or the script for repository-wide checks.

To add a module, create its manifest, add its path to `go.work`, and add versioned
workspace replacements for any new internal pins. CI discovers its lint/test
matrix automatically; `check-workspace.py` verifies membership and replacements.
Formatting, lint rules and Go version support remain shared. CI runs every module
on Go 1.24, 1.25, 1.26 and 1.27 (the latest patch in each series), including race
detection, coverage, and standalone checks. Module tooling regressions, the separate
cache stress job, and tagged-module release validation cover the same four versions.

## Pre-release standalone checks

Internal versions in this change target the **planned first releases** `v0.1.0`.
The root migration boundary targets `v0.3.0`. They are not published by this change.
Before these tags exist, tests cannot resolve those versions from the public proxy.

```sh
make deps              # tidy each module; retains only real external checksums
make check-modules     # compare tidy manifests, GOWORK=off
make test-standalone   # race/integration tests, GOWORK=off
```

These commands build a temporary file proxy from this checkout, use a fresh module
cache, and run in temporary source copies. Internal pinned versions resolve to the
current checkout. `make deps` copies tidy manifests and external sums back; it does
not copy synthetic internal checksums and preserves any existing real internal
checksums. Tidy checks compare go.mod and external
sums while ignoring temporary internal checksums, including those reached through
dependency tests. Other standalone checks leave the working tree untouched.
External modules still use the configured proxy and checksum DB.

This verifies module archives and dependency declarations independently of the
workspace. It does **not** prove that a version has been published, or that the
checkout is compatible with a previously released internal version. Verify actual
pins separately after dependencies have been published:

```sh
make deps-published     # tidy real pins and record their actual checksums
make test-published
MODULE=notion ./scripts/standalone.sh --published --check-tidy
```

Published mode uses the normal proxy/cache and checksum database, with no checkout
proxy or local replacement. It also runs in a temporary source copy, downloading
internal checksums there. Use published mode after dependencies are available to verify pinned-version
compatibility and maintain real internal checksums. The tag release workflow uses
this mode.

## First release and migration

Do not modify or delete historical tags. Existing consumers pinned to `v0.2.1`
continue to use the original combined module.

A consumer that selects both that old root module and a new child module can see
`ambiguous import`: both archives contain the same package. Publish a root version
containing the split boundaries (`v0.3.0`), and upgrade that root together with the
selected child modules. For example, **after the first releases exist**:

```sh
go get github.com/tr1v3r/pkg@v0.3.0 \
    github.com/tr1v3r/pkg/cache@v0.1.0 \
    github.com/tr1v3r/pkg/notion@v0.1.0
go mod tidy
```

Include every child module the application imports, not only those in the example.
Imports themselves stay unchanged. `go mod tidy` may remove the root requirement
when it is no longer needed; dependencies still requiring an older root may make
an explicit upgrade to the boundary version necessary.

First-release order (test dependencies also count):

1. Review and commit the split, then publish root `v0.3.0` from that split commit.
2. Publish dependency-free-within-this-repo modules: alfred, brute, cache, calendar,
   circuitbreaker, guard, hash, log, netool, pools, rss, sort, thread.
3. Once circuitbreaker is available, publish fetch.
4. Once their pins are available, publish config, notion, websocket.

Every child starts with a directory-prefixed tag, for example `cache/v0.1.0`.
Future versions are independent; a cache-only change does not require new log or
notion versions. Publish changed dependencies first, update dependent manifests
and workspace mappings, then run workspace and published-mode checks.

Pushing a `v*` or `<module>/v*` tag triggers `release.yml`. It validates the tag,
checks tidy manifests and race/integration tests against actual published pins,
and then creates a GitHub Release for the existing tag. Child releases do not
become the repository's Latest release. Go v2+ modules require a `/v2` module/import
path migration; the release validator intentionally accepts only v0/v1 tags.
No tags or releases are created by local checks or ordinary PR CI.

## Automatic releases after dev → master

After a PR from this repository's `dev` branch is merged into `master`, the push
CI must pass for the exact merge commit: every module, all four Go versions,
workspace/standalone checks, lint, tooling regressions and cache stress. The
release job then confirms the merged PR's source via the GitHub API. Other source
branches, direct master pushes, and PR validation runs do not publish automatically.
The repository currently has no dev branch; this workflow does not create one.

Only modules whose `.go`, `go.mod`, or `go.sum` files changed in that PR are
considered. Renames and deletions count. Documentation-only merges do not release.
Each module's latest stable directory-prefixed tag supplies the base version:
`cache/v0.1.2` becomes `cache/v0.1.3`; a module with no stable tag starts at
`v0.1.0`. Pre-release tags do not change this calculation. Minor/major versions
remain explicit maintainer decisions; automatic code changes use patch increments.

Unpublished internal dependencies are included when needed, even if unchanged in
the current PR. Modules publish in dependency order, including test dependencies.
The first automatic publication also creates the root `v0.3.0` migration boundary
if missing. Existing dependency pins are respected; if code needs a newer sibling
API, update its go.mod in the development PR before merging.

Before each module tag is created, real pinned dependencies are checked with
workspace disabled on Go 1.24, 1.25, 1.26 and 1.27. Then the job pushes the tag and
creates the GitHub Release directly using `GITHUB_TOKEN`. This avoids relying on
a token-created tag to trigger another workflow. No PAT or additional secret is
needed. A failed dependency publication or validation stops subsequent releases;
previously completed releases remain valid. Rerunning the same merge reuses an
existing tag and completes a missing Release instead of incrementing again.

Automatic and manual publication share a serialized `module-publication` queue
(`queue: max`). A stale run cannot publish an older snapshot over a newer module
version. Changes are also compared with each module's latest tag so earlier code
changes in a squash/rebase range are included. The queue property is documented
in [GitHub concurrency](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/control-workflow-concurrency);
`GITHUB_TOKEN` trigger behavior is documented in [workflow triggering](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow).

For a read-only local preview (all modules considered, without the merged-PR filter):

```sh
python3 scripts/auto-release.py
python3 scripts/test-release.py
```

The preview does not contact GitHub or mutate tags. Publication mode is only used
by the master push workflow after its source and test gates pass.
