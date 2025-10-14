#!/bin/bash
# Test validation script for fog project
# Runs all tests with coverage, race detection, formatting checks, and linting

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=========================================="
echo "Fog Test Validation Suite"
echo "=========================================="
echo

# Function to print status
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
    else
        echo -e "${RED}✗${NC} $2"
        exit 1
    fi
}

# 1. Run go fmt check
echo "Step 1: Checking code formatting..."
UNFORMATTED=$(find . -type f -name "*.go" -not -path "./.git/*" -not -path "./vendor/*" -exec gofmt -l {} \;)
if [ -z "$UNFORMATTED" ]; then
    print_status 0 "Code formatting check passed"
else
    echo -e "${RED}✗${NC} Code formatting check failed"
    echo "Unformatted files:"
    echo "$UNFORMATTED"
    echo
    echo "Run 'go fmt ./...' to fix formatting issues"
    exit 1
fi
echo

# 2. Run unit tests with coverage
echo "Step 2: Running unit tests with coverage..."
go test ./... -coverprofile=coverage.out -covermode=atomic
TEST_EXIT=$?
print_status $TEST_EXIT "Unit tests"
echo

# 3. Run race detection
echo "Step 3: Running tests with race detection..."
go test -race ./... > /dev/null 2>&1
RACE_EXIT=$?
print_status $RACE_EXIT "Race detection"
echo

# 4. Generate coverage HTML report
echo "Step 4: Generating coverage HTML report..."
go tool cover -html=coverage.out -o coverage.html
print_status 0 "Coverage HTML report generated (coverage.html)"
echo

# 5. Display coverage summary
echo "Step 5: Coverage Summary"
echo "----------------------------------------"
go tool cover -func=coverage.out | grep total:
echo
echo "Detailed per-package coverage:"
go tool cover -func=coverage.out | grep -E "^(github.com|total:)" | grep -v "total:"
echo

# 6. Run golangci-lint if available
echo "Step 6: Running golangci-lint..."
if command -v golangci-lint &> /dev/null; then
    golangci-lint run
    LINT_EXIT=$?
    print_status $LINT_EXIT "Linting"
else
    echo -e "${YELLOW}⚠${NC} golangci-lint not found (skipping)"
fi
echo

# 7. Run integration tests if INTEGRATION=1
if [ "$INTEGRATION" = "1" ]; then
    echo "Step 7: Running integration tests..."
    INTEGRATION=1 go test ./... -v -tags=integration
    INT_EXIT=$?
    print_status $INT_EXIT "Integration tests"
    echo
fi

echo "=========================================="
echo -e "${GREEN}All validation checks passed!${NC}"
echo "=========================================="
echo
echo "Coverage report available at: coverage.html"
echo "Coverage data available at: coverage.out"
