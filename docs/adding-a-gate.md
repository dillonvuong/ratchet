# Adding a Gate

This is the practical tutorial. The normative contract for skill bundles lives in `docs/spec.md` §5.4 and §16.

## The shape of a gate

A gate is a directory under `skills/` containing a `SKILL.md` and a `scripts/` folder.

```
skills/<gate-name>/
├── SKILL.md
└── scripts/
    └── <executable>
```

The directory name, the `name:` field in `SKILL.md`, and the parent reference in `ratchet.md`'s `gates:` list MUST all match.

## Step 1: write the SKILL.md

```markdown
---
name: lint-go
description: Runs golangci-lint on changed .go files; verdict from exit code.
severity: P1
ratchet_spec_version: "0.1"
assumptions:
  - "The model can produce code that compiles but fails standard idiomatic checks."
  - "The model can introduce dead code or unused imports without flagging."
---

# lint-go

This gate runs `golangci-lint run` on the working tree. Verdict from exit code.

## F2P / P2P semantics

This gate has no F2P (it is not a behavior verifier; it is a style verifier).
P2P is implicit: if golangci-lint passed before the change, it must pass after.

## Severity

P1: a lint failure aborts the run but the agent may retry after addressing
the findings.

## Stderr format

stderr is a single line:
  status=<pass|fail> reason=<short description>

Detailed findings are emitted on stdout in golangci-lint's default format.
```

The frontmatter `assumptions:` field is REQUIRED per spec §4.1.1 and is consulted during quarterly review. Be specific. "Catches all bugs" is wrong; "catches the model's tendency to leave unused imports" is right.

## Step 2: write the script

`scripts/check.sh` (or `check.ps1`, `check.py`, `check.go-binary` — anything executable):

```bash
#!/usr/bin/env bash
set -euo pipefail

# ratchet passes the workspace path as the first argument.
# Additional context is in env vars: RATCHET_TASK_ID, RATCHET_BASE_REF.

cd "$1"

if golangci-lint run --new-from-rev="${RATCHET_BASE_REF:-HEAD~1}" 2>&1; then
  echo "status=pass reason=lint-clean" >&2
  exit 0
else
  echo "status=fail reason=golangci-lint findings on changed files" >&2
  exit 1
fi
```

Exit codes:

- `0` — `FULL` (gate passes).
- `1` — `NO` (gate fails).
- `2` — `PARTIAL` (gate partially passes; rare for boolean gates).
- `3` — `ERROR` (infrastructure / runner error; not the agent's fault).

Any other non-zero is treated as `NO`.

## Step 3: register the gate in ratchet.md

```yaml
gates:
  - name: tdd-red-green-refactor
    severity: P0
  - name: lint-go              # ← add here
    severity: P1
```

Order matters. Cheap deterministic gates first. The TDD gate is `P0` and should always be first; lint, typecheck, coverage gates follow as `P1`; advisory gates last as `P2`.

## Step 4: add conformance tests

Per spec §16, every gate that ships a non-trivial verdict computation MUST have tests proving:

- Determinism: same inputs → same verdict.
- Idempotence: the gate does not modify the workspace.
- Reason-string stability: failure reasons are not random.

Place tests at `skills/<gate-name>/test/` or in your normal test directory. The runner's self-test (§17.4) will pick them up.

## Step 5: run ratchet on ratchet

```
$ ratchet run --gate=lint-go
[lint-go] status=pass reason=lint-clean (P1, FULL, 1.2s)
```

If the gate fails, a reflection is written. Inspect:

```
$ cat .ratchet/reflections/<task-id>/lint-go/1.md
```

## F2P / P2P pattern (worked example)

For behavior-verifier gates (TDD-shaped, integration-test-shaped, security-test-shaped), use the F2P/P2P pattern from SWE-Bench grading:

```yaml
# In a task definition (per spec §4.1.6)
task:
  id: TASK-001
  title: "Fix expired-token bypass"
  f2p:
    - tests/auth_test.go::TestLoginRejectsExpiredToken
  p2p:
    - tests/auth_test.go::TestLoginAcceptsValidToken
    - tests/session_test.go::*
  prod_code_globs: ["src/auth/**"]
  test_globs: ["tests/auth*", "tests/session*"]
```

The gate verdict is then:

- `FULL` iff all f2p tests pass AND all p2p tests pass.
- `PARTIAL` iff some f2p tests pass AND all p2p tests pass.
- `NO` if any p2p test fails (regression) regardless of f2p.

## Anti-patterns

The following are forbidden by spec §13.9 and will fail review:

- A gate that prescribes code structure ("functions must be ≤ 50 lines"). This violates the Bitter Lesson; it bakes a heuristic into the runner.
- A gate that consults the model under test for its verdict. This violates §13.1.
- A gate whose verdict depends on a learned reward model. This violates DeepSeek-R1's empirical guidance.
- A gate that modifies the agent's workspace. This violates the idempotence requirement.
- A gate whose stderr leaks secrets, tokens, or environment variables. This violates §14.1.

## Naming conventions

Per Lopopolo's repo at OpenAI Frontier (6 skills total):

- Hyphenated lowercase: `tdd-red-green-refactor`, `lint-go`, `coverage-floor`.
- Verb-first when describing an action: `check-immutability`, `enforce-imports`.
- Noun-first when describing a domain: `security-bandit`, `coverage-go`.

Avoid:

- Vendor-specific names that lock to a host (`claude-tdd`, `codex-lint`). Gates are portable.
- Cute names without semantic content (`tdd-thunder`). Future-you will not remember.

## Versioning

Each gate pins `ratchet_spec_version: "0.1"`. When the spec version bumps, gates MUST be reviewed. Spec version compatibility per §4.2: a gate pinned to `"0.1"` runs on any `0.1.x` runner; pinning to `"0.2"` requires a runner at `0.2.0` or later.

When deprecating a gate (per `docs/maintenance.md`), update the SKILL.md frontmatter:

```yaml
---
name: lint-go
description: (DEPRECATED — superseded by lint-go-v2; see RFC-007)
deprecated_since: "0.4.0"
removed_after: "0.6.0"
---
```

The runner will warn but continue to run the gate until `removed_after`.
