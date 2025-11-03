package cmd

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/ArjenSchwarz/fog/lib/testutil"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// TestShowDeploymentInfo_GoldenFiles tests deployment info display with golden files
func TestShowDeploymentInfo_GoldenFiles(t *testing.T) {
	// Initialize golden file helper
	golden := testutil.NewGoldenFileWithDir(t, "testdata/golden/cmd")

	tests := map[string]struct {
		deployment lib.DeployInfo
		awsConfig  config.AWSConfig
	}{
		"new_stack_deployment": {
			deployment: lib.DeployInfo{
				StackName: "my-new-stack",
				IsNew:     true,
				IsDryRun:  false,
			},
			awsConfig: config.AWSConfig{
				Region:       "us-east-1",
				AccountID:    "123456789012",
				AccountAlias: "production",
			},
		},
		"update_stack_deployment": {
			deployment: lib.DeployInfo{
				StackName: "existing-stack",
				IsNew:     false,
				IsDryRun:  false,
			},
			awsConfig: config.AWSConfig{
				Region:       "eu-west-1",
				AccountID:    "987654321098",
				AccountAlias: "staging",
			},
		},
		"dry_run_new_stack": {
			deployment: lib.DeployInfo{
				StackName: "test-stack",
				IsNew:     true,
				IsDryRun:  true,
			},
			awsConfig: config.AWSConfig{
				Region:    "ap-southeast-2",
				AccountID: "111111111111",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Build the expected output string
			var buf bytes.Buffer

			method := determineDeploymentMethod(tc.deployment.IsNew, tc.deployment.IsDryRun)
			account := formatAccountDisplay(tc.awsConfig.AccountID, tc.awsConfig.AccountAlias)

			if tc.deployment.IsNew {
				buf.WriteString(fmt.Sprintf("%v new stack '%v' to region %v of account %v\n",
					method, tc.deployment.StackName, tc.awsConfig.Region, account))
			} else {
				buf.WriteString(fmt.Sprintf("%v stack '%v' in region %v of account %v\n",
					method, tc.deployment.StackName, tc.awsConfig.Region, tc.awsConfig.AccountID))
			}

			// Assert against golden file (strip ANSI codes to test data correctness, not formatting)
			golden.AssertStringWithoutAnsi(name, buf.String())
		})
	}
}

