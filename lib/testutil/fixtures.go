package testutil

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// Common test fixtures for reuse across tests

// SampleTemplates provides sample CloudFormation templates for testing
var SampleTemplates = struct {
	SimpleVPC      string
	S3Bucket       string
	InvalidJSON    string
	InvalidYAML    string
	LargeTemplate  string
	NestedStack    string
	WithParameters string
	WithMappings   string
	WithConditions string
	WithOutputs    string
}{
	SimpleVPC: `{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Description": "Simple VPC Template",
  "Resources": {
    "VPC": {
      "Type": "AWS::EC2::VPC",
      "Properties": {
        "CidrBlock": "10.0.0.0/16",
        "EnableDnsHostnames": true,
        "Tags": [
          {
            "Key": "Name",
            "Value": "TestVPC"
          }
        ]
      }
    }
  }
}`,

	S3Bucket: `{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Description": "Simple S3 Bucket",
  "Resources": {
    "MyBucket": {
      "Type": "AWS::S3::Bucket",
      "Properties": {
        "BucketName": "my-test-bucket"
      }
    }
  }
}`,

	InvalidJSON: `{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Description": "Invalid JSON - missing closing brace",
  "Resources": {
    "MyBucket": {
      "Type": "AWS::S3::Bucket"
`,

	InvalidYAML: `
AWSTemplateFormatVersion: '2010-09-09'
Description: Invalid YAML - bad indentation
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
  Properties:  # This should be indented under MyBucket
    BucketName: test-bucket
`,

	LargeTemplate: generateLargeTemplate(50),

	NestedStack: `{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Description": "Parent stack with nested stack",
  "Resources": {
    "NestedStack": {
      "Type": "AWS::CloudFormation::Stack",
      "Properties": {
        "TemplateURL": "https://s3.amazonaws.com/mybucket/nested-template.json",
        "Parameters": {
          "EnvironmentName": "Test"
        }
      }
    }
  }
}`,

	WithParameters: `{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Description": "Template with parameters",
  "Parameters": {
    "EnvironmentName": {
      "Type": "String",
      "Default": "Dev",
      "AllowedValues": ["Dev", "Test", "Prod"],
      "Description": "Environment name"
    },
    "InstanceType": {
      "Type": "String",
      "Default": "t2.micro",
      "Description": "EC2 instance type"
    }
  },
  "Resources": {
    "MyBucket": {
      "Type": "AWS::S3::Bucket",
      "Properties": {
        "BucketName": {
          "Fn::Sub": "${EnvironmentName}-bucket"
        }
      }
    }
  }
}`,

	WithMappings: `{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Description": "Template with mappings",
  "Mappings": {
    "RegionMap": {
      "us-east-1": {
        "AMI": "ami-12345678"
      },
      "us-west-2": {
        "AMI": "ami-87654321"
      }
    }
  },
  "Resources": {
    "MyInstance": {
      "Type": "AWS::EC2::Instance",
      "Properties": {
        "ImageId": {
          "Fn::FindInMap": ["RegionMap", {"Ref": "AWS::Region"}, "AMI"]
        },
        "InstanceType": "t2.micro"
      }
    }
  }
}`,

	WithConditions: `{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Description": "Template with conditions",
  "Parameters": {
    "CreateProdResources": {
      "Type": "String",
      "Default": "false",
      "AllowedValues": ["true", "false"]
    }
  },
  "Conditions": {
    "CreateProdResourcesCondition": {
      "Fn::Equals": [{"Ref": "CreateProdResources"}, "true"]
    }
  },
  "Resources": {
    "ProdBucket": {
      "Type": "AWS::S3::Bucket",
      "Condition": "CreateProdResourcesCondition",
      "Properties": {
        "BucketName": "prod-bucket"
      }
    }
  }
}`,

	WithOutputs: `{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Description": "Template with outputs",
  "Resources": {
    "MyBucket": {
      "Type": "AWS::S3::Bucket"
    }
  },
  "Outputs": {
    "BucketName": {
      "Description": "Name of the S3 bucket",
      "Value": {"Ref": "MyBucket"},
      "Export": {
        "Name": "MyBucketName"
      }
    },
    "BucketArn": {
      "Description": "ARN of the S3 bucket",
      "Value": {
        "Fn::GetAtt": ["MyBucket", "Arn"]
      }
    }
  }
}`,
}

