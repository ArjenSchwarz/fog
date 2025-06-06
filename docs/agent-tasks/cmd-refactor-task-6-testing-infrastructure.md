# Task 6: Testing Infrastructure Enhancement

## Objective

Establish comprehensive testing infrastructure for the refactored command layer, including unit tests, integration tests, mocking strategies, and testing utilities to ensure code quality and reliability.

## Current State

### Problems
- Limited test coverage for command layer functionality
- No standardized testing patterns or utilities
- Missing integration tests for command workflows
- No mocking infrastructure for external dependencies
- Inconsistent test organization and structure
- Lack of testing for error scenarios and edge cases

### Current Testing Implementation
- Basic tests scattered throughout the codebase
- No command-specific testing utilities
- Limited mocking of AWS services
- No integration test framework
- Manual testing for most command functionality
- Missing test data management

### Problematic Patterns
```go
// Current: Basic testing without proper setup
func TestDeploy(t *testing.T) {
    // Manual setup without utilities
    // No proper mocking
    // Limited assertions
}
```

## Target State

### Goals
- Comprehensive test coverage for all command components
- Standardized testing utilities and patterns
- Robust mocking infrastructure for external dependencies
- Integration tests for end-to-end command workflows
- Performance testing for critical operations
- Automated test data generation and management

### Testing Architecture
```
cmd/
├── testing/
│   ├── framework/
│   │   ├── command_test.go      # Command testing utilities
│   │   ├── fixtures.go          # Test data fixtures
│   │   ├── assertions.go        # Custom assertions
│   │   └── setup.go             # Test setup utilities
│   ├── mocks/
│   │   ├── aws_service.go       # AWS service mocks
│   │   ├── ui_handler.go        # UI handler mocks
│   │   ├── file_system.go       # File system mocks
│   │   └── config.go            # Configuration mocks
│   ├── testdata/
│   │   ├── templates/           # Test CloudFormation templates
│   │   ├── parameters/          # Test parameter files
│   │   ├── configs/            # Test configuration files
│   │   └── responses/          # Mock AWS responses
│   └── integration/
│       ├── deploy_test.go       # Deploy command integration tests
│       ├── drift_test.go        # Drift command integration tests
│       └── helpers.go           # Integration test helpers
```

## Prerequisites

- Task 1: Command Structure Reorganization (provides testable command structure)
- Task 2: Business Logic Extraction (provides service layer to mock)
- Task 3: Flag Management (provides flag validation to test)
- Task 4: Output and UI Standardization (provides UI components to mock)
- Task 5: Error Handling (provides structured errors to test)

## Step-by-Step Implementation

### Step 1: Create Testing Framework Foundation

**File**: `cmd/testing/framework/setup.go`

```go
package framework

import (
    "context"
    "io"
    "os"
    "path/filepath"
    "testing"
    "github.com/ArjenSchwarz/fog/cmd/errors"
    "github.com/ArjenSchwarz/fog/cmd/services"
    "github.com/ArjenSchwarz/fog/cmd/ui"
    "github.com/ArjenSchwarz/fog/config"
    "github.com/stretchr/testify/require"
)

// TestContext provides a complete testing environment
type TestContext struct {
    T              *testing.T
    TempDir        string
    MockAWS        *MockAWSService
    MockUI         *MockUIHandler
    MockConfig     *MockConfig
    TestData       *TestDataManager
    ErrorCollector *ErrorCollector
}

// NewTestContext creates a new test context with all necessary mocks
func NewTestContext(t *testing.T) *TestContext {
    tempDir, err := os.MkdirTemp("", "fog-test-*")
    require.NoError(t, err)

    ctx := &TestContext{
        T:              t,
        TempDir:        tempDir,
        MockAWS:        NewMockAWSService(),
        MockUI:         NewMockUIHandler(),
        MockConfig:     NewMockConfig(),
        TestData:       NewTestDataManager(tempDir),
        ErrorCollector: NewErrorCollector(),
    }

    // Setup cleanup
    t.Cleanup(func() {
        os.RemoveAll(tempDir)
    })

    return ctx
}

// WithTestData creates test files and returns their paths
func (tc *TestContext) WithTestData(files map[string]string) *TestContext {
    for filename, content := range files {
        tc.TestData.CreateFile(filename, content)
    }
    return tc
}

// WithAWSResponse configures mock AWS responses
func (tc *TestContext) WithAWSResponse(operation string, response interface{}) *TestContext {
    tc.MockAWS.SetResponse(operation, response)
    return tc
}

// WithUIExpectation sets UI interaction expectations
func (tc *TestContext) WithUIExpectation(expectation UIExpectation) *TestContext {
    tc.MockUI.AddExpectation(expectation)
    return tc
}

// Context returns a context with error handling setup
func (tc *TestContext) Context() context.Context {
    ctx := context.Background()
    errorCtx := errors.NewErrorContext("test", "framework")
    return errors.WithErrorContext(ctx, errorCtx)
}

// AssertNoErrors verifies no errors were collected
func (tc *TestContext) AssertNoErrors() {
    tc.ErrorCollector.AssertNoErrors(tc.T)
}

// AssertError verifies a specific error was collected
func (tc *TestContext) AssertError(code errors.ErrorCode) {
    tc.ErrorCollector.AssertHasError(tc.T, code)
}

// TestDataManager handles test file creation and management
type TestDataManager struct {
    baseDir string
    files   map[string]string
}

// NewTestDataManager creates a new test data manager
func NewTestDataManager(baseDir string) *TestDataManager {
    return &TestDataManager{
        baseDir: baseDir,
        files:   make(map[string]string),
    }
}

// CreateFile creates a test file with content
func (tdm *TestDataManager) CreateFile(relativePath, content string) string {
    fullPath := filepath.Join(tdm.baseDir, relativePath)

    // Create directory if needed
    dir := filepath.Dir(fullPath)
    os.MkdirAll(dir, 0755)

    // Write file
    err := os.WriteFile(fullPath, []byte(content), 0644)
    if err != nil {
        panic(err)
    }

    tdm.files[relativePath] = fullPath
    return fullPath
}

// GetFilePath returns the full path to a created file
func (tdm *TestDataManager) GetFilePath(relativePath string) string {
    if fullPath, exists := tdm.files[relativePath]; exists {
        return fullPath
    }
    return filepath.Join(tdm.baseDir, relativePath)
}

// ErrorCollector collects and verifies errors during testing
type ErrorCollector struct {
    errors []errors.FogError
}

// NewErrorCollector creates a new error collector
func NewErrorCollector() *ErrorCollector {
    return &ErrorCollector{
        errors: make([]errors.FogError, 0),
    }
}

// Collect adds an error to the collection
func (ec *ErrorCollector) Collect(err error) {
    if fogErr, ok := err.(errors.FogError); ok {
        ec.errors = append(ec.errors, fogErr)
    }
}

// AssertNoErrors verifies no errors were collected
func (ec *ErrorCollector) AssertNoErrors(t *testing.T) {
    require.Empty(t, ec.errors, "Expected no errors but got: %v", ec.errors)
}

// AssertHasError verifies a specific error was collected
func (ec *ErrorCollector) AssertHasError(t *testing.T, code errors.ErrorCode) {
    for _, err := range ec.errors {
        if err.Code() == code {
            return
        }
    }
    require.Failf(t, "Expected error not found", "Expected error code %s but got: %v", code, ec.errors)
}

// GetErrors returns all collected errors
func (ec *ErrorCollector) GetErrors() []errors.FogError {
    return ec.errors
}
```

