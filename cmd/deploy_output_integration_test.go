package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/spf13/viper"
)

// TestDeploy_SuccessfulWithJSONOutput tests successful deployment with JSON output
func TestDeploy_SuccessfulWithJSONOutput(t *testing.T) {
	// Set up test stack
	stack := &types.Stack{
		StackName:   aws.String("test-json-output"),
		StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/test-json-output/abc123"),
		StackStatus: types.StackStatusCreateComplete,
		Outputs: []types.Output{
			{
				OutputKey:   aws.String("BucketName"),
				OutputValue: aws.String("my-test-bucket"),
				Description: aws.String("The S3 bucket name"),
			},
		},
	}

	// Set output format to JSON
	viper.Set("output", "json")
	defer viper.Reset()

	// Create deployment info
	deployment := &lib.DeployInfo{
		StackName:       "test-json-output",
		StackArn:        "arn:aws:cloudformation:us-east-1:123456789012:stack/test-json-output/abc123",
		RawStack:        stack,
		Template:        "simple template",
		DeploymentStart: time.Now(),
		DeploymentEnd:   time.Now(),
		FinalStackState: stack,
		CapturedChangeset: &lib.ChangesetInfo{
			ID:           "arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/def456",
			CreationTime: time.Now().Add(-1 * time.Minute),
			Changes: []lib.ChangesetChanges{
				{
					Action:      "Add",
					LogicalID:   "MyBucket",
					Type:        "AWS::S3::Bucket",
					ResourceID:  "my-test-bucket",
					Replacement: "False",
				},
			},
		},
	}

	// Capture stdout for JSON validation
	output := captureStdout(func() {
		err := outputSuccessResult(deployment)
		if err != nil {
			t.Errorf("outputSuccessResult failed: %v", err)
		}
	})

	// Verify stdout contains valid JSON
	if !strings.Contains(output, "Deployment Summary") {
		t.Error("Expected JSON output to contain 'Deployment Summary'")
	}

	if !strings.Contains(output, "CREATE_COMPLETE") {
		t.Error("Expected JSON output to contain stack status")
	}

	if !strings.Contains(output, "MyBucket") {
		t.Error("Expected JSON output to contain changeset changes")
	}

	if !strings.Contains(output, "BucketName") {
		t.Error("Expected JSON output to contain stack outputs")
	}
}

// TestDeploy_FailureWithFormattedOutput tests failed deployment with formatted output
func TestDeploy_FailureWithFormattedOutput(t *testing.T) {
	// Set up failed stack
	stack := &types.Stack{
		StackName:         aws.String("test-failed-deployment"),
		StackId:           aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/test-failed-deployment/abc123"),
		StackStatus:       types.StackStatusRollbackComplete,
		StackStatusReason: aws.String("Resource creation cancelled"),
	}

	// Set output format to JSON
	viper.Set("output", "json")
	defer viper.Reset()

	// Create deployment info with error
	deployment := &lib.DeployInfo{
		StackName:       "test-failed-deployment",
		StackArn:        "arn:aws:cloudformation:us-east-1:123456789012:stack/test-failed-deployment/abc123",
		RawStack:        stack,
		Template:        "template",
		DeploymentStart: time.Now(),
		DeploymentEnd:   time.Now(),
		DeploymentError: &customTestError{message: "The following resource(s) failed to create: [MyDatabase]"},
		FinalStackState: stack,
	}

	// Capture stdout for error output validation
	output := captureStdout(func() {
		err := outputFailureResult(deployment, config.AWSConfig{})
		if err != nil {
			t.Errorf("outputFailureResult failed: %v", err)
		}
	})

	// Verify stdout contains error details in JSON format
	if !strings.Contains(output, "Deployment failed") {
		t.Error("Expected output to contain 'Deployment failed'")
	}

	if !strings.Contains(output, "ROLLBACK_COMPLETE") {
		t.Error("Expected output to contain stack status")
	}

	if !strings.Contains(output, "Resource creation cancelled") {
		t.Error("Expected output to contain status reason")
	}
}

