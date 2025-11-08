// Package testutil provides testing utilities and helpers for the fog project.
//
// This package contains mock implementations of AWS services, test builders,
// assertion helpers, and utilities for managing test fixtures and golden files.
// It is designed to make testing fog's CloudFormation operations easier and more
// consistent across the codebase.
//
// Mock AWS Clients
//
// The package provides mock implementations for AWS service clients:
//   - MockCFNClient: Mock CloudFormation client for stack operations
//   - MockS3Client: Mock S3 client for template upload/download
//   - MockEC2Client: Mock EC2 client for VPC and subnet operations
//
// These mocks allow testing without making actual AWS API calls and support
// configuring responses for different test scenarios.
//
// Test Builders
//
// Builder types provide a fluent interface for constructing test data:
//   - StackBuilder: Create CloudFormation stack objects with custom attributes
//   - StackEventBuilder: Create stack event objects for testing event handling
//   - ChangesetBuilder: Create changeset objects for deployment tests
//
// Builders support method chaining and provide sensible defaults.
//
// Assertions
//
// Helper functions for common test assertions:
//   - AssertError/AssertNoError: Verify error conditions
//   - AssertStackStatus: Check stack status values
//   - AssertContains/AssertNotContains: String content checks
//   - AssertEqual: Value equality checks
//
// These assertions provide clear, consistent error messages when tests fail.
//
// Test Fixtures
//
// Utilities for managing test data files:
//   - LoadFixture: Read fixture files from the testdata directory
//   - SaveFixture: Write test data for debugging
//   - FixturePath: Resolve paths to fixture files
//
// Golden Files
//
// Golden file testing for comparing output against expected results:
//   - GoldenFile: Manage golden file comparisons
//   - UpdateGolden: Update golden files when output changes
//   - CompareGolden: Compare actual output against golden file
//
// Golden files are stored in testdata/golden/ and can be updated with
// the -update flag when running tests.
//
// Test Context
//
// The TestContext type provides a standard setup for tests including:
//   - Context with cancellation
//   - Mock AWS clients
//   - Golden file management
//   - Temporary directory handling
//
// Examples
//
// Using mock clients:
//
//	mockCFN := &testutil.MockCFNClient{
//	    DescribeStacksFunc: func(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
//	        return &cloudformation.DescribeStacksOutput{
//	            Stacks: []types.Stack{/* ... */},
//	        }, nil
//	    },
//	}
//
// Using builders:
//
//	stack := testutil.NewStackBuilder().
//	    WithName("my-stack").
//	    WithStatus(types.StackStatusCreateComplete).
//	    Build()
//
// Using assertions:
//
//	testutil.AssertNoError(t, err)
//	testutil.AssertEqual(t, expected, actual)
//
// Using golden files:
//
//	gf := testutil.NewGoldenFile(t, "output.json")
//	gf.Compare(actualOutput)
package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// TestContext provides a standard test context with common setup
type TestContext struct {
	T           *testing.T
	Ctx         context.Context
	Cancel      context.CancelFunc
	MockClients MockClients
	GoldenFile  *GoldenFile
	TempDir     string
}

// MockClients holds all mock AWS clients for testing
type MockClients struct {
	CFN *MockCFNClient
	EC2 *MockEC2Client
	S3  *MockS3Client
}

// NewTestContext creates a new test context with standard setup
func NewTestContext(t *testing.T) *TestContext {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	tempDir := t.TempDir()

	return &TestContext{
		T:      t,
		Ctx:    ctx,
		Cancel: cancel,
		MockClients: MockClients{
			CFN: NewMockCFNClient(),
			EC2: NewMockEC2Client(),
			S3:  NewMockS3Client(),
		},
		GoldenFile: NewGoldenFile(t),
		TempDir:    tempDir,
	}
}

// Cleanup cleans up test resources
func (tc *TestContext) Cleanup() {
	tc.Cancel()
}

// WithTimeout creates a new context with the specified timeout
func (tc *TestContext) WithTimeout(duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(tc.Ctx, duration)
}

// CreateTempFile creates a temporary file with the given content
func (tc *TestContext) CreateTempFile(name, content string) string {
	tc.T.Helper()

	path := filepath.Join(tc.TempDir, name)
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		tc.T.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		tc.T.Fatalf("Failed to write temp file %s: %v", path, err)
	}

	return path
}

// ReadTempFile reads the content of a temporary file
func (tc *TestContext) ReadTempFile(name string) string {
	tc.T.Helper()

	path := filepath.Join(tc.TempDir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		tc.T.Fatalf("Failed to read temp file %s: %v", path, err)
	}

	return string(data)
}

// CaptureOutput captures stdout/stderr during function execution
func CaptureOutput(fn func()) (string, error) {
	// Save current stdout/stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	// Create pipe for capturing output
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	// Redirect stdout and stderr to the pipe
	os.Stdout = w
	os.Stderr = w

	// Channel to capture output
	outputChan := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outputChan <- buf.String()
	}()

	// Execute the function
	fn()

	// Close the writer and wait for output
	w.Close()
	output := <-outputChan

	return output, nil
}

