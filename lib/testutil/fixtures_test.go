package testutil

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSampleTemplates(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		name     string
		template string
		validate func(t *testing.T, template string)
	}{
		"SimpleVPC is valid JSON": {
			template: SampleTemplates.SimpleVPC,
			validate: func(t *testing.T, template string) {
				var parsed map[string]any
				err := json.Unmarshal([]byte(template), &parsed)
				require.NoError(t, err, "SimpleVPC should be valid JSON")

				assert.Equal(t, "2010-09-09", parsed["AWSTemplateFormatVersion"])
				assert.Equal(t, "Simple VPC Template", parsed["Description"])

				resources, ok := parsed["Resources"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, resources, "VPC")

				vpc, ok := resources["VPC"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "AWS::EC2::VPC", vpc["Type"])
			},
		},
		"S3Bucket is valid JSON": {
			template: SampleTemplates.S3Bucket,
			validate: func(t *testing.T, template string) {
				var parsed map[string]any
				err := json.Unmarshal([]byte(template), &parsed)
				require.NoError(t, err, "S3Bucket should be valid JSON")

				resources, ok := parsed["Resources"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, resources, "MyBucket")

				bucket, ok := resources["MyBucket"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "AWS::S3::Bucket", bucket["Type"])
			},
		},
		"InvalidJSON is actually invalid": {
			template: SampleTemplates.InvalidJSON,
			validate: func(t *testing.T, template string) {
				var parsed map[string]any
				err := json.Unmarshal([]byte(template), &parsed)
				assert.Error(t, err, "InvalidJSON should fail to parse")
			},
		},
		"InvalidYAML has incorrect structure": {
			template: SampleTemplates.InvalidYAML,
			validate: func(t *testing.T, template string) {
				assert.Contains(t, template, "Properties:")
				assert.Contains(t, template, "# This should be indented")
			},
		},
		"LargeTemplate has many resources": {
			template: SampleTemplates.LargeTemplate,
			validate: func(t *testing.T, template string) {
				var parsed map[string]any
				err := json.Unmarshal([]byte(template), &parsed)
				require.NoError(t, err)

				resources, ok := parsed["Resources"].(map[string]any)
				require.True(t, ok)
				assert.Len(t, resources, 50, "LargeTemplate should have 50 resources")
			},
		},
		"NestedStack has nested stack resource": {
			template: SampleTemplates.NestedStack,
			validate: func(t *testing.T, template string) {
				var parsed map[string]any
				err := json.Unmarshal([]byte(template), &parsed)
				require.NoError(t, err)

				resources, ok := parsed["Resources"].(map[string]any)
				require.True(t, ok)
				nestedStack, ok := resources["NestedStack"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "AWS::CloudFormation::Stack", nestedStack["Type"])
			},
		},
		"WithParameters has parameter definitions": {
			template: SampleTemplates.WithParameters,
			validate: func(t *testing.T, template string) {
				var parsed map[string]any
				err := json.Unmarshal([]byte(template), &parsed)
				require.NoError(t, err)

				params, ok := parsed["Parameters"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, params, "EnvironmentName")
				assert.Contains(t, params, "InstanceType")
			},
		},
		"WithMappings has mappings section": {
			template: SampleTemplates.WithMappings,
			validate: func(t *testing.T, template string) {
				var parsed map[string]any
				err := json.Unmarshal([]byte(template), &parsed)
				require.NoError(t, err)

				mappings, ok := parsed["Mappings"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, mappings, "RegionMap")
			},
		},
		"WithConditions has conditions section": {
			template: SampleTemplates.WithConditions,
			validate: func(t *testing.T, template string) {
				var parsed map[string]any
				err := json.Unmarshal([]byte(template), &parsed)
				require.NoError(t, err)

				conditions, ok := parsed["Conditions"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, conditions, "CreateProdResourcesCondition")
			},
		},
		"WithOutputs has outputs section": {
			template: SampleTemplates.WithOutputs,
			validate: func(t *testing.T, template string) {
				var parsed map[string]any
				err := json.Unmarshal([]byte(template), &parsed)
				require.NoError(t, err)

				outputs, ok := parsed["Outputs"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, outputs, "BucketName")
				assert.Contains(t, outputs, "BucketArn")
			},
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tc.validate(t, tc.template)
		})
	}
}

