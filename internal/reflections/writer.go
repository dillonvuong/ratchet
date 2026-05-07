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
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

// Write persists a Reflection to disk under repoRoot/.ratchet/reflections/...
// Returns the absolute path written.
func Write(repoRoot, taskID, gateName string, attempt int, r Reflection) (string, error) {
	dir := filepath.Join(repoRoot, ".ratchet", "reflections", taskID, gateName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("%d.md", attempt))
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
// or empty string if none exist.
func LatestForGate(repoRoot, taskID, gateName string) string {
	dir := filepath.Join(repoRoot, ".ratchet", "reflections", taskID, gateName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	return filepath.Join(dir, names[len(names)-1])
}
