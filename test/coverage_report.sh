#!/bin/bash
# Coverage reporting script for fog project
# Generates detailed coverage analysis with per-package breakdown and exclusions

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "=========================================="
echo "Fog Coverage Analysis Report"
echo "=========================================="
echo

# Generate coverage data
echo "Generating coverage data..."
go test ./... -coverprofile=coverage.out -covermode=atomic > /dev/null 2>&1
echo

# Extract package coverage data
echo "=========================================="
echo "Per-Package Coverage Analysis"
echo "=========================================="
echo

# Coverage targets
LIB_TARGET=80.0
CONFIG_TARGET=80.0
CMD_HELPERS_TARGET=75.0
OVERALL_TARGET=80.0

# Function to extract coverage for a package
get_package_coverage() {
    local package=$1
    go tool cover -func=coverage.out | grep "^$package/" | grep -v "total:" | awk '{sum+=$3; count++} END {if(count>0) printf "%.1f", sum/count; else print "0.0"}'
}

# Function to get statement count for a package
get_statement_count() {
    local package=$1
    go tool cover -func=coverage.out | grep "^$package/" | grep -v "total:" | awk '{sum+=$2} END {print sum}'
}

# Calculate package coverages
lib_coverage=$(go test ./lib/... -coverprofile=/tmp/lib_coverage.out -covermode=atomic 2>&1 | grep "github.com/ArjenSchwarz/fog/lib" | grep "coverage:" | grep -v "testutil" | grep -v "texts" | awk '{print $5}' | tr -d '%')
config_coverage=$(go test ./config/... -coverprofile=/tmp/config_coverage.out -covermode=atomic 2>&1 | grep "coverage:" | awk '{print $5}' | tr -d '%')
cmd_coverage=$(go test ./cmd/... -coverprofile=/tmp/cmd_coverage.out -covermode=atomic 2>&1 | grep "coverage:" | awk '{print $5}' | tr -d '%')

# Function to compare coverage with target
check_target() {
    local actual=$1
    local target=$2
    local package=$3

    if (( $(echo "$actual >= $target" | bc -l) )); then
        echo -e "${GREEN}✓${NC} $package: ${GREEN}$actual%${NC} (target: $target%)"
        return 0
    else
        local diff=$(echo "$target - $actual" | bc)
        echo -e "${RED}✗${NC} $package: ${YELLOW}$actual%${NC} (target: $target%, ${RED}need +$diff%${NC})"
        return 1
    fi
}

# Check each package against targets
targets_met=0
targets_total=0

echo "Package Coverage vs Targets:"
echo "----------------------------"

if [ ! -z "$lib_coverage" ]; then
    targets_total=$((targets_total + 1))
    if check_target "$lib_coverage" "$LIB_TARGET" "lib"; then
        targets_met=$((targets_met + 1))
    fi
fi

if [ ! -z "$config_coverage" ]; then
    targets_total=$((targets_total + 1))
    if check_target "$config_coverage" "$CONFIG_TARGET" "config"; then
        targets_met=$((targets_met + 1))
    fi
fi

if [ ! -z "$cmd_coverage" ]; then
    targets_total=$((targets_total + 1))
    # Note: cmd target is for helpers only, not overall cmd coverage
    echo -e "${BLUE}ℹ${NC} cmd: ${BLUE}$cmd_coverage%${NC} (helper functions target: $CMD_HELPERS_TARGET%)"
fi

echo

# Calculate weighted overall coverage
echo "=========================================="
echo "Overall Coverage"
echo "=========================================="
echo

overall_coverage=$(go test ./... -coverprofile=coverage.out -covermode=atomic 2>&1 | grep "total:" | awk '{print $3}' | tr -d '%')

if [ ! -z "$overall_coverage" ]; then
    targets_total=$((targets_total + 1))
    if check_target "$overall_coverage" "$OVERALL_TARGET" "Overall"; then
        targets_met=$((targets_met + 1))
    fi
fi

echo

# Excluded functions documentation
echo "=========================================="
echo "Excluded Functions (Coverage Exemptions)"
echo "=========================================="
echo
echo "The following functions are excluded from coverage requirements:"
echo

