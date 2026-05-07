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
// Versions are dotted SemVer; pre-release suffixes are ignored.
func IsCompatibleCLIVersion(s string) bool {
	have := strings.SplitN(strings.SplitN(s, "-", 2)[0], ".", 4)
	want := strings.SplitN(MinimumCLIVersion, ".", 4)
	for i := 0; i < 3; i++ {
		var hi, wi int
		if i < len(have) {
			fmt.Sscanf(have[i], "%d", &hi)
		}
		if i < len(want) {
			fmt.Sscanf(want[i], "%d", &wi)
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
