# maxwell

A host-agnostic harness for code-generation agents. Enforces composable, hard-verdict gates over agent output. The first-class gate is red/green/refactor TDD enforcement on a git-topology substrate.

The spec is the durable artifact. The Go binary in this repo is one reference implementation. Anyone may re-implement in another language and claim conformance against `docs/spec.md`.

```
                  ┌──────────────────────────────────┐
                  │      Agent (Claude / Codex /     │
                  │           Cursor / ...)          │
                  └────────────┬─────────────────────┘
                               │ tool calls
                               ▼
                  ┌──────────────────────────────────┐
                  │   Host adapter (PreToolUse,      │
                  │   PostToolUse, Stop, Compact)    │
                  └────────────┬─────────────────────┘
                               │ gate dispatch
                               ▼
   ┌─────────────────────────────────────────────────────────────┐
   │                     maxwell runner                          │
   │  ┌──────────────┐  ┌──────────────┐  ┌────────────────────┐│
   │  │  TDD gate    │  │  lint gate   │  │  evaluator (P2)    ││
   │  │  (P0)        │  │  (P1)        │  │  llm_advisory      ││
   │  └──────┬───────┘  └──────┬───────┘  └─────────┬──────────┘│
   │         │                  │                    │           │
   │         ▼                  ▼                    ▼           │
   │            verdict aggregation (FULL/PARTIAL/NO/ERROR)      │
   │            reflection emission (.maxwell/reflections/)      │
   │            transcript emission (.maxwell/transcripts/)      │
   └─────────────────────────────────────────────────────────────┘
```

## Why

Self-evaluation is biased; soft review scales poorly; LLM-judged verdicts can be fooled by adversarial training (Sleeper Agents, 2024). maxwell enforces deterministic external verdicts at every step. The TDD gate is the cheapest reliable verifier: a test runner produces a binary verdict in milliseconds, and git topology makes the verdict publicly auditable.

## Quick start

```bash
# Build
go build -o maxwell ./cmd/maxwell

# Initialize a repo
./maxwell init

# Run gates against the working tree
./maxwell run

# Run a specific gate
./maxwell gate --name=tdd-red-green-refactor

# Show version + spec compatibility
./maxwell version
```

The `init` command writes `maxwell.md`, `AGENTS.md`, and `skills/tdd-red-green-refactor/` into the current directory.

## What ships in v0.1

- **Spec** (`docs/spec.md`) — Symphony-class normative document, RFC 2119, conformance matrix.
- **TDD gate** (`skills/tdd-red-green-refactor/`) — red/green/refactor with F2P/P2P semantics adopted from SWE-Bench.
- **Claude Code adapter** (`internal/adapter/claudecode`) — PreToolUse/PostToolUse hooks, `.claude/settings.json` generation, `claude --output-format stream-json` consumption.
- **Reflexion-shaped failure artifacts** (`.maxwell/reflections/`) — written on every non-FULL verdict, prepended to next attempt's prompt.
- **METR/AISI-aligned transcripts** (`.maxwell/transcripts/`) — every required field for longitudinal analysis.
- **Self-hosting CI** — maxwell runs maxwell on its own commits.

## What's reserved for later versions

- `adapter/codex`, `adapter/cursor`, `adapter/acp`, `adapter/a2a` — host adapters.
- `gates.debate.*`, `gates.interp.*`, `gates.sabotage.*`, `gates.horizon.*` — future gate namespaces.
- Brain/Hands/Session triad (Anthropic Managed Agents architecture) — v1+.
- Tracker integration (Symphony's per-issue workspace pattern) — v1+.

## Built for three consumers

Per Karpathy: humans, computers, and **agents**. maxwell's docs are agent-tuned: `AGENTS.md` is the table-of-contents; `maxwell.md` is the doctrine prompt; every gate emits structured stderr; verdicts are exit codes; failure artifacts are Reflexion-shaped.

## Status

`v0.1.0-alpha` — pre-stable. Spec breaks are documented in `CHANGELOG.md`. v1.0 ships when the conformance matrix holds across two consecutive releases without breaking changes.

## Advisors

maxwell draws on the published methodology of Boris Cherny (Anthropic / Claude Code), Ryan Lopopolo (OpenAI / Harness Engineering), Andrej Karpathy (Software 3.0), Simon Willison (TDD with AI), Peter Steinberger (gates philosophy from openclaw), Anthropic's engineering team (Managed Agents, harness design, evals), and the agent-research literature (ReAct, Reflexion, CodeAct, Let's Verify Step by Step, Sleeper Agents, DeepSeek-R1). See `docs/advisor-quotes.md` for the verbatim canon and `docs/references.md` for the full bibliography.

## Contributing

Material spec changes are RFCs in `docs/rfcs/`. The reference implementation is replaceable; the spec is not. See `docs/maintenance.md` for the protocol.

## License

Apache 2.0. See `LICENSE`.

## House rules

- No soft passes.
- Tests are spec.
- First run the tests.
- No LLM self-judgment for primary verdicts.
- Skills > prompts.
- Conventions > configuration.