// generateLargeTemplate creates a template with many resources for testing size limits
func generateLargeTemplate(resourceCount int) string {
	resources := make(map[string]any)
	for i := 0; i < resourceCount; i++ {
		resourceName := fmt.Sprintf("Bucket%d", i)
		resources[resourceName] = map[string]any{
			"Type": "AWS::S3::Bucket",
			"Properties": map[string]any{
				"BucketName": fmt.Sprintf("test-bucket-%d", i),
			},
		}
	}

	template := map[string]any{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Description":              fmt.Sprintf("Large template with %d resources", resourceCount),
		"Resources":                resources,
	}

	jsonBytes, _ := json.MarshalIndent(template, "", "  ")
	return string(jsonBytes)
}

// SampleConfigs provides sample configuration files for testing
var SampleConfigs = struct {
	ValidYAML   string
	ValidJSON   string
	ValidTOML   string
	InvalidYAML string
	InvalidJSON string
	ComplexYAML string
}{
	ValidYAML: `
region: us-west-2
profile: default
templates:
  directory: ./templates
  validate: true
parameters:
  Environment: Dev
  Application: TestApp
tags:
  Team: Platform
  CostCenter: Engineering
`,

	ValidJSON: `{
  "region": "us-east-1",
  "profile": "prod",
  "templates": {
    "directory": "./cloudformation",
    "validate": false
  },
  "parameters": {
    "Environment": "Production"
  }
}`,

	ValidTOML: `
region = "eu-west-1"
profile = "staging"

[templates]
directory = "./infrastructure"
validate = true

[parameters]
Environment = "Staging"
Application = "WebApp"

[tags]
Team = "DevOps"
CostCenter = "Operations"
`,

	InvalidYAML: `
region: us-west-2
profile: [invalid array when string expected]
`,

	InvalidJSON: `{
  "region": "us-west-2",
  "profile": "default",
  // Comments are not valid in JSON
  "templates": {
    "directory": "./templates"
  }
}`,

	ComplexYAML: `
region: us-west-2
profile: ${AWS_PROFILE}
environments:
  dev:
    region: us-west-2
    parameters:
      Environment: Development
  prod:
    region: us-east-1
    parameters:
      Environment: Production
      HighAvailability: true
templates:
  directory: ./templates
  naming:
    pattern: "{environment}-{name}-{region}"
  s3:
    bucket: cf-templates-bucket
    prefix: templates/
capabilities:
  - CAPABILITY_IAM
  - CAPABILITY_NAMED_IAM
notification_arns:
  - arn:aws:sns:us-west-2:123456789012:stack-notifications
`,
}

