package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dillon-vuong/ratchet/internal/reflections"
	"github.com/dillon-vuong/ratchet/internal/transcripts"
)

// Runner orchestrates gate dispatch for one task attempt.
// Spec §3.1, §15.1.
type Runner struct {
	cfg        *Config
	taskID     string
	opts       Options
	repoRoot   string
	transcript *transcripts.Transcript
	startedAt  time.Time
}

// New constructs a Runner. The repo root is derived from the config path.
func New(cfg *Config, taskID string, opts Options) (*Runner, error) {
	if cfg == nil {
		return nil, errors.New("nil config")
	}
	repoRoot := filepath.Dir(cfg.ConfigPath)
	if !filepath.IsAbs(repoRoot) {
		var err error
		repoRoot, err = filepath.Abs(repoRoot)
		if err != nil {
			return nil, err
		}
	}
	t := transcripts.New(taskID, repoRoot)
	return &Runner{
		cfg:        cfg,
		taskID:     taskID,
		opts:       opts,
		repoRoot:   repoRoot,
		transcript: t,
		startedAt:  time.Now().UTC(),
	}, nil
}

// Run dispatches all declared gates in order and aggregates the verdict.
// Spec §15.1.
func (r *Runner) Run() Verdict {
	gateRuns := make([]GateRun, 0, len(r.cfg.LoadedGates))
	var seenNO, seenERROR, seenPARTIAL bool
	for _, gate := range r.cfg.LoadedGates {
		if gate.Severity == SeverityP0 && seenNO {
			fmt.Fprintf(os.Stderr, "ratchet: skipping gate %q (preceded by P0 NO verdict)\n", gate.Name)
			continue
		}
		gr := r.RunGate(gate)
		gateRuns = append(gateRuns, gr)
		switch gr.Verdict.Value {
		case VerdictNO:
			seenNO = true
			if gate.Severity == SeverityP0 {
				// Abort subsequent gates per §7.3
				break
			}
		case VerdictERROR:
			seenERROR = true
		case VerdictPARTIAL:
			seenPARTIAL = true
		}
	}

	final := Aggregate(gateRuns)
	r.transcript.Verdict = string(final.Value)
	r.transcript.VerdictSource = final.VerdictSource
	r.transcript.JudgeKind = string(final.JudgeKind)
	r.transcript.WallClockEnd = time.Now().UTC()
	if err := r.transcript.Finalize(); err != nil {
		fmt.Fprintf(os.Stderr, "ratchet: transcript finalize failed: %v\n", err)
	}

	_ = seenERROR
	_ = seenPARTIAL
	return final
}

// RunGate executes one gate's scripts in order and aggregates a per-gate verdict.
func (r *Runner) RunGate(gate Gate) GateRun {
	attempt := nextAttemptNumber(r.repoRoot, r.taskID, gate.Name)
	gr := GateRun{
		GateName:      gate.Name,
		TaskID:        r.taskID,
		AttemptNumber: attempt,
		StartedAt:     time.Now().UTC(),
	}

	verdict := Verdict{
		Value:         VerdictFULL,
		F2P:           1.0,
		P2P:           1.0,
		VerdictSource: "subprocess_exit_code",
		JudgeKind:     JudgeDeterministic,
	}
	for _, script := range gate.Scripts {
		ev := r.runScript(script, gate)
		verdict.Evidence = append(verdict.Evidence, ev)
		gradeFromExit(&verdict, ev)
		if verdict.Value == VerdictNO || verdict.Value == VerdictERROR {
			break
		}
	}

	gr.Verdict = verdict
	gr.EndedAt = time.Now().UTC()

	if verdict.Value != VerdictFULL {
		path, err := reflections.Write(r.repoRoot, r.taskID, gate.Name, attempt, reflections.Reflection{
			Gate:                gate.Name,
			Verdict:             string(verdict.Value),
			AttemptNumber:       attempt,
			Observation:         buildObservation(verdict),
			SuggestedNextAction: buildSuggestion(gate, verdict),
			CreatedAt:           time.Now().UTC(),
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "ratchet: write reflection: %v\n", err)
		} else {
			gr.ReflectionPath = path
		}
	}

	r.transcript.AddGateRun(transcripts.GateRunRecord{
		GateName:      gate.Name,
		AttemptNumber: attempt,
		StartedAt:     gr.StartedAt,
		EndedAt:       gr.EndedAt,
		Verdict:       string(verdict.Value),
		Reason:        verdict.Reason,
		ScriptCount:   len(verdict.Evidence),
	})
	return gr
}

