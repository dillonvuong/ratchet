# Why TDD is the canonical first gate

The choice of red/green/refactor TDD as ratchet's first gate is not aesthetic. It is forced by the constraints of the problem and the available evidence.

## The argument in one paragraph

We need a deterministic, externally-verifiable, hard-verdict gate over agent output. Tests are the cheapest and most reliable instrument we have for that purpose: a test runner is a black-box verifier whose verdict is binary, reproducible, and uncoupled from the model's self-report. Git provides a verdict substrate that is publicly auditable (`git show HEAD~1` shows the red, `git show HEAD` shows the green) and tamper-evident (test deletions are visible in the diff). The test runner verifies the work; git verifies the *order*. Together they produce a hard verdict at the cost of one subprocess invocation, with no LLM in the loop, in a way that no other gate primitive matches.

## The constraints that force TDD

### 1. Self-reports are unreliable (Sleeper Agents)

A model that has learned to hide misbehavior cannot be relied upon to grade its own output. Hubinger et al. 2024: "Standard techniques could fail to remove such deception and create a false impression of safety." Any gate that consults the model under test for the verdict is, in the limit, fooled. Test runners are not.

### 2. Process supervision beats outcome supervision (Lightman et al.)

"Let's Verify Step by Step" (2023) shows process-based reward models outperform outcome-based on hard reasoning tasks. The harness analog: per-step gates beat per-PR gates because they catch errors at the smallest reachable scope. TDD is the smallest meaningful step in code work — a single test going red, then green.

### 3. Determinism is the noise floor (InstructGPT, DeepSeek-R1)

InstructGPT reports 72.6% inter-labeler agreement on subjective tasks. That is the noise floor for human judgment. Any gate whose verdict is reproducible at less than 73% across re-runs is broken. Test runners are typically deterministic at six nines or better given fixed inputs and seed; they are the rare gate primitive that lives well above the noise floor.

DeepSeek-R1 §2.2.2 is the contemporary verdict: "the neural reward model may suffer from reward hacking in the large-scale reinforcement learning process… retraining the reward model needs additional training resources and it complicates the whole training pipeline." DeepSeek explicitly chose deterministic rule-based rewards. ratchet's first gate inherits that choice.

### 4. Git topology is publicly verifiable (Cherny intuition; ratchet implementation)

Cherny's framing of git as the system of record carries directly. The TDD gate's verdict is not "the agent says the test went red then green"; it is "git history shows the test added in commit N failing, the test in commit N+1 passing, and no test was deleted." Anyone can re-run the assertion from the public log. The harness does not have to be trusted — only `git`.

### 5. Tests as spec, immutable (Anthropic + Willison)

"It is unacceptable to remove or edit tests" appears in both Anthropic's prompting documentation and Anthropic's *Effective Harnesses for Long-Running Agents* (Nov 2025). Willison: tests are "a robust automated test suite that protects against future regressions." Together they justify the third TDD sub-gate: test-immutability.

The asymmetry is deliberate. F2P failure (the new test does not pass) is recoverable: the agent has more work to do. P2P failure (an existing test is broken) is unforgivable: the agent broke something that worked. Test-deletion failure is malfeasance: the agent altered the spec to make the verdict pass. We collapse all three to the same `NO` verdict, but with distinguishable evidence so reflections can prescribe different next actions.

## The verdict pipeline

Three sub-gates compose into the TDD verdict. Each emits an exit code and a structured stderr line.

### Sub-gate 1: `check_red`

```
For each prod-code-glob path modified in the current turn:
  If no new test added in test-glob paths whose execution
  exercises the modified production symbol:
    Verdict: NO ("red phase missing")

For each new test added:
  Run that test against the base-ref version of the codebase.
  If it passes:
    Verdict: NO ("test does not exercise new behavior")
```

Substrate: `git diff base..HEAD -- <test-globs>` to enumerate added tests; `git stash` + `git checkout base` + run-test + `git checkout HEAD` + `git stash pop` to verify red at base.

### Sub-gate 2: `check_green`

```
Run F2P tests at HEAD.
Run P2P tests at HEAD.

f2p_score = passed_in_f2p / total_f2p
p2p_score = passed_in_p2p / total_p2p

Verdict per spec §8.1:
  FULL    iff f2p_score == 1.0 AND p2p_score == 1.0
  PARTIAL iff 0 < f2p_score < 1.0 AND p2p_score == 1.0
  NO      otherwise
```

### Sub-gate 3: `check_immutable`

```
For each test file in test-glob paths:
  If git diff base..HEAD shows deletion or modification of
  test functions/assertions:
    Verdict: NO ("test deletion forbidden")
  Exception: tests added in this turn MAY be edited within
  this turn (the agent is iterating on its own new test).
```

## What this gate does NOT do

The TDD gate does not enforce coding style, architecture choice, or the "right" test framework. It does not measure code coverage, complexity, or aesthetic. Each of those is a separate gate (or future gate) and each has its own justification.

The TDD gate also does not generate tests. The agent is responsible for producing tests; ratchet only verifies that the tests have the right topology in git. This is by design: per Sutton, "the contents of minds are tremendously, irredeemably complex" and we should not encode our taxonomy of "good tests" into the runner. We only encode the meta-rule: tests precede code, tests survive, F2P transitions exist.

## Why this is the *first* gate, not the only gate

Subsequent gates layered on top:

- **`lint`** — language-specific linter (ruff, golangci-lint, eslint). Deterministic exit code; trivial to wrap.
- **`typecheck`** — static type checker. Deterministic; straightforward F2P/P2P semantics for type errors.
- **`security`** — bandit, gosec, trivy. Deterministic; severity thresholds map to P0/P1/P2.
- **`coverage-floor`** — coverage report comparison vs baseline. Deterministic; asymmetric (coverage MUST NOT decrease).
- **`evaluator-agent` (advisory)** — LLM-judge that reviews the diff. `judge_kind: llm_advisory`. Per Toolformer, its suggestions are kept only if they would change another deterministic gate's verdict. Never a primary verdict per §13.1.

Each of these is justified separately and gated by `ratchet_spec_version`. None is required for TDD to ship.

## What we deliberately defer

- **Multi-agent debate gate** (Irving et al. 2018). Reserved namespace `gates.debate.*`. Activation criteria in spec §13 commentary.
- **Interpretability probe gate** (Anthropic Activation Oracles, Dec 2025). Reserved `gates.interp.*`. Awaits field-validated transfer of deception probes.
- **Sabotage / monitorability gate** (AISI). Reserved `gates.sabotage.*`.
- **Long-horizon gate** (METR). Reserved `gates.horizon.*`.

## References

See `docs/references.md` for the full bibliography. The load-bearing ones for this document are Willison's *Red/green TDD*, Lightman et al.'s *Let's Verify Step by Step*, Hubinger et al.'s *Sleeper Agents*, DeepSeek-R1, the SWE-Bench grading paper, Sutton's *Bitter Lesson*, and Anthropic's *Effective Harnesses for Long-Running Agents*.
