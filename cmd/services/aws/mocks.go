package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// MockCloudFormationClient is a simple mock implementing services.CloudFormationClient.
type MockCloudFormationClient struct {
	DescribeStacksFunc    func(context.Context, *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error)
	CreateChangeSetFunc   func(context.Context, *cloudformation.CreateChangeSetInput) (*cloudformation.CreateChangeSetOutput, error)
	ExecuteChangeSetFunc  func(context.Context, *cloudformation.ExecuteChangeSetInput) (*cloudformation.ExecuteChangeSetOutput, error)
	DescribeChangeSetFunc func(context.Context, *cloudformation.DescribeChangeSetInput) (*cloudformation.DescribeChangeSetOutput, error)
}

func (m *MockCloudFormationClient) DescribeStacks(ctx context.Context, in *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
	if m.DescribeStacksFunc != nil {
		return m.DescribeStacksFunc(ctx, in)
	}
	return &cloudformation.DescribeStacksOutput{}, nil
}

func (m *MockCloudFormationClient) CreateChangeSet(ctx context.Context, in *cloudformation.CreateChangeSetInput) (*cloudformation.CreateChangeSetOutput, error) {
	if m.CreateChangeSetFunc != nil {
		return m.CreateChangeSetFunc(ctx, in)
	}
	return &cloudformation.CreateChangeSetOutput{}, nil
}

func (m *MockCloudFormationClient) ExecuteChangeSet(ctx context.Context, in *cloudformation.ExecuteChangeSetInput) (*cloudformation.ExecuteChangeSetOutput, error) {
	if m.ExecuteChangeSetFunc != nil {
		return m.ExecuteChangeSetFunc(ctx, in)
	}
	return &cloudformation.ExecuteChangeSetOutput{}, nil
}

func (m *MockCloudFormationClient) DescribeChangeSet(ctx context.Context, in *cloudformation.DescribeChangeSetInput) (*cloudformation.DescribeChangeSetOutput, error) {
	if m.DescribeChangeSetFunc != nil {
		return m.DescribeChangeSetFunc(ctx, in)
	}
	return &cloudformation.DescribeChangeSetOutput{}, nil
}

// MockS3Client is a simple mock implementing services.S3Client.
type MockS3Client struct {
	PutObjectFunc func(context.Context, *s3.PutObjectInput) (*s3.PutObjectOutput, error)
	GetObjectFunc func(context.Context, *s3.GetObjectInput) (*s3.GetObjectOutput, error)
}

func (m *MockS3Client) PutObject(ctx context.Context, in *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	if m.PutObjectFunc != nil {
		return m.PutObjectFunc(ctx, in)
	}
	return &s3.PutObjectOutput{}, nil
}

func (m *MockS3Client) GetObject(ctx context.Context, in *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	if m.GetObjectFunc != nil {
		return m.GetObjectFunc(ctx, in)
	}
	return &s3.GetObjectOutput{}, nil
}
