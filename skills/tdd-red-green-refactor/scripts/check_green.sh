#!/usr/bin/env bash
# check_green.sh — Run F2P and P2P tests at HEAD; compute pass rates.
#
# Exit codes:
#   0 — F2P=1.0 AND P2P=1.0 → verdict FULL
#   1 — P2P regression or F2P=0 → verdict NO
#   2 — 0 < F2P < 1.0 AND P2P=1.0 → verdict PARTIAL
#   3 — runner error or zero-test silent collapse → verdict ERROR
#
# stderr: status=<pass|partial|fail|error> f2p=<float> p2p=<float> reason=<short>
#
# Spec §8.3: Test Absence Equals Test Failure. A passing run with zero
# detected tests is a silent test-suite collapse, not a green verdict.

set -euo pipefail

WORKSPACE="${1:-.}"

cd "$WORKSPACE"

# Auto-detect runner.
runner=""
if [ -n "${MAXWELL_TEST_RUNNER:-}" ]; then
  runner="$MAXWELL_TEST_RUNNER"
elif [ -f "go.mod" ]; then
  runner="go"
elif [ -f "pyproject.toml" ] || [ -f "pytest.ini" ] || [ -f "setup.cfg" ]; then
  runner="pytest"
elif [ -f "package.json" ]; then
  runner="npm"
elif [ -f "Cargo.toml" ]; then
  runner="cargo"
else
  echo "status=error reason=test-runner-not-detected f2p=0 p2p=0" >&2
  exit 3
fi

OUT_FILE="$(mktemp -t maxwell-test-output.XXXXXX 2>/dev/null || mktemp)"
trap 'rm -f "$OUT_FILE"' EXIT

# Run all tests at HEAD. Capture stdout+stderr to a per-run file (unique
# per invocation; avoids the world-writable /tmp shared-file footgun).
case "$runner" in
  go)
    if go test ./... >"$OUT_FILE" 2>&1; then
      pass=true
    else
      pass=false
    fi
    ;;
  pytest)
    if pytest >"$OUT_FILE" 2>&1; then
      pass=true
    else
      pass=false
    fi
    ;;
  npm)
    if npm test >"$OUT_FILE" 2>&1; then
      pass=true
    else
      pass=false
    fi
    ;;
  cargo)
    if cargo test >"$OUT_FILE" 2>&1; then
      pass=true
    else
      pass=false
    fi
    ;;
  *)
    echo "status=error reason=unknown-runner runner=$runner f2p=0 p2p=0" >&2
    exit 3
    ;;
esac

# Detect silent test-suite collapse (spec §8.3). A "pass" with zero
# detected tests is an ERROR, not FULL — this is the SWE-Bench gaming
# vector: an agent that breaks the test runner makes the gate trivially
# green. Each runner exposes a different signal.
test_count_signal=""
case "$runner" in
  go)
    # `go test` prints "ok PKG ... 0 tests" when a package has none.
    # `[no test files]` appears for packages with no _test.go.
    # We treat "all packages report no tests" as zero-test collapse.
    if grep -q "PASS" "$OUT_FILE" 2>/dev/null; then
      test_count_signal="ok"
    elif grep -qE "^ok\s" "$OUT_FILE" 2>/dev/null && ! grep -q "\[no test files\]" "$OUT_FILE" 2>/dev/null; then
      test_count_signal="ok"
    fi
    ;;
  pytest)
    # pytest exits 5 ("no tests collected") OR prints "collected 0 items".
    if grep -q "passed" "$OUT_FILE" 2>/dev/null; then
      test_count_signal="ok"
    fi
    ;;
  npm|cargo)
    # No reliable cross-tool zero-test signal; for npm and cargo we
    # require *some* indication of test execution in the output.
    if grep -qiE "(passed|passing|test result|\\.ok|✓|PASS)" "$OUT_FILE" 2>/dev/null; then
      test_count_signal="ok"
    fi
    ;;
esac

if $pass; then
  if [ -z "$test_count_signal" ]; then
    echo "status=error reason=zero-test-collapse f2p=0 p2p=0 runner=$runner" >&2
    cat "$OUT_FILE" >&2 || true
    exit 3
  fi
  echo "status=pass reason=all-tests-green f2p=1.0 p2p=1.0 runner=$runner" >&2
  exit 0
else
  # Without explicit F2P/P2P partitioning we cannot distinguish PARTIAL
  # from NO. v0.1 emits NO conservatively. v0.2+ will parse runner-specific
  # output to compute F2P/P2P separately.
  echo "status=fail reason=tests-failing f2p=0 p2p=0 runner=$runner" >&2
  cat "$OUT_FILE" >&2 || true
  exit 1
fi
