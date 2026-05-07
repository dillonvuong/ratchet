# Changelog

All notable changes to ratchet are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versioning per `docs/maintenance.md`.

## [Unreleased]

### Added
- Initial spec, advisor canon, reference implementation scaffolding.

## [0.1.0-alpha] — 2026-05-06

### Added
- `docs/spec.md` — Symphony-class normative specification, RFC 2119.
- `docs/advisor-quotes.md` — pinned verbatim canon (Willison, Cherny, Lopopolo, Karpathy, Sutton, Hubinger, Lightman, DeepSeek-R1, Anthropic engineering).
- `docs/why-tdd-first.md` — argument for TDD as the canonical first gate.
- `docs/adding-a-gate.md` — tutorial.
- `docs/maintenance.md` — versioning, deprecation, RFC process, quarterly review protocol.
- `docs/references.md` — living bibliography with Wayback recovery notes.
- `AGENTS.md` — table-of-contents (~100 lines, per Lopopolo).
- `ratchet.md` — frontmatter + doctrine prompt body.
- `.skills/tdd-red-green-refactor/` — first gate with `SKILL.md` + `scripts/`.
- Go module + `cmd/ratchet/` CLI scaffold.
- `internal/runner` — gate dispatch and verdict aggregation per spec §8.
- `internal/adapter/claudecode` — Claude Code PreToolUse/PostToolUse hook integration.
- `internal/gitsubstrate` — git-topology verdict primitives.
- `internal/reflections` — Reflexion artifact writer per spec §10.
- `internal/transcripts` — METR/AISI structured transcript emitter per spec §15.
- `.github/workflows/ci.yml` — self-hosting CI; ratchet runs ratchet on its own commits.

### Pinned commitments

- Spec version `0.1`.
- Minimum Claude CLI: `2.1.0`.
- Verdict model: `hard`. No soft passes.
- Self-judgment: `forbidden`. No LLM-judged primary verdicts (per spec §13.1).
- License: Apache 2.0.

[Unreleased]: https://github.com/dillon-vuong/ratchet/compare/v0.1.0-alpha...HEAD
[0.1.0-alpha]: https://github.com/dillon-vuong/ratchet/releases/tag/v0.1.0-alpha
