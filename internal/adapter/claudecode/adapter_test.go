package claudecode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsCompatibleCLIVersion(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"2.1.0", true},
		{"2.1.5", true},
		{"2.2.0", true},
		{"3.0.0", true},
		{"2.0.99", false},
		{"1.99.99", false},
		{"2.1.0-beta", true}, // pre-release suffix ignored
		// Unparseable input MUST refuse, not silently accept.
		{"", false},
		{"garbage", false},
		{"v2.1.0", false}, // leading 'v' is non-numeric
		{"2", false},      // single-component is not SemVer-shaped
	}
	for _, c := range cases {
		if got := IsCompatibleCLIVersion(c.in); got != c.want {
			t.Errorf("IsCompatibleCLIVersion(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestExtractVersion(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"claude code 2.1.132 (Claude Code)", "2.1.132"},
		{"2.1.132", "2.1.132"},
		{"version: v2.1.132 build foo", "2.1.132"},
		{"no version here", ""},
		{"2", ""}, // requires a dot
	}
	for _, c := range cases {
		if got := extractVersion(c.in); got != c.want {
			t.Errorf("extractVersion(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestVerifyCLIVersion_FromEnv(t *testing.T) {
	t.Setenv("RATCHET_CLAUDE_CLI_VERSION", "2.1.0")
	if err := VerifyCLIVersion(); err != nil {
		t.Errorf("expected ok, got %v", err)
	}
	t.Setenv("RATCHET_CLAUDE_CLI_VERSION", "1.5.0")
	if err := VerifyCLIVersion(); err == nil {
		t.Errorf("expected refusal on old version")
	}
}

func TestVerifyCLIVersion_SkipsWhenAsked(t *testing.T) {
	t.Setenv("RATCHET_SKIP_HOST_CHECK", "1")
	t.Setenv("RATCHET_CLAUDE_CLI_VERSION", "0.0.1") // would otherwise refuse
	if err := VerifyCLIVersion(); err != nil {
		t.Errorf("expected skip, got %v", err)
	}
}

func TestCanonicalEvent(t *testing.T) {
	cases := []struct {
		in   string
		want HookEvent
	}{
		{"PreToolUse", PreToolUse},
		{"pre_tool_use", PreToolUse},
		{"pretooluse", PreToolUse},
		{"PostToolUse", PostToolUse},
		{"post_tool_use", PostToolUse},
		{"Stop", Stop},
		{"stop", Stop},
		{"PreCompact", PreCompact},
		{"PostCompact", PostCompact},
	}
	for _, c := range cases {
		if got := canonicalEvent(c.in); got != c.want {
			t.Errorf("canonicalEvent(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseStreamLine_Valid(t *testing.T) {
	line := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}`
	e, err := ParseStreamLine(line)
	if err != nil {
		t.Fatalf("ParseStreamLine: %v", err)
	}
	if e.Type != "assistant" {
		t.Errorf("Type = %q", e.Type)
	}
}

func TestParseStreamLine_Malformed(t *testing.T) {
	_, err := ParseStreamLine("not json {")
	if err == nil {
		t.Fatal("expected error on malformed JSON")
	}
}

func TestWriteSettings(t *testing.T) {
	dir := t.TempDir()
	hooks := map[string][]HookSourceConfig{
		"PreToolUse": {
			{Matcher: "Write|Edit", Command: "ratchet gate", TimeoutSec: 30},
		},
		"pre_tool_use": { // alternate spelling should canonicalize
			{Matcher: "*", Command: "echo same-event"},
		},
	}
	perms := PermissionsSource{
		Mode:  "acceptEdits",
		Allow: []string{"Read(./**)"},
		Deny:  []string{"Write(skills/**)"},
	}
	path, err := WriteSettings(dir, perms, hooks)
	if err != nil {
		t.Fatalf("WriteSettings: %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join(".claude", "settings.json")) {
		t.Errorf("unexpected path: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got Settings
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.PermissionMode != "acceptEdits" {
		t.Errorf("PermissionMode = %q", got.PermissionMode)
	}
	if len(got.Permissions.Allow) != 1 || got.Permissions.Allow[0] != "Read(./**)" {
		t.Errorf("Allow = %+v", got.Permissions.Allow)
	}
	if len(got.Permissions.Deny) != 1 || got.Permissions.Deny[0] != "Write(skills/**)" {
		t.Errorf("Deny = %+v", got.Permissions.Deny)
	}
	// Both spellings should have been canonicalized to PreToolUse.
	if entries, ok := got.Hooks[string(PreToolUse)]; !ok || len(entries) != 2 {
		t.Errorf("PreToolUse entries = %+v", entries)
	}
}
