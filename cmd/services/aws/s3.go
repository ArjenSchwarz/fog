package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3 wraps the AWS SDK S3 client to satisfy services.S3Client.
type S3 struct{ client *s3.Client }

// NewS3 creates a new S3 wrapper.
func NewS3(c *s3.Client) *S3 { return &S3{client: c} }

func (s *S3) PutObject(ctx context.Context, input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return s.client.PutObject(ctx, input)
}

func (s *S3) GetObject(ctx context.Context, input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return s.client.GetObject(ctx, input)
}
