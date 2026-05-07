# AGENTS.md

This is the table of contents. The full content lives elsewhere; this file is intentionally short (per Lopopolo: "Context is a scarce resource. Too much guidance becomes non-guidance.").

## What ratchet is

A host-agnostic harness for code-generation agents. Enforces composable, hard-verdict gates over agent output. First-class gate is red/green/refactor TDD on a git-topology substrate.

The spec is the durable artifact. The Go binary in this repo is one reference implementation.

## Where to look

- **The spec** — `docs/spec.md`. RFC 2119 normative. Read first if you are an agent encountering ratchet.
- **Why TDD is the first gate** — `docs/why-tdd-first.md`.
- **Adding a gate** — `docs/adding-a-gate.md`.
- **Spec maintenance protocol** — `docs/maintenance.md`.
- **Pinned advisor quotes** — `docs/advisor-quotes.md`.
- **References / bibliography** — `docs/references.md`.
- **Doctrine prompt** — `ratchet.md`. The body that prepends to agents on first encounter.
- **Active gates** — `skills/<gate-name>/`.
- **Reflections directory** — `.ratchet/reflections/<task-id>/<gate-name>/<attempt>.md` (gitignored).
- **Transcripts directory** — `.ratchet/transcripts/<task-id>/<run-id>.json`.

## Active gates

- `tdd-red-green-refactor` (`P0`) — agent MUST produce a failing test before production code, then green, MUST NOT delete tests. See `skills/tdd-red-green-refactor/SKILL.md`.

## Build, run, test (this repo)

- Build: `go build -o ratchet ./cmd/ratchet`
- Test: `go test ./...`
- Run on this repo: `./ratchet run`
- Self-hosted CI: see `.github/workflows/ci.yml`. ratchet runs ratchet on its own commits.

## Doctrine (one paragraph)

You are operating under ratchet. Before editing any production-code path matching `src/**`, `lib/**`, `internal/**`, or `pkg/**`, you MUST add a failing test that exercises the new behavior. After the production edit, the test MUST pass and no previously-passing test MUST regress. You MUST NOT delete or weaken existing tests. The verdict is computed from `git diff` against the base ref plus the test runner's exit codes; there is no model in the verdict loop. If a gate fails, you will receive a Reflexion-shaped artifact at `.ratchet/reflections/<task-id>/<gate>/<attempt>.md` describing the observation and a suggested next action; read it before retrying. See `ratchet.md` for the full doctrine.

## House rules

- **No soft passes.** Every gate is hard-verdict. A regression collapses the verdict to `NO`.
- **Tests are spec.** Removing or weakening a test is malfeasance, not refactoring.
- **First run the tests.** Open every session by reading the test layout (Willison's lever).
- **No LLM self-judgment for primary verdicts.** Per spec §13.1.
- **Skills > prompts.** Encode behavior changes in skills, not in prompt edits.
- **Conventions > configuration.** If you have a choice, default.
- **Skim is not read.** When this repo asks you to read a doc, read it end to end.

## Cross-host notes

ratchet works under Claude Code (v0). Codex App Server, Cursor, ACP, and A2A adapters are reserved for future versions. The repo contract (`ratchet.md` + `AGENTS.md` + `skills/`) is portable across hosts; the adapter layer translates host-specific hooks into ratchet's gate dispatch.

If you are an agent under a host without a ratchet adapter, the gates are still runnable as standalone subprocess calls: `ratchet gate --name=tdd-red-green-refactor --task=<task-id>`. The host-adapter layer adds automatic dispatch on tool-call events; without it, the run loop is manual.

## Versioning

Spec version: `0.1` (pre-stable). Reference implementation: `0.1.0-alpha`. See `CHANGELOG.md` and `docs/maintenance.md`.

## Contributing

PRs welcome. The spec is normative; the implementation is replaceable. If you propose a material spec change, file an RFC under `docs/rfcs/`. See `docs/maintenance.md` for the protocol.