### Step 2: Create Mock Infrastructure

**File**: `cmd/testing/mocks/aws_service.go`

```go
package mocks

import (
    "context"
    "fmt"
    "github.com/ArjenSchwarz/fog/cmd/services"
    "github.com/aws/aws-sdk-go-v2/service/cloudformation"
    "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
    "github.com/stretchr/testify/mock"
)

// MockAWSService provides a mock implementation of AWS services
type MockAWSService struct {
    mock.Mock
    responses map[string]interface{}
    calls     []string
}

// NewMockAWSService creates a new mock AWS service
func NewMockAWSService() *MockAWSService {
    return &MockAWSService{
        responses: make(map[string]interface{}),
        calls:     make([]string, 0),
    }
}

// SetResponse configures a mock response for an operation
func (m *MockAWSService) SetResponse(operation string, response interface{}) {
    m.responses[operation] = response
}

// GetCalls returns all operations that were called
func (m *MockAWSService) GetCalls() []string {
    return m.calls
}

// MockDeploymentService implements services.DeploymentService
type MockDeploymentService struct {
    *MockAWSService
}

// NewMockDeploymentService creates a new mock deployment service
func NewMockDeploymentService() *MockDeploymentService {
    return &MockDeploymentService{
        MockAWSService: NewMockAWSService(),
    }
}

// PrepareDeployment mocks deployment preparation
func (m *MockDeploymentService) PrepareDeployment(ctx context.Context, opts services.DeploymentOptions) (*services.DeploymentPlan, error) {
    m.calls = append(m.calls, "PrepareDeployment")

    if response, exists := m.responses["PrepareDeployment"]; exists {
        if err, ok := response.(error); ok {
            return nil, err
        }
        if plan, ok := response.(*services.DeploymentPlan); ok {
            return plan, nil
        }
    }

    // Default response
    return &services.DeploymentPlan{
        StackName:   opts.StackName,
        IsNewStack:  true,
        Options:     opts,
        Template:    services.TemplateInfo{LocalPath: opts.TemplateSource},
    }, nil
}

// ValidateDeployment mocks deployment validation
func (m *MockDeploymentService) ValidateDeployment(ctx context.Context, plan *services.DeploymentPlan) error {
    m.calls = append(m.calls, "ValidateDeployment")

    if response, exists := m.responses["ValidateDeployment"]; exists {
        if err, ok := response.(error); ok {
            return err
        }
    }

    return nil
}

// CreateChangeset mocks changeset creation
func (m *MockDeploymentService) CreateChangeset(ctx context.Context, plan *services.DeploymentPlan) (*services.ChangesetResult, error) {
    m.calls = append(m.calls, "CreateChangeset")

    if response, exists := m.responses["CreateChangeset"]; exists {
        if err, ok := response.(error); ok {
            return nil, err
        }
        if changeset, ok := response.(*services.ChangesetResult); ok {
            return changeset, nil
        }
    }

    // Default response
    return &services.ChangesetResult{
        Name:   "test-changeset",
        Status: types.ChangeSetStatusCreateComplete,
        Changes: []types.Change{},
    }, nil
}

// ExecuteDeployment mocks deployment execution
func (m *MockDeploymentService) ExecuteDeployment(ctx context.Context, plan *services.DeploymentPlan, changeset *services.ChangesetResult) (*services.DeploymentResult, error) {
    m.calls = append(m.calls, "ExecuteDeployment")

    if response, exists := m.responses["ExecuteDeployment"]; exists {
        if err, ok := response.(error); ok {
            return nil, err
        }
        if result, ok := response.(*services.DeploymentResult); ok {
            return result, nil
        }
    }

    // Default response
    return &services.DeploymentResult{
        Success:      true,
        StackID:      "test-stack-id",
        Outputs:      []types.Output{},
    }, nil
}

// MockDriftService implements services.DriftService
type MockDriftService struct {
    *MockAWSService
}

// NewMockDriftService creates a new mock drift service
func NewMockDriftService() *MockDriftService {
    return &MockDriftService{
        MockAWSService: NewMockAWSService(),
    }
}

// DetectDrift mocks drift detection
func (m *MockDriftService) DetectDrift(ctx context.Context, stackName string) (*services.DriftResult, error) {
    m.calls = append(m.calls, "DetectDrift")

    if response, exists := m.responses["DetectDrift"]; exists {
        if err, ok := response.(error); ok {
            return nil, err
        }
        if result, ok := response.(*services.DriftResult); ok {
            return result, nil
        }
    }

    // Default response
    return &services.DriftResult{
        StackName:      stackName,
        DriftStatus:    "IN_SYNC",
        TotalResources: 1,
        DriftedCount:   0,
        Resources:      []services.DriftedResource{},
    }, nil
}

// GetDriftDetails mocks drift details retrieval
func (m *MockDriftService) GetDriftDetails(ctx context.Context, stackName string) (*services.DriftDetails, error) {
    m.calls = append(m.calls, "GetDriftDetails")

    if response, exists := m.responses["GetDriftDetails"]; exists {
        if err, ok := response.(error); ok {
            return nil, err
        }
        if details, ok := response.(*services.DriftDetails); ok {
            return details, nil
        }
    }

    // Default response
    return &services.DriftDetails{
        StackName: stackName,
        Resources: []services.ResourceDrift{},
    }, nil
}
```