// TestStackOutputFormatting_GoldenFiles tests stack output formatting
func TestStackOutputFormatting_GoldenFiles(t *testing.T) {
	// Initialize golden file helper
	golden := testutil.NewGoldenFileWithDir(t, "testdata/golden/cmd")

	tests := map[string]struct {
		outputs []types.Output
	}{
		"simple_outputs": {
			outputs: []types.Output{
				{
					OutputKey:   aws.String("VpcId"),
					OutputValue: aws.String("vpc-12345678"),
				},
				{
					OutputKey:   aws.String("SubnetId"),
					OutputValue: aws.String("subnet-87654321"),
				},
			},
		},
		"outputs_with_exports": {
			outputs: []types.Output{
				{
					OutputKey:   aws.String("VpcId"),
					OutputValue: aws.String("vpc-12345678"),
					ExportName:  aws.String("MyVpcId"),
					Description: aws.String("The VPC ID for the network"),
				},
				{
					OutputKey:   aws.String("SecurityGroup"),
					OutputValue: aws.String("sg-87654321"),
					ExportName:  aws.String("MySecurityGroup"),
					Description: aws.String("Security group for web servers"),
				},
			},
		},
		"outputs_with_descriptions": {
			outputs: []types.Output{
				{
					OutputKey:   aws.String("ApiEndpoint"),
					OutputValue: aws.String("https://api.example.com"),
					Description: aws.String("The API endpoint URL"),
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer

			for _, output := range tc.outputs {
				exportName := ""
				if output.ExportName != nil {
					exportName = *output.ExportName
				}
				description := ""
				if output.Description != nil {
					description = *output.Description
				}
				buf.WriteString(fmt.Sprintf("Key: %s\n", *output.OutputKey))
				buf.WriteString(fmt.Sprintf("Value: %s\n", *output.OutputValue))
				if description != "" {
					buf.WriteString(fmt.Sprintf("Description: %s\n", description))
				}
				if exportName != "" {
					buf.WriteString(fmt.Sprintf("ExportName: %s\n", exportName))
				}
				buf.WriteString("\n")
			}

			// Assert against golden file
			golden.AssertString(name, buf.String())
		})
	}
}

// TestChangesetChangeFormatting_GoldenFiles tests changeset change formatting
func TestChangesetChangeFormatting_GoldenFiles(t *testing.T) {
	// Initialize golden file helper
	golden := testutil.NewGoldenFileWithDir(t, "testdata/golden/cmd")

	tests := map[string]struct {
		changes []lib.ChangesetChanges
	}{
		"add_resources": {
			changes: []lib.ChangesetChanges{
				{
					Action:      "Add",
					LogicalID:   "WebServerInstance",
					ResourceID:  "i-1234567890abcdef0",
					Type:        "AWS::EC2::Instance",
					Replacement: "False",
				},
				{
					Action:      "Add",
					LogicalID:   "WebServerSecurityGroup",
					Type:        "AWS::EC2::SecurityGroup",
					Replacement: "False",
				},
			},
		},
		"modify_with_replacement": {
			changes: []lib.ChangesetChanges{
				{
					Action:      "Modify",
					LogicalID:   "DatabaseInstance",
					ResourceID:  "db-instance-123",
					Type:        "AWS::RDS::DBInstance",
					Replacement: "True",
				},
			},
		},
		"mixed_operations": {
			changes: []lib.ChangesetChanges{
				{
					Action:      "Add",
					LogicalID:   "NewBucket",
					Type:        "AWS::S3::Bucket",
					Replacement: "False",
				},
				{
					Action:      "Modify",
					LogicalID:   "ExistingQueue",
					Type:        "AWS::SQS::Queue",
					Replacement: "False",
				},
				{
					Action:      "Remove",
					LogicalID:   "OldTopic",
					Type:        "AWS::SNS::Topic",
					Replacement: "False",
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer

			for _, change := range tc.changes {
				buf.WriteString(fmt.Sprintf("Action: %s\n", change.Action))
				buf.WriteString(fmt.Sprintf("LogicalResourceId: %s\n", change.LogicalID))
				buf.WriteString(fmt.Sprintf("ResourceType: %s\n", change.Type))
				buf.WriteString(fmt.Sprintf("Replacement: %s\n", change.Replacement))
				if change.ResourceID != "" {
					buf.WriteString(fmt.Sprintf("PhysicalResourceId: %s\n", change.ResourceID))
				}
				buf.WriteString("\n")
			}

			// Assert against golden file
			golden.AssertString(name, buf.String())
		})
	}
}

// TestEventFormatting_GoldenFiles tests stack event formatting
func TestEventFormatting_GoldenFiles(t *testing.T) {
	// Initialize golden file helper
	golden := testutil.NewGoldenFileWithDir(t, "testdata/golden/cmd")

	baseTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := map[string]struct {
		events []types.StackEvent
	}{
		"successful_create_events": {
			events: []types.StackEvent{
				testutil.NewStackEventBuilder("test-stack", "test-stack").
					WithStatus(types.ResourceStatusCreateInProgress).
					WithResourceType("AWS::CloudFormation::Stack").
					WithTimestamp(baseTime).
					Build(),
				testutil.NewStackEventBuilder("test-stack", "MyBucket").
					WithStatus(types.ResourceStatusCreateInProgress).
					WithResourceType("AWS::S3::Bucket").
					WithTimestamp(baseTime.Add(5 * time.Second)).
					Build(),
				testutil.NewStackEventBuilder("test-stack", "MyBucket").
					WithStatus(types.ResourceStatusCreateComplete).
					WithResourceType("AWS::S3::Bucket").
					WithTimestamp(baseTime.Add(10 * time.Second)).
					Build(),
			},
		},
		"failed_update_events": {
			events: []types.StackEvent{
				testutil.NewStackEventBuilder("test-stack", "MyBucket").
					WithStatus(types.ResourceStatusUpdateInProgress).
					WithResourceType("AWS::S3::Bucket").
					WithTimestamp(baseTime).
					Build(),
				testutil.NewStackEventBuilder("test-stack", "MyBucket").
					WithStatus(types.ResourceStatusUpdateFailed).
					WithResourceType("AWS::S3::Bucket").
					WithStatusReason("Properties validation failed").
					WithTimestamp(baseTime.Add(5 * time.Second)).
					Build(),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer

			for _, event := range tc.events {
				buf.WriteString(fmt.Sprintf("Timestamp: %s\n", event.Timestamp.Format(time.RFC3339)))
				buf.WriteString(fmt.Sprintf("LogicalResourceId: %s\n", *event.LogicalResourceId))
				buf.WriteString(fmt.Sprintf("ResourceType: %s\n", *event.ResourceType))
				buf.WriteString(fmt.Sprintf("ResourceStatus: %s\n", string(event.ResourceStatus)))
				if event.ResourceStatusReason != nil {
					buf.WriteString(fmt.Sprintf("StatusReason: %s\n", *event.ResourceStatusReason))
				}
				buf.WriteString("\n")
			}

			// Assert against golden file
			golden.AssertString(name, buf.String())
		})
	}
}

// TestChangesetInfo_GoldenFiles tests changeset info structure formatting
func TestChangesetInfo_GoldenFiles(t *testing.T) {
	// Initialize golden file helper
	golden := testutil.NewGoldenFileWithDir(t, "testdata/golden/cmd")

	tests := map[string]struct {
		changeset lib.ChangesetInfo
	}{
		"changeset_basic": {
			changeset: lib.ChangesetInfo{
				Name:      "my-changeset",
				StackName: "my-stack",
				Status:    "CREATE_COMPLETE",
				Changes: []lib.ChangesetChanges{
					{
						Action:      "Add",
						LogicalID:   "NewResource",
						Type:        "AWS::S3::Bucket",
						Replacement: "False",
					},
				},
			},
		},
		"changeset_multiple_changes": {
			changeset: lib.ChangesetInfo{
				Name:         "update-changeset",
				StackName:    "prod-stack",
				Status:       "CREATE_COMPLETE",
				StatusReason: "All resources validated",
				Changes: []lib.ChangesetChanges{
					{
						Action:      "Add",
						LogicalID:   "NewBucket",
						Type:        "AWS::S3::Bucket",
						Replacement: "False",
					},
					{
						Action:      "Modify",
						LogicalID:   "ExistingInstance",
						ResourceID:  "i-12345",
						Type:        "AWS::EC2::Instance",
						Replacement: "True",
					},
					{
						Action:      "Remove",
						LogicalID:   "OldQueue",
						Type:        "AWS::SQS::Queue",
						Replacement: "False",
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer

			buf.WriteString(fmt.Sprintf("Changeset: %s\n", tc.changeset.Name))
			buf.WriteString(fmt.Sprintf("Stack: %s\n", tc.changeset.StackName))
			buf.WriteString(fmt.Sprintf("Status: %s\n", tc.changeset.Status))
			if tc.changeset.StatusReason != "" {
				buf.WriteString(fmt.Sprintf("StatusReason: %s\n", tc.changeset.StatusReason))
			}
			buf.WriteString("\nChanges:\n")

			for i, change := range tc.changeset.Changes {
				buf.WriteString(fmt.Sprintf("  Change %d:\n", i+1))
				buf.WriteString(fmt.Sprintf("    Action: %s\n", change.Action))
				buf.WriteString(fmt.Sprintf("    LogicalID: %s\n", change.LogicalID))
				buf.WriteString(fmt.Sprintf("    Type: %s\n", change.Type))
				buf.WriteString(fmt.Sprintf("    Replacement: %s\n", change.Replacement))
				if change.ResourceID != "" {
					buf.WriteString(fmt.Sprintf("    ResourceID: %s\n", change.ResourceID))
				}
			}

			// Assert against golden file
			golden.AssertString(name, buf.String())
		})
	}
}
