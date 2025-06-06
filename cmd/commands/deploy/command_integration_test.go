package deploy

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// buildRoot creates a root command with the deploy command registered.
func buildRoot() *cobra.Command {
	root := &cobra.Command{Use: "root"}
	builder := NewCommandBuilder()
	root.AddCommand(builder.BuildCommand())
	return root
}

func TestDeployCommandExecuteValid(t *testing.T) {
	root := buildRoot()
	root.SetArgs([]string{"deploy", "--stackname", "test"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "not yet implemented") {
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
