package deployment

import (
	"context"
	"fmt"
	"path/filepath"

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
func (ts *TemplateService) LoadTemplate(ctx context.Context, templatePath string) (*services.Template, error) {
	content, path, err := lib.ReadTemplate(&templatePath)
	if err != nil {
		return nil, err
	}
	return &services.Template{Content: content, LocalPath: path}, nil
}

// ValidateTemplate performs basic template validation.
func (ts *TemplateService) ValidateTemplate(ctx context.Context, template *services.Template) error {
	if template.Content == "" {
		return fmt.Errorf("template content is empty")
	}
	return nil
}

// UploadTemplate uploads a template to S3. Only placeholder logic implemented.
func (ts *TemplateService) UploadTemplate(ctx context.Context, template *services.Template, bucket string) (*services.TemplateReference, error) {
	key := filepath.Base(template.LocalPath)
	template.S3URL = fmt.Sprintf("s3://%s/%s", bucket, key)
	return &services.TemplateReference{URL: template.S3URL, Bucket: bucket, Key: key, Version: ""}, nil
}