// LoadFixture loads a fixture file from the testdata directory
func LoadFixture(t *testing.T, path string) []byte {
	t.Helper()

	fullPath := filepath.Join("testdata", path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to load fixture %s: %v", fullPath, err)
	}

	return data
}

// LoadFixtureString loads a fixture file as a string
func LoadFixtureString(t *testing.T, path string) string {
	t.Helper()
	return string(LoadFixture(t, path))
}

// LoadJSONFixture loads and unmarshals a JSON fixture
func LoadJSONFixture(t *testing.T, path string, target any) {
	t.Helper()

	data := LoadFixture(t, path)
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("Failed to unmarshal JSON fixture %s: %v", path, err)
	}
}

// CreateTestAWSConfig creates a test AWS config
func CreateTestAWSConfig() aws.Config {
	return aws.Config{
		Region: "us-west-2",
		Credentials: aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     "test-access-key",
				SecretAccessKey: "test-secret-key",
				SessionToken:    "test-session-token",
			}, nil
		}),
	}
}

// SkipIfIntegration skips the test if not running integration tests
func SkipIfIntegration(t *testing.T) {
	t.Helper()

	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("Skipping integration test (set INTEGRATION=1 to run)")
	}
}

// SkipIfShort skips the test if running in short mode
func SkipIfShort(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
}

// RequireEnvVar ensures an environment variable is set or skips the test
func RequireEnvVar(t *testing.T, name string) string {
	t.Helper()

	value := os.Getenv(name)
	if value == "" {
		t.Skipf("Skipping test: environment variable %s not set", name)
	}

	return value
}

// SetEnvVar sets an environment variable for the duration of the test
func SetEnvVar(t *testing.T, name, value string) {
	t.Helper()

	oldValue, exists := os.LookupEnv(name)
	os.Setenv(name, value)

	t.Cleanup(func() {
		if exists {
			os.Setenv(name, oldValue)
		} else {
			os.Unsetenv(name)
		}
	})
}

// CompareJSON compares two JSON strings, ignoring formatting
func CompareJSON(t *testing.T, got, want string) {
	t.Helper()

	var gotJSON, wantJSON any
	if err := json.Unmarshal([]byte(got), &gotJSON); err != nil {
		t.Fatalf("Failed to unmarshal got JSON: %v", err)
	}

	if err := json.Unmarshal([]byte(want), &wantJSON); err != nil {
		t.Fatalf("Failed to unmarshal want JSON: %v", err)
	}

	AssertEqual(t, gotJSON, wantJSON)
}

// FormatJSON formats a JSON string for readable output
func FormatJSON(jsonStr string) string {
	var data any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr // Return as-is if not valid JSON
	}

	formatted, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return jsonStr // Return as-is if formatting fails
	}

	return string(formatted)
}

// GenerateTestID generates a unique test ID
func GenerateTestID(prefix string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s-%d", prefix, timestamp)
}

// WaitForCondition waits for a condition to be true or times out
func WaitForCondition(t *testing.T, timeout time.Duration, check func() bool, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("Timeout waiting for condition: %s", message)
}

// NormalizeWhitespace normalizes whitespace in strings for comparison
func NormalizeWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	var normalized []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}

	return strings.Join(normalized, "\n")
}

// MockFile creates a mock file interface for testing
type MockFile struct {
	Content []byte
	Error   error
	Closed  bool
}

// Read implements io.Reader
func (m *MockFile) Read(p []byte) (n int, err error) {
	if m.Error != nil {
		return 0, m.Error
	}
	return copy(p, m.Content), io.EOF
}

// Close implements io.Closer
func (m *MockFile) Close() error {
	m.Closed = true
	return m.Error
}

// RunParallel runs a test in parallel if not disabled
func RunParallel(t *testing.T, fn func(t *testing.T)) {
	t.Helper()

	if os.Getenv("NO_PARALLEL") != "1" {
		t.Parallel()
	}

	fn(t)
}

// MustMarshalJSON marshals a value to JSON or fails the test
func MustMarshalJSON(t *testing.T, v any) []byte {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	return data
}

// MustUnmarshalJSON unmarshals JSON or fails the test
func MustUnmarshalJSON(t *testing.T, data []byte, v any) {
	t.Helper()

	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
}

// TableTestCase represents a standard table test case structure
type TableTestCase struct {
	Name       string
	Setup      func(*TestContext)
	Input      any
	Want       any
	WantErr    bool
	WantErrMsg string
	Skip       string // Skip message if test should be skipped
}

// RunTableTests runs a set of table-driven tests
func RunTableTests(t *testing.T, tests map[string]TableTestCase, testFunc func(*TestContext, TableTestCase)) {
	for name, tc := range tests {
		// Capture range variable
		t.Run(name, func(t *testing.T) {
			if tc.Skip != "" {
				t.Skip(tc.Skip)
			}

			RunParallel(t, func(t *testing.T) {
				ctx := NewTestContext(t)
				defer ctx.Cleanup()

				if tc.Setup != nil {
					tc.Setup(ctx)
				}

				testFunc(ctx, tc)
			})
		})
	}
}
