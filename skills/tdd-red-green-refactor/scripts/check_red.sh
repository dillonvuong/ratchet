#!/usr/bin/env bash
# check_red.sh — Verify a new failing test exists for any production-code edit.
#
# Spec §15.3 requires actual test execution at the base reference: the
# new tests MUST fail when run against the codebase BEFORE the agent's
# production change. This is the load-bearing Willison canon: "confirm
# that the tests fail before implementing the code to make them pass."
#
# Implementation strategy:
#   1. Identify added test files in HEAD vs BASE_REF.
#   2. Identify added test FUNCTIONS in those files (and modified test
#      files where new functions were added).
#   3. For supported runners (Go), create a git worktree at BASE_REF,
#      copy the new test files into it, and run those tests in isolation.
#      A *failing* result is the red-phase verification.
#   4. For non-Go runners in v0.1, fall back to the source-level heuristic
#      with a degraded confidence note in stderr.
#
# Exit codes:
#   0 — red phase verified (new tests added; they fail at BASE_REF)
#   1 — red phase missing or new test passes at BASE_REF; verdict NO
#   3 — runner error
#
# stderr: status=<pass|fail|error> reason=<short> f2p_count=<N> red_verified=<N>

set -euo pipefail

WORKSPACE="${1:-.}"
BASE_REF="${RATCHET_BASE_REF:-HEAD~1}"
PROD_GLOBS="${RATCHET_PROD_CODE_GLOBS:-src/**:lib/**:internal/**:pkg/**:cmd/**}"
TEST_GLOBS="${RATCHET_TEST_GLOBS:-**/*_test.go:**/*_test.py:**/*.test.ts:**/*.test.js:tests/**:test/**:spec/**}"

cd "$WORKSPACE"

if ! git rev-parse --git-dir >/dev/null 2>&1; then
  echo "status=error reason=not-a-git-repo" >&2
  exit 3
fi

# How many production-code files did the agent modify in this turn?
prod_changed=$(
  git diff --name-only --diff-filter=AM "$BASE_REF" HEAD -- $(echo "$PROD_GLOBS" | tr ':' ' ') 2>/dev/null | wc -l | tr -d ' '
)

if [ "$prod_changed" -eq 0 ]; then
  echo "status=pass reason=no-prod-edits-this-turn f2p_count=0 red_verified=0" >&2
  exit 0
fi

# Net-new test functions across all changed test files (heuristic — used
# both as a fast fail and to scope the worktree-based verification below).
mapfile -t test_changed < <(
  git diff --name-only --diff-filter=AM "$BASE_REF" HEAD -- $(echo "$TEST_GLOBS" | tr ':' ' ') 2>/dev/null
)

new_tests=0
for f in "${test_changed[@]}"; do
  if [ -f "$f" ]; then
    base_n=$(git show "$BASE_REF:$f" 2>/dev/null | grep -cE '^[[:space:]]*(func Test|def test_|it\(|test\(|@Test)' 2>/dev/null || true)
    head_n=$(grep -cE '^[[:space:]]*(func Test|def test_|it\(|test\(|@Test)' "$f" 2>/dev/null || true)
    base_n=${base_n:-0}
    head_n=${head_n:-0}
    diff=$((head_n - base_n))
    if [ "$diff" -gt 0 ]; then
      new_tests=$((new_tests + diff))
    fi
  fi
done

if [ "$new_tests" -eq 0 ]; then
  echo "status=fail reason=red-phase-missing prod_files_changed=$prod_changed f2p_count=0 red_verified=0" >&2
  exit 1
fi

# For Go we can do real verification: spin up a worktree at BASE_REF,
# overlay only the agent's added test functions, and run them. A passing
# test there means the test does NOT exercise new behavior — fail the gate.
if [ -f "go.mod" ]; then
  TEMP_WT="$(mktemp -d -t ratchet-redcheck.XXXXXX 2>/dev/null || mktemp -d)"
  cleanup() {
    git worktree remove --force "$TEMP_WT" >/dev/null 2>&1 || true
    rm -rf "$TEMP_WT"
  }
  trap cleanup EXIT

  if ! git worktree add --detach -q "$TEMP_WT" "$BASE_REF" 2>/dev/null; then
    echo "status=error reason=worktree-add-failed base_ref=$BASE_REF" >&2
    exit 3
  fi

  red_total=0
  red_passing_at_base=0

  for f in "${test_changed[@]}"; do
    [ -f "$f" ] || continue
    # Test functions added in this turn (exist in HEAD, not in BASE_REF).
    head_funcs=$(grep -oE '^func (Test[A-Za-z0-9_]+)' "$f" 2>/dev/null | awk '{print $2}' | sort -u || true)
    base_funcs=$(git show "$BASE_REF:$f" 2>/dev/null | grep -oE '^func (Test[A-Za-z0-9_]+)' 2>/dev/null | awk '{print $2}' | sort -u || true)
    new_funcs=$(comm -23 <(echo "$head_funcs") <(echo "$base_funcs") | grep -v '^$' || true)
    [ -n "$new_funcs" ] || continue

    # Overlay this file into the worktree at the same path.
    dest="$TEMP_WT/$f"
    mkdir -p "$(dirname "$dest")"
    cp "$f" "$dest"

    # Resolve the package directory.
    pkg_dir="$(dirname "$f")"
    [ "$pkg_dir" = "" ] && pkg_dir="."

    # Build a Go regex that matches exactly these test names.
    pat=$(echo "$new_funcs" | paste -sd '|' -)
    regex="^(${pat})$"

    # Run the targeted tests at BASE_REF. We expect non-zero exit.
    if (cd "$TEMP_WT" && go test -run "$regex" "./$pkg_dir" >/dev/null 2>&1); then
      # Test passed at base — does NOT exercise new behavior. Red-phase missing.
      red_passing_at_base=$((red_passing_at_base + 1))
    fi
    # Count tests we evaluated (one per file with new test funcs).
    red_total=$((red_total + 1))
  done

  if [ "$red_total" -eq 0 ]; then
    # We had new_tests > 0 by line count but couldn't isolate any function
    # names (e.g., the new tests are subtests via t.Run(...)). Fall back to
    # the heuristic-only verdict.
    echo "status=pass reason=red-phase-heuristic-only f2p_count=$new_tests red_verified=heuristic" >&2
    exit 0
  fi

  if [ "$red_passing_at_base" -gt 0 ]; then
    echo "status=fail reason=test-passes-at-base f2p_count=$red_total red_verified=$((red_total - red_passing_at_base))" >&2
    exit 1
  fi

  echo "status=pass reason=red-phase-verified-at-base f2p_count=$red_total red_verified=$red_total" >&2
  exit 0
fi

# Non-Go runners: heuristic only in v0.1. Future versions (per docs/spec.md
# §15.3) MUST extend the worktree+run pattern to pytest/jest/cargo.
echo "status=pass reason=red-phase-verified-heuristic prod_files_changed=$prod_changed f2p_count=$new_tests red_verified=heuristic" >&2
exit 0
