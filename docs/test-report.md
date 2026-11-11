# Real-Git acceptance report

Verified on 2026-11-11 with Go 1.27.1 and Git. Tests initialize temporary repositories with a `v1.0.0` anchor, create real two-parent merges, and evaluate captured Git history at every checkpoint. Git dates are deliberately pinned to the same timestamp, so version ordering depends on topology rather than time.

## Verification commands

```text
go test -race ./...                                                    PASS
go vet ./...                                                           PASS
go test ./internal/engine -run 'TestAcceptanceScenario|TestRequestedBranchWorkflow|TestCLIRealRepository' -count=1  PASS
go run ./cmd/vergener validate --config Vergener.yml                   PASS
go run ./cmd/vergener validate --config examples/major-releases.yml   PASS
go run ./cmd/vergener validate --config examples/features-only.yml    PASS
go run ./cmd/vergener validate --config examples/trunk-integration.yml PASS
```

`TestAcceptanceScenario` covers the initial stable version (`1.0.0`), work-branch direct commit counters, the first qualifying merge into develop (`1.0.1-beta.1`), additional merges including `fix/*`, release to main (`1.0.1`), and the next develop cycle (`1.0.2-beta.0`). `TestRequestedBranchWorkflow` checks `feat/*`, `fix/*`, `feature/*`, `features/*`, and `featues/*`, deleted work-branch refs, and the release reset. `TestCLIRealRepository` exercises the full acceptance sequence through the built executable, including JSON and `explain` output. Other real-Git tests cover two releases, pending work after a release source-tip, no-op merges, dirty worktrees, unknown identities, label collisions, shallow histories, octopus merges, overflow, and concurrent replay.

Limitations: matching-tag sources, multiple concurrent release-history/integration roles, merged-range inspection, and historical detection of squashes/rebases are not implemented. See the configuration reference for supported policies.
