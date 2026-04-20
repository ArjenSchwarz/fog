#!/usr/bin/env bash
# Preflight check used by `make lint` and test/validate_tests.sh.
#
# The project's .golangci.yml declares `version: "2"`, which is only understood
# by golangci-lint v2. This script verifies a v2 binary is on PATH and fails
# fast with an actionable message otherwise, so contributors don't have to
# interpret the raw "config v2 / binary v1" error from the linter itself.

set -eu

REQUIRED_MAJOR=2
INSTALL_CMD='go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest'

if ! command -v golangci-lint >/dev/null 2>&1; then
    cat >&2 <<EOF
error: golangci-lint is not installed or not on PATH.

This repository requires golangci-lint v${REQUIRED_MAJOR} (its .golangci.yml
uses the v${REQUIRED_MAJOR} schema). Install it with:

    ${INSTALL_CMD}
EOF
    exit 1
fi

# `golangci-lint --version` output lines we need to handle:
#   v1: "golangci-lint has version v1.64.8 built with go1.22.0 from ..."
#   v2: "golangci-lint has version 2.1.6 built with go1.22.0 from ..."
VERSION_OUTPUT="$(golangci-lint --version 2>&1 | head -n1)"

# Extract the first X.Y(.Z) token; tolerate an optional leading 'v'.
VERSION="$(printf '%s\n' "$VERSION_OUTPUT" | sed -E 's/.*version v?([0-9]+\.[0-9]+(\.[0-9]+)?).*/\1/')"
if [ -z "$VERSION" ] || [ "$VERSION" = "$VERSION_OUTPUT" ]; then
    cat >&2 <<EOF
error: could not determine golangci-lint version from:
    ${VERSION_OUTPUT}

This repository requires golangci-lint v${REQUIRED_MAJOR}. Install it with:

    ${INSTALL_CMD}
EOF
    exit 1
fi

MAJOR="${VERSION%%.*}"
if [ "$MAJOR" != "$REQUIRED_MAJOR" ]; then
    cat >&2 <<EOF
error: golangci-lint v${MAJOR} detected (${VERSION}), but this repository's
.golangci.yml uses the v${REQUIRED_MAJOR} schema. Install v${REQUIRED_MAJOR}
with:

    ${INSTALL_CMD}
EOF
    exit 1
fi

exit 0
