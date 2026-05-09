# maxwell — Specification

Status: Draft v0.1.0 (pre-stable)
Spec version: `0.1`
License: Apache-2.0

Purpose: Define a host-agnostic harness for code-generation agents that enforces composable, hard-verdict gates over agent output. The first-class gate is red/green/refactor TDD enforcement on a git-topology substrate. The spec is the durable artifact of this project; the reference Go implementation in this monorepo is one of many possible conforming implementations.

## Normative Language

The key words `MUST`, `MUST NOT`, `REQUIRED`, `SHALL`, `SHALL NOT`, `SHOULD`, `SHOULD NOT`, `RECOMMENDED`, `MAY`, and `OPTIONAL` are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) and [RFC 8174](https://www.rfc-editor.org/rfc/rfc8174).

`Implementation-defined` means the behavior is part of the implementation contract, but this specification does not prescribe one universal policy. Implementations MUST document the selected behavior.

`Reserved` means the field, namespace, or mechanism is set aside for a future spec version and MUST NOT be used by current implementations.

## 1. Problem Statement

Code-generation agents produce work whose quality is uneven and whose self-reports are unreliable. Two related problems follow:

1. **Self-evaluation is biased.** Models confidently rate their own output as correct even when it is not. Sleeper Agents (Hubinger et al. 2024) further demonstrates that adversarial training can teach a model to *hide* misbehavior rather than remove it; standard techniques can therefore "create a false impression of safety."
2. **Soft review scales poorly.** Human review is sampling-based and post-hoc. As agent throughput rises, the bottleneck shifts from writing code to *managing* agentic work (Lopopolo, OpenAI Harness Engineering, Feb 2026).

maxwell exists to close both problems by enforcing **hard, composable, deterministic verdicts** over agent output. Each verdict is a checkpoint with a binary outcome: nothing soft-passes. Gates are sequenced; failure of any gate fails the run. The first gate is red/green/refactor TDD: the agent MUST produce a failing test before production code, then make the test pass without breaking other tests, and MUST NOT delete or weaken existing tests.

maxwell is intentionally portable. Its policy lives in `maxwell.md` and `AGENTS.md`; its gates live in `skills/<gate-name>/`; its host integration is one thin adapter per host (Claude Code v0; Codex App Server, Cursor, ACP, A2A reserved). A user MUST be able to clone maxwell onto a new codebase or take it between employers without dependency on a specific vendor.

## 2. Goals and Non-Goals

### 2.1 Goals

- Enforce **hard verdicts** over agent output via deterministic, externally-verifiable gates.
- Sequence gates with a strict reducer: any failure fails the run; any regression collapses the verdict to `NO`.
- Keep gate **policy in-repo** (`maxwell.md`, `skills/`) so teams version their gates with their code.
- Define a **portable host-adapter protocol** so the same gate set works across Claude Code, Codex, Cursor, and any future host that respects the protocol.
- Persist **Reflexion-shaped failure artifacts** (Shinn et al. 2023) so subsequent attempts can read prior reflections and avoid repeating mistakes.
- Persist **METR/AISI-aligned structured transcripts** so longitudinal analysis (time-horizon, eval-awareness, reward-hacking detection) becomes possible without re-running.
- Stay lightweight: a single static binary, polyglot subprocess gates, no language-runtime tax on the user's repository.
- Evolve on a 90-day cadence. "Every component in a harness encodes an assumption about what the model can't do on its own; those assumptions go stale as models improve" (Anthropic, Harness Design for Long-Running Apps, Mar 2026). The spec REQUIRES quarterly review; obsolete provisions MUST be retired explicitly.

### 2.2 Non-Goals

- This specification does not prescribe a coding-style enforcement system. Gates that bake human-curated taxonomies of "good code" into the runner violate Sutton's Bitter Lesson and are out of scope (see §14.4).
- This specification does not provide a general-purpose workflow orchestrator. maxwell runs gates against agent output; it does not schedule arbitrary jobs.
- This specification does not mandate a specific test framework or language. Gates are subprocess-shaped; any executable producing an exit code qualifies.
- This specification does not provide a tracker integration in v0. Symphony's per-issue workspace pattern is reserved; v1+ MAY add a tracker layer.
- This specification does not provide a managed multi-tenant deployment. maxwell runs locally or in CI; production deployments are the user's responsibility.
- This specification does not require LLM-judged primary verdicts. Such verdicts are forbidden by §13.

## 3. System Overview

### 3.1 The Six Layers

maxwell is portable when kept in these layers. Each layer is replaceable; each gate fits exactly one layer.

1. **Policy Layer** (repo-defined)
   - `maxwell.md` configuration + prompt body.
   - `AGENTS.md` table-of-contents into `docs/`.
   - `skills/<gate-name>/SKILL.md` per-gate definitions.
   - Team-specific rules for gate selection, ordering, severity thresholds.

2. **Configuration Layer** (typed getters)
   - Parses `maxwell.md` frontmatter into typed runtime settings.
   - Handles defaults, environment-variable resolution, path normalization, settings precedence (`user | project | local`).

3. **Coordination Layer** (runner)
   - Gate dispatch in declared order, fail-fast on hard verdicts.
   - Verdict aggregation per §9.
   - Reflection artifact emission per §10.
   - Transcript emission per §15.

4. **Execution Layer** (workspace + subprocess)
   - Per-task workspace isolation (filesystem boundary; safety invariants per §11).
   - Subprocess execution of gate scripts with timeouts and resource limits.

5. **Adaptation Layer** (host adapter)
   - One adapter per host. Currently REQUIRED: Claude Code (`adapter/claudecode`).
   - Reserved: Codex App Server (`adapter/codex`), Cursor (`adapter/cursor`), ACP (`adapter/acp`), A2A (`adapter/a2a`).

6. **Observability Layer** (logs + transcripts)
   - Structured logs (key=value) with stable identifiers.
   - METR/AISI-aligned per-run transcripts.
   - Optional snapshot interface for live monitoring.

### 3.2 The Brain / Hands / Session Triad (Reserved Architecture)

Per Anthropic's Managed Agents (Apr 2026), a mature long-running harness decomposes into:
- **Session** — append-only event log, the durable system of record.
- **Brain** — the stateless harness loop that calls the model and routes tool calls.
- **Hands** — interchangeable sandbox (the workspace).

v0 implementations MAY collapse all three into a single in-process runner. v1+ implementations SHOULD separate them so the harness leaves the container, the container becomes interchangeable cattle, and `wake(sessionId)` + `getEvents()` resumes from the last event after a crash.

The `Session.events[]` log MUST be append-only and durable (filesystem in v0; opaque storage backend in v1+). Implementations MUST NOT depend on in-memory state surviving process restart.

### 3.3 External Dependencies

- A POSIX-like filesystem for workspaces, skills, and reflections.
- `git` 2.40+ for the topology substrate (§9.3).
- A host CLI (Claude Code 2.1.x+ for v0).
- The host's hook integration mechanism (Claude Code `PreToolUse` / `PostToolUse` for v0).

maxwell MUST NOT require Docker, Kubernetes, or a specific cloud provider.

## 4. Core Domain Model

### 4.1 Entities

#### 4.1.1 Gate

A `Gate` is a deterministic external verifier over agent output. Fields:

- `name` (string) — `[a-z0-9-]`, 1–64 chars, no leading/trailing/consecutive hyphens, MUST equal the parent directory name in `skills/<name>/`.
- `description` (string) — what + when, 1–1024 chars.
- `severity` (enum: `P0` | `P1` | `P2`) — `P0` failures are unrecoverable; `P1` failures fail the run with retry permitted; `P2` failures are advisory and recorded but do not fail the run.
- `scripts` (list) — ordered list of executables under `scripts/`. Verdict aggregation per §9.
- `assumptions` (list of strings) — REQUIRED documentation of which model failure modes this gate covers (per Anthropic's "stress-test the assumptions" rule). Used during quarterly review.
- `maxwell_spec_version` (string) — the spec version this gate is pinned to (e.g., `"0.1"`). Runners MUST refuse incompatible pins.

#### 4.1.2 Verdict

A `Verdict` is the aggregate result of running a Gate's scripts. Fields:

- `value` (enum: `FULL` | `PARTIAL` | `NO` | `ERROR`) — see §9.
- `f2p` (float, 0.0–1.0) — fraction of FAIL_TO_PASS criteria met.
- `p2p` (float, 0.0–1.0) — fraction of PASS_TO_PASS criteria preserved.
- `evidence` (list of structured records) — per-script outputs, exit codes, captured stderr/stdout.
- `reason` (string) — human-readable summary, 1–512 chars.
- `verdict_source` (string) — the oracle that produced the verdict. MUST be one of: `git`, `subprocess_exit_code`, `parser`, `compiler`, `test_runner`, `linter`, `type_checker`, `llm_advisory`. The value `llm_advisory` MUST NOT appear in primary verdicts.
- `judge_kind` (enum: `deterministic` | `llm_advisory`) — `llm_advisory` is permitted only for secondary advisory signals (§13.4).

#### 4.1.3 Reflection

A `Reflection` is an artifact written after a gate failure that the next attempt's prompt prepends (Shinn et al. 2023). Fields:

- `gate` (string) — the gate name.
- `verdict` (Verdict) — the failure verdict.
- `observation` (string) — what the gate saw; concrete, evidence-derived.
- `suggested_next_action` (string) — single sentence.
- `attempt_number` (integer, ≥1).
- `created_at` (UTC timestamp).

Reflections persist at `.maxwell/reflections/<task-id>/<gate-name>/<attempt>.md`.

#### 4.1.4 Transcript

A `Transcript` is a per-run structured record consumed by METR/AISI-aligned longitudinal analysis. REQUIRED fields enumerated in §15.

#### 4.1.5 Workspace

A `Workspace` is the per-task filesystem boundary in which gate scripts run. Fields:

- `path` (absolute) — workspace path on disk.
- `task_id` (string) — sanitized task identifier.
- `created_at` (UTC timestamp).

Workspaces MUST be reused across attempts of the same task and MUST NOT be reused across distinct tasks. Sanitization rules are normative in §11.3.

#### 4.1.6 Task

A `Task` is the unit of work being gated. Fields:

- `id` (string, sanitized) — see §11.3.
- `title` (string).
- `f2p` (list of test identifiers) — FAIL_TO_PASS criteria. The agent's work is verified by these tests transitioning from failing to passing.
- `p2p` (list of test identifiers) — PASS_TO_PASS criteria. These tests MUST remain passing throughout.
- `human_baseline_minutes` (integer or null) — REQUIRED for METR-style longitudinal analysis. Implementations MAY emit `null` when unknown.
- `prod_code_globs` (list of glob patterns) — paths whose modification triggers the TDD gate. Default: `["src/**", "lib/**", "internal/**", "pkg/**"]`.
- `test_globs` (list of glob patterns) — paths matching test files. Default: `["**/*_test.*", "tests/**", "test/**", "spec/**"]`.

#### 4.1.7 GateRun

A `GateRun` is one execution of one gate against one task attempt. Fields:

- `gate_name`
- `task_id`
- `attempt_number`
- `started_at` / `ended_at` (UTC)
- `verdict` (Verdict)
- `evidence_path` (path to captured artifacts)
- `reflection_path` (path to reflection if verdict ≠ FULL)

#### 4.1.8 Session and EventLog (Reserved for v1+)

`Session` and its append-only `EventLog` are reserved fields for the Brain/Hands/Session triad. v0 implementations MAY treat the in-memory run state as the session.

### 4.2 Stable Identifiers and Normalization

- **Task ID sanitization**: Replace any character not in `[A-Za-z0-9._-]` with `_`. Result MUST NOT be empty.
- **Gate name normalization**: Lowercase, hyphenated. MUST match the parent directory name.
- **Spec version comparison**: SemVer minor-version-tolerant. A gate pinned to `"0.1"` runs on any `0.1.x` runner.

## 5. Repository Contract

### 5.1 Required Files at Repository Root

A repository using maxwell MUST contain:

- `maxwell.md` — configuration + doctrine prompt body (§5.2).
- `AGENTS.md` — table-of-contents into `docs/` (§5.3).
- `skills/<gate-name>/SKILL.md` for each active gate (§5.4).

A repository using maxwell SHOULD contain:

- `docs/spec.md` — pinned reference to the spec version in use.
- `.maxwell/settings.toml` (project scope).
- `.maxwell/reflections/` (gitignored; runtime artifact directory).
- `.maxwell/transcripts/` (gitignored or committed per team policy).

### 5.2 `maxwell.md` Format

`maxwell.md` is a Markdown file with REQUIRED YAML front matter.

```markdown
---
maxwell_spec_version: "0.1"
verdict_model: hard         # hard | advisory; "advisory" forbidden in v0 except for P2 gates
self_judgment: forbidden    # forbidden | permitted; "permitted" reserved for future versions
gates:
  - name: tdd-red-green-refactor
    severity: P0
hooks:
  PreToolUse:
    - matcher: "Write|Edit"
      command: "maxwell gate --on=PreToolUse"
  PostToolUse:
    - matcher: "Write|Edit"
      command: "maxwell gate --on=PostToolUse"
runner:
  timeout_ms: 60000
  flake_budget_pct: 0.5
workspace:
  isolation: per-task
  cleanup: never            # never | on-completion | on-success
---

# maxwell doctrine

(Markdown body — the prompt prepended to any agent encountering maxwell for the first time.
Contains the TDD canon and core invariants. See maxwell.md in this repo for the canonical body.)
```

Parsing rules:

- The file MUST start with `---` and a closing `---` on a line by itself. Front matter is REQUIRED.
- Front matter MUST decode to a YAML map. Non-map front matter is an error.
- The Markdown body is the doctrine prompt. Implementations MUST NOT silently substitute a default; an empty body is an error.

### 5.3 `AGENTS.md` Format

Per Lopopolo's verbatim rule: `AGENTS.md` is a **table-of-contents**, not an encyclopedia. Target ~100 lines. It MUST link out to `docs/` and `skills/` rather than inlining content.

`AGENTS.md` MUST be standard Markdown with no required frontmatter (matching the open AGENTS.md standard at agents.md). Closest-file-wins precedence applies for nested AGENTS.md within a repository.

### 5.4 Skill Bundle Layout

Each gate is a skill bundle:

```
skills/<gate-name>/
├── SKILL.md            # YAML frontmatter (name, description, severity, maxwell_spec_version)
└── scripts/
    └── <executable>    # exit code 0 = pass; non-zero = fail; stderr = reason
```

`SKILL.md` frontmatter:

- `name` (REQUIRED) — MUST equal `<gate-name>`.
- `description` (REQUIRED) — what + when.
- `severity` (OPTIONAL) — `P0` | `P1` | `P2`. Default: `P1`.
- `maxwell_spec_version` (REQUIRED) — e.g., `"0.1"`.
- `assumptions` (REQUIRED) — list of model-failure modes this gate covers. Used in quarterly review.

Body is unrestricted Markdown but SHOULD describe the gate's contract: inputs, outputs, exit-code semantics, F2P/P2P interpretation if applicable.

### 5.5 Settings Precedence

Settings resolve in this order (later overrides earlier):

1. `~/.maxwell/settings.toml` (user scope)
2. `<repo>/.maxwell/settings.toml` (project scope)
3. `<repo>/.maxwell/settings.local.toml` (local scope; gitignored)

This matches the claude-agent-sdk `SettingSource` precedence.

## 6. Configuration Specification

### 6.1 Resolution Pipeline

1. Locate `maxwell.md` (cwd default; `--config <path>` override).
2. Parse front matter into raw config map.
3. Apply built-in defaults for missing OPTIONAL fields.
4. Resolve `$VAR_NAME` indirection only for values that explicitly contain it.
5. Coerce and validate typed values.
6. Merge in settings precedence (§5.5).

Environment variables MUST NOT globally override YAML values. They MAY only be substituted where a value explicitly references them.

### 6.2 Validation

- Invalid front matter MUST fail startup with an operator-visible error.
- Unknown top-level keys SHOULD be ignored for forward compatibility.
- A gate referenced by `gates:` whose `skills/<name>/SKILL.md` does not exist MUST fail validation.
- A `maxwell_spec_version` outside the runner's compatibility range MUST fail validation.

### 6.3 Dynamic Reload (Reserved for v1+)

v0 reads configuration at startup. v1+ implementations MAY watch `maxwell.md` for changes and re-apply.

## 7. Gate Lifecycle State Machine

### 7.1 Gate States

1. `Pending` — gate scheduled but not yet running.
2. `Running` — gate scripts executing.
3. `EvidenceCollecting` — scripts complete; runner aggregating outputs.
4. `Verdict` — aggregate verdict computed; one of `FULL` / `PARTIAL` / `NO` / `ERROR`.
5. `ReflectionWritten` — if verdict ≠ FULL, reflection persisted.
6. `Done` — terminal.

### 7.2 Transition Triggers

- `Pending → Running`: dispatcher picks next gate per declared order.
- `Running → EvidenceCollecting`: all scripts terminated (success, failure, timeout, or kill).
- `EvidenceCollecting → Verdict`: aggregation per §9.
- `Verdict → ReflectionWritten`: writer per §10.
- `Any → Done`.

### 7.3 Sequencing

Gates MUST run in the order declared in `maxwell.md`'s `gates:` list. Cheaper deterministic gates SHOULD precede expensive LLM-advisory gates so the run fails fast.

A `P0` failure MUST abort all subsequent gates in the run. A `P1` failure MUST abort subsequent `P0` gates and MAY proceed with `P1`/`P2` gates per implementation policy. A `P2` failure MUST NOT abort the run.

## 8. Verdict Model

### 8.1 The FULL/PARTIAL/NO Ladder

Adopted from SWE-Bench's grading methodology. Per-gate verdict is computed as:

- `FULL` iff `f2p == 1.0` AND `p2p == 1.0`.
- `PARTIAL` iff `0 < f2p < 1.0` AND `p2p == 1.0`.
- `NO` otherwise.
- `ERROR` if any script failed to execute (timeout, infrastructure error, malformed output) and could not be retried within `flake_budget_pct`.

### 8.2 Reducer

Aggregating multiple gates into a run-level verdict uses a strict reducer:

- Run is `FULL` iff all gates are `FULL`.
- Run is `PARTIAL` iff at least one gate is `PARTIAL` AND no gate is `NO` or `ERROR`.
- Run is `NO` iff any gate is `NO`.
- Run is `ERROR` iff any gate is `ERROR` and no prior gate was `NO`.

**Any regression collapses to `NO`.** This asymmetry is deliberate: F2P failure is recoverable (the agent has more work to do); P2P failure is unforgivable (the agent broke something that worked).

### 8.3 Test Absence Equals Test Failure

If a gate script declares an F2P or P2P criterion that produces no test result (silent test-suite collapse, missing test runner output), the runner MUST treat that criterion as failed. Silent-collapse gaming is the #1 vector for false-pass verdicts (SWE-Bench observation).

### 8.4 Pass@k vs Pass^k

Implementations SHOULD report both:

- `Pass@k` — fraction of tasks where at least one of `k` attempts succeeded.
- `Pass^k` — fraction of tasks where all of `k` attempts succeeded (consistency).

Per Anthropic's *Demystifying Evals* (Jan 2026): at per-trial 75% × 3 trials, Pass@3 ≈ 98% but Pass^3 ≈ 42%. Both numbers MUST be available in the transcript.

### 8.5 Flake Budget

Implementations MAY rerun a gate script up to a `flake_budget_pct` (default `0.5%`) of total runtime if non-determinism is suspected. Reruns MUST be logged. Determinism failures beyond the flake budget MUST surface as `ERROR` verdicts.

## 9. Reflection Artifact Contract

### 9.1 When to Write

A reflection MUST be written when a gate's verdict is `PARTIAL`, `NO`, or `ERROR`. Reflections MUST NOT be written for `FULL` verdicts.

### 9.2 Persistence Path

`<repo>/.maxwell/reflections/<task-id>/<gate-name>/<attempt-number>.md`

The directory `.maxwell/reflections/` SHOULD be gitignored.

### 9.3 Format

```markdown
---
gate: tdd-red-green-refactor
verdict: NO
attempt_number: 2
created_at: 2026-05-06T19:14:00Z
---

# Observation

The agent edited src/auth.go but no test in tests/ matches the modified
production symbol. f2p criterion `TestLoginRejectsExpiredToken` was not
added in this turn (no new test file or test function in the diff).

# Suggested next action

Add a test case for the auth-expiration logic in `tests/auth_test.go`
that exercises the modified code path. Confirm it fails before editing
any production file.
```

### 9.4 Retry Semantics

The next attempt's prompt MUST prepend the most recent reflection for each failed gate. Implementations MAY truncate older reflections per a configured `max_reflections_per_gate` (default: 3).

Reflections MUST NOT be modified after writing. Append-only.

## 10. Host Adapter Protocol

### 10.1 Adapter Responsibilities

A host adapter MUST:

- Translate the host's hook events into maxwell's gate dispatch (§3.2).
- Generate the host's permission/settings file with values derived from `maxwell.md` (§10.2).
- Stream agent activity (turns, tool calls) to the runner for transcript emission.

A host adapter MUST NOT:

- Modify gate verdicts based on host-specific signals.
- Forward agent output to a learned judge for primary verdict (§13.2).

### 10.2 Claude Code Adapter (v0)

The Claude Code adapter MUST:

- Generate `.claude/settings.json` (or `.claude_settings.json` in legacy projects) per attempt with:
  - `permissions.allow` derived from `maxwell.md` (default: `["Read(./**)", "Write(./**)", "Bash(*)"]` with Bash gated by hooks).
  - `permissions.deny` for any path outside the workspace.
  - `permissionMode: "acceptEdits"` by default.
  - Hooks for `PreToolUse`, `PostToolUse`, `Stop`, `PreCompact`, `PostCompact` (PascalCase, exact match).
- Pin minimum Claude CLI version `2.1.0`. The runner MUST refuse to launch against older CLIs.
- Consume `claude --output-format stream-json` and parse JSONL events.
- Emit maxwell transcript fields (§15) from the JSONL stream.

### 10.3 Reserved Adapter Namespaces

- `adapter/codex` — Codex App Server (Item / Turn / Thread primitives over JSON-RPC).
- `adapter/cursor` — Cursor (`.cursor/rules/*.mdc` policy + ToolUse hooks).
- `adapter/acp` — Agent Client Protocol (Zed Industries).
- `adapter/a2a` — Agent2Agent Protocol (Linux Foundation), with `Task` lifecycle (`WORKING → INPUT_REQUIRED → COMPLETED|FAILED|REJECTED`) mapping to gate verdicts.

## 11. Workspace Management

### 11.1 Layout

`workspace.root` is normalized to an absolute path. Per-task workspace is at `<workspace.root>/<sanitized-task-id>`.

### 11.2 Reuse and Cleanup

- Workspaces MUST be reused across attempts of the same task.
- Workspaces MUST NOT be reused across distinct tasks.
- Cleanup policy (`workspace.cleanup`):
  - `never` (default) — workspaces persist across runs.
  - `on-completion` — cleaned after any terminal verdict.
  - `on-success` — cleaned only when verdict is `FULL`.

### 11.3 Safety Invariants (Mandatory)

- **Invariant 1**: Gate scripts MUST run with `cwd == workspace.path`.
- **Invariant 2**: `workspace.path` MUST resolve to an absolute path under `workspace.root`. Implementations MUST normalize and reject paths outside the root.
- **Invariant 3**: Task IDs MUST be sanitized: characters outside `[A-Za-z0-9._-]` MUST be replaced with `_`. Empty results are an error.
- **Invariant 4**: Gate scripts MUST NOT receive credentials beyond what is explicitly configured in `hooks` or `runner` sections.
- **Invariant 5**: The agent under test MUST NOT have read or write access to gate scripts, gate evidence, or reflection artifacts during the run (§13.5).

## 12. Failure Model and Recovery

### 12.1 Failure Classes

1. **Configuration failures**: missing or invalid `maxwell.md`; missing referenced skill; spec-version mismatch.
2. **Workspace failures**: directory creation failure; path-containment violation; permission error.
3. **Gate-script failures**: timeout; infrastructure error; non-zero exit code; malformed output.
4. **Host-adapter failures**: host CLI not found; version mismatch; protocol error.
5. **Observability failures**: log sink failure; transcript serialization error.

### 12.2 Recovery Behavior

- Configuration failures: refuse to start; emit operator-visible error.
- Workspace failures: fail the run; do not retry.
- Gate-script timeouts: kill subprocess; record `ERROR` verdict; do not retry beyond flake budget.
- Host-adapter failures: surface to operator; runner remains alive.
- Observability failures: log a warning to remaining sinks; do not fail the run.

### 12.3 Reset vs Resume

On gate failure (`PARTIAL` / `NO`):
- The default policy is **reset**: the next attempt starts fresh with reflections prepended.
- v1+ implementations MAY support resume from event log (§3.2) for transient failures.

Per Anthropic's *harness-design-long-running-apps*: "context resets > compaction" for hard failures; resume only for transient.

## 13. Security and Safety Invariants

This section is the most important constraint on any conforming implementation. The invariants below are **MUSTs**, not SHOULDs.

### 13.1 No LLM Self-Judgment of Primary Verdicts

A primary verdict MUST be produced by a non-LLM oracle: compilation, test execution, type checker, linter, exit code, structural diff, or git topology check. The model under test (or any of its sibling/derivative checkpoints) MUST NOT be the primary grader of its own outputs.

This invariant is grounded in Sleeper Agents (Hubinger et al. 2024): "Standard techniques could fail to remove such deception and create a false impression of safety." A model that has learned to hide misbehavior cannot be relied upon to judge whether it has done so.

LLM judgment MAY be used only as `judge_kind: llm_advisory` for secondary signals, and MUST be flagged as such in the transcript. Advisory verdicts MUST NOT change the run's primary verdict.

### 13.2 Adversarial Training Is Not a Substitute

Passing adversarial / red-team training MUST NOT be treated as evidence that a behavior has been removed. Per Hubinger et al.: adversarial training "effectively hiding unwanted behavior during adversarial training and evaluation, rather than training it away."

Red-team gates MUST be **additive** to ground-truth gates, never substitutable.

### 13.3 Chain-of-Thought Preservation

When the agent emits a chain-of-thought, scratchpad, or reasoning trace, the runner MUST preserve it as a separate transcript field (`cot_or_scratchpad`), distinct from the agent's final output. Distillation that hides CoT is forbidden by this spec.

### 13.4 Honeypot Subset

Implementations SHOULD reserve a fraction of every run for **honeypot probes**: inputs designed to detect deceptive or eval-aware behavior. Honeypot results MUST be transcript-flagged (`honeypot_subset: true`). Honeypots MUST NOT overlap the standard task distribution.

### 13.5 Grader Isolation

The agent under test MUST NOT be able to read or write:

- Gate scripts (`skills/<gate-name>/scripts/*`).
- Gate evidence (`.maxwell/evidence/`).
- Reflection artifacts of the current run (until written and frozen).
- Transcript files (until finalized).

Implementations MUST enforce this through filesystem permissions, sandboxing, or workspace isolation. Per METR's June 2025 reward-hacking finding: agents try to "cheat and get impossibly high scores by exploiting bugs in our scoring code."

### 13.6 Eval-Awareness Recording

The runner MUST scan agent CoT and tool calls for eval-awareness markers (substrings matching `/test|eval|grader|rubric|benchmark|scoring|harness/i`) and record matches as `eval_awareness_flags[]` in the transcript. This is a first-class artifact, not noise (METR August 2025).

### 13.7 No Soft Safety Claims

Verdict language MUST be: "no failures observed under config X". Verdict language MUST NOT be: "the agent is safe". Per Anthropic Core Views: safety claims require external auditability.

### 13.8 Auditable Run Artifacts

A `FULL` verdict MUST be reproducible by a third party from the published transcript. A run whose transcript is incomplete, redacted, or unavailable MUST NOT count as `FULL`.

### 13.9 Bitter Lesson Hygiene

Gates are **verifiers over agent output** (allowed). Gates are NOT **heuristics inside agent reasoning** (forbidden). A gate that asserts "tests went red then green" is a verifier; a gate that asserts "use the Repository pattern" is a heuristic and MUST NOT exist in this spec.

Per Sutton's Bitter Lesson: "We want AI agents that can discover like we can, not which contain what we have discovered."

## 14. Logging, Transcripts, Observability

### 14.1 Structured Logs

Logs MUST use stable `key=value` phrasing. REQUIRED context fields:

- `task_id`, `gate_name`, `attempt_number`, `verdict`, `reason`.

Implementations MAY emit additional fields. Logs MUST NOT contain raw secrets, tokens, or full prompts unless explicitly enabled by an operator-visible flag.

### 14.2 Transcripts (METR/AISI-Aligned)

A transcript MUST be emitted per run at `.maxwell/transcripts/<task-id>/<run-id>.json`. REQUIRED fields:

```json
{
  "task_id": "string",
  "task_family": "string|null",
  "human_baseline_minutes": "integer|null",
  "wall_clock_seconds_start": "number",
  "wall_clock_seconds_end": "number",
  "wall_clock_duration_seconds": "number",
  "turn_count": "integer",
  "tool_call_count": "integer",
  "tokens_input": "integer",
  "tokens_output": "integer",
  "tokens_thinking": "integer|null",
  "model_id": "string",
  "model_version": "string",
  "harness_version": "string",
  "scaffold_config_hash": "string",
  "maxwell_spec_version": "string",
  "temperature": "number|null",
  "seed": "integer|null",
  "verdict": "FULL|PARTIAL|NO|ERROR",
  "verdict_source": "string",
  "judge_kind": "deterministic|llm_advisory",
  "success_threshold_used": "number",
  "transcript": [
    {
      "role": "user|assistant|tool",
      "content": "string",
      "cot_or_scratchpad": "string|null",
      "timestamp": "ISO-8601 UTC",
      "tools_invoked": ["string"]
    }
  ],
  "eval_awareness_flags": ["string"],
  "honeypot_subset": "boolean",
  "sandbox_escape_attempted": "boolean",
  "safety_case_id": "string|null"
}
```

`cot_or_scratchpad` MUST be a separate field per §13.3. It MUST NOT be merged into `content`.

### 14.3 Snapshot Interface (OPTIONAL)

Implementations MAY expose a synchronous runtime snapshot. If present, the snapshot SHOULD return:

- Currently running gates (with task_id, gate_name, started_at).
- Recent verdicts (last N runs).
- Aggregate token totals (input/output/thinking).
- Last operator-visible error.

The snapshot interface MUST NOT be required for correctness.

### 14.4 Resource Reporting

Per Anthropic *Quantifying Infrastructure Noise* (Feb 2026):

- Implementations MUST publish guaranteed-allocation AND hard-limit per task, not a single value.
- The recommended hard-limit is **3× the guaranteed allocation** (5.8% → 2.1% infra-error drop, p<0.001).
- Leaderboard deltas under **3 percentage points** SHOULD be flagged as suspect until configs are matched.

## 15. Reference Algorithms

### 15.1 Run Dispatch

```text
function run(task, gates):
  workspace = ensure_workspace(task.id)
  validate_workspace_invariants(workspace)
  transcript = new_transcript(task)

  for gate in gates:
    if gate.severity == "P0" and any prior verdict was NO:
      record_skipped(gate, "preceding P0 failure")
      continue

    gate_run = run_gate(gate, task, workspace, transcript)
    transcript.gate_runs.append(gate_run)

    if gate_run.verdict.value in {"PARTIAL", "NO", "ERROR"}:
      reflection = build_reflection(gate, gate_run)
      write_reflection(task, gate, reflection)

      if gate.severity == "P0":
        break

  finalize_transcript(transcript)
  return aggregate_verdict(transcript.gate_runs)
```

### 15.2 Verdict Aggregation

```text
function aggregate_verdict(gate_runs):
  if any run.verdict == "NO": return "NO"
  if any run.verdict == "ERROR" and no prior "NO": return "ERROR"
  if any run.verdict == "PARTIAL": return "PARTIAL"
  return "FULL"
```

### 15.3 TDD Red-Green-Refactor Verdict

```text
function tdd_verdict(diff, base_ref, task):
  added_tests = git_diff_added_tests(base_ref, diff, task.test_globs)
  modified_prod = git_diff_modified(base_ref, diff, task.prod_code_globs)
  removed_tests = git_diff_removed_tests(base_ref, diff, task.test_globs)

  if removed_tests is non-empty:
    return verdict NO with reason "test deletion forbidden"

  if modified_prod is non-empty and added_tests is empty:
    return verdict NO with reason "production edit without new test (red phase missing)"

  red_pass = run_tests_at(base_ref + added_tests)
  if red_pass.f2p_count > 0:
    return verdict NO with reason "added tests pass at base_ref (test does not exercise new behavior)"

  green = run_tests_at(HEAD)
  f2p = green.passed_in_f2p / total_f2p
  p2p = green.passed_in_p2p / total_p2p

  if f2p == 1.0 and p2p == 1.0: return FULL
  if f2p > 0 and p2p == 1.0: return PARTIAL
  return NO
```

### 15.4 Reflection Composition

```text
function build_reflection(gate, gate_run):
  return Reflection {
    gate: gate.name,
    verdict: gate_run.verdict,
    observation: extract_observation(gate_run.evidence),
    suggested_next_action: derive_action(gate, gate_run.verdict),
    attempt_number: gate_run.attempt_number,
    created_at: now_utc()
  }
```

`extract_observation` is implementation-defined but MUST cite concrete evidence (file paths, test names, line numbers).

## 16. Test and Validation Matrix

A conforming implementation MUST include tests covering the behaviors below. Profiles:

- **Core Conformance**: REQUIRED for all conforming implementations.
- **Extension Conformance**: REQUIRED only for OPTIONAL features the implementation chooses to ship.
- **Real Integration Profile**: environment-dependent smoke tests RECOMMENDED before production use.

### 16.1 Configuration and Repository Contract (Core)

- C-1.1: Missing `maxwell.md` returns typed error.
- C-1.2: Missing front matter delimiter returns typed error.
- C-1.3: Front matter non-map returns typed error.
- C-1.4: Empty Markdown body returns typed error.
- C-1.5: Spec-version mismatch returns typed error.
- C-1.6: Gate referenced without skill bundle returns typed error.
- C-1.7: Settings precedence (user/project/local) is honored.
- C-1.8: `$VAR` resolution works for explicit references; does not affect non-referenced values.

### 16.2 Workspace Safety (Core)

- C-2.1: Workspace path is resolved to absolute under root.
- C-2.2: Path outside workspace root is rejected.
- C-2.3: Task ID with disallowed characters is sanitized; empty result is rejected.
- C-2.4: Workspace reused across attempts of same task; new workspace per task.

### 16.3 Verdict Aggregation (Core)

- C-3.1: All FULL → FULL.
- C-3.2: Any NO → NO.
- C-3.3: Any ERROR with no prior NO → ERROR.
- C-3.4: PARTIAL with no NO/ERROR → PARTIAL.
- C-3.5: Test absence treated as test failure.
- C-3.6: Flake budget retries logged; exceeding budget → ERROR.

### 16.4 Reflection Artifact (Core)

- C-4.1: Reflection written for non-FULL verdicts.
- C-4.2: Reflection NOT written for FULL verdicts.
- C-4.3: Reflection persistence path matches spec.
- C-4.4: Reflection format includes all REQUIRED fields.
- C-4.5: Reflections are append-only (not modified after write).

### 16.5 Safety Invariants (Core)

- C-5.1: Primary verdict source is never `llm_advisory`.
- C-5.2: CoT preserved as separate transcript field when emitted.
- C-5.3: Eval-awareness flags populated when CoT contains markers.
- C-5.4: Agent cannot read gate scripts during run (filesystem permission test).
- C-5.5: Run with incomplete transcript MUST NOT be reported as FULL.

### 16.6 Transcript (Core)

- C-6.1: All REQUIRED transcript fields present.
- C-6.2: `cot_or_scratchpad` is distinct from `content`.
- C-6.3: Wall-clock timestamps are UTC.
- C-6.4: Pass@k and Pass^k computable from transcript.

### 16.7 TDD Gate (Extension; REQUIRED if `tdd-red-green-refactor` is shipped)

- E-T.1: Production edit without new test → NO with reason "red phase missing".
- E-T.2: Test deletion → NO with reason "test deletion forbidden".
- E-T.3: Added test passes at base_ref → NO with reason "test does not exercise new behavior".
- E-T.4: F2P=1.0 AND P2P=1.0 → FULL.
- E-T.5: F2P=0.5 AND P2P=1.0 → PARTIAL.
- E-T.6: P2P<1.0 → NO regardless of F2P.

### 16.8 Claude Code Adapter (Extension; REQUIRED if `adapter/claudecode` is shipped)

- E-CC.1: Refuses Claude CLI version <2.1.0.
- E-CC.2: Generates `.claude/settings.json` with REQUIRED hook events.
- E-CC.3: Hook event names match PascalCase canon (`PreToolUse`, `PostToolUse`, `Stop`, `PreCompact`, `PostCompact`).
- E-CC.4: Permission grammar uses `Tool(pattern)` form.
- E-CC.5: Stream-JSON parsing handles malformed lines without crashing.

### 16.9 Real Integration Profile

- R-1: End-to-end run against a fixture repo with a known-failing TDD scenario produces NO.
- R-2: End-to-end run against a fixture repo with a correct red→green flow produces FULL.
- R-3: End-to-end run with the agent attempting to delete a test produces NO.

## 17. Implementation Checklist (Definition of Done)

### 17.1 v0.1 Core Conformance

- [ ] `maxwell.md` parser with frontmatter + body split.
- [ ] `AGENTS.md` discovery (closest-file-wins).
- [ ] Skill bundle loader (`skills/<name>/SKILL.md`).
- [ ] Settings precedence (user/project/local).
- [ ] Gate dispatch in declared order with severity-aware abort.
- [ ] Verdict aggregation per §8.
- [ ] Reflection writer per §10.
- [ ] Transcript emitter per §15.
- [ ] Workspace manager with sanitized per-task workspaces.
- [ ] Filesystem safety invariants (§11.3).
- [ ] Operator-visible structured logs.
- [ ] Test matrix §16.1–§16.6 passing.

### 17.2 v0.1 TDD Gate

- [ ] `skills/tdd-red-green-refactor/SKILL.md` with REQUIRED frontmatter.
- [ ] `scripts/check_red.sh`, `check_green.sh`, `check_immutable.sh`.
- [ ] F2P/P2P semantics implemented.
- [ ] Test matrix §16.7 passing.

### 17.3 v0.1 Claude Code Adapter

- [ ] `.claude/settings.json` generation per attempt.
- [ ] PreToolUse / PostToolUse / Stop / PreCompact / PostCompact hooks.
- [ ] Stream-JSON event consumption.
- [ ] CLI version pinning.
- [ ] Test matrix §16.8 passing.

### 17.4 v0.1 Self-Hosting

- [ ] maxwell runs maxwell on its own commits in CI.
- [ ] Conformance report published per release.

## 18. Glossary

- **Gate**: a deterministic external verifier over agent output.
- **Verdict**: the aggregate result of running a gate's scripts. One of FULL / PARTIAL / NO / ERROR.
- **F2P (FAIL_TO_PASS)**: tests that should transition from failing to passing as a result of the agent's work.
- **P2P (PASS_TO_PASS)**: tests that MUST remain passing throughout the agent's work.
- **Reflection**: a Reflexion-shaped artifact written after a non-FULL verdict; prepended to the next attempt's prompt.
- **Transcript**: per-run structured record consumed by longitudinal analysis.
- **Workspace**: per-task filesystem boundary in which gate scripts run.
- **Hard verdict**: a verdict that cannot be soft-passed; a regression collapses to NO.
- **Honeypot**: an input designed to detect deceptive or eval-aware behavior, off-distribution from the standard task corpus.
- **Skill bundle**: `skills/<gate-name>/` directory containing `SKILL.md` and `scripts/`.
- **Adapter**: a host-specific shim translating between the host's hook mechanism and maxwell's gate dispatch.
- **Brain / Hands / Session**: the stateless-harness / interchangeable-sandbox / append-only-event-log triad reserved for v1+ implementations (Anthropic Managed Agents, Apr 2026).

## 19. Normative References

- [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) — Key words for use in RFCs to Indicate Requirement Levels.
- [RFC 8174](https://www.rfc-editor.org/rfc/rfc8174) — Ambiguity of Uppercase vs Lowercase in RFC 2119 Key Words.
- Hubinger et al. 2024, *Sleeper Agents* — basis for §13.1 and §13.2.
- Lightman et al. 2023, *Let's Verify Step by Step* — basis for per-gate process supervision.
- Shinn et al. 2023, *Reflexion* — basis for §10.
- DeepSeek 2025, *DeepSeek-R1* — basis for deterministic verifiable rewards in §8.
- SWE-Bench grading methodology — basis for FULL/PARTIAL/NO ladder and F2P/P2P split.
- Sutton 2019, *The Bitter Lesson* — basis for §13.9.
- Anthropic 2026, *Harness Design for Long-Running Apps* — basis for context-reset semantics in §12.3 and stress-test rule in §2.1.
- Anthropic 2026, *Managed Agents* — basis for Brain/Hands/Session reserved architecture in §3.2.
- Anthropic 2026, *Demystifying Evals for AI Agents* — basis for Pass@k/Pass^k in §8.4.
- Anthropic 2026, *Quantifying Infrastructure Noise* — basis for §14.4.
- Willison 2025, *Red/green TDD* — basis for the TDD gate canon.
- Lopopolo 2026, *Harness Engineering* — basis for AGENTS.md table-of-contents rule in §5.3.
- Symphony SPEC.md (OpenAI 2026) — structural model for this specification.
- METR — basis for `human_baseline_minutes` and run-artifact requirements.
- UK AI Security Institute — basis for safety-case framing in §13 and §15.

---

*This specification is the durable artifact of the maxwell project. Reference implementations come and go; the spec persists. See `docs/maintenance.md` for the versioning, deprecation, and quarterly review protocol.*
