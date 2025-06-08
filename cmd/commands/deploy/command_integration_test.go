package deploy

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"

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
	viper.Set("templates.extensions", []string{".yaml"})
	viper.Set("deployments.extensions", []string{".yaml"})
	builder := NewCommandBuilder(factory)
	root.AddCommand(builder.BuildCommand())
	return root
}

func TestDeployCommandExecuteValid(t *testing.T) {
	root := buildRoot()
	tmp := t.TempDir() + "/tmpl.yaml"
	_ = os.WriteFile(tmp, []byte("x"), 0o644)
	root.SetArgs([]string{"deploy", "--stackname", "test", "--template", tmp})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "failed to create changeset") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeployCommandExecuteMissingStack(t *testing.T) {
	root := buildRoot()
	tmp := t.TempDir() + "/tmpl.yaml"
	_ = os.WriteFile(tmp, []byte("x"), 0o644)
	root.SetArgs([]string{"deploy", "--template", tmp})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "stackname") {
		t.Fatalf("expected validation error, got: %v", err)
	}
}

func TestDeployCommandExecuteConflictingFlags(t *testing.T) {
	root := buildRoot()
	dir := t.TempDir()
	tf := dir + "/tmpl.yaml"
	df := dir + "/deploy.yaml"
	_ = os.WriteFile(tf, []byte("x"), 0o644)
	_ = os.WriteFile(df, []byte("x"), 0o644)
	root.SetArgs([]string{"deploy", "--stackname", "s", "--deployment-file", df, "--template", tf})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "conflicting flags") {
		t.Fatalf("expected conflict error, got: %v", err)
	}
}
