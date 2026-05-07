---
name: tdd-red-green-refactor
description: Enforce red/green/refactor TDD via git topology. Verdict from F2P/P2P test runner exit codes; no LLM in the verdict loop. Triggered when the agent edits production code.
severity: P0
ratchet_spec_version: "0.1"
assumptions:
  - "The model can produce production code without first writing a test that exercises the new behavior."
  - "The model can write a test that already passes against the base reference (does not exercise new behavior)."
  - "The model can be tempted to delete or weaken an existing test to pass a verdict."
  - "The model's self-report of 'I wrote tests' is not reliable; git diff is."
---

# tdd-red-green-refactor

The first-class ratchet gate. Enforces three sub-properties of agent diffs:

1. **Red phase exists** — every production-code edit is preceded (in this turn) by a new failing test.
2. **Green phase achieved** — F2P tests pass and P2P tests still pass.
3. **Tests are immutable** — no test deletion or weakening relative to the base reference.

The verdict is computed from `git diff` and the test runner's exit codes. There is no model in the verdict loop.

## Verdict ladder

Per `docs/spec.md` §8.1:

- `FULL` iff F2P=1.0 AND P2P=1.0 AND no test deletion.
- `PARTIAL` iff 0 < F2P < 1.0 AND P2P=1.0 AND no test deletion.
- `NO` if any of: P2P regression, test deletion, no new test added, added test passes at base ref.
- `ERROR` if test runner failed to execute (timeout, infra error).

## Sub-gate scripts

Three scripts compose the verdict. Each emits exit code + structured stderr.

### `scripts/check_immutable.sh` (runs first; fastest)

Detects test deletions or assertion-weakening modifications relative to the base reference.

- Exit `0` — no test deletions; no test functions removed; no `assert` lines removed.
- Exit `1` — test deletion or weakening detected; verdict `NO`.

stderr format: `status=<pass|fail> reason=<short> evidence=<file:line>`

### `scripts/check_red.sh` (runs second; cheap)

For every production-code edit in this turn, verify a new test was added and that test fails at the base reference.

- Exit `0` — new test exists for each prod-code edit; new tests fail at base.
- Exit `1` — production edit without new test, or new test passes at base; verdict `NO`.

stderr format: `status=<pass|fail> reason=<short> f2p_count=<N> red_verified=<N>`

### `scripts/check_green.sh` (runs last; most expensive)

Run F2P and P2P tests at HEAD and compute pass-rates.

- Exit `0` — F2P=1.0 AND P2P=1.0 → verdict `FULL`.
- Exit `2` — 0 < F2P < 1.0 AND P2P=1.0 → verdict `PARTIAL`.
- Exit `1` — P2P regression or F2P=0 → verdict `NO`.
- Exit `3` — test runner error → verdict `ERROR`.

stderr format: `status=<pass|partial|fail|error> f2p=<float> p2p=<float> reason=<short>`

## Environment contract

ratchet passes the workspace path as `$1`. Additional context via env vars:

- `RATCHET_TASK_ID` — sanitized task identifier.
- `RATCHET_BASE_REF` — git ref to diff against (default: `HEAD~1` or merge base).
- `RATCHET_F2P_TESTS` — space-separated list of FAIL_TO_PASS test identifiers (optional; defaults inferred from diff).
- `RATCHET_P2P_TESTS` — space-separated list of PASS_TO_PASS test identifiers (optional; defaults to "all not in F2P").
- `RATCHET_PROD_CODE_GLOBS` — colon-separated globs for production code paths.
- `RATCHET_TEST_GLOBS` — colon-separated globs for test paths.
- `RATCHET_TEST_RUNNER` — auto-detected (go|pytest|npm|etc.) or set explicitly.

## Test runner detection

Auto-detection in priority order:

1. `go.mod` exists → `go test ./...` with `-run` filter for F2P/P2P.
2. `pyproject.toml` or `pytest.ini` exists → `pytest` with `-k` filter.
3. `package.json` with `scripts.test` → `npm test`.
4. `Cargo.toml` → `cargo test`.
5. Otherwise `RATCHET_TEST_RUNNER` MUST be set.

The runner detection is implemented in the Go binary; the shell scripts here invoke `ratchet --internal-test-runner` for portability.

## What this gate does NOT do

- It does not prescribe a test framework. Any framework whose runner emits exit codes works.
- It does not measure coverage; that's a separate gate.
- It does not enforce test quality (assertion specificity, fixture discipline). The verdict is binary; quality is the user's concern.
- It does not gate aesthetics, naming, or architecture. Those would violate spec §13.9 (Bitter Lesson hygiene).

## Why this gate is `P0`

Per spec severity semantics, `P0` failures abort the run and prevent subsequent gates. The TDD gate is `P0` because:

1. Every other gate's signal is muddled if the underlying TDD topology is broken. There is no point lint-checking code that should not have been written without a test.
2. Skipping the red phase is a malpractice signal: the agent is not following the basic discipline.
3. The cost of the gate is low (typically <5 seconds for a small diff); the cost of letting an untested change through is unbounded.

## References

- `docs/spec.md` §8 (verdict model), §11 (workspace), §13 (safety).
- `docs/why-tdd-first.md` — full argument.
- Willison, *Red/green TDD* — the canonical pattern.
- SWE-Bench grading — F2P/P2P split.
- Lightman et al. 2023, *Let's Verify Step by Step* — process supervision.
- Hubinger et al. 2024, *Sleeper Agents* — why no LLM self-judgment.