func TestSampleConfigs(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		name     string
		config   string
		validate func(t *testing.T, config string)
	}{
		"ValidYAML has correct structure": {
			config: SampleConfigs.ValidYAML,
			validate: func(t *testing.T, config string) {
				assert.Contains(t, config, "region: us-west-2")
				assert.Contains(t, config, "profile: default")
				assert.Contains(t, config, "templates:")
				assert.Contains(t, config, "parameters:")
				assert.Contains(t, config, "tags:")
			},
		},
		"ValidJSON is parseable": {
			config: SampleConfigs.ValidJSON,
			validate: func(t *testing.T, config string) {
				var parsed map[string]any
				err := json.Unmarshal([]byte(config), &parsed)
				require.NoError(t, err)

				assert.Equal(t, "us-east-1", parsed["region"])
				assert.Equal(t, "prod", parsed["profile"])

				templates, ok := parsed["templates"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "./cloudformation", templates["directory"])
			},
		},
		"ValidTOML has TOML structure": {
			config: SampleConfigs.ValidTOML,
			validate: func(t *testing.T, config string) {
				assert.Contains(t, config, `region = "eu-west-1"`)
				assert.Contains(t, config, `profile = "staging"`)
				assert.Contains(t, config, "[templates]")
				assert.Contains(t, config, "[parameters]")
				assert.Contains(t, config, "[tags]")
			},
		},
		"InvalidYAML has invalid structure": {
			config: SampleConfigs.InvalidYAML,
			validate: func(t *testing.T, config string) {
				assert.Contains(t, config, "[invalid array when string expected]")
			},
		},
		"InvalidJSON has comments": {
			config: SampleConfigs.InvalidJSON,
			validate: func(t *testing.T, config string) {
				assert.Contains(t, config, "// Comments are not valid in JSON")

				var parsed map[string]any
				err := json.Unmarshal([]byte(config), &parsed)
				assert.Error(t, err, "Should fail to parse due to comments")
			},
		},
		"ComplexYAML has nested structure": {
			config: SampleConfigs.ComplexYAML,
			validate: func(t *testing.T, config string) {
				assert.Contains(t, config, "environments:")
				assert.Contains(t, config, "dev:")
				assert.Contains(t, config, "prod:")
				assert.Contains(t, config, "${AWS_PROFILE}")
				assert.Contains(t, config, "capabilities:")
				assert.Contains(t, config, "notification_arns:")
			},
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tc.validate(t, tc.config)
		})
	}
}

func TestSampleStackResponses(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		name  string
		stack *types.Stack
		check func(t *testing.T, stack *types.Stack)
	}{
		"CreateInProgress": {
			stack: SampleStackResponses.CreateInProgress,
			check: func(t *testing.T, stack *types.Stack) {
				assert.Equal(t, "test-stack", *stack.StackName)
				assert.Equal(t, types.StackStatusCreateInProgress, stack.StackStatus)
				assert.Equal(t, "Stack creation in progress", *stack.Description)
			},
		},
		"CreateComplete": {
			stack: SampleStackResponses.CreateComplete,
			check: func(t *testing.T, stack *types.Stack) {
				assert.Equal(t, types.StackStatusCreateComplete, stack.StackStatus)
				assert.Equal(t, "Stack successfully created", *stack.Description)
			},
		},
		"UpdateInProgress": {
			stack: SampleStackResponses.UpdateInProgress,
			check: func(t *testing.T, stack *types.Stack) {
				assert.Equal(t, types.StackStatusUpdateInProgress, stack.StackStatus)
				assert.Equal(t, "Stack update in progress", *stack.Description)
			},
		},
		"UpdateComplete": {
			stack: SampleStackResponses.UpdateComplete,
			check: func(t *testing.T, stack *types.Stack) {
				assert.Equal(t, types.StackStatusUpdateComplete, stack.StackStatus)
				assert.Equal(t, "Stack successfully updated", *stack.Description)
			},
		},
		"DeleteInProgress": {
			stack: SampleStackResponses.DeleteInProgress,
			check: func(t *testing.T, stack *types.Stack) {
				assert.Equal(t, types.StackStatusDeleteInProgress, stack.StackStatus)
			},
		},
		"DeleteComplete": {
			stack: SampleStackResponses.DeleteComplete,
			check: func(t *testing.T, stack *types.Stack) {
				assert.Equal(t, types.StackStatusDeleteComplete, stack.StackStatus)
			},
		},
		"RollbackComplete": {
			stack: SampleStackResponses.RollbackComplete,
			check: func(t *testing.T, stack *types.Stack) {
				assert.Equal(t, types.StackStatusRollbackComplete, stack.StackStatus)
			},
		},
		"Failed": {
			stack: SampleStackResponses.Failed,
			check: func(t *testing.T, stack *types.Stack) {
				assert.Equal(t, types.StackStatusCreateFailed, stack.StackStatus)
			},
		},
		"WithOutputs": {
			stack: SampleStackResponses.WithOutputs,
			check: func(t *testing.T, stack *types.Stack) {
				assert.Equal(t, types.StackStatusCreateComplete, stack.StackStatus)
				assert.Len(t, stack.Outputs, 2)

				outputs := make(map[string]string)
				for _, o := range stack.Outputs {
					outputs[*o.OutputKey] = *o.OutputValue
				}

				assert.Equal(t, "my-bucket", outputs["BucketName"])
				assert.Equal(t, "arn:aws:s3:::my-bucket", outputs["BucketArn"])
			},
		},
		"WithDrift": {
			stack: SampleStackResponses.WithDrift,
			check: func(t *testing.T, stack *types.Stack) {
				assert.Equal(t, types.StackStatusCreateComplete, stack.StackStatus)
				require.NotNil(t, stack.DriftInformation)
				assert.Equal(t, types.StackDriftStatusDrifted, stack.DriftInformation.StackDriftStatus)
			},
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.NotNil(t, tc.stack)
			require.NotNil(t, tc.stack.StackName)
			tc.check(t, tc.stack)
		})
	}
}

