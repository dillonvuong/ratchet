// Package gitsubstrate provides git-topology primitives that the TDD gate
// uses to compute its verdict. The substrate is git itself: `git diff`
// against the base reference, plus `git log` for branch derivation.
//
// Per spec §9.3, the verdict-source for the TDD gate is `git`; this
// package wraps the git CLI and never interprets agent self-reports.
package gitsubstrate

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// IsRepo returns true iff path (or its parents) contains a .git directory.
func IsRepo(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = abs
	return cmd.Run() == nil
}

// DeriveTaskID returns a sanitized identifier for the current task.
// Strategy: use the current branch name; sanitize per spec §11.3.
// Falls back to "default" if no branch info is available.
func DeriveTaskID(repoRoot string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "default"
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" || branch == "HEAD" {
		return "default"
	}
	return SanitizeID(branch)
}

// SanitizeID replaces any character not in [A-Za-z0-9._-] with `_`.
// Per spec §11.3.
func SanitizeID(s string) string {
	if s == "" {
		return "_"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z',
			r >= 'a' && r <= 'z',
			r >= '0' && r <= '9',
			r == '.' || r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		out = "_"
	}
	return out
}

// BaseRef returns the merge-base between HEAD and the given target branch
// (default: "main", falling back to "master"). If neither exists, returns
// "HEAD~1" as a conservative fallback.
func BaseRef(repoRoot string) string {
	for _, target := range []string{"main", "master"} {
		out, err := runGit(repoRoot, "merge-base", "HEAD", target)
		if err == nil && strings.TrimSpace(out) != "" {
			return strings.TrimSpace(out)
		}
	}
	return "HEAD~1"
}

// DiffNames returns the list of file paths changed between baseRef and HEAD,
// filtered by the given pathspecs. An empty pathspec list returns all changes.
func DiffNames(repoRoot, baseRef string, pathspecs []string) ([]string, error) {
	args := []string{"diff", "--name-only", baseRef, "HEAD"}
	if len(pathspecs) > 0 {
		args = append(args, "--")
		args = append(args, pathspecs...)
	}
	out, err := runGit(repoRoot, args...)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var paths []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			paths = append(paths, l)
		}
	}
	return paths, nil
}

// DiffNamesAddedModified is DiffNames restricted to added (A) or modified (M) files.
func DiffNamesAddedModified(repoRoot, baseRef string, pathspecs []string) ([]string, error) {
	args := []string{"diff", "--name-only", "--diff-filter=AM", baseRef, "HEAD"}
	if len(pathspecs) > 0 {
		args = append(args, "--")
		args = append(args, pathspecs...)
	}
	out, err := runGit(repoRoot, args...)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var paths []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			paths = append(paths, l)
		}
	}
	return paths, nil
}

// DiffNamesDeleted returns the list of files deleted between baseRef and HEAD.
func DiffNamesDeleted(repoRoot, baseRef string, pathspecs []string) ([]string, error) {
	args := []string{"diff", "--name-only", "--diff-filter=D", baseRef, "HEAD"}
	if len(pathspecs) > 0 {
		args = append(args, "--")
		args = append(args, pathspecs...)
	}
	out, err := runGit(repoRoot, args...)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var paths []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			paths = append(paths, l)
		}
	}
	return paths, nil
}

// FileAtRef returns the contents of path at the given git ref.
func FileAtRef(repoRoot, ref, path string) (string, error) {
	return runGit(repoRoot, "show", ref+":"+path)
}

func runGit(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	return string(out), err
}
