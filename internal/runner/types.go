// Package runner implements the gate dispatch and verdict aggregation
// behaviors normative in docs/spec.md §7, §8, §15.
//
// This package depends only on the Go standard library. The contract with
// gates is subprocess-shaped: gate scripts are any executable that emits
// an exit code and structured stderr. No language-runtime tax is imposed
// on the user's repository.
package runner

import "time"

// VerdictValue is one of FULL, PARTIAL, NO, ERROR.
type VerdictValue string

const (
	VerdictFULL    VerdictValue = "FULL"
	VerdictPARTIAL VerdictValue = "PARTIAL"
	VerdictNO      VerdictValue = "NO"
	VerdictERROR   VerdictValue = "ERROR"
)

// Severity is one of P0, P1, P2 per spec §4.1.1.
type Severity string

const (
	SeverityP0 Severity = "P0"
	SeverityP1 Severity = "P1"
	SeverityP2 Severity = "P2"
)

// JudgeKind discriminates deterministic from llm_advisory verdict sources.
// Per spec §13.1, primary verdicts MUST be deterministic.
type JudgeKind string

const (
	JudgeDeterministic JudgeKind = "deterministic"
	JudgeLLMAdvisory   JudgeKind = "llm_advisory"
)

// Verdict is the aggregate result of running a Gate's scripts.
// Spec §4.1.2.
type Verdict struct {
	Value         VerdictValue `json:"value"`
	F2P           float64      `json:"f2p"`
	P2P           float64      `json:"p2p"`
	Reason        string       `json:"reason"`
	VerdictSource string       `json:"verdict_source"`
	JudgeKind     JudgeKind    `json:"judge_kind"`
	Evidence      []Evidence   `json:"evidence"`
}

// Evidence captures one script's output for verdict aggregation and reflection.
type Evidence struct {
	Script   string `json:"script"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	DurationMS int64 `json:"duration_ms"`
}

// Gate is a deterministic external verifier over agent output.
// Spec §4.1.1.
type Gate struct {
	Name               string   `yaml:"name"`
	Description        string   `yaml:"description"`
	Severity           Severity `yaml:"severity"`
	RatchetSpecVersion string   `yaml:"ratchet_spec_version"`
	Assumptions        []string `yaml:"assumptions"`
	// Scripts is the ordered list of executables under skills/<name>/scripts/.
	// Populated by LoadSkill from the directory, sorted alphabetically.
	Scripts []string `yaml:"-"`
	// SkillRoot is the absolute path to the skill bundle directory.
	SkillRoot string `yaml:"-"`
}

// GateRun captures one execution of one gate against one task attempt.
// Spec §4.1.7.
type GateRun struct {
	GateName       string    `json:"gate_name"`
	TaskID         string    `json:"task_id"`
	AttemptNumber  int       `json:"attempt_number"`
	StartedAt      time.Time `json:"started_at"`
	EndedAt        time.Time `json:"ended_at"`
	Verdict        Verdict   `json:"verdict"`
	EvidencePath   string    `json:"evidence_path,omitempty"`
	ReflectionPath string    `json:"reflection_path,omitempty"`
}

// HookConfig describes a single host-hook event handler.
type HookConfig struct {
	Matcher    string `yaml:"matcher"`
	Command    string `yaml:"command"`
	TimeoutSec int    `yaml:"timeoutSec"`
}

// Permissions models the host permission grammar (spec §10.2).
type Permissions struct {
	Mode  string   `yaml:"mode"`
	Allow []string `yaml:"allow"`
	Deny  []string `yaml:"deny"`
}

// WorkspaceConfig governs per-task workspace behavior. Spec §11.
type WorkspaceConfig struct {
	Isolation       string   `yaml:"isolation"`
	Cleanup         string   `yaml:"cleanup"`
	ProdCodeGlobs   []string `yaml:"prod_code_globs"`
	TestGlobs       []string `yaml:"test_globs"`
}

// RunnerConfig controls timeouts, flake budget, retry behavior. Spec §6, §8.5.
type RunnerConfig struct {
	TimeoutMS             int     `yaml:"timeout_ms"`
	FlakeBudgetPct        float64 `yaml:"flake_budget_pct"`
	MaxReflectionsPerGate int     `yaml:"max_reflections_per_gate"`
}

// Observability flags transcript and CoT preservation. Spec §13.3, §15.
type Observability struct {
	Transcripts            string `yaml:"transcripts"`
	CoTPreservation        string `yaml:"cot_preservation"`
	EvalAwarenessRecording string `yaml:"eval_awareness_recording"`
}

// Config is the typed view of ratchet.md's frontmatter.
type Config struct {
	RatchetSpecVersion string                  `yaml:"ratchet_spec_version"`
	VerdictModel       string                  `yaml:"verdict_model"`
	SelfJudgment       string                  `yaml:"self_judgment"`
	Gates              []GateRef               `yaml:"gates"`
	Hooks              map[string][]HookConfig `yaml:"hooks"`
	Runner             RunnerConfig            `yaml:"runner"`
	Workspace          WorkspaceConfig         `yaml:"workspace"`
	Observability      Observability           `yaml:"observability"`
	Permissions        Permissions             `yaml:"permissions"`

	// PromptBody is the Markdown body after the frontmatter.
	PromptBody string `yaml:"-"`

	// ConfigPath is where this config was loaded from.
	ConfigPath string `yaml:"-"`

	// LoadedGates is the resolved list of Gates (after walking skills/).
	LoadedGates []Gate `yaml:"-"`
}

// GateRef is the entry under `gates:` in ratchet.md frontmatter.
type GateRef struct {
	Name     string   `yaml:"name"`
	Severity Severity `yaml:"severity"`
}

// Options govern runner behavior at construction time.
type Options struct {
	JSONOut      bool
	TriggerEvent string
	BaseRef      string
}