// runScript executes one gate script with the spec-mandated env vars and
// captures stdout, stderr, and exit code.
func (r *Runner) runScript(script string, gate Gate) Evidence {
	timeout := time.Duration(r.cfg.Runner.TimeoutMS) * time.Millisecond
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	startedAt := time.Now()
	cmd := exec.CommandContext(ctx, script, r.repoRoot)
	cmd.Dir = r.repoRoot
	cmd.Env = append(os.Environ(),
		"RATCHET_TASK_ID="+r.taskID,
		"RATCHET_BASE_REF="+r.baseRef(),
		"RATCHET_PROD_CODE_GLOBS="+strings.Join(r.cfg.Workspace.ProdCodeGlobs, ":"),
		"RATCHET_TEST_GLOBS="+strings.Join(r.cfg.Workspace.TestGlobs, ":"),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return Evidence{
		Script:     filepath.Base(script),
		ExitCode:   exitCode,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		DurationMS: time.Since(startedAt).Milliseconds(),
	}
}

// gradeFromExit translates a script's exit code into a verdict update.
// Convention defined in skills/tdd-red-green-refactor/SKILL.md and spec §16.7.
//   0 → FULL (no change to current verdict)
//   1 → NO
//   2 → PARTIAL
//   3 → ERROR
//   other → NO
func gradeFromExit(v *Verdict, ev Evidence) {
	switch ev.ExitCode {
	case 0:
		// No-op; verdict stays at FULL or whatever a prior script set.
	case 2:
		if v.Value == VerdictFULL {
			v.Value = VerdictPARTIAL
		}
		v.F2P = 0.5
		v.Reason = extractReason(ev.Stderr, "PARTIAL verdict from "+ev.Script)
	case 3:
		v.Value = VerdictERROR
		v.Reason = extractReason(ev.Stderr, "ERROR from "+ev.Script)
	default:
		v.Value = VerdictNO
		v.F2P = 0.0
		// P2P stays at 1.0 unless the script signals a regression in stderr.
		if strings.Contains(ev.Stderr, "p2p=") {
			v.P2P = parseStderrFloat(ev.Stderr, "p2p=", 1.0)
		} else {
			// Conservative: a NO from an unspecified script keeps P2P at
			// its prior value (the runner cannot infer a regression).
		}
		v.Reason = extractReason(ev.Stderr, "NO verdict from "+ev.Script)
	}
}

func extractReason(stderr, fallback string) string {
	for _, line := range strings.Split(stderr, "\n") {
		if i := strings.Index(line, "reason="); i >= 0 {
			return strings.TrimSpace(line[i+len("reason="):])
		}
	}
	return fallback
}

func parseStderrFloat(stderr, key string, fallback float64) float64 {
	for _, line := range strings.Split(stderr, "\n") {
		idx := strings.Index(line, key)
		if idx < 0 {
			continue
		}
		val := line[idx+len(key):]
		val = strings.SplitN(val, " ", 2)[0]
		var f float64
		if _, err := fmt.Sscanf(val, "%f", &f); err == nil {
			return f
		}
	}
	return fallback
}

// Aggregate composes per-gate verdicts into a run-level verdict.
// Spec §8.2.
func Aggregate(runs []GateRun) Verdict {
	final := Verdict{
		Value:         VerdictFULL,
		F2P:           1.0,
		P2P:           1.0,
		VerdictSource: "aggregate",
		JudgeKind:     JudgeDeterministic,
	}
	for _, gr := range runs {
		switch gr.Verdict.Value {
		case VerdictNO:
			final.Value = VerdictNO
			final.F2P = gr.Verdict.F2P
			final.P2P = gr.Verdict.P2P
			final.Reason = gr.Verdict.Reason
			return final
		case VerdictERROR:
			if final.Value == VerdictFULL {
				final.Value = VerdictERROR
				final.Reason = gr.Verdict.Reason
			}
		case VerdictPARTIAL:
			if final.Value == VerdictFULL {
				final.Value = VerdictPARTIAL
				final.F2P = gr.Verdict.F2P
				final.Reason = gr.Verdict.Reason
			}
		}
	}
	return final
}

func (r *Runner) baseRef() string {
	if r.opts.BaseRef != "" {
		return r.opts.BaseRef
	}
	return "HEAD~1"
}

func nextAttemptNumber(repoRoot, taskID, gateName string) int {
	dir := filepath.Join(repoRoot, ".ratchet", "reflections", taskID, gateName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 1
	}
	return len(entries) + 1
}

func buildObservation(v Verdict) string {
	if len(v.Evidence) == 0 {
		return v.Reason
	}
	var b strings.Builder
	for _, ev := range v.Evidence {
		fmt.Fprintf(&b, "Script %s exited %d after %dms.\n", ev.Script, ev.ExitCode, ev.DurationMS)
		if ev.Stderr != "" {
			fmt.Fprintf(&b, "stderr: %s\n", strings.TrimSpace(ev.Stderr))
		}
	}
	return b.String()
}

func buildSuggestion(gate Gate, v Verdict) string {
	switch v.Value {
	case VerdictNO:
		if strings.Contains(v.Reason, "red-phase-missing") {
			return "Add a failing test for the production-code edit. Confirm it fails at the base reference before re-running the gate."
		}
		if strings.Contains(v.Reason, "test-file-deleted") || strings.Contains(v.Reason, "test-weakened") {
			return "Restore the deleted/weakened tests. Tests are spec; modifying them invalidates the verdict."
		}
		if strings.Contains(v.Reason, "tests-failing") {
			return "Run the test suite and address failures. P2P regression is unrecoverable in this verdict; F2P partial is recoverable."
		}
		return fmt.Sprintf("Address the failing condition in gate %q and retry.", gate.Name)
	case VerdictPARTIAL:
		return "Some F2P criteria pass; remaining criteria do not. Inspect the failing tests and add or fix them. P2P is intact."
	case VerdictERROR:
		return "Infrastructure or runner error. Re-run; if persistent, investigate the gate scripts or sandbox configuration."
	}
	return ""
}
