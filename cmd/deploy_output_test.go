package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStdout is defined in describe_changeset_test.go and reused here

// createTestDeployment creates a mock deployment for testing
func createTestDeployment() *lib.DeployInfo {
	now := time.Now()
	return &lib.DeployInfo{
		StackName:       "test-stack",
		StackArn:        "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/12345",
		DeploymentStart: now.Add(-5 * time.Minute),
		DeploymentEnd:   now,
		CapturedChangeset: &lib.ChangesetInfo{
			ID:           "arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset",
			CreationTime: now.Add(-6 * time.Minute),
			Changes: []lib.ChangesetChanges{
				{
					Action:      "Add",
					LogicalID:   "TestResource",
					Type:        "AWS::S3::Bucket",
					ResourceID:  "test-bucket",
					Replacement: "False",
				},
			},
		},
		FinalStackState: &types.Stack{
			StackStatus: types.StackStatusCreateComplete,
			Outputs: []types.Output{
				{
					OutputKey:   aws.String("BucketName"),
					OutputValue: aws.String("test-bucket"),
					Description: aws.String("The bucket name"),
				},
			},
		},
		RawStack: &types.Stack{
			StackStatus:     types.StackStatusCreateComplete,
			LastUpdatedTime: aws.Time(now.Add(-1 * time.Hour)),
		},
	}
}

// TestOutputSuccessResult tests the successful deployment output
func TestOutputSuccessResult(t *testing.T) {
	tests := map[string]struct {
		format       string
		deployment   *lib.DeployInfo
		expectError  bool
		validateFunc func(t *testing.T, output string)
	}{
		"successful deployment with JSON format": {
			format:     "json",
			deployment: createTestDeployment(),
			validateFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "Deployment Summary")
				assert.Contains(t, output, "CREATE_COMPLETE")
				assert.Contains(t, output, "test-stack")
				assert.Contains(t, output, "TestResource")
				assert.Contains(t, output, "BucketName")
			},
		},
		"successful deployment without changeset": {
			format: "json",
			deployment: func() *lib.DeployInfo {
				d := createTestDeployment()
				d.CapturedChangeset = nil
				return d
			}(),
			validateFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "Deployment Summary")
				assert.Contains(t, output, "CREATE_COMPLETE")
			},
		},
		"successful deployment without outputs": {
			format: "json",
			deployment: func() *lib.DeployInfo {
				d := createTestDeployment()
				d.FinalStackState.Outputs = []types.Output{}
				return d
			}(),
			validateFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "Deployment Summary")
				assert.Contains(t, output, "CREATE_COMPLETE")
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Set output format
			viper.Set("output", tc.format)
			defer viper.Reset()

			// Capture stdout
			output := captureStdout(func() {
				err := outputSuccessResult(tc.deployment)
				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})

			if tc.validateFunc != nil {
				tc.validateFunc(t, output)
			}
		})
	}
}

// TestOutputNoChangesResult tests the no-changes output
func TestOutputNoChangesResult(t *testing.T) {
	tests := map[string]struct {
		format       string
		deployment   *lib.DeployInfo
		validateFunc func(t *testing.T, output string)
	}{
		"no changes with JSON format": {
			format:     "json",
			deployment: createTestDeployment(),
			validateFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "No changes to deploy")
				assert.Contains(t, output, "test-stack")
				assert.Contains(t, output, "CREATE_COMPLETE")
			},
		},
		"no changes without LastUpdatedTime": {
			format: "json",
			deployment: func() *lib.DeployInfo {
				d := createTestDeployment()
				d.RawStack.LastUpdatedTime = nil
				d.RawStack.CreationTime = aws.Time(time.Now().Add(-2 * time.Hour))
				return d
			}(),
			validateFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "No changes to deploy")
				assert.Contains(t, output, "test-stack")
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Set output format
			viper.Set("output", tc.format)
			defer viper.Reset()

			// Capture stdout
			output := captureStdout(func() {
				err := outputNoChangesResult(tc.deployment)
				assert.NoError(t, err)
			})

			if tc.validateFunc != nil {
				tc.validateFunc(t, output)
			}
		})
	}
}

