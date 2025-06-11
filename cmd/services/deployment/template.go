package deployment

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	ferr "github.com/ArjenSchwarz/fog/cmd/errors"
	"github.com/ArjenSchwarz/fog/cmd/services"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// TemplateService implements services.TemplateService.
type TemplateService struct {
	s3  services.S3Client
	cfn services.CloudFormationClient
}

// NewTemplateService returns a new TemplateService.
func NewTemplateService(s3 services.S3Client, cfn services.CloudFormationClient) *TemplateService {
	return &TemplateService{s3: s3, cfn: cfn}
}

// LoadTemplate loads a template from disk using lib.ReadTemplate.
func (ts *TemplateService) LoadTemplate(ctx context.Context, templatePath string) (*services.Template, ferr.FogError) {
	content, path, err := lib.ReadTemplate(&templatePath)
	if err != nil {
		errorCtx := ferr.GetErrorContext(ctx)
		return nil, ferr.ContextualError(errorCtx, ferr.ErrTemplateNotFound, err.Error())
	}
	return &services.Template{Content: content, LocalPath: path}, nil
}

// ValidateTemplate performs comprehensive template validation using CloudFormation APIs.
func (ts *TemplateService) ValidateTemplate(ctx context.Context, template *services.Template) ferr.FogError {
	// Basic validation first
	if template.Content == "" {
		errorCtx := ferr.GetErrorContext(ctx)
		return ferr.ContextualError(errorCtx, ferr.ErrTemplateInvalid, "template content is empty")
	}

	// Use CloudFormation ValidateTemplate API for comprehensive validation
	input := &cloudformation.ValidateTemplateInput{
		TemplateBody: &template.Content,
	}

	_, err := ts.cfn.ValidateTemplate(ctx, input)
	if err != nil {
		errorCtx := ferr.GetErrorContext(ctx)
		return ferr.ContextualError(errorCtx, ferr.ErrTemplateInvalid,
			fmt.Sprintf("CloudFormation template validation failed: %v", err))
	}

	return nil
}

// UploadTemplate uploads a template to S3 using the S3Client interface.
func (ts *TemplateService) UploadTemplate(ctx context.Context, template *services.Template, bucket string) (*services.TemplateReference, ferr.FogError) {
	templateName := filepath.Base(template.LocalPath)

	// Generate a unique key similar to lib.UploadTemplate logic
	// Use the same naming pattern as lib.UploadTemplate for consistency
	key := fmt.Sprintf("fog/%v-%v", templateName, time.Now().UnixNano())

	// Upload using the S3Client interface
	_, err := ts.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   strings.NewReader(template.Content),
	})
	if err != nil {
		errorCtx := ferr.GetErrorContext(ctx)
		return nil, ferr.ContextualError(errorCtx, ferr.ErrTemplateUploadFailed,
			fmt.Sprintf("failed to upload template to S3: %v", err))
	}

	// Build the S3 URL
	s3URL := fmt.Sprintf("s3://%s/%s", bucket, key)
	template.S3URL = s3URL

	return &services.TemplateReference{
		URL:     s3URL,
		Bucket:  bucket,
		Key:     key,
		Version: "", // S3 versioning not implemented yet
	}, nil
}
