// Package claudecode is the Claude Code host adapter (spec §10.2).
//
// The adapter:
//   * Generates .claude/settings.json from ratchet.md's permissions and hooks.
//   * Pins the minimum claude CLI version (2.1.0).
//   * Consumes `claude --output-format stream-json` JSONL events.
//   * Translates host hook events (PreToolUse, PostToolUse, Stop, PreCompact,
//     PostCompact) into ratchet gate dispatches.
//
// Hook event names match the claude-agent-sdk PascalCase canon exactly.
// Permission grammar uses the `Tool(pattern)` form.
package claudecode

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AdapterName is exposed so the binary can keep the package linked.
const AdapterName = "claudecode"

// MinimumCLIVersion pins the minimum supported claude CLI (spec §10.2).
const MinimumCLIVersion = "2.1.0"

// HookEvent is the canonical PascalCase event name. Spec §10.2.
type HookEvent string

const (
	PreToolUse  HookEvent = "PreToolUse"
	PostToolUse HookEvent = "PostToolUse"
	Stop        HookEvent = "Stop"
	PreCompact  HookEvent = "PreCompact"
	PostCompact HookEvent = "PostCompact"
)

// HookEntry is the JSON shape Claude Code expects in .claude/settings.json
// for a single hook event.
type HookEntry struct {
	Matcher    string         `json:"matcher,omitempty"`
	Hooks      []HookCommand  `json:"hooks"`
}

// HookCommand is one executable command bound to a hook entry.
type HookCommand struct {
	Type       string `json:"type"`
	Command    string `json:"command"`
	TimeoutSec int    `json:"timeoutSec,omitempty"`
}

// Settings models the subset of .claude/settings.json that ratchet writes.
type Settings struct {
	Permissions    SettingsPermissions       `json:"permissions"`
	Hooks          map[string][]HookEntry    `json:"hooks,omitempty"`
	PermissionMode string                    `json:"permissionMode,omitempty"`
}

// SettingsPermissions models the allow/deny grammar Claude Code consumes.
type SettingsPermissions struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

// HookSourceConfig is the input shape coming from ratchet.md frontmatter.
type HookSourceConfig struct {
	Matcher    string
	Command    string
	TimeoutSec int
}

// PermissionsSource is the input from ratchet.md frontmatter.
type PermissionsSource struct {
	Mode  string
	Allow []string
	Deny  []string
}

