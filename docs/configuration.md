# Configuration reference (schema 1)

`Vergener.yml` is strict YAML. Unrecognized fields are errors. Lists replace rather than append. An `extends` array loads local files left to right, merges mappings recursively, then applies the current file, then the explicitly selected `profiles.<name>` mapping. Cyclic extends and unknown profiles fail. `config --effective` prints the resulting typed configuration; `config --schema` prints JSON Schema (structural shape; the Go validator handles semantic and graph checks). A profile is never selected implicitly. The hash is SHA-256 of normalized effective YAML, not the raw file bytes.

## Top-level

| Field | Meaning |
|---|---|
| `schema-version` | Must be `1`. |
| `version.initial` | Stable `Major.Minor.Patch` at the initial anchor. |
| `version.anchor` | Required Git ref or SHA resolving to a commit on the stable branch's first-parent chain. |
| `version.develop-anchor` | Optional explicit commit on the integration branch's first-parent chain, used when the initial boundary cannot be derived (field name retained for schema-1 compatibility). |
| `defaults.increment` | Concrete `Major`, `Minor`, `Patch`, or `None`; final inheritance fallback. |
| `defaults.unknown-commit` | `ignore` or `error` when a counter commit matches no rule. |
| `defaults.unknown-merge` | `ignore` or `error` when merge source identity cannot be resolved. |
| `git.traversal`, `git.source-identity` | `first-parent` and `merge-message`, respectively. |
| `git.merge-message-formats.<name>` | Go RE2 regex with named `source` group and optional `target`; built-in Git/GitLab parsers also apply. Conflicting matches fail. |
| `git.history.*` | `shallow`, `missing-parent`, `octopus`, `squash`, `rebase`: only `error` supported. Shallow/missing parent/octopus can be detected; historical squash/rebase generally cannot. |
| `tags.discovery`, `tags.create` | Use `anchor-only`, `false`. `matching` currently validates but matching-tag evaluation is incomplete; do not select it. No per-release tags are required. |
| `branches.<role>` | Role policies below. Exactly one role with `core.source: releases` and one with `origin.kind: release-boundary` define the stable/integration workflow; both must match exactly one local branch. |
| `rule-sets.<name>` | Reusable list of rules; referenced via `use-rule-sets`. An unused set is rejected. |
| `rules.evaluation`, `rules.scope-order` | `first-match`, `[branch, shared]`. |
| `rules.shared` | Fallback branch rules. Counter actions belong to counter rules, not shared rules. |
| `rules.no-match` | Both `commit` and `recognized-merge` must be `ignore`. |
| `output.default-format`, `output.metadata` | `json` or `semver`; metadata must be `[]`. |
| `validation.*` | `unknown-keys`, `ambiguous-branch-match`, `ambiguous-rule-priority`, `inheritance-cycle`, `label-collision` must each be `error`. |

## Branch and counter fields

`match` must contain exactly one of `exact` (literal), `glob` (Go filepath glob) or `regex` (Go RE2). `priority` is an integer; higher wins; equal-priority simultaneous matches fail. `increment` accepts `Major`, `Minor`, `Patch`, `None`, `Inherit`. `inherits` names another role and supplies **only its resolved increment**; it does not copy a matcher, core graph, rules, or counter. An absent increment follows `inherits` and finally `defaults.increment`. Inheritance cycles and missing roles fail.

`core.source` supports `releases` (current stable replay), `next-release` (apply `core.increment` to `core.of` role), `branch-core` (reuse `core.of` role), and `fixed` (parse `core.value`). `core.increment: Inherit` resolves against the referenced release role, not the current branch. Dependencies must exist and be acyclic. This implementation processes branch event rules only on the stable role; `bump-core` bumps that role's core without recording a release boundary, whereas `release` records one.

`prerelease.enabled` selects a prerelease. `prerelease.label` can include named regex matcher captures such as `{kind}-{name}`. It is lowercased and non-ASCII/non-alphanumeric sequences normalize to hyphens; empty labels, unresolved placeholders, and collisions among captured branch labels fail. `prerelease.counter.initial` is a nonnegative integer. `origin.kind` supports `release-boundary` (`branch` naming the stable role, `projection: released-source-tip`, on the integration role), `fork` (`branch` naming the integration role, `algorithm: common-first-parent`), and `anchor` (`ref` equal to `version.anchor`). Fork counting traverses the work branch's first-parent chain and ignores integration synchronization merges; merges from work branches into work branches are unsupported. `reset` for a release-boundary counter requires `on: release`, `branch` naming the stable role, `requires-source-role` naming the integration role, `value` equal to initial and `preserve-pending-events: true`; other resets are rejected. The released source-tip, not the destination release commit, is the new boundary.

`branches.<role>.rules` match stable release events. `branches.<role>.prerelease.counter.rules` match counter events. Both can include `use-rule-sets` names appropriate for their scope. A counter-scoped `ignore` does not suppress later stable releases.

## Rule syntax

Each rule has unique `id`, integer `priority` (higher first), `when`, and nonempty `actions`. A condition accepts `event: commit|merge`; `source`/`target` with `roles: [...]` and optional RE2 `pattern` against the branch name; regex `message` and `author`; `paths: [...]` of filepath glob patterns (any matching changed path; compare merge with first parent); `trailers: {Key: 'value-regex'}`; and recursively `all`, `any`, `not`. Conditions combine by AND unless a boolean operator is used. `message` inspects the event commit (for a merge, the merge's own message), not the merged commit range. Paths are read only when a configured rule needs them; the Git adapter loads them before the pure engine runs.

Within each scope, rules are ordered by descending priority; first matching rule wins. Two matching rules at the same highest priority fail. Branch scope precedes shared scope. A rule may contain explicit actions: `release: {increment: Patch|Minor|Major|None|Inherit, boundary: source-tip}`, `increment-counter: {by: positive-integer}`, `set-counter: {value: nonnegative-integer}`, `ignore: {}`, and `fail: {message: text}`. `release` is for branch rules; counter mutations are for counter rules. `Inherit` on a release uses the destination role's resolved increment. `bump-core: {increment: ...}` changes the stable role's core on a matching event without creating a release event; `Inherit` uses that role's increment. The current `stableVersion` field reflects the resulting stable-branch core, even for a standalone bump; `releases` remains unchanged. `None` never changes a counter. No hidden Conventional Commit or CI-skip semantics exist.

## Consequential assumptions

A merge's first parent is the destination tip and second parent is the included source tip. A source message must match a configured role; the default `Vergener.yml` counts `feat/*`, `feature/*`, `features/*`, `featues/*`, and `fix/*` merges into develop (and retains `ci/*`). To exclude a source such as `docs/*`, define a role for it with no matching counter rule. Stable/integration refs are both captured before replay and checked again after reading Git history; changed refs cause a retryable error. The label collision check considers refs available in the local snapshot. Trailers are parsed from `Key: value` lines in the event message. Deleted branch refs remain identifiable only if the merge message names their source.