**File**: `cmd/testing/mocks/ui_handler.go`

```go
package mocks

import (
    "io"
    "strings"
    "testing"
    "github.com/ArjenSchwarz/fog/cmd/ui"
    "github.com/stretchr/testify/require"
)

// MockUIHandler provides a mock implementation of ui.OutputHandler
type MockUIHandler struct {
    expectations []UIExpectation
    messages     []UIMessage
    confirmResponses []bool
    quiet        bool
    verbose      bool
    format       ui.OutputFormat
}

// UIExpectation represents an expected UI interaction
type UIExpectation struct {
    Type     UIMessageType
    Message  string
    Required bool
}

// UIMessage represents a captured UI message
type UIMessage struct {
    Type    UIMessageType
    Message string
}

// UIMessageType represents the type of UI message
type UIMessageType int

const (
    UIMessageSuccess UIMessageType = iota
    UIMessageInfo
    UIMessageWarning
    UIMessageError
    UIMessageDebug
    UIMessageConfirm
)

// NewMockUIHandler creates a new mock UI handler
func NewMockUIHandler() *MockUIHandler {
    return &MockUIHandler{
        expectations:     make([]UIExpectation, 0),
        messages:         make([]UIMessage, 0),
        confirmResponses: make([]bool, 0),
    }
}

// AddExpectation adds a UI expectation
func (m *MockUIHandler) AddExpectation(expectation UIExpectation) {
    m.expectations = append(m.expectations, expectation)
}

// SetConfirmResponse sets the response for confirmation prompts
func (m *MockUIHandler) SetConfirmResponse(response bool) {
    m.confirmResponses = append(m.confirmResponses, response)
}

// Success captures success messages
func (m *MockUIHandler) Success(message string) {
    m.messages = append(m.messages, UIMessage{
        Type:    UIMessageSuccess,
        Message: message,
    })
}

// Info captures info messages
func (m *MockUIHandler) Info(message string) {
    if m.quiet {
        return
    }
    m.messages = append(m.messages, UIMessage{
        Type:    UIMessageInfo,
        Message: message,
    })
}

// Warning captures warning messages
func (m *MockUIHandler) Warning(message string) {
    m.messages = append(m.messages, UIMessage{
        Type:    UIMessageWarning,
        Message: message,
    })
}

// Error captures error messages
func (m *MockUIHandler) Error(message string) {
    m.messages = append(m.messages, UIMessage{
        Type:    UIMessageError,
        Message: message,
    })
}

// Debug captures debug messages
func (m *MockUIHandler) Debug(message string) {
    if !m.verbose {
        return
    }
    m.messages = append(m.messages, UIMessage{
        Type:    UIMessageDebug,
        Message: message,
    })
}

// Table captures table display requests
func (m *MockUIHandler) Table(data interface{}, options ui.TableOptions) error {
    m.messages = append(m.messages, UIMessage{
        Type:    UIMessageInfo,
        Message: "table_displayed",
    })
    return nil
}

// JSON captures JSON display requests
func (m *MockUIHandler) JSON(data interface{}) error {
    m.messages = append(m.messages, UIMessage{
        Type:    UIMessageInfo,
        Message: "json_displayed",
    })
    return nil
}

// StartProgress creates a mock progress indicator
func (m *MockUIHandler) StartProgress(message string) ui.ProgressIndicator {
    return &MockProgressIndicator{
        ui:      m,
        message: message,
    }
}

// SetStatus captures status messages
func (m *MockUIHandler) SetStatus(message string) {
    if m.quiet {
        return
    }
    m.messages = append(m.messages, UIMessage{
        Type:    UIMessageInfo,
        Message: message,
    })
}

// Confirm handles confirmation prompts
func (m *MockUIHandler) Confirm(message string) bool {
    return m.ConfirmWithDefault(message, false)
}

// ConfirmWithDefault handles confirmation prompts with defaults
func (m *MockUIHandler) ConfirmWithDefault(message string, defaultValue bool) bool {
    m.messages = append(m.messages, UIMessage{
        Type:    UIMessageConfirm,
        Message: message,
    })

    if len(m.confirmResponses) > 0 {
        response := m.confirmResponses[0]
        m.confirmResponses = m.confirmResponses[1:]
        return response
    }

    return defaultValue
}

// SetVerbose sets verbose mode
func (m *MockUIHandler) SetVerbose(verbose bool) {
    m.verbose = verbose
}

// SetQuiet sets quiet mode
func (m *MockUIHandler) SetQuiet(quiet bool) {
    m.quiet = quiet
}

// SetOutputFormat sets output format
func (m *MockUIHandler) SetOutputFormat(format ui.OutputFormat) {
    m.format = format
}

// GetWriter returns a string writer
func (m *MockUIHandler) GetWriter() io.Writer {
    return &strings.Builder{}
}

// GetErrorWriter returns a string writer
func (m *MockUIHandler) GetErrorWriter() io.Writer {
    return &strings.Builder{}
}

// Verification methods

// AssertExpectationsMet verifies all expectations were met
func (m *MockUIHandler) AssertExpectationsMet(t *testing.T) {
    for _, expectation := range m.expectations {
        if expectation.Required {
            found := false
            for _, message := range m.messages {
                if message.Type == expectation.Type && strings.Contains(message.Message, expectation.Message) {
                    found = true
                    break
                }
            }
            require.True(t, found, "Expected UI message not found: %v", expectation)
        }
    }
}

// AssertMessageContains verifies a message was displayed
func (m *MockUIHandler) AssertMessageContains(t *testing.T, msgType UIMessageType, content string) {
    for _, message := range m.messages {
        if message.Type == msgType && strings.Contains(message.Message, content) {
            return
        }
    }
    require.Failf(t, "Message not found", "Expected message of type %v containing '%s'", msgType, content)
}

// GetMessages returns all captured messages
func (m *MockUIHandler) GetMessages() []UIMessage {
    return m.messages
}

// MockProgressIndicator provides a mock progress indicator
type MockProgressIndicator struct {
    ui      *MockUIHandler
    message string
    stopped bool
}

// Update updates the progress message
func (m *MockProgressIndicator) Update(message string) {
    m.message = message
}

// Success completes with success
func (m *MockProgressIndicator) Success(message string) {
    m.stopped = true
    m.ui.Success(message)
}

// Error completes with error
func (m *MockProgressIndicator) Error(message string) {
    m.stopped = true
    m.ui.Error(message)
}

// Stop stops the progress indicator
func (m *MockProgressIndicator) Stop() {
    m.stopped = true
}
```

