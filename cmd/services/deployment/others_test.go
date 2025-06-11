package deployment

import (
	"context"
	"fmt"
	"testing"

	"github.com/ArjenSchwarz/fog/config"

	"github.com/ArjenSchwarz/fog/cmd/services"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Local test mocks
type testS3Client struct{}

func (c *testS3Client) PutObject(ctx context.Context, input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return &s3.PutObjectOutput{}, nil
}

func (c *testS3Client) GetObject(ctx context.Context, input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return &s3.GetObjectOutput{}, nil
}

type testCfnClient struct{}

func (c *testCfnClient) DescribeStacks(ctx context.Context, input *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
	return &cloudformation.DescribeStacksOutput{}, nil
}

func (c *testCfnClient) CreateChangeSet(ctx context.Context, input *cloudformation.CreateChangeSetInput) (*cloudformation.CreateChangeSetOutput, error) {
	// Return an error to simulate changeset creation failure for testing
	return nil, fmt.Errorf("test changeset creation error")
}

func (c *testCfnClient) ExecuteChangeSet(ctx context.Context, input *cloudformation.ExecuteChangeSetInput) (*cloudformation.ExecuteChangeSetOutput, error) {
	// Return an error to simulate execution failure for testing
	return nil, fmt.Errorf("test changeset execution error")
}

func (c *testCfnClient) DescribeChangeSet(ctx context.Context, input *cloudformation.DescribeChangeSetInput) (*cloudformation.DescribeChangeSetOutput, error) {
	return &cloudformation.DescribeChangeSetOutput{}, nil
}

func (c *testCfnClient) ValidateTemplate(ctx context.Context, input *cloudformation.ValidateTemplateInput) (*cloudformation.ValidateTemplateOutput, error) {
	return &cloudformation.ValidateTemplateOutput{}, nil
}

// TestParameterAndTagServices exercises the basic load and validate helpers for
// parameters and tags.
func TestParameterAndTagServices(t *testing.T) {
	ps := NewParameterService()
	params, err := ps.LoadParameters(context.Background(), []string{"p.json"})
	if err != nil || len(params) != 0 {
		t.Fatalf("unexpected parameters result")
	}
	if err := ps.ValidateParameters(context.Background(), params, &services.Template{}); err != nil {
		t.Fatalf("unexpected validate error")
	}

	ts := NewTagService()
	tags, err := ts.LoadTags(context.Background(), nil, map[string]string{"k": "v"})
	if err != nil || len(tags) != 1 || *tags[0].Key != "k" {
		t.Fatalf("tags not loaded")
	}
	if err := ts.ValidateTags(context.Background(), tags); err != nil {
		t.Fatalf("unexpected tag validate error")
	}
}

// TestTemplateServiceUploadAndCreateExecute checks template upload and error
// handling for change set creation and execution.
func TestTemplateServiceUploadAndCreateExecute(t *testing.T) {
	// Create mock clients for testing
	mockS3 := &testS3Client{}
	mockCfn := &testCfnClient{}
	tmplSvc := NewTemplateService(mockS3, mockCfn)
	tmpl := &services.Template{LocalPath: "/tmp/test.yaml"}
	ref, err := tmplSvc.UploadTemplate(context.Background(), tmpl, "b")
	if err != nil {
		t.Fatalf("unexpected upload error: %v", err)
	}
	// Check basic ref values (key includes timestamp so just check it's not empty)
	if ref.URL != tmpl.S3URL || ref.Key == "" || ref.Bucket != "b" {
		t.Fatalf("unexpected ref values: URL=%s, Key=%s, Bucket=%s", ref.URL, ref.Key, ref.Bucket)
	}

	svc := NewService(tmplSvc, NewParameterService(), NewTagService(), mockCfn, mockS3, &config.Config{})
	// Create a valid deployment plan with proper template
	plan := &services.DeploymentPlan{
		StackName:     "test-stack",
		ChangesetName: "test-changeset",
		Template:      &services.Template{Content: "Resources: {}"},
	}
	if _, err := svc.CreateChangeset(context.Background(), plan); err == nil {
		t.Fatalf("expected error from changeset")
	}
	// Create a valid changeset for testing execution
	changeset := &services.ChangesetResult{
		Name: "test-changeset",
		ID:   "test-changeset-id",
	}
	if _, err := svc.ExecuteDeployment(context.Background(), plan, changeset); err == nil {
		t.Fatalf("expected error from execute")
	}
}
