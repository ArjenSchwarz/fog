package deploy

import (
	"context"
	"testing"

	ferr "github.com/ArjenSchwarz/fog/cmd/errors"
	"github.com/ArjenSchwarz/fog/cmd/services"
	"github.com/ArjenSchwarz/fog/config"
)

type mockHandlerDeploymentService struct{}

func (m mockHandlerDeploymentService) PrepareDeployment(ctx context.Context, opts services.DeploymentOptions) (*services.DeploymentPlan, ferr.FogError) {
	return &services.DeploymentPlan{}, nil
}
func (m mockHandlerDeploymentService) ValidateDeployment(ctx context.Context, plan *services.DeploymentPlan) ferr.FogError {
	return nil
}
func (m mockHandlerDeploymentService) CreateChangeset(ctx context.Context, plan *services.DeploymentPlan) (*services.ChangesetResult, ferr.FogError) {
	return nil, ferr.NewError(ferr.ErrNotImplemented, "changeset logic not implemented")
}
func (m mockHandlerDeploymentService) ExecuteDeployment(ctx context.Context, plan *services.DeploymentPlan, cs *services.ChangesetResult) (*services.DeploymentResult, ferr.FogError) {
	return &services.DeploymentResult{Success: true}, nil
}

// TestValidateFlags verifies that ValidateFlags returns any errors from the Flags
// validation logic.
func TestValidateFlags(t *testing.T) {
	h := NewHandler(&Flags{StackName: "test"}, mockHandlerDeploymentService{}, &config.Config{})
	if err := h.ValidateFlags(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	h = NewHandler(&Flags{}, mockHandlerDeploymentService{}, &config.Config{})
	err := h.ValidateFlags()
	if err == nil {
		t.Fatalf("expected validation error when stack name missing")
	}
	fe, ok := err.(ferr.FogError)
	if !ok || fe.Code() != ferr.ErrRequiredField {
		t.Fatalf("unexpected error type: %#v", err)
	}
}

// TestExecute verifies that Execute currently returns the not implemented error.
func TestExecute(t *testing.T) {
	h := NewHandler(&Flags{StackName: "test"}, mockHandlerDeploymentService{}, &config.Config{})
	err := h.Execute(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
	fe, ok := err.(ferr.FogError)
	if !ok || fe.Code() != ferr.ErrChangesetFailed {
		t.Fatalf("unexpected error: %#v", err)
	}
}