### Step 3: Create Test Fixtures and Data

**File**: `cmd/testing/framework/fixtures.go`

```go
package framework

import (
    "encoding/json"
    "fmt"
    "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// TestTemplates provides common CloudFormation template content
var TestTemplates = struct {
    SimpleVPC     string
    S3Bucket      string
    LambdaFunction string
    Invalid       string
}{
    SimpleVPC: `{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Description": "Simple VPC template for testing",
  "Resources": {
    "VPC": {
      "Type": "AWS::EC2::VPC",
      "Properties": {
        "CidrBlock": "10.0.0.0/16",
        "EnableDnsHostnames": true,
        "EnableDnsSupport": true
      }
    }
  },
  "Outputs": {
    "VPCId": {
      "Description": "VPC ID",
      "Value": {"Ref": "VPC"},
      "Export": {"Name": "TestVPC-ID"}
    }
  }
}`,

    S3Bucket: `{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Description": "Simple S3 bucket template",
  "Parameters": {
    "BucketName": {
      "Type": "String",
      "Description": "Name of the S3 bucket"
    }
  },
  "Resources": {
    "S3Bucket": {
      "Type": "AWS::S3::Bucket",
      "Properties": {
        "BucketName": {"Ref": "BucketName"}
      }
    }
  }
}`,

    LambdaFunction: `{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Resources": {
    "LambdaRole": {
      "Type": "AWS::IAM::Role",
      "Properties": {
        "AssumeRolePolicyDocument": {
          "Version": "2012-10-17",
          "Statement": [{
            "Effect": "Allow",
            "Principal": {"Service": "lambda.amazonaws.com"},
            "Action": "sts:AssumeRole"
          }]
        }
      }
    },
    "LambdaFunction": {
      "Type": "AWS::Lambda::Function",
      "Properties": {
        "Runtime": "python3.9",
        "Handler": "index.handler",
        "Role": {"Fn::GetAtt": ["LambdaRole", "Arn"]},
        "Code": {"ZipFile": "def handler(event, context): return 'Hello'"}
      }
    }
  }
}`,

    Invalid: `{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Resources": {
    "InvalidResource": {
      "Type": "AWS::INVALID::Resource"
    }
  }
}`,
}

// TestParameters provides common parameter file content
var TestParameters = struct {
    VPCParams    string
    S3Params     string
    EmptyParams  string
    InvalidParams string
}{
    VPCParams: `[
  {
    "ParameterKey": "Environment",
    "ParameterValue": "test"
  },
  {
    "ParameterKey": "VPCCidr",
    "ParameterValue": "10.0.0.0/16"
  }
]`,

    S3Params: `[
  {
    "ParameterKey": "BucketName",
    "ParameterValue": "test-bucket-12345"
  }
]`,

    EmptyParams: `[]`,

    InvalidParams: `[
  {
    "ParameterKey": "InvalidParam"
  }
]`,
}

// TestTags provides common tag file content
var TestTags = struct {
    ProjectTags   string
    EnvironmentTags string
    EmptyTags     string
}{
    ProjectTags: `[
  {
    "Key": "Project",
    "Value": "fog-testing"
  },
  {
    "Key": "Owner",
    "Value": "test-team"
  }
]`,

    EnvironmentTags: `[
  {
    "Key": "Environment",
    "Value": "test"
  },
  {
    "Key": "CostCenter",
    "Value": "12345"
  }
]`,

    EmptyTags: `[]`,
}

// TestConfigs provides common configuration file content
var TestConfigs = struct {
    BasicConfig    string
    CompleteConfig string
    InvalidConfig  string
}{
    BasicConfig: `defaulttags:
  - Key: Project
    Value: fog
  - Key: Environment
    Value: test`,

    CompleteConfig: `defaulttags:
  - Key: Project
    Value: fog
  - Key: Environment
    Value: test
