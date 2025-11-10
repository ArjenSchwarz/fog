package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ArjenSchwarz/fog/lib"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// createGoldenDeployment creates a deployment with realistic test data
func createGoldenDeployment() *lib.DeployInfo {
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

// createGoldenFailedDeployment creates a failed deployment with realistic test data
func createGoldenFailedDeployment() *lib.DeployInfo {
	startTime := time.Date(2025, 11, 7, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 11, 7, 10, 3, 45, 0, time.UTC)

	return &lib.DeployInfo{
		StackName:       "test-stack",
		StackArn:        "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123",
		DeploymentStart: startTime,
		DeploymentEnd:   endTime,
		DeploymentError: errors.New("The following resource(s) failed to create: [MyDatabase]"),
		FinalStackState: &types.Stack{
			StackStatus:       types.StackStatusRollbackComplete,
			StackStatusReason: aws.String("Resource creation cancelled"),
		},
	}
}

// createGoldenNoChangesDeployment creates a no-changes deployment with realistic test data
func createGoldenNoChangesDeployment() *lib.DeployInfo {
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

// TestGenerateGoldenFiles generates golden files for output format testing
// Run with: UPDATE_GOLDEN=1 go test ./cmd -run TestGenerateGoldenFiles -v
func TestGenerateGoldenFiles(t *testing.T) {
	if os.Getenv("UPDATE_GOLDEN") != "1" {
		t.Skip("Skipping golden file generation. Set UPDATE_GOLDEN=1 to regenerate.")
	}

	baseDir := "testdata/deploy-output"

	// Test cases for golden file generation
	tests := []struct {
		name       string
		format     string
		deployment *lib.DeployInfo
		outputFunc func(*lib.DeployInfo) error
	}{
		// Success scenarios
		{
			name:       "success-output.json",
			format:     "json",
			deployment: createGoldenDeployment(),
			outputFunc: outputSuccessResult,
		},
		{
			name:       "success-output.yaml",
			format:     "yaml",
			deployment: createGoldenDeployment(),
			outputFunc: outputSuccessResult,
		},
		{
			name:       "success-output.csv",
			format:     "csv",
			deployment: createGoldenDeployment(),
			outputFunc: outputSuccessResult,
		},
		{
			name:       "success-output.md",
			format:     "markdown",
			deployment: createGoldenDeployment(),
			outputFunc: outputSuccessResult,
		},
		// Failure scenarios
		{
			name:       "failure-output.json",
			format:     "json",
			deployment: createGoldenFailedDeployment(),
			outputFunc: func(d *lib.DeployInfo) error {
				return outputFailureResult(d, nil)
			},
		},
		{
			name:       "failure-output.yaml",
			format:     "yaml",
			deployment: createGoldenFailedDeployment(),
			outputFunc: func(d *lib.DeployInfo) error {
				return outputFailureResult(d, nil)
			},
		},
		// No changes scenarios
		{
			name:       "no-changes-output.json",
			format:     "json",
			deployment: createGoldenNoChangesDeployment(),
			outputFunc: outputNoChangesResult,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set format
			viper.Set("output", tc.format)
			defer viper.Reset()

			// Capture output
			output := captureStdout(func() {
				err := tc.outputFunc(tc.deployment)
				require.NoError(t, err)
			})

			// Write golden file
			goldenPath := filepath.Join(baseDir, tc.name)
			err := os.WriteFile(goldenPath, []byte(output), 0644)
			require.NoError(t, err)

			t.Logf("Generated golden file: %s", goldenPath)
		})
	}
}
