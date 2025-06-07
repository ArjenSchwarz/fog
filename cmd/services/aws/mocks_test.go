package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// TestCloudFormationClient and TestS3Client are aliases so other packages can
// use the mock implementations defined in mocks.go.
type TestCloudFormationClient = MockCloudFormationClient

type TestS3Client = MockS3Client

// TestMockCloudFormationClient ensures the mock CloudFormation client returns
// default values and that optional custom functions are invoked when set.
func TestMockCloudFormationClient(t *testing.T) {
	m := &MockCloudFormationClient{}
	// default branches
	if out, err := m.DescribeStacks(context.Background(), &cloudformation.DescribeStacksInput{}); err != nil || out == nil {
		t.Fatalf("unexpected default DescribeStacks result")
	}
	if _, err := m.CreateChangeSet(context.Background(), &cloudformation.CreateChangeSetInput{}); err != nil {
		t.Fatalf("unexpected default CreateChangeSet result")
	}
	if _, err := m.ExecuteChangeSet(context.Background(), &cloudformation.ExecuteChangeSetInput{}); err != nil {
		t.Fatalf("unexpected default ExecuteChangeSet result")
	}
	if _, err := m.DescribeChangeSet(context.Background(), &cloudformation.DescribeChangeSetInput{}); err != nil {
		t.Fatalf("unexpected default DescribeChangeSet result")
	}
	// custom functions
	called := struct{ create, exec, desc bool }{}
	m.CreateChangeSetFunc = func(ctx context.Context, in *cloudformation.CreateChangeSetInput) (*cloudformation.CreateChangeSetOutput, error) {
		called.create = true
		return &cloudformation.CreateChangeSetOutput{}, nil
	}
	m.ExecuteChangeSetFunc = func(ctx context.Context, in *cloudformation.ExecuteChangeSetInput) (*cloudformation.ExecuteChangeSetOutput, error) {
		called.exec = true
		return &cloudformation.ExecuteChangeSetOutput{}, nil
	}
	m.DescribeChangeSetFunc = func(ctx context.Context, in *cloudformation.DescribeChangeSetInput) (*cloudformation.DescribeChangeSetOutput, error) {
		called.desc = true
		return &cloudformation.DescribeChangeSetOutput{}, nil
	}
	m.DescribeStacksFunc = func(ctx context.Context, in *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
		return &cloudformation.DescribeStacksOutput{}, nil
	}
	_, _ = m.CreateChangeSet(context.Background(), &cloudformation.CreateChangeSetInput{})
	_, _ = m.ExecuteChangeSet(context.Background(), &cloudformation.ExecuteChangeSetInput{})
	_, _ = m.DescribeChangeSet(context.Background(), &cloudformation.DescribeChangeSetInput{})
	_, _ = m.DescribeStacks(context.Background(), &cloudformation.DescribeStacksInput{})
	if !called.create || !called.exec || !called.desc {
		t.Fatalf("expected all custom functions invoked")
	}
}

// TestMockS3Client verifies the S3 client mock returns default outputs and that
// optional override functions are executed.
func TestMockS3Client(t *testing.T) {
	m := &MockS3Client{}
	if out, err := m.PutObject(context.Background(), &s3.PutObjectInput{}); err != nil || out == nil {
		t.Fatalf("unexpected default PutObject result")
	}
	if out, err := m.GetObject(context.Background(), &s3.GetObjectInput{}); err != nil || out == nil {
		t.Fatalf("unexpected default GetObject result")
	}
	called := struct{ put, get bool }{}
	m.PutObjectFunc = func(ctx context.Context, in *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
		called.put = true
		return &s3.PutObjectOutput{}, nil
	}
	m.GetObjectFunc = func(ctx context.Context, in *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
		called.get = true
		return &s3.GetObjectOutput{}, nil
	}
	_, _ = m.PutObject(context.Background(), &s3.PutObjectInput{})
	_, _ = m.GetObject(context.Background(), &s3.GetObjectInput{})
	if !called.put || !called.get {
		t.Fatalf("expected custom functions invoked")
	}
}
