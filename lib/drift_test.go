package lib

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type mockCloudFormationClient struct {
	DescribeStackDriftDetectionStatusFunc func(ctx context.Context, params *cloudformation.DescribeStackDriftDetectionStatusInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackDriftDetectionStatusOutput, error)
}

func (m *mockCloudFormationClient) DescribeStackDriftDetectionStatus(ctx context.Context, params *cloudformation.DescribeStackDriftDetectionStatusInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackDriftDetectionStatusOutput, error) {
	return m.DescribeStackDriftDetectionStatusFunc(ctx, params, optFns...)
}

func TestWaitForDriftDetectionToFinish(t *testing.T) {
	// Test case 1: detection status is already complete
	mockSvc1 := &mockCloudFormationClient{
		DescribeStackDriftDetectionStatusFunc: func(ctx context.Context, params *cloudformation.DescribeStackDriftDetectionStatusInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackDriftDetectionStatusOutput, error) {
			return &cloudformation.DescribeStackDriftDetectionStatusOutput{
				DetectionStatus: types.StackDriftDetectionStatusDetectionComplete,
			}, nil
		},
	}
	status1 := WaitForDriftDetectionToFinish(aws.String("test-id"), mockSvc1)
	if status1 != types.StackDriftDetectionStatusDetectionComplete {
		t.Errorf("Expected detection status to be DetectionComplete, but got %v", status1)
	}

	// Test case 2: detection status is in progress
	mockSvc2 := &mockCloudFormationClient{
		DescribeStackDriftDetectionStatusFunc: func(ctx context.Context, params *cloudformation.DescribeStackDriftDetectionStatusInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackDriftDetectionStatusOutput, error) {
			return &cloudformation.DescribeStackDriftDetectionStatusOutput{
				DetectionStatus: types.StackDriftDetectionStatusDetectionInProgress,
			}, nil
		},
	}
	status2 := WaitForDriftDetectionToFinish(aws.String("test-id"), mockSvc2)
	if status2 != types.StackDriftDetectionStatusDetectionComplete {
		t.Errorf("Expected detection status to be DetectionComplete, but got %v", status2)
	}

	// Test case 3: DescribeStackDriftDetectionStatus returns an error
	mockSvc3 := &mockCloudFormationClient{
		DescribeStackDriftDetectionStatusFunc: func(ctx context.Context, params *cloudformation.DescribeStackDriftDetectionStatusInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackDriftDetectionStatusOutput, error) {
			return nil, errors.New("error")
		},
	}
	status3 := WaitForDriftDetectionToFinish(aws.String("test-id"), mockSvc3)
	if status3 != "" {
		t.Errorf("Expected detection status to be empty, but got %v", status3)
	}
}
