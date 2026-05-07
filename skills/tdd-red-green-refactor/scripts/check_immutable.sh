#!/usr/bin/env bash
# check_immutable.sh — Detect test deletions or assertion-weakening relative to base ref.
#
# Exit codes:
#   0 — no test deletions detected
#   1 — test deletion or weakening; verdict NO
#   3 — runner error
#
# stderr: status=<pass|fail> reason=<short> evidence=<file:line>

set -euo pipefail

WORKSPACE="${1:-.}"
BASE_REF="${RATCHET_BASE_REF:-HEAD~1}"
TEST_GLOBS="${RATCHET_TEST_GLOBS:-**/*_test.go:**/*_test.py:**/*.test.ts:**/*.test.js:tests/**:test/**:spec/**}"

cd "$WORKSPACE"

if ! git rev-parse --git-dir >/dev/null 2>&1; then
  echo "status=error reason=not-a-git-repo" >&2
  exit 3
fi

if ! git rev-parse --verify "$BASE_REF" >/dev/null 2>&1; then
  echo "status=error reason=base-ref-not-found base_ref=$BASE_REF" >&2
  exit 3
fi

# Build the set of test files at base ref.
mapfile -t base_test_files < <(
  git diff --name-only --diff-filter=DM "$BASE_REF" HEAD -- $(echo "$TEST_GLOBS" | tr ':' ' ') 2>/dev/null || true
)

deletions=()
weakenings=()

for f in "${base_test_files[@]}"; do
  # If the file existed at base but not at HEAD → deletion.
  if git cat-file -e "$BASE_REF:$f" 2>/dev/null && ! [ -f "$f" ]; then
    deletions+=("$f")
    continue
  fi

  # If the file exists at both, check for removed test functions / assertions.
  if [ -f "$f" ]; then
    # Count assert-style lines at base vs HEAD. Heuristic across languages.
    base_asserts=$(git show "$BASE_REF:$f" 2>/dev/null | grep -cE '^[[:space:]]*(assert|expect|Assert|Expect|ASSERT|XCTAssert|require\.|t\.(Errorf|Fatalf|Fatal|Error)|self\.assert)' || echo 0)
    head_asserts=$(grep -cE '^[[:space:]]*(assert|expect|Assert|Expect|ASSERT|XCTAssert|require\.|t\.(Errorf|Fatalf|Fatal|Error)|self\.assert)' "$f" || echo 0)
    if [ "$head_asserts" -lt "$base_asserts" ]; then
      weakenings+=("$f (asserts: $base_asserts → $head_asserts)")
    fi

    # Detect removed test functions. Heuristic: lines starting with "func Test" / "def test_" / "it(".
    base_tests=$(git show "$BASE_REF:$f" 2>/dev/null | grep -cE '^[[:space:]]*(func Test|def test_|it\(|test\(|@Test)' || echo 0)
    head_tests=$(grep -cE '^[[:space:]]*(func Test|def test_|it\(|test\(|@Test)' "$f" || echo 0)
    if [ "$head_tests" -lt "$base_tests" ]; then
      weakenings+=("$f (test fns: $base_tests → $head_tests)")
    fi
  fi
done

if [ "${#deletions[@]}" -gt 0 ]; then
  ev=$(printf '%s ' "${deletions[@]}")
  echo "status=fail reason=test-file-deleted evidence=${ev% }" >&2
  exit 1
fi

if [ "${#weakenings[@]}" -gt 0 ]; then
  ev=$(printf '%s ' "${weakenings[@]}")
  echo "status=fail reason=test-weakened evidence=${ev% }" >&2
  exit 1
fi

echo "status=pass reason=tests-immutable" >&2
exit 0
