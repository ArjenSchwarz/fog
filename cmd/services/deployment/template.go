package deployment

import (
	"context"
	"fmt"
	"path/filepath"

	ferr "github.com/ArjenSchwarz/fog/cmd/errors"
	"github.com/ArjenSchwarz/fog/cmd/services"
	"github.com/ArjenSchwarz/fog/lib"
)

// TemplateService implements services.TemplateService.
type TemplateService struct {
	s3 services.S3Client
}

// NewTemplateService returns a new TemplateService.
func NewTemplateService(s3 services.S3Client) *TemplateService {
	return &TemplateService{s3: s3}
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

// ValidateTemplate performs basic template validation.
func (ts *TemplateService) ValidateTemplate(ctx context.Context, template *services.Template) ferr.FogError {
	if template.Content == "" {
		errorCtx := ferr.GetErrorContext(ctx)
		return ferr.ContextualError(errorCtx, ferr.ErrTemplateInvalid, "template content is empty")
	}
	return nil
}

// UploadTemplate uploads a template to S3. Only placeholder logic implemented.
func (ts *TemplateService) UploadTemplate(ctx context.Context, template *services.Template, bucket string) (*services.TemplateReference, ferr.FogError) {
	key := filepath.Base(template.LocalPath)
	template.S3URL = fmt.Sprintf("s3://%s/%s", bucket, key)
	return &services.TemplateReference{URL: template.S3URL, Bucket: bucket, Key: key, Version: ""}, nil
}
