package deployment

import (
	"context"
	"testing"

	ferr "github.com/ArjenSchwarz/fog/cmd/errors"
	"github.com/ArjenSchwarz/fog/cmd/services"
	svcaws "github.com/ArjenSchwarz/fog/cmd/services/aws"
	"github.com/ArjenSchwarz/fog/config"
	cfnTypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type stubTemplateService struct {
	loadFunc     func(context.Context, string) (*services.Template, ferr.FogError)
	validateFunc func(context.Context, *services.Template) ferr.FogError
}

func (s stubTemplateService) LoadTemplate(ctx context.Context, path string) (*services.Template, ferr.FogError) {
	if s.loadFunc != nil {
		return s.loadFunc(ctx, path)
	}
	return &services.Template{}, nil
}

func (s stubTemplateService) ValidateTemplate(ctx context.Context, t *services.Template) ferr.FogError {
	if s.validateFunc != nil {
		return s.validateFunc(ctx, t)
	}
	return nil
}

func (s stubTemplateService) UploadTemplate(ctx context.Context, t *services.Template, bucket string) (*services.TemplateReference, ferr.FogError) {
	return &services.TemplateReference{URL: t.S3URL}, nil
}

type stubParameterService struct {
	loadFunc     func(context.Context, []string) ([]cfnTypes.Parameter, ferr.FogError)
	validateFunc func(context.Context, []cfnTypes.Parameter, *services.Template) ferr.FogError
}

func (s stubParameterService) LoadParameters(ctx context.Context, files []string) ([]cfnTypes.Parameter, ferr.FogError) {
	if s.loadFunc != nil {
		return s.loadFunc(ctx, files)
	}
	return nil, nil
}

func (s stubParameterService) ValidateParameters(ctx context.Context, params []cfnTypes.Parameter, t *services.Template) ferr.FogError {
	if s.validateFunc != nil {
		return s.validateFunc(ctx, params, t)
	}
	return nil
}

type stubTagService struct {
	loadFunc     func(context.Context, []string, map[string]string) ([]cfnTypes.Tag, ferr.FogError)
	validateFunc func(context.Context, []cfnTypes.Tag) ferr.FogError
}

func (s stubTagService) LoadTags(ctx context.Context, files []string, defaults map[string]string) ([]cfnTypes.Tag, ferr.FogError) {
	if s.loadFunc != nil {
		return s.loadFunc(ctx, files, defaults)
	}
	return nil, nil
}

func (s stubTagService) ValidateTags(ctx context.Context, tags []cfnTypes.Tag) ferr.FogError {
	if s.validateFunc != nil {
		return s.validateFunc(ctx, tags)
	}
	return nil
}

// TestServicePrepareDeployment exercises PrepareDeployment for both the happy
// path and when template loading fails.
func TestServicePrepareDeployment(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		called := struct{ tmpl, params, tags bool }{}
		tmplSvc := stubTemplateService{loadFunc: func(ctx context.Context, p string) (*services.Template, ferr.FogError) {
			called.tmpl = true
			if p != "tpl" {
				t.Errorf("expected template path tpl, got %s", p)
			}
			return &services.Template{Content: "c", LocalPath: "p"}, nil
		}}
		paramSvc := stubParameterService{loadFunc: func(ctx context.Context, f []string) ([]cfnTypes.Parameter, ferr.FogError) {
			called.params = true
			return []cfnTypes.Parameter{{ParameterKey: strPtr("k")}}, nil
		}}
		tagSvc := stubTagService{loadFunc: func(ctx context.Context, f []string, d map[string]string) ([]cfnTypes.Tag, ferr.FogError) {
			called.tags = true
			return []cfnTypes.Tag{}, nil
		}}
		svc := NewService(tmplSvc, paramSvc, tagSvc, &svcaws.MockCloudFormationClient{}, &svcaws.MockS3Client{}, &config.Config{})
		plan, err := svc.PrepareDeployment(ctx, services.DeploymentOptions{StackName: "stack", TemplateSource: "tpl"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called.tmpl || !called.params || !called.tags {
			t.Fatalf("expected all loaders called")
		}
		if plan.StackName != "stack" || plan.Template.Content != "c" {
			t.Errorf("plan fields not populated")
		}
		if plan.ChangesetName != "fog-changeset" {
			t.Errorf("default changeset name not set")
		}
	})

	t.Run("template error", func(t *testing.T) {
		tmplSvc := stubTemplateService{loadFunc: func(ctx context.Context, p string) (*services.Template, ferr.FogError) {
			return nil, ferr.NewError(ferr.ErrTemplateNotFound, "boom")
		}}
		svc := NewService(tmplSvc, stubParameterService{}, stubTagService{}, &svcaws.MockCloudFormationClient{}, &svcaws.MockS3Client{}, &config.Config{})
		_, err := svc.PrepareDeployment(ctx, services.DeploymentOptions{TemplateSource: "tpl"})
		if err == nil {
			t.Fatalf("expected error")
		}
	})
}

// TestServiceValidateDeployment checks that ValidateDeployment accepts valid
// plans and returns errors for invalid ones.
func TestServiceValidateDeployment(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		tmplSvc := stubTemplateService{validateFunc: func(context.Context, *services.Template) ferr.FogError { return nil }}
		paramSvc := stubParameterService{validateFunc: func(context.Context, []cfnTypes.Parameter, *services.Template) ferr.FogError { return nil }}
		tagSvc := stubTagService{validateFunc: func(context.Context, []cfnTypes.Tag) ferr.FogError { return nil }}
		svc := NewService(tmplSvc, paramSvc, tagSvc, &svcaws.MockCloudFormationClient{}, &svcaws.MockS3Client{}, &config.Config{})
		plan := &services.DeploymentPlan{Template: &services.Template{Content: "c"}}
		if err := svc.ValidateDeployment(ctx, plan); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("template validation error", func(t *testing.T) {
		tmplSvc := stubTemplateService{validateFunc: func(context.Context, *services.Template) ferr.FogError {
			return ferr.NewError(ferr.ErrTemplateInvalid, "bad")
		}}
		svc := NewService(tmplSvc, stubParameterService{}, stubTagService{}, &svcaws.MockCloudFormationClient{}, &svcaws.MockS3Client{}, &config.Config{})
		plan := &services.DeploymentPlan{Template: &services.Template{Content: ""}}
		if err := svc.ValidateDeployment(ctx, plan); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func strPtr(s string) *string { return &s }
