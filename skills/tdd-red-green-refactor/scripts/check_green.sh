#!/usr/bin/env bash
# check_green.sh — Run F2P and P2P tests at HEAD; compute pass rates.
#
# Exit codes:
#   0 — F2P=1.0 AND P2P=1.0 → verdict FULL
#   1 — P2P regression or F2P=0 → verdict NO
#   2 — 0 < F2P < 1.0 AND P2P=1.0 → verdict PARTIAL
#   3 — runner error → verdict ERROR
#
# stderr: status=<pass|partial|fail|error> f2p=<float> p2p=<float> reason=<short>

set -euo pipefail

WORKSPACE="${1:-.}"

cd "$WORKSPACE"

# Auto-detect runner.
runner=""
if [ -n "${RATCHET_TEST_RUNNER:-}" ]; then
  runner="$RATCHET_TEST_RUNNER"
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

# Run all tests at HEAD.
case "$runner" in
  go)
    if go test ./... 2>&1 | tee /tmp/ratchet-test-output.log; then
      pass=true
    else
      pass=false
    fi
    ;;
  pytest)
    if pytest 2>&1 | tee /tmp/ratchet-test-output.log; then
      pass=true
    else
      pass=false
    fi
    ;;
  npm)
    if npm test 2>&1 | tee /tmp/ratchet-test-output.log; then
      pass=true
    else
      pass=false
    fi
    ;;
  cargo)
    if cargo test 2>&1 | tee /tmp/ratchet-test-output.log; then
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

if $pass; then
  echo "status=pass reason=all-tests-green f2p=1.0 p2p=1.0 runner=$runner" >&2
  exit 0
else
  # Without explicit F2P/P2P partitioning we cannot distinguish PARTIAL from NO.
  # In v0.1, any test failure → NO (conservative). v0.2+ will parse runner-specific
  # output to compute F2P/P2P separately.
  echo "status=fail reason=tests-failing f2p=0 p2p=0 runner=$runner see=/tmp/ratchet-test-output.log" >&2
  exit 1
fi