outputs:
  table: true
  verbose: false
deployment:
  defaultbucket: fog-test-bucket
  autoconfirm: false`,

    InvalidConfig: `invalid: yaml: content`,
}

// MockAWSResponses provides mock AWS service responses
type MockAWSResponses struct{}

// GetMockChangeSet returns a mock changeset response
func (MockAWSResponses) GetMockChangeSet(stackName string, changes []types.Change) *types.DescribeChangeSetOutput {
    return &types.DescribeChangeSetOutput{
        ChangeSetName: stringPtr("test-changeset"),
        StackName:     &stackName,
        Status:        types.ChangeSetStatusCreateComplete,
        Changes:       changes,
    }
}

// GetMockStack returns a mock stack description
func (MockAWSResponses) GetMockStack(stackName string, status types.StackStatus) *types.Stack {
    return &types.Stack{
        StackName:   &stackName,
        StackStatus: status,
        StackId:     stringPtr(fmt.Sprintf("arn:aws:cloudformation:us-east-1:123456789012:stack/%s/test-id", stackName)),
    }
}

// GetMockStackResources returns mock stack resources
func (MockAWSResponses) GetMockStackResources(stackName string) []types.StackResource {
    return []types.StackResource{
        {
            LogicalResourceId:  stringPtr("VPC"),
            PhysicalResourceId: stringPtr("vpc-12345"),
            ResourceType:       stringPtr("AWS::EC2::VPC"),
            ResourceStatus:     types.ResourceStatusCreateComplete,
        },
    }
}

// GetMockDriftResult returns a mock drift detection result
func (MockAWSResponses) GetMockDriftResult(stackName string, drifted bool) *types.DescribeStackDriftDetectionStatusOutput {
    status := types.StackDriftStatusInSync
    if drifted {
        status = types.StackDriftStatusDrifted
    }

    return &types.DescribeStackDriftDetectionStatusOutput{
        StackId:            stringPtr(fmt.Sprintf("arn:aws:cloudformation:us-east-1:123456789012:stack/%s/test-id", stackName)),
        StackDriftStatus:   status,
        DetectionStatus:    types.StackDriftDetectionStatusDetectionComplete,
        DriftedStackResourceCount: intPtr(0),
    }
}

// Helper functions
func stringPtr(s string) *string {
    return &s
}

func intPtr(i int32) *int32 {
    return &i
}

// TestDataBuilder helps build test data structures
type TestDataBuilder struct {
    data map[string]interface{}
}

// NewTestDataBuilder creates a new test data builder
func NewTestDataBuilder() *TestDataBuilder {
    return &TestDataBuilder{
        data: make(map[string]interface{}),
    }
}

// WithField adds a field to the test data
func (tdb *TestDataBuilder) WithField(key string, value interface{}) *TestDataBuilder {
    tdb.data[key] = value
    return tdb
}

// Build returns the built test data
func (tdb *TestDataBuilder) Build() map[string]interface{} {
    return tdb.data
}

// ToJSON converts the test data to JSON
func (tdb *TestDataBuilder) ToJSON() string {
    data, _ := json.MarshalIndent(tdb.data, "", "  ")
    return string(data)
}
```

### Step 4: Create Command Testing Utilities

**File**: `cmd/testing/framework/command_test.go`

