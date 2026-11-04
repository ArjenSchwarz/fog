package lib

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTestFile creates parent directories and writes content to path.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// TestLoadDeploymentFile_TemplatePathRelativeToDeploymentFile is a regression
// test for T-1414: template-file-path is documented as relative to the
// deployment file, but LoadDeploymentFile discarded the resolved deployment
// file path so the template path was later resolved against the working
// directory. These tests use the real viper deployments settings and the
// working directory, so they cannot run in parallel.
func TestLoadDeploymentFile_TemplatePathRelativeToDeploymentFile(t *testing.T) {
	absTemplate := filepath.Join(t.TempDir(), "abs", "vpc.yaml")

	tests := map[string]struct {
		deploymentDir    string
		templateFilePath string
		wantTemplatePath string
	}{
		"parent-relative path from deployments directory": {
			deploymentDir:    "deployments",
			templateFilePath: "../templates/vpc.yaml",
			wantTemplatePath: filepath.Join("templates", "vpc.yaml"),
		},
		"subdirectory-relative path from deployments directory": {
			deploymentDir:    "deployments",
			templateFilePath: "templates/vpc.yaml",
			wantTemplatePath: filepath.Join("deployments", "templates", "vpc.yaml"),
		},
		"deployment file in working directory is unchanged": {
			deploymentDir:    ".",
			templateFilePath: "templates/vpc.yaml",
			wantTemplatePath: filepath.Join("templates", "vpc.yaml"),
		},
		"absolute path is kept as is": {
			deploymentDir:    "deployments",
			templateFilePath: absTemplate,
			wantTemplatePath: absTemplate,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("deployments.directory", tc.deploymentDir)
			viper.Set("deployments.extensions", []string{"", ".yaml"})
			writeTestFile(t, filepath.Join(tc.deploymentDir, "vpc-dev.yaml"),
				"template-file-path: "+tc.templateFilePath+"\n")

			deployment := DeployInfo{}
			require.NoError(t, deployment.LoadDeploymentFile("vpc-dev"))

			require.NotNil(t, deployment.StackDeploymentFile)
			assert.Equal(t, tc.wantTemplatePath, deployment.StackDeploymentFile.TemplateFilePath)
		})
	}
}

// TestLoadDeploymentFile_ExplicitDeploymentFilePath verifies that a deployment
// file passed as a direct path (outside the configured deployments directory)
// anchors the template path to that file's own directory.
func TestLoadDeploymentFile_ExplicitDeploymentFilePath(t *testing.T) {
	t.Chdir(t.TempDir())
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("deployments.directory", "unused")
	writeTestFile(t, filepath.Join("stacks", "network", "vpc.yaml"),
		`{"template-file-path": "../../templates/vpc.yaml"}`)

	deployment := DeployInfo{}
	require.NoError(t, deployment.LoadDeploymentFile(filepath.Join("stacks", "network", "vpc.yaml")))

	assert.Equal(t, filepath.Join("templates", "vpc.yaml"), deployment.StackDeploymentFile.TemplateFilePath)
}