// SampleStackResponses provides pre-configured stack responses for testing
var SampleStackResponses = struct {
	CreateInProgress *types.Stack
	CreateComplete   *types.Stack
	UpdateInProgress *types.Stack
	UpdateComplete   *types.Stack
	DeleteInProgress *types.Stack
	DeleteComplete   *types.Stack
	RollbackComplete *types.Stack
	Failed           *types.Stack
	WithOutputs      *types.Stack
	WithDrift        *types.Stack
}{
	CreateInProgress: NewStackBuilder("test-stack").
		WithStatus(types.StackStatusCreateInProgress).
		WithDescription("Stack creation in progress").
		Build(),

	CreateComplete: NewStackBuilder("test-stack").
		WithStatus(types.StackStatusCreateComplete).
		WithDescription("Stack successfully created").
		Build(),

	UpdateInProgress: NewStackBuilder("test-stack").
		WithStatus(types.StackStatusUpdateInProgress).
		WithDescription("Stack update in progress").
		Build(),

	UpdateComplete: NewStackBuilder("test-stack").
		WithStatus(types.StackStatusUpdateComplete).
		WithDescription("Stack successfully updated").
		Build(),

	DeleteInProgress: NewStackBuilder("test-stack").
		WithStatus(types.StackStatusDeleteInProgress).
		WithDescription("Stack deletion in progress").
		Build(),

	DeleteComplete: NewStackBuilder("test-stack").
		WithStatus(types.StackStatusDeleteComplete).
		WithDescription("Stack successfully deleted").
		Build(),

	RollbackComplete: NewStackBuilder("test-stack").
		WithStatus(types.StackStatusRollbackComplete).
		WithDescription("Stack rolled back").
		Build(),

	Failed: NewStackBuilder("test-stack").
		WithStatus(types.StackStatusCreateFailed).
		WithDescription("Stack creation failed").
		Build(),

	WithOutputs: NewStackBuilder("test-stack").
		WithStatus(types.StackStatusCreateComplete).
		WithOutput("BucketName", "my-bucket").
		WithOutput("BucketArn", "arn:aws:s3:::my-bucket").
		Build(),

	WithDrift: NewStackBuilder("test-stack").
		WithStatus(types.StackStatusCreateComplete).
		WithDriftStatus(types.StackDriftStatusDrifted).
		Build(),
}

// SampleChangesets provides sample changeset data for testing
func SampleChangesets() map[string]*cloudformation.DescribeChangeSetOutput {
	return map[string]*cloudformation.DescribeChangeSetOutput{
		"add-resource": {
			ChangeSetName:   aws.String("add-resource"),
			ChangeSetId:     aws.String("arn:aws:cloudformation:us-west-2:123456789012:changeSet/add-resource/12345"),
			StackName:       aws.String("test-stack"),
			Status:          types.ChangeSetStatusCreateComplete,
			ExecutionStatus: types.ExecutionStatusAvailable,
			Changes: []types.Change{
				{
					Type: types.ChangeTypeResource,
					ResourceChange: &types.ResourceChange{
						Action:            types.ChangeActionAdd,
						LogicalResourceId: aws.String("MyBucket"),
						ResourceType:      aws.String("AWS::S3::Bucket"),
						Details: []types.ResourceChangeDetail{
							{
								Target: &types.ResourceTargetDefinition{
									Attribute: types.ResourceAttributeProperties,
									Name:      aws.String("BucketName"),
								},
								Evaluation:   types.EvaluationTypeStatic,
								ChangeSource: types.ChangeSourceParameterReference,
							},
						},
					},
				},
			},
		},
		"modify-resource": {
			ChangeSetName:   aws.String("modify-resource"),
			ChangeSetId:     aws.String("arn:aws:cloudformation:us-west-2:123456789012:changeSet/modify-resource/67890"),
			StackName:       aws.String("test-stack"),
			Status:          types.ChangeSetStatusCreateComplete,
			ExecutionStatus: types.ExecutionStatusAvailable,
			Changes: []types.Change{
				{
					Type: types.ChangeTypeResource,
					ResourceChange: &types.ResourceChange{
						Action:            types.ChangeActionModify,
						LogicalResourceId: aws.String("MyBucket"),
						ResourceType:      aws.String("AWS::S3::Bucket"),
						Replacement:       types.ReplacementFalse,
						Details: []types.ResourceChangeDetail{
							{
								Target: &types.ResourceTargetDefinition{
									Attribute: types.ResourceAttributeProperties,
									Name:      aws.String("VersioningConfiguration"),
								},
								Evaluation:   types.EvaluationTypeStatic,
								ChangeSource: types.ChangeSourceDirectModification,
							},
						},
					},
				},
			},
		},
		"remove-resource": {
			ChangeSetName:   aws.String("remove-resource"),
			ChangeSetId:     aws.String("arn:aws:cloudformation:us-west-2:123456789012:changeSet/remove-resource/54321"),
			StackName:       aws.String("test-stack"),
			Status:          types.ChangeSetStatusCreateComplete,
			ExecutionStatus: types.ExecutionStatusAvailable,
			Changes: []types.Change{
				{
					Type: types.ChangeTypeResource,
					ResourceChange: &types.ResourceChange{
						Action:            types.ChangeActionRemove,
						LogicalResourceId: aws.String("OldBucket"),
						ResourceType:      aws.String("AWS::S3::Bucket"),
					},
				},
			},
		},
		"no-changes": {
			ChangeSetName:   aws.String("no-changes"),
			ChangeSetId:     aws.String("arn:aws:cloudformation:us-west-2:123456789012:changeSet/no-changes/99999"),
			StackName:       aws.String("test-stack"),
			Status:          types.ChangeSetStatusFailed,
			StatusReason:    aws.String("The submitted information didn't contain changes."),
			ExecutionStatus: types.ExecutionStatusUnavailable,
		},
	}
}