```go
package framework

import (
    "context"
    "testing"
    "github.com/ArjenSchwarz/fog/cmd/commands/deploy"
    "github.com/ArjenSchwarz/fog/cmd/flags/groups"
    "github.com/ArjenSchwarz/fog/cmd/registry"
    "github.com/ArjenSchwarz/fog/cmd/testing/mocks"
    "github.com/stretchr/testify/require"
)

// CommandTestSuite provides utilities for testing commands
type CommandTestSuite struct {
    *TestContext
    command registry.Command
    flags   interface{}
}

// NewCommandTestSuite creates a new command test suite
func NewCommandTestSuite(t *testing.T) *CommandTestSuite {
    return &CommandTestSuite{
        TestContext: NewTestContext(t),
    }
}

// WithDeployCommand sets up testing for the deploy command
func (cts *CommandTestSuite) WithDeployCommand() *CommandTestSuite {
    flags := groups.NewDeploymentFlags()
    handler := deploy.NewHandler(
        flags,
        mocks.NewMockDeploymentService(),
        cts.MockConfig,
        cts.MockUI,
    )

    cts.command = handler
    cts.flags = flags
    return cts
}

// WithFlags configures command flags
func (cts *CommandTestSuite) WithFlags(flagValues map[string]interface{}) *CommandTestSuite {
    if deployFlags, ok := cts.flags.(*groups.DeploymentFlags); ok {
        cts.setDeploymentFlags(deployFlags, flagValues)
    }
    return cts
}

// Execute runs the command and captures results
func (cts *CommandTestSuite) Execute() error {
    if handler, ok := cts.command.(interface{ Execute(context.Context) error }); ok {
        return handler.Execute(cts.Context())
    }
    return nil
}

// AssertSuccess verifies command executed successfully
func (cts *CommandTestSuite) AssertSuccess() *CommandTestSuite {
    err := cts.Execute()
    require.NoError(cts.T, err, "Command should execute successfully")
    return cts
}

// AssertError verifies command failed with expected error
func (cts *CommandTestSuite) AssertError(expectedError string) *CommandTestSuite {
    err := cts.Execute()
    require.Error(cts.T, err, "Command should fail")
    require.Contains(cts.T, err.Error(), expectedError, "Error message should contain expected text")
    return cts
}

// AssertUIMessage verifies a UI message was displayed
func (cts *CommandTestSuite) AssertUIMessage(msgType mocks.UIMessageType, content string) *CommandTestSuite {
    cts.MockUI.AssertMessageContains(cts.T, msgType, content)
    return cts
}

// AssertAWSCall verifies an AWS service was called
func (cts *CommandTestSuite) AssertAWSCall(operation string) *CommandTestSuite {
    if deployService, ok := cts.MockAWS.(*mocks.MockDeploymentService); ok {
        calls := deployService.GetCalls()
        require.Contains(cts.T, calls, operation, "Expected AWS operation was not called")
    }
    return cts
}

// setDeploymentFlags sets deployment-specific flags
func (cts *CommandTestSuite) setDeploymentFlags(flags *groups.DeploymentFlags, values map[string]interface{}) {
    for key, value := range values {
        switch key {
        case "stackname":
            if v, ok := value.(string); ok {
                flags.StackName = v
            }
        case "template":
            if v, ok := value.(string); ok {
                flags.Template = v
            }
        case "parameters":
            if v, ok := value.(string); ok {
                flags.Parameters = v
            }
        case "tags":
            if v, ok := value.(string); ok {
                flags.Tags = v
            }
        case "dry-run":
            if v, ok := value.(bool); ok {
                flags.Dryrun = v
            }
        case "non-interactive":
            if v, ok := value.(bool); ok {
                flags.NonInteractive = v
            }
        }
    }
}

// FlagValidationTestSuite provides utilities for testing flag validation
type FlagValidationTestSuite struct {
    *TestContext
    flags interface{}
}

// NewFlagValidationTestSuite creates a new flag validation test suite
func NewFlagValidationTestSuite(t *testing.T) *FlagValidationTestSuite {
    return &FlagValidationTestSuite{
        TestContext: NewTestContext(t),
    }
}

// WithDeploymentFlags sets up deployment flags for testing
func (fvts *FlagValidationTestSuite) WithDeploymentFlags() *FlagValidationTestSuite {
    fvts.flags = groups.NewDeploymentFlags()
    return fvts
}

// SetFlag sets a flag value
func (fvts *FlagValidationTestSuite) SetFlag(name string, value interface{}) *FlagValidationTestSuite {
    if deployFlags, ok := fvts.flags.(*groups.DeploymentFlags); ok {
        switch name {
        case "stackname":
            if v, ok := value.(string); ok {
                deployFlags.StackName = v
            }
        case "template":
            if v, ok := value.(string); ok {
                deployFlags.Template = v
            }
        }
    }
    return fvts
}

// AssertValidationSuccess verifies validation passes
func (fvts *FlagValidationTestSuite) AssertValidationSuccess() *FlagValidationTestSuite {
    if validator, ok := fvts.flags.(interface{ Validate(context.Context) error }); ok {
        err := validator.Validate(fvts.Context())
        require.NoError(fvts.T, err, "Flag validation should pass")
    }
    return fvts
}

// AssertValidationError verifies validation fails with expected error
func (fvts *FlagValidationTestSuite) AssertValidationError(expectedError string) *FlagValidationTestSuite {
    if validator, ok := fvts.flags.(interface{ Validate(context.Context) error }); ok {
        err := validator.Validate(fvts.Context())
        require.Error(fvts.T, err, "Flag validation should fail")
        require.Contains(fvts.T, err.Error(), expectedError, "Validation error should contain expected text")
    }
    return fvts
}
```

### Step 5: Create Integration Test Framework

**File**: `cmd/testing/integration/helpers.go`

```go
package integration

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    "github.com/ArjenSchwarz/fog/cmd/testing/framework"
    "github.com/stretchr/testify/require"
)

// IntegrationTestSuite provides end-to-end testing capabilities
type IntegrationTestSuite struct {
    *framework.TestContext
    WorkingDir string
}

// NewIntegrationTestSuite creates a new integration test suite
func NewIntegrationTestSuite(t *testing.T) *IntegrationTestSuite {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    ctx := framework.NewTestContext(t)

    // Create a working directory for the test
    workingDir := filepath.Join(ctx.TempDir, "working")
    err := os.MkdirAll(workingDir, 0755)
    require.NoError(t, err)

    return &IntegrationTestSuite{
        TestContext: ctx,
        WorkingDir:  workingDir,
    }
}

// SetupTemplateFiles creates the necessary template and parameter files
func (its *IntegrationTestSuite) SetupTemplateFiles() *IntegrationTestSuite {
    files := map[string]string{
        "template.json":    framework.TestTemplates.SimpleVPC,
        "parameters.json":  framework.TestParameters.VPCParams,
        "tags.json":        framework.TestTags.ProjectTags,
        "config.yaml":      framework.TestConfigs.BasicConfig,
    }

    its.WithTestData(files)
    return its
}

// SetupComplexScenario creates files for complex testing scenarios
func (its *IntegrationTestSuite) SetupComplexScenario() *IntegrationTestSuite {
    files := map[string]string{
        "templates/vpc.json":           framework.TestTemplates.SimpleVPC,
        "templates/s3.json":            framework.TestTemplates.S3Bucket,
        "templates/lambda.json":        framework.TestTemplates.LambdaFunction,
        "parameters/vpc-params.json":   framework.TestParameters.VPCParams,
        "parameters/s3-params.json":    framework.TestParameters.S3Params,
        "tags/project.json":            framework.TestTags.ProjectTags,
        "tags/environment.json":        framework.TestTags.EnvironmentTags,
        "configs/complete.yaml":        framework.TestConfigs.CompleteConfig,
        "deployment.yaml": `