func TestSampleChangesets(t *testing.T) {
	t.Helper()

	changesets := SampleChangesets()

	tests := map[string]struct {
		name     string
		csName   string
		validate func(t *testing.T, changeset *cloudformation.DescribeChangeSetOutput)
	}{
		"add-resource changeset": {
			csName: "add-resource",
			validate: func(t *testing.T, changeset *cloudformation.DescribeChangeSetOutput) {
				assert.Equal(t, "add-resource", *changeset.ChangeSetName)
				assert.Equal(t, types.ChangeSetStatusCreateComplete, changeset.Status)
				assert.Equal(t, types.ExecutionStatusAvailable, changeset.ExecutionStatus)

				require.Len(t, changeset.Changes, 1)
				change := changeset.Changes[0]
				assert.Equal(t, types.ChangeTypeResource, change.Type)
				assert.Equal(t, types.ChangeActionAdd, change.ResourceChange.Action)
				assert.Equal(t, "MyBucket", *change.ResourceChange.LogicalResourceId)
			},
		},
		"modify-resource changeset": {
			csName: "modify-resource",
			validate: func(t *testing.T, changeset *cloudformation.DescribeChangeSetOutput) {
				assert.Equal(t, "modify-resource", *changeset.ChangeSetName)
				assert.Equal(t, types.ChangeActionModify, changeset.Changes[0].ResourceChange.Action)
				assert.Equal(t, types.ReplacementFalse, changeset.Changes[0].ResourceChange.Replacement)
			},
		},
		"remove-resource changeset": {
			csName: "remove-resource",
			validate: func(t *testing.T, changeset *cloudformation.DescribeChangeSetOutput) {
				assert.Equal(t, "remove-resource", *changeset.ChangeSetName)
				assert.Equal(t, types.ChangeActionRemove, changeset.Changes[0].ResourceChange.Action)
				assert.Equal(t, "OldBucket", *changeset.Changes[0].ResourceChange.LogicalResourceId)
			},
		},
		"no-changes changeset": {
			csName: "no-changes",
			validate: func(t *testing.T, changeset *cloudformation.DescribeChangeSetOutput) {
				assert.Equal(t, "no-changes", *changeset.ChangeSetName)
				assert.Equal(t, types.ChangeSetStatusFailed, changeset.Status)
				assert.Equal(t, types.ExecutionStatusUnavailable, changeset.ExecutionStatus)
				assert.Contains(t, *changeset.StatusReason, "didn't contain changes")
			},
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			changeset, exists := changesets[tc.csName]
			require.True(t, exists, "Changeset %s should exist", tc.csName)
			tc.validate(t, changeset)
		})
	}
}

