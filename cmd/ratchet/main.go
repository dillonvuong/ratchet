// ratchet — host-agnostic harness enforcing hard-verdict gates over agent output.
//
// See docs/spec.md for the normative specification. This binary is one
// reference implementation; conformance to the spec is the durable contract.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dillon-vuong/ratchet/internal/adapter/claudecode"
	"github.com/dillon-vuong/ratchet/internal/gitsubstrate"
	"github.com/dillon-vuong/ratchet/internal/runner"
)

const (
	binaryVersion = "0.1.0-alpha"
	specVersion   = "0.1"
)

func main() {
	if len(os.Args) < 2 {
		printUsage(os.Stderr)
		os.Exit(2)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "run":
		os.Exit(cmdRun(args))
	case "gate":
		os.Exit(cmdGate(args))
	case "init":
		os.Exit(cmdInit(args))
	case "version", "--version", "-v":
		cmdVersion(os.Stdout)
	case "finalize-transcript":
		os.Exit(cmdFinalizeTranscript(args))
	case "help", "--help", "-h":
		printUsage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "ratchet: unknown command %q\n\n", cmd)
		printUsage(os.Stderr)
		os.Exit(2)
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `ratchet %s — spec %s

Usage:
  ratchet run [--config <path>] [--task <id>]
      Run all gates declared in ratchet.md against the working tree.

  ratchet gate --name <gate-name> [--on <event>] [--task <id>]
      Run a single named gate. Used by host adapters from hook events.
      Events: PreToolUse | PostToolUse | Stop | PreCompact | PostCompact

  ratchet init [<dir>]
      Scaffold ratchet.md, AGENTS.md, and skills/tdd-red-green-refactor/
      into the current or specified directory.

  ratchet version
      Print the binary version and the spec version it implements.

  ratchet finalize-transcript
      Close out the current run's transcript artifact.

Documentation:
  docs/spec.md             Normative specification (RFC 2119)
  docs/why-tdd-first.md    The argument for TDD as the canonical first gate
  docs/adding-a-gate.md    How to write a new gate
  docs/maintenance.md      Versioning, deprecation, RFC process
  AGENTS.md                Table of contents
  ratchet.md               Doctrine prompt body

`, binaryVersion, specVersion)
}

func cmdVersion(w io.Writer) {
	fmt.Fprintf(w, "ratchet %s\n", binaryVersion)
	fmt.Fprintf(w, "spec %s\n", specVersion)
	fmt.Fprintf(w, "ref-impl: github.com/dillon-vuong/ratchet\n")
	fmt.Fprintf(w, "license: Apache-2.0\n")
}

func cmdRun(args []string) int {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", "ratchet.md", "path to ratchet.md")
	taskID := fs.String("task", "", "task identifier (optional; auto-derived from branch if empty)")
	skipGitRepoCheck := fs.Bool("skip-git-repo-check", false, "skip the git repo precondition")
	jsonOut := fs.Bool("json", false, "emit JSONL run events on stdout")
	_ = fs.Parse(args)

	cfg, err := runner.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ratchet: %v\n", err)
		return 1
	}
	if err := cfg.Validate(specVersion); err != nil {
		fmt.Fprintf(os.Stderr, "ratchet: %v\n", err)
		return 1
	}

	if !*skipGitRepoCheck {
		if !gitsubstrate.IsRepo(".") {
			fmt.Fprintln(os.Stderr, "ratchet: not a git repository (use --skip-git-repo-check to override)")
			return 1
		}
	}

	if err := claudecode.VerifyCLIVersion(); err != nil {
		fmt.Fprintf(os.Stderr, "ratchet: %v\n", err)
		return 1
	}

	if *taskID == "" {
		*taskID = gitsubstrate.DeriveTaskID(".")
	}

	r, err := runner.New(cfg, *taskID, runner.Options{JSONOut: *jsonOut})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ratchet: %v\n", err)
		return 1
	}

	verdict := r.Run()
	switch verdict.Value {
	case runner.VerdictFULL:
		return 0
	case runner.VerdictPARTIAL:
		return 2
	case runner.VerdictNO:
		return 1
	case runner.VerdictERROR:
		return 3
	default:
		return 1
	}
}

func cmdGate(args []string) int {
	fs := flag.NewFlagSet("gate", flag.ExitOnError)
	gateName := fs.String("name", "", "gate name (required)")
	onEvent := fs.String("on", "", "host event that triggered this run (optional)")
	taskID := fs.String("task", "", "task identifier")
	_ = fs.Parse(args)

	if *gateName == "" {
		fmt.Fprintln(os.Stderr, "ratchet gate: --name is required")
		return 2
	}

	cfg, err := runner.LoadConfig("ratchet.md")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ratchet: %v\n", err)
		return 1
	}

	if err := cfg.Validate(specVersion); err != nil {
		fmt.Fprintf(os.Stderr, "ratchet: %v\n", err)
		return 1
	}

	gate, err := cfg.FindGate(*gateName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ratchet: %v\n", err)
		return 1
	}

	if err := claudecode.VerifyCLIVersion(); err != nil {
		fmt.Fprintf(os.Stderr, "ratchet: %v\n", err)
		return 1
	}

	if *taskID == "" {
		*taskID = gitsubstrate.DeriveTaskID(".")
	}

	r, err := runner.New(cfg, *taskID, runner.Options{TriggerEvent: *onEvent})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ratchet: %v\n", err)
		return 1
	}

	gateRun := r.RunGate(gate)
	switch gateRun.Verdict.Value {
	case runner.VerdictFULL:
		return 0
	case runner.VerdictPARTIAL:
		return 2
	case runner.VerdictNO:
		return 1
	case runner.VerdictERROR:
		return 3
	default:
		return 1
	}
}

func cmdInit(args []string) int {
	target := "."
	if len(args) > 0 {
		target = args[0]
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ratchet: %v\n", err)
		return 1
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "ratchet: %v\n", err)
		return 1
	}

	// In v0.1, init copies the canonical files from the binary's embedded fs.
	// For now, print guidance.
	fmt.Fprintf(os.Stdout, `ratchet init: scaffold the v0.1 layout into %s

Required files (copy from this repository):
  ratchet.md                                    — frontmatter + doctrine
  AGENTS.md                                     — table-of-contents
  skills/tdd-red-green-refactor/SKILL.md       — first gate
  skills/tdd-red-green-refactor/scripts/*.sh   — gate scripts (chmod +x)

After scaffolding, run:
  ratchet run

Embedded init will land in v0.1.1; for now, see github.com/dillon-vuong/ratchet
for the canonical layout.
`, abs)
	return 0
}

func cmdFinalizeTranscript(args []string) int {
	_ = args
	// In v0.1, transcripts are finalized by the runner on Run() completion.
	// This entry point exists for hook integration (Stop event).
	if os.Getenv("RATCHET_RUN_ID") == "" {
		fmt.Fprintln(os.Stderr, "ratchet finalize-transcript: no active run")
		return 0
	}
	fmt.Fprintln(os.Stderr, "ratchet finalize-transcript: ok")
	return 0
}

