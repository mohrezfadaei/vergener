# VerGener

VerGener calculates semantic versions from immutable Git commits, two-parent merge events and an explicit initial anchor. Counters are replayed from first-parent history; there is no mutable version database. This implementation targets the documented main/develop/work workflow and rejects unsupported policy values rather than silently emulating GitVersion.

## Install

Requires Go 1.27+ and Git. Run `./scripts/install.sh` from this checkout to build and install into `~/.local/bin`, or set `INSTALL_DIR=/path/to/bin` to choose another location. Make sure the install directory is on your `PATH`. Alternatively, run `go install ./cmd/vergener` or `go build -o vergener ./cmd/vergener`. Copy `Vergener.yml` to the root of the repository being versioned; create the initial `v1.0.0` tag once at its `1.0.0` anchor. Stable releases after that need no tags. Work with full, non-shallow history.

## Docker image for GitLab CI

Build with `just docker-build` (local tag `vergener:local`) or `just docker-build registry.example/namespace/vergener:0.0.1` for a registry tag. Other recipes include `just check`, `just validate`, and `just build`. The multi-stage image contains `vergener`, Git, CA certificates, and `/bin/sh`; it has no entrypoint so GitLab Runner can start its job shell. The image does not contain `Vergener.yml` or repository history: the CI checkout supplies both.

In a GitLab branch pipeline, use `GIT_DEPTH: "0"` and make sure the checkout has the initial tag plus local `refs/heads/main`, `refs/heads/develop`, and the branch being evaluated. GitLab commonly checks out a detached commit; pass `--branch "$CI_COMMIT_REF_NAME"` explicitly. For example, before running `vergener version`, fetch the required refs and point the local target branch at the pipeline commit:

```sh
git fetch origin '+refs/heads/main:refs/heads/main' '+refs/heads/develop:refs/heads/develop' '+refs/tags/v1.0.0:refs/tags/v1.0.0'
git update-ref "refs/heads/$CI_COMMIT_REF_NAME" "$CI_COMMIT_SHA"
vergener version --branch "$CI_COMMIT_REF_NAME" --format semver
```

This example is for branch pipelines with the default detached checkout; merge-request pipelines need an explicit source-branch ref strategy. The job's `Vergener.yml` must be present at its repository root.

## Commands

```sh
vergener validate --config Vergener.yml
vergener version --branch develop --format semver
vergener version --branch develop                # JSON
vergener explain --branch develop
vergener config --effective --profile minor
vergener config --schema
```

The config path defaults to the Git repository root, independent of the current directory. `--profile` selects an explicitly defined profile. `version`, `explain`, and `validate` never write Git objects. Nonzero exit status indicates a configuration, Git-history, or evaluation error. A detached HEAD requires `--branch`.

## Versioning model

`main` starts at the version of the tagged anchor. Every matching develop→main merge after that anchor registers a stable release and increments the **core** according to the release action; `None` registers a release boundary without bumping the core. `develop` projects the *next* stable core independently of its prerelease counter. Its beta counter increments once for each matching work→develop merge commit, not for each commit merged. Direct commits and excluded source roles do not increment beta. On release, the boundary moves to the released develop source-tip, preserving qualifying merges after that commit. Merging main back into develop does not increment beta. Work branches count non-merge commits along their own first-parent chain after the latest first-parent commit shared with develop; synchronization merges don't reset that boundary. No-op merges create no merge commit and therefore no event. Git commit timestamps, CI skip messages, and uncommitted changes do not influence versions.

A branch core may depend on the current develop/main refs. Repeatable results require an identical *complete* ref snapshot and effective configuration, not merely the same feature SHA. The JSON reports captured refs, config hash, release evidence, counted SHAs, cycle base and dirty status. The `dirty` flag is diagnostic only.

## Supported policy and limitations

See [configuration reference](docs/configuration.md) and [real-Git test report](docs/test-report.md). The machine-readable JSON Schema is checked in at [`docs/schema.json`](docs/schema.json) and printed by `config --schema`. Alternate examples live under `examples/`. This release supports concrete core sources (`releases`, `next-release`, `branch-core`, `fixed`), fork/release/anchor counter origins, first-match rules, named rule sets, profiles and extends. It does **not** yet support matching-tag core/counter evaluation, merged-commit-range message rules, multiple concurrent release-history/integration roles, or generalized counter reset algorithms. Matching-tag settings currently pass configuration validation but fail during evaluation; do not use them. Other unsupported policy values are rejected. Stable and integration roles are inferred from the single `core.source: releases` role and the single `origin.kind: release-boundary` role; their Git branch names are matched by configuration rather than hardcoded. See `examples/trunk-integration.yml` for different roles, branch names, and a Minor release policy. Unsupported work→work merges fail explicitly.

Identity is extracted from merge messages; a deleted source ref does not affect replay, but a missing or misleading identity cannot be reconstructed reliably from topology. Matching identity does not itself increment anything. Git cannot generally detect a historical squash or rebase when only the resulting graph remains. The configured `squash: error` and `rebase: error` settings mean the tool does not infer them; they are **not** a guarantee of detection. Unknown merge identity is an error by default. For an anchor outside develop's first-parent history, set `version.develop-anchor` unless the anchor itself is a recognized develop→main release merge. An initial tag must identify the initial version truthfully; Git cannot independently verify the historical SemVer meaning of a tag.

## Development

```sh
gofmt -w cmd internal
go build ./cmd/vergener
go test ./...
go test -race ./...
go vet ./...
```

Acceptance tests create real temporary Git repositories, enforce two-parent merges, compare the full sequence, and check pending changes after release. See the report for observed outputs. No persistent state is stored.
