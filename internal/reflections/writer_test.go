package reflections

import (
	"errors"
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
	// Files are zero-padded so lexicographic sort matches numeric.
	if !strings.HasSuffix(path, filepath.Join(".maxwell", "reflections", "task-1", "tdd", "00001.md")) {
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
	if !strings.HasSuffix(latest, "00003.md") {
		t.Errorf("expected latest = 00003.md, got %s", latest)
	}
}

// TestLatestForGate_NumericSort defends against the lexicographic-sort bug
// where attempt 10 would be ordered before attempt 2.
func TestLatestForGate_NumericSort(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []int{1, 2, 3, 9, 10, 11} {
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
	if !strings.HasSuffix(latest, "00011.md") {
		t.Errorf("expected latest = 00011.md, got %s", latest)
	}
}

func TestLatestForGate_Missing(t *testing.T) {
	dir := t.TempDir()
	if got := LatestForGate(dir, "no-such-task", "no-gate"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// TestLatestForGateBody returns the file contents so the next attempt's
// prompt can prepend the prior reflection (spec §10).
func TestLatestForGateBody(t *testing.T) {
	dir := t.TempDir()
	_, err := Write(dir, "task-1", "tdd", 1, Reflection{
		Gate:          "tdd",
		Verdict:       "NO",
		AttemptNumber: 1,
		Observation:   "specific observation here",
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	body := LatestForGateBody(dir, "task-1", "tdd")
	if !strings.Contains(body, "specific observation here") {
		t.Errorf("body missing expected content: %q", body)
	}
}

func TestLatestForGateBody_Missing(t *testing.T) {
	dir := t.TempDir()
	if got := LatestForGateBody(dir, "no-such-task", "no-gate"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// TestWrite_RejectsTaskIDPathTraversal ensures a crafted taskID cannot
// escape the reflections root. Spec §11.3 Invariant 2 extended to
// reflection paths.
func TestWrite_RejectsTaskIDPathTraversal(t *testing.T) {
	dir := t.TempDir()
	_, err := Write(dir, "../../etc", "tdd", 1, Reflection{
		Gate: "tdd", Verdict: "NO", AttemptNumber: 1,
	})
	if err == nil {
		t.Fatal("expected ErrPathEscape, got nil")
	}
	if !errors.Is(err, ErrPathEscape) {
		t.Errorf("expected ErrPathEscape, got %v", err)
	}
}

func TestWrite_RejectsGateNamePathTraversal(t *testing.T) {
	dir := t.TempDir()
	_, err := Write(dir, "task-1", "../tdd", 1, Reflection{
		Gate: "tdd", Verdict: "NO", AttemptNumber: 1,
	})
	if err == nil {
		t.Fatal("expected ErrPathEscape, got nil")
	}
	if !errors.Is(err, ErrPathEscape) {
		t.Errorf("expected ErrPathEscape, got %v", err)
	}
}

func TestWrite_AllowsLegitNestedSlashesInGateName(t *testing.T) {
	// gate names should be hyphenated lowercase; this test checks that a
	// legitimate name with trailing slash semantics doesn't accidentally
	// trip the traversal guard.
	dir := t.TempDir()
	_, err := Write(dir, "task-1", "tdd-red-green-refactor", 1, Reflection{
		Gate: "tdd-red-green-refactor", Verdict: "NO", AttemptNumber: 1,
	})
	if err != nil {
		t.Errorf("legitimate gate name rejected: %v", err)
	}
}