// TestOutputFailureResult tests the failed deployment output
func TestOutputFailureResult(t *testing.T) {
	tests := map[string]struct {
		format       string
		deployment   *lib.DeployInfo
		awsConfig    config.AWSConfig
		validateFunc func(t *testing.T, output string)
	}{
		"failed deployment with error": {
			format: "json",
			deployment: func() *lib.DeployInfo {
				d := createTestDeployment()
				d.DeploymentError = assert.AnError
				d.FinalStackState = &types.Stack{
					StackStatus:       types.StackStatusRollbackComplete,
					StackStatusReason: aws.String("Resource creation failed"),
				}
				return d
			}(),
			awsConfig: config.AWSConfig{},
			validateFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "Deployment failed")
				assert.Contains(t, output, "ROLLBACK_COMPLETE")
				assert.Contains(t, output, "Resource creation failed")
			},
		},
		"failed deployment without FinalStackState": {
			format: "json",
			deployment: func() *lib.DeployInfo {
				d := createTestDeployment()
				d.DeploymentError = assert.AnError
				d.FinalStackState = nil
				return d
			}(),
			awsConfig: config.AWSConfig{},
			validateFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "Deployment failed")
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Set output format
			viper.Set("output", tc.format)
			defer viper.Reset()

			// Capture stdout
			output := captureStdout(func() {
				err := outputFailureResult(tc.deployment, tc.awsConfig)
				assert.NoError(t, err)
			})

			if tc.validateFunc != nil {
				tc.validateFunc(t, output)
			}
		})
	}
}

// TestOutputSuccessResult_AllFormats tests all supported output formats
func TestOutputSuccessResult_AllFormats(t *testing.T) {
	formats := []string{"json", "yaml", "csv", "markdown", "table"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			// Set output format
			viper.Set("output", format)
			defer viper.Reset()

			deployment := createTestDeployment()

			// Capture stdout
			output := captureStdout(func() {
				err := outputSuccessResult(deployment)
				assert.NoError(t, err)
			})

			// Basic validation - output should not be empty
			assert.NotEmpty(t, output)

			// Format-specific validation
			switch format {
			case "json":
				// JSON should be parseable (go-output handles this)
				assert.Contains(t, output, "Deployment Summary")
			case "yaml":
				assert.Contains(t, output, "Deployment Summary")
			case "csv":
				// CSV should have header row
				assert.Contains(t, output, "Status")
			case "markdown":
				// Markdown should have headers
				assert.Contains(t, output, "#")
			case "table":
				// Table should have visual structure
				assert.Contains(t, output, "Deployment Summary")
			}
		})
	}
}

// TestOutputDryRunResult tests the dry-run output integration
func TestOutputDryRunResult(t *testing.T) {
	t.Skip("Integration test for outputDryRunResult requires buildAndRenderChangeset mock")
}

// TestExtractFailedResources tests failed resource extraction
func TestExtractFailedResources(t *testing.T) {
	t.Skip("Requires mock CloudFormation client for event retrieval")
}

// TestOutputBuilders_DataStructureCorrectness verifies data structure correctness
func TestOutputBuilders_DataStructureCorrectness(t *testing.T) {
	deployment := createTestDeployment()

	// Test that we can build the document without errors
	t.Run("successful deployment structure", func(t *testing.T) {
		duration := deployment.DeploymentEnd.Sub(deployment.DeploymentStart)
		changesetID := deployment.CapturedChangeset.ID

		summaryData := []map[string]any{
			{
				"Status":     string(deployment.FinalStackState.StackStatus),
				"Stack ARN":  deployment.StackArn,
				"Changeset":  changesetID,
				"Start Time": deployment.DeploymentStart.Format(time.RFC3339),
				"End Time":   deployment.DeploymentEnd.Format(time.RFC3339),
				"Duration":   duration.String(),
			},
		}

		builder := output.New().
			Table(
				"Deployment Summary",
				summaryData,
				output.WithKeys("Status", "Stack ARN", "Changeset", "Start Time", "End Time", "Duration"),
			)

		doc := builder.Build()
		assert.NotNil(t, doc)

		// Verify we can render it
		out := output.NewOutput(
			output.WithFormat(output.Table()),
			output.WithWriter(output.NewStdoutWriter()),
		)

		err := out.Render(context.Background(), doc)
		assert.NoError(t, err)
	})

	t.Run("no changes structure", func(t *testing.T) {
		message := "No changes to deploy - stack is already up to date"

		stackData := []map[string]any{
			{
				"Stack Name":   deployment.StackName,
				"Status":       string(deployment.RawStack.StackStatus),
				"ARN":          deployment.StackArn,
				"Last Updated": deployment.RawStack.LastUpdatedTime.Format(time.RFC3339),
			},
		}

		builder := output.New().
			Text(message).
			Table(
				"Stack Information",
				stackData,
				output.WithKeys("Stack Name", "Status", "ARN", "Last Updated"),
			)

		doc := builder.Build()
		assert.NotNil(t, doc)
	})

	t.Run("failure structure", func(t *testing.T) {
		deployment.DeploymentError = assert.AnError
		deployment.FinalStackState = &types.Stack{
			StackStatus:       types.StackStatusRollbackComplete,
			StackStatusReason: aws.String("Resource creation failed"),
		}

		statusData := []map[string]any{
			{
				"Stack ARN":     deployment.StackArn,
				"Status":        string(deployment.FinalStackState.StackStatus),
				"Status Reason": aws.ToString(deployment.FinalStackState.StackStatusReason),
				"Timestamp":     time.Now().Format(time.RFC3339),
			},
		}

		builder := output.New().
			Text("Deployment failed: "+deployment.DeploymentError.Error()).
			Table(
				"Stack Status",
				statusData,
				output.WithKeys("Stack ARN", "Status", "Status Reason", "Timestamp"),
			)

		doc := builder.Build()
		assert.NotNil(t, doc)
	})
}

