package deployment

import (
	"context"
	"github.com/ArjenSchwarz/fog/config"
	"testing"

	"github.com/ArjenSchwarz/fog/cmd/services"
)

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
	tmplSvc := NewTemplateService(nil)
	tmpl := &services.Template{LocalPath: "/tmp/test.yaml"}
	ref, err := tmplSvc.UploadTemplate(context.Background(), tmpl, "b")
	if err != nil {
		t.Fatalf("unexpected upload error: %v", err)
	}
	if ref.URL != tmpl.S3URL || ref.Key != "test.yaml" || ref.Bucket != "b" {
		t.Fatalf("unexpected ref values")
	}

	svc := NewService(tmplSvc, NewParameterService(), NewTagService(), nil, nil, &config.Config{})
	if _, err := svc.CreateChangeset(context.Background(), &services.DeploymentPlan{}); err == nil {
		t.Fatalf("expected error from changeset")
	}
	if _, err := svc.ExecuteDeployment(context.Background(), &services.DeploymentPlan{}, nil); err == nil {
		t.Fatalf("expected error from execute")
	}
}
