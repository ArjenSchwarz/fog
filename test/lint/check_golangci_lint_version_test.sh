#!/usr/bin/env bash
# Regression test for T-846: make lint should fail with a clear v2 requirement
# message when the installed golangci-lint is a v1 binary (the repo config uses
# `version: "2"`, so a v1 binary cannot lint the project).
#
# The test stubs `golangci-lint` on PATH with fake binaries reporting different
# versions and invokes the shared preflight script (scripts/check-golangci-lint.sh).
#
# Expected behaviour after the fix:
#   - v1 binary  -> preflight exits non-zero with a message mentioning "v2"
#   - v2 binary  -> preflight exits zero (invocation succeeds, linter runs)
#   - missing    -> preflight exits non-zero with an install hint

set -u

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CHECK_SCRIPT="$REPO_ROOT/scripts/check-golangci-lint.sh"

FAIL=0
TEST_TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TEST_TMPDIR"' EXIT

assert() {
    local description="$1"
    local expected="$2"
    local actual="$3"
    if [ "$expected" = "$actual" ]; then
        echo "  PASS: $description"
    else
        echo "  FAIL: $description"
        echo "    expected: $expected"
        echo "    actual:   $actual"
        FAIL=1
    fi
}

assert_contains() {
    local description="$1"
    local needle="$2"
    local haystack="$3"
    case "$haystack" in
        *"$needle"*) echo "  PASS: $description" ;;
        *)
            echo "  FAIL: $description"
            echo "    expected to contain: $needle"
            echo "    actual output: $haystack"
            FAIL=1
            ;;
    esac
}

make_fake_binary() {
    local dir="$1"
    local version_output="$2"
    mkdir -p "$dir"
    cat > "$dir/golangci-lint" <<EOF
#!/usr/bin/env bash
case "\$1" in
    --version|version)
        printf '%s\n' "$version_output"
        exit 0
        ;;
    *)
        # Pretend run succeeds so we can see the preflight decides.
        exit 0
        ;;
esac
EOF
    chmod +x "$dir/golangci-lint"
}

echo "T-846 regression: golangci-lint version preflight"
echo "=================================================="

if [ ! -x "$CHECK_SCRIPT" ]; then
    echo "  FAIL: preflight script missing or not executable: $CHECK_SCRIPT"
    exit 1
fi

# Case 1: v1 binary on PATH -> must fail with a v2 requirement message.
V1_DIR="$TEST_TMPDIR/v1"
make_fake_binary "$V1_DIR" "golangci-lint has version v1.64.8 built with go1.22.0 from unknown"
set +e
OUT_V1="$(PATH="$V1_DIR:/usr/bin:/bin" "$CHECK_SCRIPT" 2>&1)"
RC_V1=$?
set -e
assert "v1 binary causes preflight to fail" "1" "$([ $RC_V1 -ne 0 ] && echo 1 || echo 0)"
assert_contains "v1 failure mentions v2 requirement" "v2" "$OUT_V1"

# Case 2: v2 binary on PATH -> preflight succeeds.
V2_DIR="$TEST_TMPDIR/v2"
make_fake_binary "$V2_DIR" "golangci-lint has version 2.1.6 built with go1.22.0 from abc123"
set +e
OUT_V2="$(PATH="$V2_DIR:/usr/bin:/bin" "$CHECK_SCRIPT" 2>&1)"
RC_V2=$?
set -e
assert "v2 binary is accepted" "0" "$RC_V2"

# Case 3: no binary on PATH -> fail with install hint.
EMPTY_DIR="$TEST_TMPDIR/empty"
mkdir -p "$EMPTY_DIR"
set +e
OUT_MISSING="$(PATH="$EMPTY_DIR:/usr/bin:/bin" "$CHECK_SCRIPT" 2>&1)"
RC_MISSING=$?
set -e
assert "missing binary causes preflight to fail" "1" "$([ $RC_MISSING -ne 0 ] && echo 1 || echo 0)"
assert_contains "missing binary output mentions install hint" "install" "$OUT_MISSING"

if [ $FAIL -ne 0 ]; then
    echo "FAIL"
    exit 1
fi

echo "OK"
