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

	"github.com/dillonvuong/maxwell/internal/gitsubstrate"
	"github.com/dillonvuong/maxwell/internal/reflections"
	"github.com/dillonvuong/maxwell/internal/transcripts"
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
// Spec §7.3 (severity-aware abort) and §15.1.
func (r *Runner) Run() Verdict {
	gateRuns := make([]GateRun, 0, len(r.cfg.LoadedGates))
	var p0Failed bool
gateLoop:
	for _, gate := range r.cfg.LoadedGates {
		gr := r.RunGate(gate)
		gateRuns = append(gateRuns, gr)
		if gr.Verdict.Value == VerdictNO && gate.Severity == SeverityP0 {
			p0Failed = true
			fmt.Fprintf(os.Stderr, "maxwell: aborting subsequent gates (P0 NO from %q)\n", gate.Name)
			break gateLoop
		}
	}
	_ = p0Failed

	final := Aggregate(gateRuns)
	r.transcript.Verdict = string(final.Value)
	r.transcript.VerdictSource = final.VerdictSource
	r.transcript.JudgeKind = string(final.JudgeKind)
	r.transcript.WallClockEnd = time.Now().UTC()
	if err := r.transcript.Finalize(); err != nil {
		// Spec §13.8: a FULL verdict MUST be reproducible from the published
		// transcript. If we cannot persist the transcript, demote the verdict
		// rather than silently emit a green that no third party can audit.
		fmt.Fprintf(os.Stderr, "maxwell: transcript finalize failed: %v\n", err)
		if final.Value == VerdictFULL || final.Value == VerdictPARTIAL {
			final.Value = VerdictERROR
			final.Reason = fmt.Sprintf("transcript finalize failed: %v", err)
			final.VerdictSource = "transcript_failure"
		}
	}

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
			fmt.Fprintf(os.Stderr, "maxwell: write reflection: %v\n", err)
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
	// Spec §13.5 (grader isolation): scrub library-injection vectors so the
	// agent under test cannot intercept gate-script subprocess execution by
	// planting a shared library and pointing LD_PRELOAD at it. PATH stays
	// because gate scripts must locate git/go/etc.; future versions SHOULD
	// pin a workspace-PATH allowlist (see docs/maintenance.md quarterly
	// review). Defense in depth: workspace also has Read(./skills/**) +
	// Write(./skills/**) deny rules in the Claude Code permission grammar.
	cmd.Env = append(scrubInjectionEnv(os.Environ()),
		"MAXWELL_TASK_ID="+r.taskID,
		"MAXWELL_BASE_REF="+r.baseRef(),
		"MAXWELL_PROD_CODE_GLOBS="+strings.Join(r.cfg.Workspace.ProdCodeGlobs, ":"),
		"MAXWELL_TEST_GLOBS="+strings.Join(r.cfg.Workspace.TestGlobs, ":"),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		// Spec §8.1: timeout MUST yield ERROR, not NO.
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			exitCode = 3
		} else {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exitCode = exitErr.ExitCode()
			} else {
				// Subprocess could not be launched / killed by signal.
				// Treat as ERROR; the agent isn't responsible.
				exitCode = 3
			}
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

// baseRef returns the git ref to diff against. Spec §15.3: the TDD gate
// must compare HEAD against the merge-base with main/master, not HEAD~1.
// Diffing against HEAD~1 is the gate-inverting bug from the v0.1 review:
// a properly disciplined red-then-green commit sequence (test in HEAD~1,
// prod in HEAD) would *fail* check_red because HEAD~1 already contains
// the test.
// scrubInjectionEnv removes LD_PRELOAD, LD_LIBRARY_PATH, DYLD_*, and any
// other shared-library-injection vectors from the inherited env before
// passing it to gate-script subprocesses. Spec §13.5.
//
// This is partial defense. A complete grader-isolation implementation
// would also pin PATH to a workspace-independent allowlist and run gates
// in a sandbox (Docker/firejail/macOS sandbox-exec). Tracked for v0.2.
func scrubInjectionEnv(env []string) []string {
	dropPrefixes := []string{
		"LD_PRELOAD=",
		"LD_LIBRARY_PATH=",
		"LD_AUDIT=",
		"LD_BIND_NOW=",
		"DYLD_INSERT_LIBRARIES=",
		"DYLD_LIBRARY_PATH=",
		"DYLD_FRAMEWORK_PATH=",
		"DYLD_FALLBACK_LIBRARY_PATH=",
		"DYLD_FALLBACK_FRAMEWORK_PATH=",
		"DYLD_VERSIONED_LIBRARY_PATH=",
		"DYLD_VERSIONED_FRAMEWORK_PATH=",
		"DYLD_PRINT_LIBRARIES=",
		"GIT_EXEC_PATH=",
		"GIT_TEMPLATE_DIR=",
		"GIT_DIR=",
		"GIT_WORK_TREE=",
		"GIT_INDEX_FILE=",
	}
	out := make([]string, 0, len(env))
loop:
	for _, kv := range env {
		for _, p := range dropPrefixes {
			if strings.HasPrefix(kv, p) {
				continue loop
			}
		}
		out = append(out, kv)
	}
	return out
}

func (r *Runner) baseRef() string {
	if r.opts.BaseRef != "" {
		return r.opts.BaseRef
	}
	return gitsubstrate.BaseRef(r.repoRoot)
}

func nextAttemptNumber(repoRoot, taskID, gateName string) int {
	dir := filepath.Join(repoRoot, ".maxwell", "reflections", taskID, gateName)
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
