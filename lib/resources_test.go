package lib

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type mockCloudFormationClient struct {
	describeStacksOutput         cloudformation.DescribeStacksOutput
	describeStackResourcesOutput cloudformation.DescribeStackResourcesOutput
}

func (m *mockCloudFormationClient) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	return &m.describeStacksOutput, nil
}

func (m *mockCloudFormationClient) DescribeStackResources(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error) {
	return &m.describeStackResourcesOutput, nil
}

func TestGetResources(t *testing.T) {
	mock := &mockCloudFormationClient{
		describeStacksOutput: cloudformation.DescribeStacksOutput{
			Stacks: []types.Stack{
				{StackName: aws.String("test-stack")},
			},
		},
		describeStackResourcesOutput: cloudformation.DescribeStackResourcesOutput{
			StackResources: []types.StackResource{
				{
					PhysicalResourceId: aws.String("resource-id"),
					LogicalResourceId:  aws.String("logical-id"),
					ResourceType:       aws.String("AWS::S3::Bucket"),
					ResourceStatus:     types.ResourceStatusCreateComplete,
				},
			},
		},
	}

	stackName := "test-stack"
	got := GetResources(&stackName, mock)

	want := []CfnResource{
		{
			StackName:  "test-stack",
			Type:       "AWS::S3::Bucket",
			ResourceID: "resource-id",
			LogicalID:  "logical-id",
			Status:     "CREATE_COMPLETE",
		},
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d resources, got %d", len(want), len(got))
	}

	if got[0] != want[0] {
		t.Errorf("unexpected resource: got %+v, want %+v", got[0], want[0])
	}
}
