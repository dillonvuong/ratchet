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
	}
	for _, c := range cases {
		if got := IsCompatibleCLIVersion(c.in); got != c.want {
			t.Errorf("IsCompatibleCLIVersion(%q) = %v, want %v", c.in, got, c.want)
		}
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
