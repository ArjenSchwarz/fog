package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

// CloudFormation wraps the AWS SDK CloudFormation client to satisfy services.CloudFormationClient.
type CloudFormation struct{ client *cloudformation.Client }

// NewCloudFormation creates a new CloudFormation wrapper.
func NewCloudFormation(c *cloudformation.Client) *CloudFormation { return &CloudFormation{client: c} }

func (c *CloudFormation) DescribeStacks(ctx context.Context, input *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
	return c.client.DescribeStacks(ctx, input)
}

func (c *CloudFormation) CreateChangeSet(ctx context.Context, input *cloudformation.CreateChangeSetInput) (*cloudformation.CreateChangeSetOutput, error) {
	return c.client.CreateChangeSet(ctx, input)
}

func (c *CloudFormation) ExecuteChangeSet(ctx context.Context, input *cloudformation.ExecuteChangeSetInput) (*cloudformation.ExecuteChangeSetOutput, error) {
	return c.client.ExecuteChangeSet(ctx, input)
}

func (c *CloudFormation) DescribeChangeSet(ctx context.Context, input *cloudformation.DescribeChangeSetInput) (*cloudformation.DescribeChangeSetOutput, error) {
	return c.client.DescribeChangeSet(ctx, input)
}
