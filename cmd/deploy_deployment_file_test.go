package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ArjenSchwarz/fog/lib"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReadTemplateFromSource_DeploymentFileRelativeTemplate is a regression
// test for T-1414. It uses the documented layout of deployments/vpc-dev.yaml
// pointing at ../templates/vpc.yaml and runs from the repository root. A decoy
// template is placed at ../templates/vpc.yaml relative to the working
// directory: before the fix that decoy was deployed instead of the template
// next to the deployment file. Not parallel: it changes the working directory
// and global viper settings.
func TestReadTemplateFromSource_DeploymentFileRelativeTemplate(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	writeFile := func(path, content string) {
		t.Helper()
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	writeFile(filepath.Join(repo, "deployments", "vpc-dev.yaml"), "template-file-path: ../templates/vpc.yaml\n")
	writeFile(filepath.Join(repo, "templates", "vpc.yaml"), "Description: intended\n")
	writeFile(filepath.Join(root, "templates", "vpc.yaml"), "Description: decoy\n")

	t.Chdir(repo)
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("deployments.directory", "deployments")
	viper.Set("deployments.extensions", []string{"", ".yaml"})
	viper.Set("templates.directory", "templates")
	viper.Set("templates.extensions", []string{"", ".yaml"})

	deployment := lib.DeployInfo{}
	require.NoError(t, deployment.LoadDeploymentFile("vpc-dev"))

	gotTemplate, gotPath := readTemplateFromSource(&deployment)

	assert.Equal(t, "Description: intended\n", gotTemplate)
	assert.Equal(t, filepath.Join("templates", "vpc.yaml"), gotPath)
}

// TestReadTemplateFromSource_DeploymentFileMissingTemplateFails is a
// regression test for T-1414 review feedback: a deployment file reference that
// does not exist next to the deployment file must fail, rather than falling
// back to templates.directory and silently reading an unrelated template
// (here the decoy templates/deployments/missing.yaml).
func TestReadTemplateFromSource_DeploymentFileMissingTemplateFails(t *testing.T) {
	repo := t.TempDir()
	writeFile := func(path, content string) {
		t.Helper()
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	writeFile(filepath.Join(repo, "deployments", "dev.yaml"), "template-file-path: missing.yaml\n")
	writeFile(filepath.Join(repo, "templates", "deployments", "missing.yaml"), "Description: decoy\n")

	t.Chdir(repo)
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("debug", false)
	viper.Set("deployments.directory", "deployments")
	viper.Set("deployments.extensions", []string{"", ".yaml"})
	viper.Set("templates.directory", "templates")
	viper.Set("templates.extensions", []string{"", ".yaml"})

	deployment := lib.DeployInfo{}
	require.NoError(t, deployment.LoadDeploymentFile("dev"))

	var gotTemplate string
	code, called, stdout, stderr := runDeployWithInterceptedExit(t, func() {
		gotTemplate, _ = readTemplateFromSource(&deployment)
	})

	assert.True(t, called, "expected the missing template to fail via failWithError")
	assert.Equal(t, 1, code)
	assert.Empty(t, gotTemplate, "decoy template must not be read")
	assert.Empty(t, stdout)
	assert.Contains(t, stderr, filepath.Join("deployments", "missing.yaml"))
}
