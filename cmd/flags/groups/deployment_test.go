package groups

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TestNewDeploymentFlags verifies validation rules are configured and flags registered.
func TestNewDeploymentFlags(t *testing.T) {
	origTmpl := viper.Get("templates.extensions")
	origDep := viper.Get("deployments.extensions")
	viper.Set("templates.extensions", []string{".yaml"})
	viper.Set("deployments.extensions", []string{".yaml"})
	t.Cleanup(func() {
		viper.Set("templates.extensions", origTmpl)
		viper.Set("deployments.extensions", origDep)
	})

	df := NewDeploymentFlags()
	if got := len(df.GetValidationRules()); got != 11 {
		t.Fatalf("expected 11 rules, got %d", got)
	}

	cmd := &cobra.Command{Use: "test"}
	df.RegisterFlags(cmd)
	flags := []string{"stackname", "template", "parameters", "tags", "bucket", "changeset", "deployment-file", "dry-run", "non-interactive", "create-changeset", "deploy-changeset", "default-tags"}
	for _, f := range flags {
		if cmd.Flags().Lookup(f) == nil {
			t.Errorf("flag %s not registered", f)
		}
	}
}
