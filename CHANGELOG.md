# Changelog

All notable changes to ratchet are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versioning per `docs/maintenance.md`.

## [Unreleased]

### Added
- `ratchet recall` subcommand — prints the latest reflection for each (or one) gate of the current task. Closes the anterograde-amnesia loop the v0.1 review surfaced.
- `SessionStart` hook in `ratchet.md` invokes `ratchet recall` so prior failures re-enter context automatically.
- `LatestForGateBody` helper in `internal/reflections`.
- `VerifyCLIVersion` in `internal/adapter/claudecode` — runtime CLI version check; was helper-only before.
- `scrubInjectionEnv` in `internal/runner` — strips LD_PRELOAD/DYLD_*/GIT_* before launching gate scripts (spec §13.5 partial).
- Real red-phase verification in `check_red.sh` for Go (worktree-based; runs newly-added tests at the base ref and confirms they fail). Non-Go runners retain the heuristic with explicit `red_verified=heuristic` flag.
- Zero-test detection in `check_green.sh` — silent test-suite collapse now produces ERROR instead of FULL (spec §8.3).
- Path-segment validation in `internal/reflections.Write` — rejects taskID/gateName containing path separators or `..`.
- Numeric ordering of reflections (zero-padded `00001.md` filenames) so attempt 10 follows attempt 9.

### Fixed
- P0 abort `break` no longer escapes only the `switch` — labeled `gateLoop` break stops subsequent gates per spec §7.3.
- Subprocess timeout produces ERROR (exit 3) instead of NO per spec §8.1.
- `baseRef()` now uses merge-base against main/master (was hardcoded `HEAD~1`, which inverted the canon by failing properly disciplined red-then-green commit sequences).
- Transcript Finalize failure demotes a FULL/PARTIAL verdict to ERROR per spec §13.8 (a green verdict with no on-disk transcript is unauditable).
- `cmdFinalizeTranscript` no-op condition (`strings.HasPrefix(s, "")` was always true).
- `IsCompatibleCLIVersion` now refuses unparseable input (was silently accepting "garbage" as compatible).
- `PreToolUse` hook removed for the TDD gate — running the full test suite twice per tool call violated Cherny's <5s pre-commit rule and PreToolUse cannot verify a diff that hasn't been written.

## [0.1.0-alpha] — 2026-05-06

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
- `skills/tdd-red-green-refactor/` — first gate with `SKILL.md` + `scripts/`.
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