// TestOutputBuilders_MissingFieldHandling tests error handling for missing fields
func TestOutputBuilders_MissingFieldHandling(t *testing.T) {
	t.Run("success without FinalStackState", func(t *testing.T) {
		deployment := createTestDeployment()
		deployment.FinalStackState = nil

		viper.Set("output", "json")
		defer viper.Reset()

		// This should panic or error because FinalStackState is required
		assert.Panics(t, func() {
			_ = outputSuccessResult(deployment)
		})
	})

	t.Run("no changes without RawStack", func(t *testing.T) {
		deployment := createTestDeployment()
		deployment.RawStack = nil

		viper.Set("output", "json")
		defer viper.Reset()

		// Should handle gracefully with empty time
		output := captureStdout(func() {
			err := outputNoChangesResult(deployment)
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "No changes to deploy")
	})

	t.Run("failure without DeploymentError", func(t *testing.T) {
		deployment := createTestDeployment()
		deployment.DeploymentError = nil
		deployment.FinalStackState = &types.Stack{
			StackStatus: types.StackStatusRollbackComplete,
		}

		viper.Set("output", "json")
		defer viper.Reset()

		// Should use default message
		output := captureStdout(func() {
			err := outputFailureResult(deployment, config.AWSConfig{})
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "Deployment failed")
	})
}

// compareWithGolden compares output against a golden file
func compareWithGolden(t *testing.T, goldenFile string, actual string) {
	t.Helper()

	goldenPath := filepath.Join("testdata", "deploy-output", goldenFile)

	// Check if we should update golden files
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		err := os.WriteFile(goldenPath, []byte(actual), 0644)
		require.NoError(t, err)
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	// Read expected content
	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "Failed to read golden file: %s", goldenPath)

	// Compare
	assert.Equal(t, string(expected), actual, "Output does not match golden file: %s", goldenPath)
}

// TestGoldenFile_SuccessOutput tests successful deployment output against golden files
func TestGoldenFile_SuccessOutput(t *testing.T) {
	tests := map[string]struct {
		format     string
		goldenFile string
	}{
		"json format": {
			format:     "json",
			goldenFile: "success-output.json",
		},
		"yaml format": {
			format:     "yaml",
			goldenFile: "success-output.yaml",
		},
		"csv format": {
			format:     "csv",
			goldenFile: "success-output.csv",
		},
		"markdown format": {
			format:     "markdown",
			goldenFile: "success-output.md",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Use the same deployment data as golden file generation
			deployment := createGoldenTestDeployment()

			// Set format
			viper.Set("output", tc.format)
			defer viper.Reset()

			// Capture output
			actual := captureStdout(func() {
				err := outputSuccessResult(deployment)
				require.NoError(t, err)
			})

			// Compare with golden file
			compareWithGolden(t, tc.goldenFile, actual)
		})
	}
}

// TestGoldenFile_FailureOutput tests failed deployment output against golden files
func TestGoldenFile_FailureOutput(t *testing.T) {
	tests := map[string]struct {
		format     string
		goldenFile string
	}{
		"json format": {
			format:     "json",
			goldenFile: "failure-output.json",
		},
		"yaml format": {
			format:     "yaml",
			goldenFile: "failure-output.yaml",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Use the same deployment data as golden file generation
			deployment := createGoldenFailedDeploymentTest()

			// Set format
			viper.Set("output", tc.format)
			defer viper.Reset()

			// Capture output
			actual := captureStdout(func() {
				err := outputFailureResult(deployment, config.AWSConfig{})
				require.NoError(t, err)
			})

			// Compare with golden file
			compareWithGolden(t, tc.goldenFile, actual)
		})
	}
}

