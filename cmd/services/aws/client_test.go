package aws

import (
	"context"
	"testing"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// TestNewClientsAndWrappers verifies that the AWS client constructors return
// valid client instances and that wrapper methods propagate errors when no
// region has been configured.

func TestNewClientsAndWrappers(t *testing.T) {
	cfg := config.AWSConfig{}
	cfn := NewCloudFormationClient(cfg)
	if cfn == nil {
		t.Fatalf("expected client")
	}
	ctx := context.Background()
	if _, err := cfn.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{}); err == nil {
		t.Errorf("expected error due to missing region")
	}
	if _, err := cfn.CreateChangeSet(ctx, &cloudformation.CreateChangeSetInput{}); err == nil {
		t.Errorf("expected error")
	}
	if _, err := cfn.ExecuteChangeSet(ctx, &cloudformation.ExecuteChangeSetInput{}); err == nil {
		t.Errorf("expected error")
	}
	if _, err := cfn.DescribeChangeSet(ctx, &cloudformation.DescribeChangeSetInput{}); err == nil {
		t.Errorf("expected error")
	}

	s3c := NewS3Client(cfg)
	if s3c == nil {
		t.Fatalf("expected s3 client")
	}
	if _, err := s3c.GetObject(ctx, &s3.GetObjectInput{}); err == nil {
		t.Errorf("expected error due to missing region")
	}
	if _, err := s3c.PutObject(ctx, &s3.PutObjectInput{}); err == nil {
		t.Errorf("expected error")
	}
}
