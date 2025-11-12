package cmd

import (
	"path/filepath"
	"testing"

	"github.com/ArjenSchwarz/fog/lib"
	"github.com/spf13/viper"
)

// TestLoadDefaultTags tests loading tags from default configuration
func TestLoadDefaultTags(t *testing.T) {
	// Save original viper config and restore after test
	originalTags := viper.GetStringMapString("tags.default")
	defer func() {
		viper.Set("tags.default", originalTags)
	}()

	tests := []struct {
		name        string
		defaultTags map[string]string
		deployment  *lib.DeployInfo
		wantCount   int
		wantKey     string
		wantValue   string
	}{
		{
			name: "single tag",
			defaultTags: map[string]string{
				"Environment": "test",
			},
			deployment: &lib.DeployInfo{},
			wantCount:  1,
			wantKey:    "Environment",
			wantValue:  "test",
		},
		{
			name: "multiple tags",
			defaultTags: map[string]string{
				"Environment": "production",
				"Team":        "platform",
			},
			deployment: &lib.DeployInfo{},
			wantCount:  2,
		},
		{
			name:        "no default tags",
			defaultTags: map[string]string{},
			deployment:  &lib.DeployInfo{},
			wantCount:   0,
		},
		{
			name: "tag with placeholder",
			defaultTags: map[string]string{
				"TemplatePath": "$TEMPLATEPATH",
			},
			deployment: &lib.DeployInfo{
				TemplateLocalPath: "templates/vpc.yaml",
			},
			wantCount: 1,
			wantKey:   "TemplatePath",
			wantValue: "templates/vpc.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set("tags.default", tt.defaultTags)
			got := loadDefaultTags(tt.deployment)

			if len(got) != tt.wantCount {
				t.Errorf("loadDefaultTags() returned %d tags, want %d", len(got), tt.wantCount)
			}

			if tt.wantKey != "" {
				found := false
				for _, tag := range got {
					if *tag.Key == tt.wantKey && *tag.Value == tt.wantValue {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("loadDefaultTags() did not return tag %s=%s", tt.wantKey, tt.wantValue)
				}
			}
		})
	}
}

