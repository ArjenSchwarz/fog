package lib

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// These tests verify that context cancellation propagates through lib functions
// to the underlying AWS API calls. A cancelled context should result in an error.

// mockDescribeStacksCancelAPI returns the context error when the context is cancelled.
type mockDescribeStacksCancelAPI struct{}

func (m mockDescribeStacksCancelAPI) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &cloudformation.DescribeStacksOutput{}, nil
}

func TestContextPropagation_GetStack_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	stackName := "test-stack"
	_, err := GetStack(ctx, &stackName, mockDescribeStacksCancelAPI{})
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

// mockDetectDriftCancelAPI returns context error when cancelled.
type mockDetectDriftCancelAPI struct{}

func (m mockDetectDriftCancelAPI) DetectStackDrift(ctx context.Context, params *cloudformation.DetectStackDriftInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DetectStackDriftOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to start drift detection: %w", err)
	}
	return &cloudformation.DetectStackDriftOutput{}, nil
}

func TestContextPropagation_StartDriftDetection_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stackName := "test-stack"
	_, err := StartDriftDetection(ctx, &stackName, mockDetectDriftCancelAPI{})
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

// mockDescribeNaclsCancelAPI returns context error when cancelled.
type mockDescribeNaclsCancelAPI func(ctx context.Context, params *ec2.DescribeNetworkAclsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkAclsOutput, error)

func (m mockDescribeNaclsCancelAPI) DescribeNetworkAcls(ctx context.Context, params *ec2.DescribeNetworkAclsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkAclsOutput, error) {
	return m(ctx, params, optFns...)
}

func TestContextPropagation_GetNacl_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mock := mockDescribeNaclsCancelAPI(func(ctx context.Context, params *ec2.DescribeNetworkAclsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkAclsOutput, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return &ec2.DescribeNetworkAclsOutput{
			NetworkAcls: []ec2types.NetworkAcl{{}},
		}, nil
	})

	_, err := GetNacl(ctx, "nacl-123", mock)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

// mockS3CancelAPI returns context error when cancelled.
type mockS3CancelAPI struct{}

func (m mockS3CancelAPI) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &s3.PutObjectOutput{}, nil
}

func TestContextPropagation_UploadTemplate_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	templateName := "test-template"
	bucketName := "test-bucket"
	_, err := UploadTemplate(ctx, &templateName, "template-body", &bucketName, mockS3CancelAPI{})
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

// mockCreateChangeSetCancelAPI returns context error when cancelled.
type mockCreateChangeSetCancelAPI struct{}

func (m mockCreateChangeSetCancelAPI) CreateChangeSet(ctx context.Context, params *cloudformation.CreateChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateChangeSetOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id := "arn:aws:cloudformation:us-east-1:123456789012:changeSet/test"
	return &cloudformation.CreateChangeSetOutput{Id: &id}, nil
}

func TestContextPropagation_CreateChangeSet_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	deployment := &DeployInfo{
		StackName:     "test-stack",
		ChangesetName: "test-changeset",
		Template:      "{}",
	}
	_, err := deployment.CreateChangeSet(ctx, mockCreateChangeSetCancelAPI{})
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

// mockDeleteStackCancelAPI returns context error when cancelled.
type mockDeleteStackCancelAPI struct{}

func (m mockDeleteStackCancelAPI) DeleteStack(ctx context.Context, params *cloudformation.DeleteStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &cloudformation.DeleteStackOutput{}, nil
}

func TestContextPropagation_DeleteStack_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	deployment := &DeployInfo{StackName: "test-stack"}
	result := deployment.DeleteStack(ctx, mockDeleteStackCancelAPI{})
	if result {
		t.Fatal("expected DeleteStack to return false with cancelled context")
	}
}

// mockDescribeStackEventsCancelAPI returns context error when cancelled.
type mockDescribeStackEventsCancelAPI struct{}

func (m mockDescribeStackEventsCancelAPI) DescribeStackEvents(ctx context.Context, params *cloudformation.DescribeStackEventsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackEventsOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &cloudformation.DescribeStackEventsOutput{
		StackEvents: []types.StackEvent{},
	}, nil
}

func TestContextPropagation_GetEvents_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	deployment := &DeployInfo{StackName: "test-stack"}
	_, err := deployment.GetEvents(ctx, mockDescribeStackEventsCancelAPI{})
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

func TestContextPropagation_LiveContext_GetStack(t *testing.T) {
	// Verify that a valid context passes through correctly (no error)
	stackName := "test-stack"
	name := stackName
	mock := mockDescribeStacksCancelAPI{}
	// With a live (non-cancelled) context, GetStack should work (though return "no stacks found")
	_, err := GetStack(context.Background(), &name, mock)
	if err == nil {
		t.Fatal("expected 'no stacks found' error, got nil")
	}
	// The error should NOT be about context cancellation
	if err == context.Canceled || err == context.DeadlineExceeded {
		t.Fatalf("unexpected context error: %v", err)
	}
}
