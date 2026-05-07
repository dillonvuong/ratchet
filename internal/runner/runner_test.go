package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestAggregate_AllFull(t *testing.T) {
	runs := []GateRun{
		{Verdict: Verdict{Value: VerdictFULL, F2P: 1.0, P2P: 1.0}},
		{Verdict: Verdict{Value: VerdictFULL, F2P: 1.0, P2P: 1.0}},
	}
	v := Aggregate(runs)
	if v.Value != VerdictFULL {
		t.Errorf("expected FULL, got %v", v.Value)
	}
}

func TestAggregate_AnyNoCollapses(t *testing.T) {
	runs := []GateRun{
		{Verdict: Verdict{Value: VerdictFULL, F2P: 1.0, P2P: 1.0}},
		{Verdict: Verdict{Value: VerdictNO, F2P: 0.0, P2P: 0.5, Reason: "regression"}},
		{Verdict: Verdict{Value: VerdictFULL, F2P: 1.0, P2P: 1.0}},
	}
	v := Aggregate(runs)
	if v.Value != VerdictNO {
		t.Errorf("expected NO, got %v", v.Value)
	}
	if v.Reason != "regression" {
		t.Errorf("expected reason='regression', got %q", v.Reason)
	}
}

func TestAggregate_PartialWithoutNo(t *testing.T) {
	runs := []GateRun{
		{Verdict: Verdict{Value: VerdictFULL, F2P: 1.0, P2P: 1.0}},
		{Verdict: Verdict{Value: VerdictPARTIAL, F2P: 0.5, P2P: 1.0}},
	}
	v := Aggregate(runs)
	if v.Value != VerdictPARTIAL {
		t.Errorf("expected PARTIAL, got %v", v.Value)
	}
}

func TestAggregate_ErrorWithoutNo(t *testing.T) {
	runs := []GateRun{
		{Verdict: Verdict{Value: VerdictFULL, F2P: 1.0, P2P: 1.0}},
		{Verdict: Verdict{Value: VerdictERROR, Reason: "infra"}},
	}
	v := Aggregate(runs)
	if v.Value != VerdictERROR {
		t.Errorf("expected ERROR, got %v", v.Value)
	}
}

func TestAggregate_NoBeatsError(t *testing.T) {
	runs := []GateRun{
		{Verdict: Verdict{Value: VerdictNO, Reason: "broken"}},
		{Verdict: Verdict{Value: VerdictERROR, Reason: "infra"}},
	}
	v := Aggregate(runs)
	if v.Value != VerdictNO {
		t.Errorf("NO must take precedence over ERROR; got %v", v.Value)
	}
}

func TestAggregate_DefaultIsDeterministic(t *testing.T) {
	v := Aggregate(nil)
	if v.JudgeKind != JudgeDeterministic {
		t.Errorf("default judge_kind must be deterministic, got %q", v.JudgeKind)
	}
}

func TestExtractReason(t *testing.T) {
	stderr := "status=fail reason=red-phase-missing prod_files_changed=2"
	got := extractReason(stderr, "fallback")
	if got != "red-phase-missing prod_files_changed=2" {
		t.Errorf("extractReason = %q", got)
	}
}

func TestExtractReason_Fallback(t *testing.T) {
	got := extractReason("no key=value here", "fb")
	if got != "fb" {
		t.Errorf("extractReason fallback = %q", got)
	}
}

func TestParseStderrFloat(t *testing.T) {
	stderr := "status=pass f2p=0.75 p2p=1.0"
	got := parseStderrFloat(stderr, "f2p=", -1.0)
	if got != 0.75 {
		t.Errorf("parseStderrFloat(f2p) = %v", got)
	}
	got = parseStderrFloat(stderr, "p2p=", -1.0)
	if got != 1.0 {
		t.Errorf("parseStderrFloat(p2p) = %v", got)
	}
}

// TestRun_P0AbortStopsSubsequentGates exercises the labeled-break fix
// (review finding #3): a P0 NO must abort all later gates, not just other
// P0 gates.
func TestRun_P0AbortStopsSubsequentGates(t *testing.T) {
	dir := t.TempDir()

	// Two gates: gate-a is P0 and will fail; gate-b is P1 and would
	// otherwise run.
	mkSkill := func(name string, exit int) {
		root := filepath.Join(dir, "skills", name, "scripts")
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		script := filepath.Join(root, "check.sh")
		body := fmt.Sprintf("#!/bin/sh\necho \"reason=test-script\" >&2\nexit %d\n", exit)
		if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
		md := fmt.Sprintf(`---
name: %s
description: test gate.
ratchet_spec_version: "0.1"
assumptions:
  - "test"
---
# body
`, name)
		if err := os.WriteFile(filepath.Join(dir, "skills", name, "SKILL.md"), []byte(md), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mkSkill("gate-a", 1) // NO
	mkSkill("gate-b", 0) // would be FULL

	ratchetMD := `---
ratchet_spec_version: "0.1"
verdict_model: hard
self_judgment: forbidden

gates:
  - name: gate-a
    severity: P0
  - name: gate-b
    severity: P1

runner:
  timeout_ms: 30000

workspace:
  prod_code_globs: ["src/**"]
  test_globs: ["tests/**"]
---
# body
`
	cfgPath := filepath.Join(dir, "ratchet.md")
	if err := os.WriteFile(cfgPath, []byte(ratchetMD), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(cfg, "task-x", Options{})
	if err != nil {
		t.Fatal(err)
	}
	final := r.Run()
	if final.Value != VerdictNO {
		t.Errorf("expected NO, got %v", final.Value)
	}
	// gate-b should NOT have produced a transcript entry.
	if len(r.transcript.GateRuns) != 1 {
		t.Errorf("expected 1 gate run (P0 abort), got %d", len(r.transcript.GateRuns))
	}
	if len(r.transcript.GateRuns) > 0 && r.transcript.GateRuns[0].GateName != "gate-a" {
		t.Errorf("expected gate-a to be the only run, got %q", r.transcript.GateRuns[0].GateName)
	}
}

// TestRunScript_TimeoutYieldsERROR defends spec §8.1: timeout MUST be ERROR,
// not NO.
func TestRunScript_TimeoutYieldsERROR(t *testing.T) {
	dir := t.TempDir()
	skillRoot := filepath.Join(dir, "skills", "slow", "scripts")
	if err := os.MkdirAll(skillRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	// A script that sleeps longer than the runner timeout.
	body := "#!/bin/sh\nsleep 5\n"
	script := filepath.Join(skillRoot, "check.sh")
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skills", "slow", "SKILL.md"), []byte(`---
name: slow
description: test gate.
ratchet_spec_version: "0.1"
assumptions:
  - "test"
---
# b
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(dir, "ratchet.md")
	if err := os.WriteFile(cfgPath, []byte(`---
ratchet_spec_version: "0.1"
verdict_model: hard
self_judgment: forbidden

gates:
  - name: slow
    severity: P1

runner:
  timeout_ms: 200

workspace:
  prod_code_globs: ["src/**"]
  test_globs: ["tests/**"]
---
# body
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(cfg, "task-x", Options{})
	if err != nil {
		t.Fatal(err)
	}
	gr := r.RunGate(cfg.LoadedGates[0])
	if gr.Verdict.Value != VerdictERROR {
		t.Errorf("expected ERROR on timeout, got %v (reason=%s)", gr.Verdict.Value, gr.Verdict.Reason)
	}
}