// TestLoadDeploymentFileTags tests loading tags from deployment file
func TestLoadDeploymentFileTags(t *testing.T) {
	tests := []struct {
		name       string
		deployment *lib.DeployInfo
		wantCount  int
		wantKey    string
		wantValue  string
	}{
		{
			name: "deployment file with tags",
			deployment: &lib.DeployInfo{
				StackDeploymentFile: &lib.StackDeploymentFile{
					Tags: map[string]string{
						"Project": "my-project",
						"Owner":   "team-a",
					},
				},
			},
			wantCount: 2,
		},
		{
			name: "deployment file with placeholder",
			deployment: &lib.DeployInfo{
				StackDeploymentFile: &lib.StackDeploymentFile{
					Tags: map[string]string{
						"Template": "$TEMPLATEPATH",
					},
				},
				TemplateLocalPath: "infra/main.yaml",
			},
			wantCount: 1,
			wantKey:   "Template",
			wantValue: "infra/main.yaml",
		},
		{
			name: "deployment file with no tags",
			deployment: &lib.DeployInfo{
				StackDeploymentFile: &lib.StackDeploymentFile{
					Tags: map[string]string{},
				},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := loadDeploymentFileTags(tt.deployment)

			if len(got) != tt.wantCount {
				t.Errorf("loadDeploymentFileTags() returned %d tags, want %d", len(got), tt.wantCount)
			}

			if tt.wantKey != "" {
				found := false
				for _, tag := range got {
					if *tag.Key == tt.wantKey && *tag.Value == tt.wantValue {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("loadDeploymentFileTags() did not return tag %s=%s", tt.wantKey, tt.wantValue)
				}
			}
		})
	}
}

// TestLoadDeploymentFileParameters tests loading parameters from deployment file
func TestLoadDeploymentFileParameters(t *testing.T) {
	tests := []struct {
		name       string
		deployment *lib.DeployInfo
		wantCount  int
		wantKey    string
		wantValue  string
	}{
		{
			name: "deployment file with parameters",
			deployment: &lib.DeployInfo{
				StackDeploymentFile: &lib.StackDeploymentFile{
					Parameters: map[string]string{
						"VpcCidr":    "10.0.0.0/16",
						"SubnetCidr": "10.0.1.0/24",
					},
				},
			},
			wantCount: 2,
		},
		{
			name: "single parameter",
			deployment: &lib.DeployInfo{
				StackDeploymentFile: &lib.StackDeploymentFile{
					Parameters: map[string]string{
						"Environment": "production",
					},
				},
			},
			wantCount: 1,
			wantKey:   "Environment",
			wantValue: "production",
		},
		{
			name: "deployment file with no parameters",
			deployment: &lib.DeployInfo{
				StackDeploymentFile: &lib.StackDeploymentFile{
					Parameters: map[string]string{},
				},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := loadDeploymentFileParameters(tt.deployment)

			if len(got) != tt.wantCount {
				t.Errorf("loadDeploymentFileParameters() returned %d parameters, want %d", len(got), tt.wantCount)
			}

			if tt.wantKey != "" {
				found := false
				for _, param := range got {
					if *param.ParameterKey == tt.wantKey && *param.ParameterValue == tt.wantValue {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("loadDeploymentFileParameters() did not return parameter %s=%s", tt.wantKey, tt.wantValue)
				}
			}
		})
	}
}

// TestCalculateTemplateLocalPath tests relative path calculation
func TestCalculateTemplateLocalPath(t *testing.T) {
	// Save original values
	originalCfgFile := cfgFile
	originalRootdir := viper.GetString("rootdir")
	defer func() {
		cfgFile = originalCfgFile
		viper.Set("rootdir", originalRootdir)
	}()

	// Create a temp directory for testing
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		path        string
		cfgFile     string
		rootdir     string
		wantContain string
	}{
		{
			name:        "simple relative path",
			path:        filepath.Join(tmpDir, "templates", "vpc.yaml"),
			cfgFile:     "",
			rootdir:     tmpDir,
			wantContain: "vpc.yaml",
		},
		{
			name:        "absolute path with config file",
			path:        filepath.Join(tmpDir, "infra", "main.yaml"),
			cfgFile:     filepath.Join(tmpDir, "config.yaml"),
			rootdir:     ".",
			wantContain: "main.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgFile = tt.cfgFile
			viper.Set("rootdir", tt.rootdir)

			got := calculateTemplateLocalPath(tt.path)

			// Should not be empty
			if got == "" {
				t.Error("calculateTemplateLocalPath() returned empty string")
			}

			// Should contain the filename
			if tt.wantContain != "" && !containsSubstring(got, filepath.Base(tt.wantContain)) {
				t.Errorf("calculateTemplateLocalPath() = %v, want to contain %v", got, tt.wantContain)
			}
		})
	}
}

// TestPlaceholderParser tests placeholder replacement
func TestPlaceholderParser(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		deployment  *lib.DeployInfo
		wantContain string
	}{
		{
			name:  "TEMPLATEPATH placeholder",
			value: "Template: $TEMPLATEPATH",
			deployment: &lib.DeployInfo{
				TemplateLocalPath: "templates/vpc.yaml",
			},
			wantContain: "templates/vpc.yaml",
		},
		{
			name:        "TIMESTAMP placeholder",
			value:       "Deployed: $TIMESTAMP",
			deployment:  &lib.DeployInfo{},
			wantContain: "Deployed:",
		},
		{
			name:        "no placeholders",
			value:       "static value",
			deployment:  &lib.DeployInfo{},
			wantContain: "static value",
		},
		{
			name:        "nil deployment",
			value:       "$TEMPLATEPATH",
			deployment:  nil,
			wantContain: "$TEMPLATEPATH",
		},
		{
			name:  "multiple placeholders",
			value: "$TEMPLATEPATH-$TIMESTAMP",
			deployment: &lib.DeployInfo{
				TemplateLocalPath: "test.yaml",
			},
			wantContain: "test.yaml-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := placeholderParser(tt.value, tt.deployment)

			if !containsSubstring(got, tt.wantContain) {
				t.Errorf("placeholderParser() = %v, want to contain %v", got, tt.wantContain)
			}
		})
	}
}

// Helper function for substring checking
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && hasSubstringHelper(s, substr)
}

func hasSubstringHelper(s, substr string) bool {
	if substr == "" {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