stackName: test-stack
template: templates/vpc.json
parameters:
  - parameters/vpc-params.json
tags:
  - tags/project.json
  - tags/environment.json
`,
    }

    its.WithTestData(files)
    return its
}

// ExecuteCommand simulates command execution
func (its *IntegrationTestSuite) ExecuteCommand(command string, args ...string) error {
    // This would integrate with the actual command execution
    // For now, we'll simulate it
    switch command {
    case "deploy":
        return its.simulateDeployCommand(args)
    case "drift":
        return its.simulateDriftCommand(args)
    default:
        return nil
    }
}

// simulateDeployCommand simulates deploy command execution
func (its *IntegrationTestSuite) simulateDeployCommand(args []string) error {
    // Parse arguments and execute deploy logic
    // This would use the actual command infrastructure
    return nil
}

// simulateDriftCommand simulates drift command execution
func (its *IntegrationTestSuite) simulateDriftCommand(args []string) error {
    // Parse arguments and execute drift logic
    return nil
}

// AssertFilesCreated verifies expected files were created
func (its *IntegrationTestSuite) AssertFilesCreated(files []string) *IntegrationTestSuite {
    for _, file := range files {
        fullPath := filepath.Join(its.WorkingDir, file)
        require.FileExists(its.T, fullPath, "Expected file should exist: %s", file)
    }
    return its
}

// AssertFileContains verifies file contains expected content
func (its *IntegrationTestSuite) AssertFileContains(filename, content string) *IntegrationTestSuite {
    fullPath := filepath.Join(its.WorkingDir, filename)
    data, err := os.ReadFile(fullPath)
    require.NoError(its.T, err, "Should be able to read file: %s", filename)
    require.Contains(its.T, string(data), content, "File should contain expected content")
    return its
}

// PerformanceTestSuite provides performance testing capabilities
type PerformanceTestSuite struct {
    *framework.TestContext
    benchmarks map[string]func() error
}

// NewPerformanceTestSuite creates a new performance test suite
func NewPerformanceTestSuite(t *testing.T) *PerformanceTestSuite {
    return &PerformanceTestSuite{
        TestContext: framework.NewTestContext(t),
        benchmarks:  make(map[string]func() error),
    }
}

// AddBenchmark adds a benchmark test
func (pts *PerformanceTestSuite) AddBenchmark(name string, fn func() error) *PerformanceTestSuite {
    pts.benchmarks[name] = fn
    return pts
}

// RunBenchmarks executes all benchmarks
func (pts *PerformanceTestSuite) RunBenchmarks() *PerformanceTestSuite {
    for name, fn := range pts.benchmarks {
        pts.T.Run(name, func(t *testing.T) {
            // This would include timing and performance measurements
            err := fn()
            require.NoError(t, err, "Benchmark should complete without error")
        })
    }
    return pts
}
```

### Step 6: Create Example Test Files

**File**: `cmd/commands/deploy/handler_test.go`

```go
package deploy

import (
    "testing"
    "github.com/ArjenSchwarz/fog/cmd/testing/framework"
    "github.com/ArjenSchwarz/fog/cmd/testing/mocks"
    "github.com/stretchr/testify/require"
)

func TestDeployHandler_Execute_Success(t *testing.T) {
    suite := framework.NewCommandTestSuite(t).
        WithDeployCommand().
        WithTestData(map[string]string{
            "template.json": framework.TestTemplates.SimpleVPC,
        }).
        WithFlags(map[string]interface{}{
            "stackname":       "test-stack",
            "template":        "template.json",
            "non-interactive": true,
        })

    suite.AssertSuccess().
        AssertUIMessage(mocks.UIMessageSuccess, "deployed successfully").
        AssertAWSCall("PrepareDeployment").
        AssertAWSCall("CreateChangeset").
        AssertAWSCall("ExecuteDeployment")
}

func TestDeployHandler_Execute_DryRun(t *testing.T) {
    suite := framework.NewCommandTestSuite(t).
        WithDeployCommand().
        WithTestData(map[string]string{
            "template.json": framework.TestTemplates.SimpleVPC,
        }).
        WithFlags(map[string]interface{}{
            "stackname": "test-stack",
            "template":  "template.json",
            "dry-run":   true,
        })

    suite.AssertSuccess().
        AssertUIMessage(mocks.UIMessageSuccess, "Dry run completed").
        AssertAWSCall("PrepareDeployment").
        AssertAWSCall("CreateChangeset")

    // Should not call ExecuteDeployment in dry run mode
    require.NotContains(t, suite.MockAWS.GetCalls(), "ExecuteDeployment")
}

func TestDeployHandler_Execute_MissingTemplate(t *testing.T) {
    suite := framework.NewCommandTestSuite(t).
        WithDeployCommand().
        WithFlags(map[string]interface{}{
            "stackname": "test-stack",
            "template":  "nonexistent.json",
        })

    suite.AssertError("file not found")
}

func TestDeployHandler_Execute_InvalidTemplate(t *testing.T) {
    suite := framework.NewCommandTestSuite(t).
        WithDeployCommand().
        WithTestData(map[string]string{
            "invalid.json": framework.TestTemplates.Invalid,
        }).
        WithFlags(map[string]interface{}{
            "stackname": "test-stack",
            "template":  "invalid.json",
        }).
        WithAWSResponse("ValidateDeployment", fmt.Errorf("template validation failed"))

    suite.AssertError("template validation failed")
}
```