// WriteSettings emits .claude/settings.json into repoRoot.
// Returns the path written.
func WriteSettings(repoRoot string, perms PermissionsSource, hooks map[string][]HookSourceConfig) (string, error) {
	dir := filepath.Join(repoRoot, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	out := Settings{
		Permissions: SettingsPermissions{
			Allow: perms.Allow,
			Deny:  perms.Deny,
		},
		PermissionMode: perms.Mode,
		Hooks:          map[string][]HookEntry{},
	}
	for ev, list := range hooks {
		canonical := string(canonicalEvent(ev))
		for _, h := range list {
			out.Hooks[canonical] = append(out.Hooks[canonical], HookEntry{
				Matcher: h.Matcher,
				Hooks: []HookCommand{
					{Type: "command", Command: h.Command, TimeoutSec: h.TimeoutSec},
				},
			})
		}
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// canonicalEvent normalizes an incoming event name to the spec's PascalCase canon.
// Tolerant of snake_case input from ratchet.md but emits PascalCase to settings.json.
func canonicalEvent(name string) HookEvent {
	switch strings.ToLower(name) {
	case "pretooluse", "pre_tool_use":
		return PreToolUse
	case "posttooluse", "post_tool_use":
		return PostToolUse
	case "stop":
		return Stop
	case "precompact", "pre_compact":
		return PreCompact
	case "postcompact", "post_compact":
		return PostCompact
	default:
		// Pass through unrecognized; user is responsible.
		return HookEvent(name)
	}
}

// IsCompatibleCLIVersion compares the claude CLI version string to MinimumCLIVersion.
// Versions are dotted SemVer; pre-release suffixes are ignored. An unparseable
// version string returns false (refuse to proceed) — the prior behavior of
// returning true on parse failure created a false-positive compatibility claim.
func IsCompatibleCLIVersion(s string) bool {
	core := strings.SplitN(s, "-", 2)[0]
	have := strings.SplitN(core, ".", 4)
	want := strings.SplitN(MinimumCLIVersion, ".", 4)
	if len(have) < 2 {
		return false
	}
	for i := 0; i < 3; i++ {
		var hi, wi int
		if i < len(have) {
			if _, err := fmt.Sscanf(have[i], "%d", &hi); err != nil {
				return false
			}
		}
		if i < len(want) {
			if _, err := fmt.Sscanf(want[i], "%d", &wi); err != nil {
				return false
			}
		}
		if hi > wi {
			return true
		}
		if hi < wi {
			return false
		}
	}
	return true
}

// VerifyCLIVersion attempts to discover the local claude CLI version and
// verify it meets MinimumCLIVersion. Returns nil if compatible.
//
// Behavior:
//   * If the env var `RATCHET_SKIP_HOST_CHECK=1` is set, returns nil.
//   * If the env var `RATCHET_CLAUDE_CLI_VERSION` is set, uses that value
//     directly (useful for CI and hook contexts where the host has injected it).
//   * Otherwise attempts `claude --version` via PATH lookup. If `claude`
//     is not on PATH, returns nil (operator may be running ratchet outside
//     a Claude Code context — Codex, Cursor, CI). Only refuses when claude
//     IS present and the version is too old.
func VerifyCLIVersion() error {
	if os.Getenv("RATCHET_SKIP_HOST_CHECK") == "1" {
		return nil
	}
	if v := os.Getenv("RATCHET_CLAUDE_CLI_VERSION"); v != "" {
		if !IsCompatibleCLIVersion(v) {
			return fmt.Errorf("claude CLI version %q < minimum %q", v, MinimumCLIVersion)
		}
		return nil
	}
	bin, err := exec.LookPath("claude")
	if err != nil {
		// Not on PATH: ratchet is running outside Claude Code. No-op.
		return nil
	}
	cmd := exec.Command(bin, "--version")
	out, err := cmd.Output()
	if err != nil {
		// Couldn't ask the CLI; soft-fail rather than block.
		return nil
	}
	// Output shapes seen in the wild:
	//   "claude code 2.1.132 (Claude Code)"
	//   "2.1.132"
	// Extract the first dotted number.
	v := extractVersion(string(out))
	if v == "" {
		return nil
	}
	if !IsCompatibleCLIVersion(v) {
		return fmt.Errorf("claude CLI version %q < minimum %q (from %q)", v, MinimumCLIVersion, strings.TrimSpace(string(out)))
	}
	return nil
}

func extractVersion(s string) string {
	// Find the first run of digits.dots.digits.
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			continue
		}
		j := i
		for j < len(s) && (s[j] == '.' || (s[j] >= '0' && s[j] <= '9') || s[j] == '-') {
			j++
		}
		seg := s[i:j]
		if strings.Contains(seg, ".") {
			return seg
		}
	}
	return ""
}

// StreamEvent is one JSONL record emitted by `claude --output-format stream-json`.
type StreamEvent struct {
	Type    string          `json:"type"`
	Subtype string          `json:"subtype,omitempty"`
	Message json.RawMessage `json:"message,omitempty"`
}

// ParseStreamLine parses a single JSONL line. Malformed lines return an error
// without crashing the caller.
func ParseStreamLine(line string) (StreamEvent, error) {
	var e StreamEvent
	if err := json.Unmarshal([]byte(line), &e); err != nil {
		return StreamEvent{}, err
	}
	return e, nil
}
