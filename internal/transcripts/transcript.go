// Package transcripts implements the per-run structured transcript
// contract from spec §15.2. Transcripts are aligned with METR/AISI
// run-artifact requirements so longitudinal analysis (time-horizon,
// eval-awareness, reward-hacking detection) becomes possible without
// re-running.
//
// Required fields are enumerated in spec §15.2; the JSON tags below
// match the spec exactly.
package transcripts

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// GateRunRecord is the per-gate record embedded in a Transcript.
type GateRunRecord struct {
	GateName      string    `json:"gate_name"`
	AttemptNumber int       `json:"attempt_number"`
	StartedAt     time.Time `json:"started_at"`
	EndedAt       time.Time `json:"ended_at"`
	Verdict       string    `json:"verdict"`
	Reason        string    `json:"reason"`
	ScriptCount   int       `json:"script_count"`
}

// Transcript captures a complete run for downstream analysis.
type Transcript struct {
	TaskID                string `json:"task_id"`
	TaskFamily            string `json:"task_family,omitempty"`
	HumanBaselineMinutes  *int   `json:"human_baseline_minutes"`
	WallClockStart        time.Time `json:"wall_clock_seconds_start"`
	WallClockEnd          time.Time `json:"wall_clock_seconds_end"`
	WallClockDurationSec  float64   `json:"wall_clock_duration_seconds"`
	TurnCount             int       `json:"turn_count"`
	ToolCallCount         int       `json:"tool_call_count"`
	TokensInput           int       `json:"tokens_input"`
	TokensOutput          int       `json:"tokens_output"`
	TokensThinking        *int      `json:"tokens_thinking"`
	ModelID               string    `json:"model_id"`
	ModelVersion          string    `json:"model_version"`
	HarnessVersion        string    `json:"harness_version"`
	ScaffoldConfigHash    string    `json:"scaffold_config_hash"`
	MaxwellSpecVersion    string    `json:"maxwell_spec_version"`
	Temperature           *float64  `json:"temperature"`
	Seed                  *int      `json:"seed"`
	Verdict               string    `json:"verdict"`
	VerdictSource         string    `json:"verdict_source"`
	JudgeKind             string    `json:"judge_kind"`
	SuccessThresholdUsed  float64   `json:"success_threshold_used"`
	GateRuns              []GateRunRecord `json:"gate_runs"`
	Transcript            []TranscriptEvent `json:"transcript"`
	EvalAwarenessFlags    []string  `json:"eval_awareness_flags"`
	HoneypotSubset        bool      `json:"honeypot_subset"`
	SandboxEscapeAttempted bool     `json:"sandbox_escape_attempted"`
	SafetyCaseID          string    `json:"safety_case_id,omitempty"`

	// internal
	repoRoot string
	runID    string
}

// TranscriptEvent captures one agent ↔ environment exchange. Spec §15.2.
// CoTOrScratchpad MUST be a separate field per §13.3 — never merged into Content.
type TranscriptEvent struct {
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	CoTOrScratchpad string   `json:"cot_or_scratchpad,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	ToolsInvoked   []string  `json:"tools_invoked,omitempty"`
}

// New creates a Transcript with default values and a random run-id.
func New(taskID, repoRoot string) *Transcript {
	return &Transcript{
		TaskID:               taskID,
		WallClockStart:       time.Now().UTC(),
		HarnessVersion:       "maxwell-0.1.0-alpha",
		MaxwellSpecVersion:   "0.1",
		Verdict:              "FULL",
		VerdictSource:        "aggregate",
		JudgeKind:            "deterministic",
		SuccessThresholdUsed: 1.0,
		EvalAwarenessFlags:   []string{},
		repoRoot:             repoRoot,
		runID:                randID(),
	}
}

// AddGateRun appends a per-gate record.
func (t *Transcript) AddGateRun(g GateRunRecord) {
	t.GateRuns = append(t.GateRuns, g)
}

// AddEvent appends an agent ↔ environment exchange.
func (t *Transcript) AddEvent(e TranscriptEvent) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	t.Transcript = append(t.Transcript, e)
}

// Finalize writes the transcript to .maxwell/transcripts/<task-id>/<run-id>.json.
func (t *Transcript) Finalize() error {
	if t.WallClockEnd.IsZero() {
		t.WallClockEnd = time.Now().UTC()
	}
	t.WallClockDurationSec = t.WallClockEnd.Sub(t.WallClockStart).Seconds()

	dir := filepath.Join(t.repoRoot, ".maxwell", "transcripts", t.TaskID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, t.runID+".json")
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func randID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return time.Now().UTC().Format("20060102T150405") + "-" + hex.EncodeToString(b)
}
