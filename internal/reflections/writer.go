// Package reflections implements the Reflexion-shaped failure-artifact
// contract from spec §10.
//
// On a non-FULL gate verdict, the runner writes one Markdown file under
// .ratchet/reflections/<task-id>/<gate-name>/<attempt>.md. The next
// attempt's prompt prepends the most recent reflection so the agent can
// learn from prior failure (Shinn et al. 2023).
//
// Reflections are append-only. Implementations MUST NOT modify a
// reflection after writing.
package reflections

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Reflection is the structured payload written to disk.
type Reflection struct {
	Gate                string
	Verdict             string
	AttemptNumber       int
	Observation         string
	SuggestedNextAction string
	CreatedAt           time.Time
}

// ErrPathEscape is returned when a taskID or gateName would resolve to a
// filesystem location outside repoRoot/.ratchet/reflections/. Spec §11.3
// Invariant 2: workspace path MUST stay inside workspace root. We extend
// that invariant to reflection paths because they are co-located.
var ErrPathEscape = errors.New("reflection path escapes repo reflections root")

// Write persists a Reflection to disk under repoRoot/.ratchet/reflections/...
// Returns the absolute path written. Rejects taskID/gateName that traverse
// outside the reflections root.
func Write(repoRoot, taskID, gateName string, attempt int, r Reflection) (string, error) {
	root, err := filepath.Abs(filepath.Join(repoRoot, ".ratchet", "reflections"))
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, taskID, gateName)
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") || strings.HasPrefix(rel, string(filepath.Separator)+"..") {
		return "", fmt.Errorf("%w: taskID=%q gateName=%q resolves to %q", ErrPathEscape, taskID, gateName, rel)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", err
	}
	dir = abs
	path := filepath.Join(dir, fmt.Sprintf("%05d.md", attempt))
	body := fmt.Sprintf(`---
gate: %s
verdict: %s
attempt_number: %d
created_at: %s
---

# Observation

%s

# Suggested next action

%s
`,
		r.Gate, r.Verdict, r.AttemptNumber, r.CreatedAt.UTC().Format(time.RFC3339),
		strings.TrimSpace(r.Observation),
		strings.TrimSpace(r.SuggestedNextAction),
	)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// LatestForGate returns the most recent reflection's path for a given gate,
// or empty string if none exist. Reflections are named "<attempt>.md";
// sort numerically so attempt 10 follows attempt 9 (lexicographic sort
// would order "10.md" before "2.md").
func LatestForGate(repoRoot, taskID, gateName string) string {
	dir := filepath.Join(repoRoot, ".ratchet", "reflections", taskID, gateName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	type indexedName struct {
		n    int
		name string
	}
	var named []indexedName
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		n, err := strconv.Atoi(base)
		if err != nil {
			continue
		}
		named = append(named, indexedName{n: n, name: e.Name()})
	}
	if len(named) == 0 {
		return ""
	}
	sort.Slice(named, func(i, j int) bool { return named[i].n < named[j].n })
	return filepath.Join(dir, named[len(named)-1].name)
}

// LatestForGateBody is LatestForGate plus reading the file contents.
// Returns empty string on any error or absence. Used to surface the prior
// reflection into the next attempt's prompt (spec §10).
func LatestForGateBody(repoRoot, taskID, gateName string) string {
	path := LatestForGate(repoRoot, taskID, gateName)
	if path == "" {
		return ""
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(body)
}
