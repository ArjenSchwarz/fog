package aws

import "github.com/ArjenSchwarz/fog/config"

// NewCloudFormationClient returns a wrapped CloudFormation client using the provided AWS config.
func NewCloudFormationClient(cfg config.AWSConfig) *CloudFormation {
	return NewCloudFormation(cfg.CloudformationClient())
}

// NewS3Client returns a wrapped S3 client using the provided AWS config.
func NewS3Client(cfg config.AWSConfig) *S3 {
	return NewS3(cfg.S3Client())
}
