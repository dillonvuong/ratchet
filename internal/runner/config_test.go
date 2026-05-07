package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSplitFrontmatter_Valid(t *testing.T) {
	in := "---\nname: foo\nseverity: P0\n---\n\n# Body\n"
	front, body, err := splitFrontmatter(in)
	if err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
	if front == "" {
		t.Fatalf("expected frontmatter, got empty")
	}
	if body == "" {
		t.Fatalf("expected body, got empty")
	}
}

func TestSplitFrontmatter_MissingLeadingDelimiter(t *testing.T) {
	_, _, err := splitFrontmatter("name: foo\n---\n")
	if err == nil {
		t.Fatal("expected error for missing leading delimiter")
	}
}

func TestSplitFrontmatter_MissingClosingDelimiter(t *testing.T) {
	_, _, err := splitFrontmatter("---\nname: foo\nbody\n")
	if err == nil {
		t.Fatal("expected error for missing closing delimiter")
	}
}

func TestSpecVersionsCompatible(t *testing.T) {
	cases := []struct {
		gate, runner string
		want         bool
	}{
		{"0.1", "0.1.0", true},
		{"0.1", "0.1.5", true},
		{"0.1.0", "0.1.5", true},
		{"0.2", "0.1.0", false},
		{"1.0", "0.1.0", false},
		{"0.1", "1.0.0", false},
	}
	for _, c := range cases {
		if got := specVersionsCompatible(c.gate, c.runner); got != c.want {
			t.Errorf("specVersionsCompatible(%q, %q) = %v, want %v", c.gate, c.runner, got, c.want)
		}
	}
}

func TestDecodeScalar(t *testing.T) {
	cases := []struct {
		in   string
		want interface{}
	}{
		{"true", true},
		{"false", false},
		{"null", nil},
		{"~", nil},
		{"42", 42},
		{"3.14", 3.14},
		{`"quoted"`, "quoted"},
		{"'single'", "single"},
		{"plain", "plain"},
	}
	for _, c := range cases {
		got := decodeScalar(c.in)
		if got != c.want {
			t.Errorf("decodeScalar(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestSanitize_Identity(t *testing.T) {
	// SanitizeID lives in gitsubstrate; this test asserts the rule it
	// implements (spec §11.3) is what runner expects.
	if isExecutableName("check.sh") != true {
		t.Error("expected .sh to be executable")
	}
	if isExecutableName("README.md") {
		t.Error("expected .md NOT to be executable")
	}
}

func TestLoadConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	skill := filepath.Join(dir, ".skills", "tdd-red-green-refactor", "scripts")
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "check.sh"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	skillMD := `---
name: tdd-red-green-refactor
description: First gate.
severity: P0
ratchet_spec_version: "0.1"
assumptions:
  - "Model can produce code without tests."
---
# Body
`
	if err := os.WriteFile(filepath.Join(dir, ".skills", "tdd-red-green-refactor", "SKILL.md"), []byte(skillMD), 0o644); err != nil {
		t.Fatal(err)
	}

	ratchetMD := `---
ratchet_spec_version: "0.1"
verdict_model: hard
self_judgment: forbidden

gates:
  - name: tdd-red-green-refactor
    severity: P0

runner:
  timeout_ms: 30000
  flake_budget_pct: 0.5

workspace:
  isolation: per-task
  cleanup: never
  prod_code_globs:
    - "src/**"
  test_globs:
    - "tests/**"
---

# doctrine

You are operating under ratchet.
`
	cfgPath := filepath.Join(dir, "ratchet.md")
	if err := os.WriteFile(cfgPath, []byte(ratchetMD), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.RatchetSpecVersion != "0.1" {
		t.Errorf("RatchetSpecVersion = %q", cfg.RatchetSpecVersion)
	}
	if cfg.VerdictModel != "hard" {
		t.Errorf("VerdictModel = %q", cfg.VerdictModel)
	}
	if cfg.SelfJudgment != "forbidden" {
		t.Errorf("SelfJudgment = %q", cfg.SelfJudgment)
	}
	if len(cfg.Gates) != 1 || cfg.Gates[0].Name != "tdd-red-green-refactor" {
		t.Errorf("Gates = %+v", cfg.Gates)
	}
	if cfg.Runner.TimeoutMS != 30000 {
		t.Errorf("Runner.TimeoutMS = %d", cfg.Runner.TimeoutMS)
	}
	if len(cfg.Workspace.ProdCodeGlobs) != 1 || cfg.Workspace.ProdCodeGlobs[0] != "src/**" {
		t.Errorf("Workspace.ProdCodeGlobs = %+v", cfg.Workspace.ProdCodeGlobs)
	}
	if len(cfg.LoadedGates) != 1 {
		t.Fatalf("LoadedGates = %+v", cfg.LoadedGates)
	}
	g := cfg.LoadedGates[0]
	if g.Name != "tdd-red-green-refactor" {
		t.Errorf("gate name = %q", g.Name)
	}
	if g.Severity != SeverityP0 {
		t.Errorf("gate severity = %q", g.Severity)
	}
	if len(g.Scripts) != 1 {
		t.Errorf("gate scripts = %+v", g.Scripts)
	}
	if err := cfg.Validate("0.1.0"); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

func TestLoadConfig_RejectsEmptyBody(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "ratchet.md")
	body := "---\nratchet_spec_version: \"0.1\"\ngates:\n  - name: foo\n---\n\n"
	if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(cfgPath)
	if err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestValidate_RejectsSelfJudgmentPermitted(t *testing.T) {
	cfg := &Config{
		RatchetSpecVersion: "0.1",
		SelfJudgment:       "permitted",
		Gates:              []GateRef{{Name: "x"}},
		LoadedGates: []Gate{
			{Name: "x", RatchetSpecVersion: "0.1", Scripts: []string{"a"}},
		},
		PromptBody: "x",
	}
	if err := cfg.Validate("0.1.0"); err == nil {
		t.Fatal("expected validate error for self_judgment=permitted")
	}
}
