package deploy

import (
	"context"
	"strings"
	"testing"

	ferr "github.com/ArjenSchwarz/fog/cmd/errors"
	"github.com/ArjenSchwarz/fog/cmd/services"
	"github.com/ArjenSchwarz/fog/config"
	"github.com/spf13/cobra"
)

type integrationDeploymentService struct{}

func (m integrationDeploymentService) PrepareDeployment(ctx context.Context, opts services.DeploymentOptions) (*services.DeploymentPlan, ferr.FogError) {
	return &services.DeploymentPlan{}, nil
}
func (m integrationDeploymentService) ValidateDeployment(ctx context.Context, plan *services.DeploymentPlan) ferr.FogError {
	return nil
}
func (m integrationDeploymentService) CreateChangeset(ctx context.Context, plan *services.DeploymentPlan) (*services.ChangesetResult, ferr.FogError) {
	return nil, ferr.NewError(ferr.ErrNotImplemented, "changeset logic not implemented")
}
func (m integrationDeploymentService) ExecuteDeployment(ctx context.Context, plan *services.DeploymentPlan, cs *services.ChangesetResult) (*services.DeploymentResult, ferr.FogError) {
	return &services.DeploymentResult{Success: true}, nil
}

type stubFactory struct{ cfg *config.Config }

func (f stubFactory) CreateDeploymentService() services.DeploymentService {
	return integrationDeploymentService{}
}
func (f stubFactory) CreateDriftService() services.DriftService { return nil }
func (f stubFactory) CreateStackService() services.StackService { return nil }
func (f stubFactory) AppConfig() *config.Config                 { return f.cfg }
func (f stubFactory) AWSConfig() *config.AWSConfig              { return &config.AWSConfig{} }

// buildRoot creates a root command with the deploy command registered.
func buildRoot() *cobra.Command {
	root := &cobra.Command{Use: "root"}
	factory := stubFactory{cfg: &config.Config{}}
	builder := NewCommandBuilder(factory)
	root.AddCommand(builder.BuildCommand())
	return root
}

func TestDeployCommandExecuteValid(t *testing.T) {
	root := buildRoot()
	root.SetArgs([]string{"deploy", "--stackname", "test"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "failed to create changeset") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeployCommandExecuteMissingStack(t *testing.T) {
	root := buildRoot()
	root.SetArgs([]string{"deploy"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "stack name is required") {
		t.Fatalf("expected validation error, got: %v", err)
	}
}

func TestDeployCommandExecuteConflictingFlags(t *testing.T) {
	root := buildRoot()
	root.SetArgs([]string{"deploy", "--stackname", "s", "--deployment-file", "f", "--template", "t"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "deployment file") {
		t.Fatalf("expected conflict error, got: %v", err)
	}
}