cat << 'EOF'
1. cmd/deploy.go (409 lines)
   - Function: deployTemplate()
   - Reason: Large orchestration function (>350 lines, >80% AWS API calls)
   - Recommendation: Refactor into smaller, testable functions
   - Justification: Heavy Cobra framework integration, minimal testable business logic

2. cmd/report.go (360 lines)
   - Function: reportCmd() and related reporting functions
   - Reason: Complex reporting orchestration with heavy AWS API usage
   - Recommendation: Extract report generation logic into separate testable functions
   - Justification: AWS API orchestration, complex output formatting

3. cmd/drift.go
   - Function: detectDrift()
   - Reason: Large orchestration function with AWS SDK integration
   - Recommendation: Extract drift analysis logic into lib package
   - Justification: Minimal testable business logic, heavy AWS integration

4. cmd/describe_changeset.go
   - Functions: describeChangeset(), showChangeset()
   - Reason: Cobra command orchestration
   - Recommendation: Extract changeset formatting into lib package
   - Justification: Framework integration, output-focused

5. cmd/exports.go
   - Function: listExports()
   - Reason: Simple AWS API wrapper with output formatting
   - Recommendation: Extract filtering logic if it grows
   - Justification: Minimal business logic

6. Other cmd/*_cmd.go main command functions
   - Reason: Cobra command setup and orchestration
   - Recommendation: Focus on testing extracted helper functions
   - Justification: Framework integration code
EOF

echo
echo

# Coverage gaps and improvement opportunities
echo "=========================================="
echo "Coverage Gaps & Improvement Opportunities"
echo "=========================================="
echo

# Identify low coverage files in lib package
echo "Low Coverage Files in lib Package (< 80%):"
echo "-------------------------------------------"
go tool cover -func=/tmp/lib_coverage.out 2>/dev/null | grep -E "^github.com/ArjenSchwarz/fog/lib/" | grep -v "testutil" | awk '{
    file=$1;
    coverage=$3;
    gsub(/%/, "", coverage);
    if (coverage < 80.0 && coverage > 0) {
        printf "%-60s %6.1f%%\n", file, coverage
    }
}' | sort -t: -k2 -n || echo "No low coverage files found (or lib coverage data not available)"

echo
echo "Low Coverage Files in config Package (< 80%):"
echo "----------------------------------------------"
go tool cover -func=/tmp/config_coverage.out 2>/dev/null | grep -E "^github.com/ArjenSchwarz/fog/config/" | awk '{
    file=$1;
    coverage=$3;
    gsub(/%/, "", coverage);
    if (coverage < 80.0 && coverage > 0) {
        printf "%-60s %6.1f%%\n", file, coverage
    }
}' | sort -t: -k2 -n || echo "No low coverage files found (or config coverage data not available)"

echo
echo "Untested Functions in lib Package (0% coverage):"
echo "------------------------------------------------"
go tool cover -func=/tmp/lib_coverage.out 2>/dev/null | grep -E "^github.com/ArjenSchwarz/fog/lib/" | grep "0.0%" | grep -v "testutil" | awk '{print $1}' | head -20 || echo "None found"

echo
echo

# Summary
echo "=========================================="
echo "Summary"
echo "=========================================="
echo
echo "Targets Met: $targets_met/$targets_total"
echo

if [ $targets_met -eq $targets_total ]; then
    echo -e "${GREEN}✓ All coverage targets met!${NC}"
    echo
    echo "Next Steps:"
    echo "  1. Review and address any remaining linting issues"
    echo "  2. Update documentation with testing patterns"
    echo "  3. Consider increasing coverage for files < 90%"
else
    echo -e "${YELLOW}⚠ Some coverage targets not met${NC}"
    echo
    echo "Next Steps:"
    echo "  1. Review low coverage files listed above"
    echo "  2. Add tests for untested functions"
    echo "  3. Ensure helper functions in cmd package have adequate coverage"
    echo "  4. Re-run this script after adding tests"
fi

echo
echo "Coverage data available at: coverage.out"
echo "HTML report available at: coverage.html (run: go tool cover -html=coverage.out -o coverage.html)"
echo "Per-package data: /tmp/{lib,config,cmd}_coverage.out"
