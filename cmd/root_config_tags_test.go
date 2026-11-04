package cmd

import (
	"maps"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// TestLoadConfigFile_DefaultTagsEnvOverride verifies how TAGS_DEFAULT
// interacts with configured tags.default when read by loadDefaultTags, the
// function deploy uses to apply default tags. A JSON object replaces the
// configured map; any other value, including an object with non-string
// values, must fail config loading instead of silently producing an empty map
// or empty tag values.
func TestLoadConfigFile_DefaultTagsEnvOverride(t *testing.T) {
	configContent := "tags:\n  default:\n    Environment: production\n    ManagedBy: fog\n"

	tests := map[string]struct {
		env         *string
		wantTags    map[string]string
		wantErr     bool
		wantErrText string
	}{
		"unset env keeps configured tags": {
			// Viper lowercases map keys read from config files; this is
			// pre-existing behaviour and unrelated to the env override.
			wantTags: map[string]string{"environment": "production", "managedby": "fog"},
		},
		"JSON object replaces configured tags": {
			env:      new(`{"Environment":"staging","Owner":"platform team"}`),
			wantTags: map[string]string{"Environment": "staging", "Owner": "platform team"},
		},
		"numeric-looking string values are kept": {
			env:      new(`{"CostCenter":"123","Enabled":"true"}`),
			wantTags: map[string]string{"CostCenter": "123", "Enabled": "true"},
		},
		"number value is rejected": {
			env:         new(`{"Environment":"staging","CostCenter":123}`),
			wantErr:     true,
			wantErrText: `tag "CostCenter"`,
		},
		"boolean value is rejected": {
			env:         new(`{"Environment":"staging","Enabled":true}`),
			wantErr:     true,
			wantErrText: `tag "Enabled"`,
		},
		"null value is rejected": {
			env:         new(`{"Environment":"staging","Owner":null}`),
			wantErr:     true,
			wantErrText: `tag "Owner"`,
		},
		"key=value form is rejected": {
			env:     new("Environment=staging"),
			wantErr: true,
		},
		"JSON array is rejected": {
			env:     new(`["Environment"]`),
			wantErr: true,
		},
		"JSON null is rejected": {
			env:     new("null"),
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			withResetViper(t)
			path := writeConfig(t, configContent)
			t.Setenv("TAGS_DEFAULT", "")
			if tc.env != nil {
				t.Setenv("TAGS_DEFAULT", *tc.env)
			} else if err := os.Unsetenv("TAGS_DEFAULT"); err != nil {
				t.Fatalf("failed to unset TAGS_DEFAULT: %v", err)
			}

			err := loadConfigFile(path)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got nil; tags would be %v", tagsToMap(loadDefaultTags(nil)))
				}
				if !strings.Contains(err.Error(), "TAGS_DEFAULT") {
					t.Errorf("expected error to mention TAGS_DEFAULT, got %q", err)
				}
				if !strings.Contains(err.Error(), tc.wantErrText) {
					t.Errorf("expected error to contain %q, got %q", tc.wantErrText, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if got := tagsToMap(loadDefaultTags(nil)); !maps.Equal(got, tc.wantTags) {
				t.Errorf("default tags: got %v, want %v", got, tc.wantTags)
			}
		})
	}
}

// tagsToMap converts CloudFormation tags to a key/value map for comparison.
func tagsToMap(tags []types.Tag) map[string]string {
	result := make(map[string]string, len(tags))
	for _, tag := range tags {
		result[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return result
}