func TestSampleStackEvents(t *testing.T) {
	t.Helper()

	events := SampleStackEvents()

	assert.Len(t, events, 4, "Should have 4 sample events")

	// Check the sequence of events
	expectedSequence := []struct {
		logicalID      string
		resourceStatus types.ResourceStatus
	}{
		{"test-stack", types.ResourceStatusCreateInProgress},
		{"MyBucket", types.ResourceStatusCreateInProgress},
		{"MyBucket", types.ResourceStatusCreateComplete},
		{"test-stack", types.ResourceStatusCreateComplete},
	}

	for i, expected := range expectedSequence {
		assert.Equal(t, expected.logicalID, *events[i].LogicalResourceId)
		assert.Equal(t, expected.resourceStatus, events[i].ResourceStatus)
	}
}

func TestSampleStackResources(t *testing.T) {
	t.Helper()

	resources := SampleStackResources()

	assert.Len(t, resources, 4, "Should have 4 sample resources")

	// Check resource types
	expectedResources := map[string]string{
		"VPC":             "AWS::EC2::VPC",
		"PublicSubnet":    "AWS::EC2::Subnet",
		"InternetGateway": "AWS::EC2::InternetGateway",
		"RouteTable":      "AWS::EC2::RouteTable",
	}

	for _, resource := range resources {
		expectedType, ok := expectedResources[*resource.LogicalResourceId]
		assert.True(t, ok, "Unexpected resource: %s", *resource.LogicalResourceId)
		assert.Equal(t, expectedType, *resource.ResourceType)
		assert.Equal(t, types.ResourceStatusCreateComplete, resource.ResourceStatus)
		assert.NotNil(t, resource.PhysicalResourceId)
	}
}

func TestSampleValidationError(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		message string
		check   func(t *testing.T, err error)
	}{
		"simple error message": {
			message: "Template validation failed",
			check: func(t *testing.T, err error) {
				assert.Equal(t, "Template validation failed", err.Error())
			},
		},
		"detailed error message": {
			message: "Property BucketName cannot be empty",
			check: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "BucketName")
				assert.Contains(t, err.Error(), "cannot be empty")
			},
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := SampleValidationError(tc.message)
			require.Error(t, err)
			tc.check(t, err)
		})
	}
}

func TestLoadTestTemplate(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		name     string
		input    string
		wantErr  bool
		validate func(t *testing.T, result map[string]any)
	}{
		"valid JSON template": {
			input: `{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Description": "Test template",
				"Resources": {
					"Bucket": {
						"Type": "AWS::S3::Bucket"
					}
				}
			}`,
			validate: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "2010-09-09", result["AWSTemplateFormatVersion"])
				assert.Equal(t, "Test template", result["Description"])

				resources, ok := result["Resources"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, resources, "Bucket")
			},
		},
		"YAML-like template": {
			input: `AWSTemplateFormatVersion: '2010-09-09'
Description: YAML template
Resources:
  Bucket:
    Type: AWS::S3::Bucket`,
			validate: func(t *testing.T, result map[string]any) {
				// Since we're mocking YAML parsing, check for expected mock response
				assert.Equal(t, "2010-09-09", result["AWSTemplateFormatVersion"])
				assert.Equal(t, "Mock YAML template", result["Description"])

				resources, ok := result["Resources"].(map[string]any)
				require.True(t, ok)
				assert.NotNil(t, resources)
			},
		},
		"invalid template": {
			input:   "not a valid template",
			wantErr: true,
		},
		"empty template": {
			input:   "",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := LoadTestTemplate(tc.input)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			tc.validate(t, result)
		})
	}
}

func TestGenerateLargeTemplate(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		resourceCount int
		validate      func(t *testing.T, template string)
	}{
		"generate 10 resources": {
			resourceCount: 10,
			validate: func(t *testing.T, template string) {
				var parsed map[string]any
				err := json.Unmarshal([]byte(template), &parsed)
				require.NoError(t, err)

				resources, ok := parsed["Resources"].(map[string]any)
				require.True(t, ok)
				assert.Len(t, resources, 10)

				// Check specific resources exist
				assert.Contains(t, resources, "Bucket0")
				assert.Contains(t, resources, "Bucket9")
			},
		},
		"generate 100 resources": {
			resourceCount: 100,
			validate: func(t *testing.T, template string) {
				var parsed map[string]any
				err := json.Unmarshal([]byte(template), &parsed)
				require.NoError(t, err)

				resources, ok := parsed["Resources"].(map[string]any)
				require.True(t, ok)
				assert.Len(t, resources, 100)

				// Check description
				desc, ok := parsed["Description"].(string)
				require.True(t, ok)
				assert.Contains(t, desc, "100 resources")
			},
		},
		"generate 0 resources": {
			resourceCount: 0,
			validate: func(t *testing.T, template string) {
				var parsed map[string]any
				err := json.Unmarshal([]byte(template), &parsed)
				require.NoError(t, err)

				resources, ok := parsed["Resources"].(map[string]any)
				require.True(t, ok)
				assert.Empty(t, resources)
			},
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			template := generateLargeTemplate(tc.resourceCount)
			tc.validate(t, template)
		})
	}
}