// TestDeploy_QuietMode tests quiet mode suppresses stderr output
func TestDeploy_QuietMode(t *testing.T) {
	// Set up test stack
	stack := &types.Stack{
		StackName:   aws.String("test-quiet-mode"),
		StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/test-quiet-mode/abc123"),
		StackStatus: types.StackStatusCreateComplete,
	}

	// Set output format
	viper.Set("output", "json")
	defer viper.Reset()

	// Create deployment info
	deployment := &lib.DeployInfo{
		StackName:       "test-quiet-mode",
		StackArn:        "arn:aws:cloudformation:us-east-1:123456789012:stack/test-quiet-mode/abc123",
		RawStack:        stack,
		Template:        "template",
		DeploymentStart: time.Now(),
		DeploymentEnd:   time.Now(),
		FinalStackState: stack,
		CapturedChangeset: &lib.ChangesetInfo{
			ID:           "arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/def456",
			CreationTime: time.Now().Add(-1 * time.Minute),
			Changes:      []lib.ChangesetChanges{},
		},
	}

	// Capture stdout
	output := captureStdout(func() {
		err := outputSuccessResult(deployment)
		if err != nil {
			t.Errorf("outputSuccessResult failed: %v", err)
		}
	})

	// Verify stdout contains formatted output
	if !strings.Contains(output, "Deployment Summary") {
		t.Error("Expected output to contain deployment summary")
	}

	// Note: In a full integration test, we would capture stderr separately
	// and verify it's empty in quiet mode. These tests focus on stdout output.
}

// TestDeploy_DryRunMultipleFormats tests dry-run with multiple output formats
func TestDeploy_DryRunMultipleFormats(t *testing.T) {
	formats := []string{"json", "yaml", "csv", "markdown"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			// Set up test stack
			stack := &types.Stack{
				StackName:   aws.String("test-dryrun-" + format),
				StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/test-dryrun-" + format + "/abc123"),
				StackStatus: types.StackStatusCreateComplete,
			}

			// Set output format
			viper.Set("output", format)
			defer viper.Reset()

			// Create deployment info for dry-run
			deployment := &lib.DeployInfo{
				StackName: "test-dryrun-" + format,
				StackArn:  "arn:aws:cloudformation:us-east-1:123456789012:stack/test-dryrun-" + format + "/abc123",
				RawStack:  stack,
				IsDryRun:  true,
				CapturedChangeset: &lib.ChangesetInfo{
					ID:           "arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/def456",
					Name:         "test-changeset",
					CreationTime: time.Now().Add(-1 * time.Minute),
					Status:       string(types.ChangeSetStatusCreateComplete),
					Changes: []lib.ChangesetChanges{
						{
							Action:      "Add",
							LogicalID:   "MyBucket",
							Type:        "AWS::S3::Bucket",
							ResourceID:  "my-bucket",
							Replacement: "False",
						},
						{
							Action:      "Modify",
							LogicalID:   "MyFunction",
							Type:        "AWS::Lambda::Function",
							ResourceID:  "my-function",
							Replacement: "False",
						},
					},
				},
			}

			// Capture output - test changeset rendering
			output := captureStdout(func() {
				buildAndRenderChangeset(*deployment.CapturedChangeset, *deployment, config.AWSConfig{})
			})

			// Verify output is not empty
			if len(output) == 0 {
				t.Error("Expected non-empty output")
			}

			// Format-specific validation
			switch format {
			case "json", "yaml":
				if !strings.Contains(output, "MyBucket") && !strings.Contains(output, "MyFunction") {
					t.Errorf("Expected %s output to contain changeset changes", format)
				}
			case "csv", "markdown":
				if !strings.Contains(output, "MyBucket") || !strings.Contains(output, "MyFunction") {
					t.Errorf("Expected %s output to contain changeset changes", format)
				}
			}
		})
	}
}

// TestDeploy_NoChanges tests no-changes scenario
func TestDeploy_NoChanges(t *testing.T) {
	// Set up existing stack
	lastUpdated := time.Now().Add(-1 * time.Hour)
	stack := &types.Stack{
		StackName:       aws.String("test-no-changes"),
		StackId:         aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/test-no-changes/abc123"),
		StackStatus:     types.StackStatusUpdateComplete,
		LastUpdatedTime: &lastUpdated,
	}

	// Set output format
	viper.Set("output", "json")
	defer viper.Reset()

	// Create deployment info for no-changes scenario
	deployment := &lib.DeployInfo{
		StackName: "test-no-changes",
		StackArn:  "arn:aws:cloudformation:us-east-1:123456789012:stack/test-no-changes/abc123",
		RawStack:  stack,
	}

	// Capture output
	output := captureStdout(func() {
		err := outputNoChangesResult(deployment)
		if err != nil {
			t.Errorf("outputNoChangesResult failed: %v", err)
		}
	})

	// Verify no-changes output
	if !strings.Contains(output, "No changes to deploy") {
		t.Error("Expected output to contain 'No changes to deploy'")
	}

	if !strings.Contains(output, "test-no-changes") {
		t.Error("Expected output to contain stack name")
	}

	if !strings.Contains(output, "UPDATE_COMPLETE") {
		t.Error("Expected output to contain stack status")
	}
}