// SampleStackEvents provides sample stack events for testing
func SampleStackEvents() []types.StackEvent {
	return []types.StackEvent{
		NewStackEventBuilder("test-stack", "test-stack").
			WithStatus(types.ResourceStatusCreateInProgress).
			WithResourceType("AWS::CloudFormation::Stack").
			WithStatusReason("User Initiated").
			Build(),
		NewStackEventBuilder("test-stack", "MyBucket").
			WithStatus(types.ResourceStatusCreateInProgress).
			WithResourceType("AWS::S3::Bucket").
			Build(),
		NewStackEventBuilder("test-stack", "MyBucket").
			WithStatus(types.ResourceStatusCreateComplete).
			WithResourceType("AWS::S3::Bucket").
			Build(),
		NewStackEventBuilder("test-stack", "test-stack").
			WithStatus(types.ResourceStatusCreateComplete).
			WithResourceType("AWS::CloudFormation::Stack").
			Build(),
	}
}

// SampleStackResources provides sample stack resources for testing
func SampleStackResources() []types.StackResource {
	return []types.StackResource{
		{
			LogicalResourceId:  aws.String("VPC"),
			PhysicalResourceId: aws.String("vpc-12345678"),
			ResourceType:       aws.String("AWS::EC2::VPC"),
			ResourceStatus:     types.ResourceStatusCreateComplete,
		},
		{
			LogicalResourceId:  aws.String("PublicSubnet"),
			PhysicalResourceId: aws.String("subnet-12345678"),
			ResourceType:       aws.String("AWS::EC2::Subnet"),
			ResourceStatus:     types.ResourceStatusCreateComplete,
		},
		{
			LogicalResourceId:  aws.String("InternetGateway"),
			PhysicalResourceId: aws.String("igw-12345678"),
			ResourceType:       aws.String("AWS::EC2::InternetGateway"),
			ResourceStatus:     types.ResourceStatusCreateComplete,
		},
		{
			LogicalResourceId:  aws.String("RouteTable"),
			PhysicalResourceId: aws.String("rtb-12345678"),
			ResourceType:       aws.String("AWS::EC2::RouteTable"),
			ResourceStatus:     types.ResourceStatusCreateComplete,
		},
	}
}

// SampleValidationError creates a sample CloudFormation validation error
func SampleValidationError(message string) error {
	return fmt.Errorf("%s", message)
}

// LoadTestTemplate loads a test template from a string, handling both JSON and YAML
func LoadTestTemplate(templateContent string) (map[string]any, error) {
	var result map[string]any

	// Try to parse as JSON first
	if err := json.Unmarshal([]byte(templateContent), &result); err == nil {
		return result, nil
	}

	// If not JSON, assume it's YAML and return a simple mock
	// (In a real implementation, you'd use a YAML parser)
	if strings.Contains(templateContent, "AWSTemplateFormatVersion") {
		return map[string]any{
			"AWSTemplateFormatVersion": "2010-09-09",
			"Description":              "Mock YAML template",
			"Resources":                map[string]any{},
		}, nil
	}

	return nil, fmt.Errorf("invalid template format")
}