// TestFixtureConsistency verifies that all fixtures maintain consistency
func TestFixtureConsistency(t *testing.T) {
	t.Helper()

	t.Run("all stack responses have required fields", func(t *testing.T) {
		stacks := []*types.Stack{
			SampleStackResponses.CreateInProgress,
			SampleStackResponses.CreateComplete,
			SampleStackResponses.UpdateInProgress,
			SampleStackResponses.UpdateComplete,
			SampleStackResponses.DeleteInProgress,
			SampleStackResponses.DeleteComplete,
			SampleStackResponses.RollbackComplete,
			SampleStackResponses.Failed,
			SampleStackResponses.WithOutputs,
			SampleStackResponses.WithDrift,
		}

		for _, stack := range stacks {
			assert.NotNil(t, stack.StackName, "Stack should have a name")
			assert.NotNil(t, stack.StackId, "Stack should have an ID")
			assert.NotNil(t, stack.CreationTime, "Stack should have creation time")
			assert.NotEmpty(t, stack.StackStatus, "Stack should have a status")
		}
	})

	t.Run("template fixtures are distinct", func(t *testing.T) {
		templates := []string{
			SampleTemplates.SimpleVPC,
			SampleTemplates.S3Bucket,
			SampleTemplates.NestedStack,
			SampleTemplates.WithParameters,
			SampleTemplates.WithMappings,
			SampleTemplates.WithConditions,
			SampleTemplates.WithOutputs,
		}

		seen := make(map[string]bool)
		for _, template := range templates {
			assert.False(t, seen[template], "Template should be unique")
			seen[template] = true
		}
	})

	t.Run("config fixtures are well-formed", func(t *testing.T) {
		configs := []struct {
			name   string
			config string
		}{
			{"ValidYAML", SampleConfigs.ValidYAML},
			{"ValidJSON", SampleConfigs.ValidJSON},
			{"ValidTOML", SampleConfigs.ValidTOML},
			{"ComplexYAML", SampleConfigs.ComplexYAML},
		}

		for _, cfg := range configs {
			assert.NotEmpty(t, cfg.config, "%s should not be empty", cfg.name)
			assert.True(t, strings.Contains(cfg.config, "region") || strings.Contains(cfg.config, "profile"),
				"%s should contain region or profile", cfg.name)
		}
	})
}

// TestFixtureHelpers tests helper functions that work with fixtures
func TestFixtureHelpers(t *testing.T) {
	t.Helper()

	t.Run("SampleChangesets returns map with expected keys", func(t *testing.T) {
		changesets := SampleChangesets()

		expectedKeys := []string{
			"add-resource",
			"modify-resource",
			"remove-resource",
			"no-changes",
		}

		for _, key := range expectedKeys {
			_, exists := changesets[key]
			assert.True(t, exists, "Should have changeset: %s", key)
		}

		assert.Len(t, changesets, len(expectedKeys))
	})

	t.Run("SampleStackEvents returns chronological sequence", func(t *testing.T) {
		events := SampleStackEvents()

		// Events should represent a logical sequence
		assert.Equal(t, types.ResourceStatusCreateInProgress, events[0].ResourceStatus)
		assert.Equal(t, types.ResourceStatusCreateComplete, events[len(events)-1].ResourceStatus)

		// All events should have timestamps
		for i, event := range events {
			assert.NotNil(t, event.Timestamp, "Event %d should have timestamp", i)
			assert.NotNil(t, event.EventId, "Event %d should have ID", i)
		}
	})

	t.Run("SampleStackResources returns complete resources", func(t *testing.T) {
		resources := SampleStackResources()

		for _, resource := range resources {
			assert.NotNil(t, resource.LogicalResourceId)
			assert.NotNil(t, resource.PhysicalResourceId)
			assert.NotNil(t, resource.ResourceType)
			assert.NotEmpty(t, resource.ResourceStatus)
		}
	})
}