// TestGoldenFile_NoChangesOutput tests no-changes output against golden file
func TestGoldenFile_NoChangesOutput(t *testing.T) {
	// Use the same deployment data as golden file generation
	deployment := createGoldenNoChangesDeploymentTest()

	// Set format
	viper.Set("output", "json")
	defer viper.Reset()

	// Capture output
	actual := captureStdout(func() {
		err := outputNoChangesResult(deployment)
		require.NoError(t, err)
	})

	// Compare with golden file
	compareWithGolden(t, "no-changes-output.json", actual)
}

// createGoldenTestDeployment creates the same test data used in generate_golden_files_test.go
func createGoldenTestDeployment() *lib.DeployInfo {
	// Fixed timestamp for consistent golden files
	startTime := time.Date(2025, 11, 7, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 11, 7, 10, 5, 30, 0, time.UTC)
	changesetTime := time.Date(2025, 11, 7, 9, 58, 0, 0, time.UTC)
	lastUpdated := time.Date(2025, 11, 6, 14, 30, 0, 0, time.UTC)

	return &lib.DeployInfo{
		StackName:       "test-stack",
		StackArn:        "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123",
		DeploymentStart: startTime,
		DeploymentEnd:   endTime,
		CapturedChangeset: &lib.ChangesetInfo{
			ID:           "arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/def456",
			CreationTime: changesetTime,
			Changes: []lib.ChangesetChanges{
				{
					Action:      "Add",
					LogicalID:   "MyBucket",
					Type:        "AWS::S3::Bucket",
					ResourceID:  "my-test-bucket-123",
					Replacement: "False",
				},
				{
					Action:      "Modify",
					LogicalID:   "MyFunction",
					Type:        "AWS::Lambda::Function",
					ResourceID:  "test-stack-MyFunction-ABC123",
					Replacement: "False",
				},
			},
		},
		FinalStackState: &types.Stack{
			StackStatus: types.StackStatusUpdateComplete,
			Outputs: []types.Output{
				{
					OutputKey:   aws.String("BucketName"),
					OutputValue: aws.String("my-test-bucket-123"),
					Description: aws.String("The S3 bucket name"),
				},
				{
					OutputKey:   aws.String("FunctionArn"),
					OutputValue: aws.String("arn:aws:lambda:us-east-1:123456789012:function:test-stack-MyFunction-ABC123"),
					Description: aws.String("The Lambda function ARN"),
				},
			},
		},
		RawStack: &types.Stack{
			StackStatus:     types.StackStatusUpdateComplete,
			LastUpdatedTime: &lastUpdated,
		},
	}
}

// createGoldenFailedDeploymentTest creates the same failed deployment data used in generate_golden_files_test.go
func createGoldenFailedDeploymentTest() *lib.DeployInfo {
	startTime := time.Date(2025, 11, 7, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 11, 7, 10, 3, 45, 0, time.UTC)

	deployment := &lib.DeployInfo{
		StackName:       "test-stack",
		StackArn:        "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123",
		DeploymentStart: startTime,
		DeploymentEnd:   endTime,
		FinalStackState: &types.Stack{
			StackStatus:       types.StackStatusRollbackComplete,
			StackStatusReason: aws.String("Resource creation cancelled"),
		},
	}

	// Use a custom error type that matches the golden file generation
	deployment.DeploymentError = &customTestError{message: "The following resource(s) failed to create: [MyDatabase]"}

	return deployment
}

// createGoldenNoChangesDeploymentTest creates the same no-changes deployment data used in generate_golden_files_test.go
func createGoldenNoChangesDeploymentTest() *lib.DeployInfo {
	lastUpdated := time.Date(2025, 11, 6, 14, 30, 0, 0, time.UTC)

	return &lib.DeployInfo{
		StackName: "test-stack",
		StackArn:  "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123",
		RawStack: &types.Stack{
			StackStatus:     types.StackStatusUpdateComplete,
			LastUpdatedTime: &lastUpdated,
		},
	}
}

// customTestError is a simple error type for testing
type customTestError struct {
	message string
}

func (e *customTestError) Error() string {
	return e.message
}
