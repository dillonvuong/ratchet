package reflections

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWrite_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path, err := Write(dir, "task-1", "tdd", 1, Reflection{
		Gate:                "tdd",
		Verdict:             "NO",
		AttemptNumber:       1,
		Observation:         "test deletion detected",
		SuggestedNextAction: "restore the deleted test",
		CreatedAt:           time.Date(2026, 5, 6, 19, 14, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join(".ratchet", "reflections", "task-1", "tdd", "1.md")) {
		t.Errorf("unexpected path: %s", path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"gate: tdd",
		"verdict: NO",
		"attempt_number: 1",
		"created_at: 2026-05-06T19:14:00Z",
		"# Observation",
		"test deletion detected",
		"# Suggested next action",
		"restore the deleted test",
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("body missing %q\n--- body ---\n%s", want, string(body))
		}
	}
}

func TestLatestForGate(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []int{1, 2, 3} {
		_, err := Write(dir, "task-1", "tdd", n, Reflection{
			Gate:          "tdd",
			Verdict:       "NO",
			AttemptNumber: n,
			Observation:   "obs",
			CreatedAt:     time.Now().UTC(),
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	latest := LatestForGate(dir, "task-1", "tdd")
	if !strings.HasSuffix(latest, "3.md") {
		t.Errorf("expected latest = 3.md, got %s", latest)
	}
}

func TestLatestForGate_Missing(t *testing.T) {
	dir := t.TempDir()
	if got := LatestForGate(dir, "no-such-task", "no-gate"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}
