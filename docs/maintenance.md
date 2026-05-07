# Spec Maintenance Protocol

The ratchet spec is a living document with a deliberate cadence. The premise is Anthropic's: every harness component encodes an assumption about model weakness; those assumptions go stale as models improve. The spec must reflect current assumptions, not historical ones.

## Versioning

ratchet uses Semantic Versioning of the spec itself.

- **v0.x** — pre-stable. Breaking changes are permitted with a `CHANGELOG.md` entry.
- **v1.0+** — stable. Breaking changes require a major version bump and are subject to the deprecation policy below.
- Spec version comparison is minor-version-tolerant: a gate pinned to `"0.1"` runs on any `0.1.x` runner.

The reference Go implementation in this monorepo is tagged in lockstep: `ratchet@0.1.0` runs `spec@0.1.x`.

## Deprecation policy

Once at v1.0+:

1. A spec provision marked `Deprecated since v<X>.<Y>` MUST survive at least two minor versions before removal.
2. Deprecation notices appear in `CHANGELOG.md` and in the spec section header itself.
3. Implementations encountering a deprecated provision MUST emit an operator-visible warning but MUST NOT fail.
4. Removal happens in `v<X>.<Y+2>` at the earliest. The removal is announced in `CHANGELOG.md` with a `BREAKING` flag.

Pre-1.0, deprecation may be shorter, but a one-version overlap is RECOMMENDED.

## RFC process

Material changes to the spec are proposed as RFCs in `docs/rfcs/`. The shape:

```
docs/rfcs/
├── 0001-introduce-tdd-gate.md
├── 0002-reflexion-artifact.md
└── template.md
```

RFC structure:

```markdown
# RFC <N>: <title>

Status: Draft | Accepted | Rejected | Superseded
Author: <name>
Created: <YYYY-MM-DD>
Last updated: <YYYY-MM-DD>
Supersedes: (none | RFC-<N>)
Superseded by: (none | RFC-<N>)

## Summary

(One paragraph.)

## Motivation

(What problem does this solve? Cite sources.)

## Detailed design

(The change to the spec, in normative language.)

## Alternatives considered

(What else was on the table; why rejected.)

## Backwards compatibility

(Breaking? Deprecating?)

## Adoption plan

(How does this roll out across implementations?)

## Open questions

(Anything unresolved.)
```

RFCs progress: `Draft` → public discussion (issue/PR comments) → `Accepted` (merged with `BREAKING`/`SPEC` tag) → realized in next minor release.

This is modeled on Python PEPs, MCP RFCs, and Symphony's commit-as-RFC pattern.

## Conformance report

`docs/conformance.md` is auto-generated from the spec's normative claims. Every `MUST` and `SHOULD` in the spec is mapped to a test ID. CI publishes a conformance report per release, listing pass/fail for each claim.

The mapping schema:

```yaml
# Generated; do not hand-edit.
- claim_id: SPEC-13.1
  text: "Primary verdicts MUST be produced by a non-LLM oracle."
  rfc2119_level: MUST
  test_ids:
    - test/runner/verdict_test.go::TestPrimaryVerdictRejectsLLMSource
  status: pass
  last_run: 2026-05-06T19:14:00Z
```

## Quarterly spec review

Every quarter (90-day half-life ratchet), maintainers conduct an explicit review:

1. **Stress-test assumptions** (Anthropic's rule). For each gate, re-evaluate the `assumptions:` frontmatter field. Are those failure modes still occurring on current models? If a model no longer makes the mistake the gate was designed to catch, the gate is a candidate for retirement.
2. **Consult the corpus.** Recent posts on Anthropic Engineering, OpenAI Index, LangChain Blog. Latent Space episodes. Any harness-relevant paper.
3. **Update `docs/references.md`.** Add new sources, retire stale ones.
4. **File RFCs** for any material change.
5. **Tag a release.** Even if the spec is unchanged, tag a `v0.<N+1>.0` release so the conformance report runs against current implementations.

The review is documented in `docs/quarterly-reviews/<YYYY-Q<N>>.md`.

## Reference-implementation lockstep

The Go runner in `cmd/ratchet/` and the spec are tagged together. A spec change that requires a runner change MUST land both in the same release. If the runner cannot be updated in time, the spec change is held.

This prevents "spec ahead of impl" drift, which is a common failure mode in spec-driven projects.

## Broken-link CI

`docs/references.md` is a living index. URLs rot. CI runs a link checker on every PR; broken links are flagged but do not fail the build. A weekly job opens an issue listing currently-broken links for maintainer review.

When a primary source goes offline, prefer recovering it via Wayback Machine and archiving the snapshot URL in `docs/references.md` alongside the live link.

## Self-hosting

ratchet runs ratchet on its own commits. The `.github/workflows/ci.yml` invokes the runner against every PR; the TDD gate gates ratchet's own development.

This is the strongest form of dogfooding: the spec's first-class gate gates the spec itself.

## Versioning the doctrine

The `ratchet.md` Markdown body — the doctrine prompt prepended to agents — also versions. Major doctrine changes bump the minor version of the spec. The doctrine is the most user-visible artifact and changes to it ripple into every agent's behavior; treat it with the same care as a normative spec section.

## What we will not do

- We will not retroactively edit accepted RFCs. Superseded RFCs are marked, not rewritten.
- We will not silently change normative claims. A normative change is a versioned change.
- We will not accumulate gates indefinitely. Stale gates are retired per the quarterly review.
- We will not couple the spec to a single host. Adding `adapter/<host>` to the spec is permitted; making any normative claim depend on a specific host is forbidden.
