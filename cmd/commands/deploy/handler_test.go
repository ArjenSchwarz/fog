package deploy

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/fog/cmd/services"
	"github.com/ArjenSchwarz/fog/config"
)

type mockHandlerDeploymentService struct{}

func (m mockHandlerDeploymentService) PrepareDeployment(ctx context.Context, opts services.DeploymentOptions) (*services.DeploymentPlan, error) {
	return &services.DeploymentPlan{}, nil
}
func (m mockHandlerDeploymentService) ValidateDeployment(ctx context.Context, plan *services.DeploymentPlan) error {
	return nil
}
func (m mockHandlerDeploymentService) CreateChangeset(ctx context.Context, plan *services.DeploymentPlan) (*services.ChangesetResult, error) {
	return nil, fmt.Errorf("changeset logic not implemented")
}
func (m mockHandlerDeploymentService) ExecuteDeployment(ctx context.Context, plan *services.DeploymentPlan, cs *services.ChangesetResult) (*services.DeploymentResult, error) {
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
	if err := h.ValidateFlags(); err == nil {
		t.Fatalf("expected validation error when stack name missing")
	}
}

// TestExecute verifies that Execute currently returns the not implemented error.
func TestExecute(t *testing.T) {
	h := NewHandler(&Flags{StackName: "test"}, mockHandlerDeploymentService{}, &config.Config{})
	err := h.Execute(context.Background())
	if err == nil || !strings.Contains(err.Error(), "failed to create changeset") {
		t.Fatalf("unexpected error: %v", err)
	}
}
