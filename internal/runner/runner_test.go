package runner

import "testing"

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
