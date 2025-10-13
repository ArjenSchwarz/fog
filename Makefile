# fog Makefile
# Provides standard targets for common development tasks

# Version information
VERSION ?= dev
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags for version injection
LDFLAGS := -X github.com/ArjenSchwarz/fog/cmd.Version=$(VERSION) \
           -X github.com/ArjenSchwarz/fog/cmd.BuildTime=$(BUILD_TIME) \
           -X github.com/ArjenSchwarz/fog/cmd.GitCommit=$(GIT_COMMIT)

# Build the fog application
build:
	go build -ldflags "$(LDFLAGS)" .

# Build the fog application with version information
build-release:
	@if [ "$(VERSION)" = "dev" ]; then \
		echo "Error: VERSION must be set for release builds. Usage: make build-release VERSION=1.2.3"; \
		exit 1; \
	fi
	go build -ldflags "$(LDFLAGS)" -o fog .

# Run all tests
test:
	go test ./...

# Run tests with verbose output and coverage
test-verbose:
	go test -v -cover ./...

# Run only integration tests
test-integration:
	INTEGRATION=1 go test ./...

# Run integration tests with verbose output
test-integration-verbose:
	INTEGRATION=1 go test -v ./...

# Run both unit and integration tests
test-all: test test-integration

# Generate test coverage report
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

# Run benchmark tests
benchmarks:
	go test -bench=. ./...

# Run benchmarks with memory profiling
benchmarks-mem:
	go test -bench=. -benchmem ./...

# Run benchmarks multiple times for statistical analysis
benchmarks-stats:
	@echo "Running benchmarks 10 times for statistical analysis..."
	go test -bench=. -benchmem -count=10 ./lib/plan/ > bench_results.txt
	@echo "Results saved to bench_results.txt"
	@echo "Use 'benchstat bench_results.txt' for statistical analysis"

# Run only analysis benchmarks
benchmarks-analysis:
	go test -bench=BenchmarkAnalysis -benchmem ./lib/plan/

# Run only formatting benchmarks
benchmarks-formatting:
	go test -bench=BenchmarkFormatting -benchmem ./lib/plan/

# Run only property analysis benchmarks
benchmarks-property:
	go test -bench=BenchmarkPropertyAnalysis -benchmem ./lib/plan/

# Run performance validation tests
test-performance:
	go test -run=TestPerformanceTargets ./lib/plan/

# Run memory usage tests
test-memory:
	go test -run=TestMemoryUsage ./lib/plan/

# Run golden file tests
test-golden:
	go test ./cmd/... -run Golden

# Update golden files (use when output format changes intentionally)
test-golden-update:
	go test ./cmd/... -run Golden -update
	@echo "Golden files updated in cmd/testdata/golden/cmd/"

# Run golden file tests with verbose output
test-golden-verbose:
	go test ./cmd/... -run Golden -v

# Verify golden files exist and tests pass
test-golden-check: test-golden
	@echo "Verifying golden files exist..."
	@test -d cmd/testdata/golden/cmd || (echo "Error: Golden file directory not found" && exit 1)
	@test -n "$$(ls -A cmd/testdata/golden/cmd 2>/dev/null)" || (echo "Error: No golden files found" && exit 1)
	@echo "✓ Golden files verified"

# Compare benchmark results (requires BASELINE and CURRENT files)
benchmarks-compare:
	@if [ -z "$(BASELINE)" ] || [ -z "$(CURRENT)" ]; then \
		echo "Error: Both BASELINE and CURRENT parameters required."; \
		echo "Usage: make benchmarks-compare BASELINE=bench_before.txt CURRENT=bench_after.txt"; \
		exit 1; \
	fi
	@if [ ! -f "$(BASELINE)" ] || [ ! -f "$(CURRENT)" ]; then \
		echo "Error: Benchmark files not found"; \
		exit 1; \
	fi
	benchstat $(BASELINE) $(CURRENT)

# Format Go code
fmt:
	go fmt ./...

# Run go vet for static analysis
vet:
	go vet ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Run modernize to update code to modern Go patterns (requires modernize)
modernize:
	@which modernize > /dev/null || (echo "modernize not installed. Run: go install github.com/gaissmai/modernize@latest" && exit 1)
	modernize -fix -test ./...