**File**: `cmd/flags/groups/deployment_test.go`

```go
package groups

import (
    "testing"
    "github.com/ArjenSchwarz/fog/cmd/testing/framework"
)

func TestDeploymentFlags_Validation_Success(t *testing.T) {
    suite := framework.NewFlagValidationTestSuite(t).
        WithDeploymentFlags().
        SetFlag("stackname", "test-stack").
        SetFlag("template", "template.json")

    suite.TestData.CreateFile("template.json", framework.TestTemplates.SimpleVPC)

    suite.AssertValidationSuccess()
}

func TestDeploymentFlags_Validation_MissingStackName(t *testing.T) {
    suite := framework.NewFlagValidationTestSuite(t).
        WithDeploymentFlags().
        SetFlag("template", "template.json")

    suite.AssertValidationError("stackname is required")
}

func TestDeploymentFlags_Validation_MissingTemplate(t *testing.T) {
    suite := framework.NewFlagValidationTestSuite(t).
        WithDeploymentFlags().
        SetFlag("stackname", "test-stack")

    suite.AssertValidationError("template or deployment-file must be provided")
}

func TestDeploymentFlags_Validation_ConflictingFlags(t *testing.T) {
    suite := framework.NewFlagValidationTestSuite(t).
        WithDeploymentFlags().
        SetFlag("stackname", "test-stack").
        SetFlag("template", "template.json").
        SetFlag("deployment-file", "deployment.yaml")

    suite.AssertValidationError("conflicting flags")
}
```

## Files to Create/Modify

### New Files
- `cmd/testing/framework/setup.go`
- `cmd/testing/framework/command_test.go`
- `cmd/testing/framework/fixtures.go`
- `cmd/testing/framework/assertions.go`
- `cmd/testing/mocks/aws_service.go`
- `cmd/testing/mocks/ui_handler.go`
- `cmd/testing/mocks/file_system.go`
- `cmd/testing/mocks/config.go`
- `cmd/testing/testdata/` (directory with test files)
- `cmd/testing/integration/helpers.go`
- `cmd/testing/integration/deploy_test.go`
- `cmd/testing/integration/drift_test.go`

### Test Files for Existing Components
- `cmd/commands/deploy/handler_test.go`
- `cmd/commands/deploy/command_test.go`
- `cmd/flags/groups/deployment_test.go`
- `cmd/services/deployment/service_test.go`
- `cmd/ui/console/output_test.go`
- `cmd/errors/types_test.go`

### Modified Files
- `go.mod` - Add testing dependencies
- `Makefile` - Add test targets
- `.github/workflows/` - Add test automation

## Testing Strategy

### Unit Tests
- Test individual components in isolation
- Mock all external dependencies
- Focus on business logic and error handling
- Achieve >90% code coverage

### Integration Tests
- Test complete command workflows
- Use real file system interactions
- Mock AWS services but test service integration
- Verify end-to-end functionality

### Performance Tests
- Benchmark critical operations
- Test with large templates and parameter files
- Measure memory usage and execution time
- Identify performance regressions

### Test Categories
```go
// Unit tests - fast, isolated
func TestUnit_*(t *testing.T) { ... }

// Integration tests - slower, more realistic
func TestIntegration_*(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    ...
}

// Performance tests - benchmarking
func BenchmarkPerformance_*(b *testing.B) { ... }
```

## Success Criteria

### Coverage Requirements
- [ ] >90% unit test coverage for all new code
- [ ] Integration tests for all major command workflows
- [ ] Error scenario testing for all error types
- [ ] Performance benchmarks for critical operations

### Quality Requirements
- [ ] All tests pass consistently
- [ ] Tests run in reasonable time (<30s for unit, <5m for integration)
- [ ] Clear, maintainable test code
- [ ] Comprehensive mocking of external dependencies

### Documentation Requirements
- [ ] Testing guidelines and patterns documented
- [ ] Examples of how to write tests for new features
- [ ] Performance benchmarking procedures
- [ ] CI/CD integration documentation

## Migration Timeline

### Phase 1: Foundation
- Create testing framework and utilities
- Implement mock infrastructure
- Setup test data management

### Phase 2: Core Tests
- Add unit tests for existing commands
- Create integration test framework
- Implement flag validation tests

### Phase 3: Comprehensive Coverage
- Add tests for all error scenarios
- Implement performance benchmarks
- Create end-to-end integration tests

## Dependencies

### Upstream Dependencies
- Task 1: Command Structure Reorganization (provides testable structure)
- Task 2: Business Logic Extraction (provides services to mock)
- Task 3: Flag Management (provides validation to test)
- Task 4: Output and UI Standardization (provides UI to mock)
- Task 5: Error Handling (provides errors to test)

### External Dependencies
- `github.com/stretchr/testify` - Testing utilities and assertions
- `github.com/golang/mock` - Mock generation (optional)
- `github.com/aws/aws-sdk-go-v2` - AWS SDK for mocking

## Risk Mitigation

### Potential Issues
- Test execution time becoming too long
- Flaky tests due to external dependencies
- Maintaining test data and fixtures
- Mock complexity for AWS services

### Mitigation Strategies
- Parallel test execution where possible
- Comprehensive mocking of external services
- Automated test data generation
- Clear separation of unit and integration tests
- Regular test maintenance and cleanup