# Run full validation suite
check: fmt vet lint test

# Clean build artifacts
clean:
	rm -f fog
	rm -rf dist/
	rm -f coverage.out coverage.html

# Install the application
install:
	go install .

# Clean up go.mod and go.sum
deps-tidy:
	go mod tidy

# Update dependencies to latest versions
deps-update:
	go get -u ./...
	go mod tidy

# Run security scan (requires gosec)
security-scan:
	@which gosec > /dev/null || (echo "gosec not installed. Run: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest" && exit 1)
	gosec ./...

go-functions:
	@echo "Finding all functions in the project..."
	@grep -r "^func " . --include="*.go" | grep -v vendor/

# Update v1 tag to latest commit and push to GitHub
update-v1-tag:
	git tag -f v1
	git push origin v1 --force

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build                 - Build the fog application with version info"
	@echo "  build-release         - Build release version (requires VERSION=x.y.z)"
	@echo "  install               - Install the application"
	@echo "  clean                 - Clean build artifacts and coverage files"
	@echo ""
	@echo "Testing targets:"
	@echo "  test                  - Run Go unit tests"
	@echo "  test-verbose          - Run tests with verbose output and coverage"
	@echo "  test-integration      - Run only integration tests"
	@echo "  test-integration-verbose - Run integration tests with verbose output"
	@echo "  test-all              - Run both unit and integration tests"
	@echo "  test-coverage         - Generate test coverage report (HTML)"
	@echo "  test-performance      - Run performance validation tests"
	@echo "  test-memory           - Run memory usage tests"
	@echo "  test-golden           - Run golden file tests for output validation"
	@echo "  test-golden-update    - Update golden files (use when output format changes)"
	@echo "  test-golden-verbose   - Run golden file tests with verbose output"
	@echo "  test-golden-check     - Verify golden files exist and tests pass"
	@echo ""
	@echo "Benchmark targets:"
	@echo "  benchmarks            - Run benchmark tests"
	@echo "  benchmarks-mem        - Run benchmarks with memory profiling"
	@echo "  benchmarks-stats      - Run benchmarks 10 times for statistical analysis"
	@echo "  benchmarks-analysis   - Run only analysis benchmarks"
	@echo "  benchmarks-formatting - Run only formatting benchmarks"
	@echo "  benchmarks-property   - Run only property analysis benchmarks"
	@echo "  benchmarks-compare    - Compare benchmark results (requires BASELINE and CURRENT files)"
	@echo ""
	@echo "Code quality targets:"
	@echo "  fmt                   - Format Go code"
	@echo "  vet                   - Run go vet for static analysis"
	@echo "  lint                  - Run linter (requires golangci-lint)"
	@echo "  modernize             - Update code to modern Go patterns (requires modernize)"
	@echo "  check                 - Run full validation suite (fmt, vet, lint, test)"
	@echo "  security-scan         - Run security analysis (requires gosec)"
	@echo ""
	@echo "Dependency management:"
	@echo "  deps-tidy             - Clean up go.mod and go.sum"
	@echo "  deps-update           - Update dependencies to latest versions"
	@echo ""
	@echo "Development utilities:"
	@echo "  go-functions          - List all Go functions in the project"
	@echo "  update-v1-tag         - Update v1 tag to latest commit and push to GitHub"
	@echo "  help                  - Show this help message"
	@echo ""
	@echo "Build examples:"
	@echo "  make build                    - Build with dev version"
	@echo "  make build VERSION=1.2.3     - Build with specific version"
	@echo "  make build-release VERSION=1.2.3 - Build release version"

# Declare all targets as phony (not real files)
.PHONY: $(MAKECMDGOALS) build build-release test test-verbose test-integration \
	test-integration-verbose test-all test-coverage test-performance test-memory \
	test-golden test-golden-update test-golden-verbose test-golden-check benchmarks \
	benchmarks-mem benchmarks-stats benchmarks-analysis benchmarks-formatting \
	benchmarks-property benchmarks-compare fmt vet lint modernize check clean install \
	deps-tidy deps-update security-scan go-functions update-v1-tag help
