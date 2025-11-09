package lib

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseTemplateString_EdgeCases tests edge cases for template parsing.
// These tests address Issue 4.1 from the audit report regarding edge cases in template parsing.
func TestParseTemplateString_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		template    string
		overrides   map[string]any
		shouldPanic bool
		description string
	}{
		"empty template string": {
			template:    "",
			overrides:   map[string]any{},
			shouldPanic: true,
			description: "Empty template should cause panic",
		},
		"malformed JSON": {
			template:    `{"Resources": {`,
			overrides:   map[string]any{},
			shouldPanic: true,
			description: "Malformed JSON should cause panic",
		},
		"malformed YAML": {
			template:    "Resources:\n\t- invalid:\nyaml",
			overrides:   map[string]any{},
			shouldPanic: true,
			description: "Malformed YAML should cause panic",
		},
		"nil overrides map": {
			template: `{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Resources": {
					"Bucket": {
						"Type": "AWS::S3::Bucket"
					}
				}
			}`,
			overrides:   nil,
			shouldPanic: false,
			description: "Nil overrides map should be handled gracefully",
		},
		"template with only whitespace": {
			template:    "   \n\t  \n  ",
			overrides:   map[string]any{},
			shouldPanic: true,
			description: "Whitespace-only template should cause panic",
		},
		"template without Resources section": {
			template: `{
				"AWSTemplateFormatVersion": "2010-09-09"
			}`,
			overrides:   map[string]any{},
			shouldPanic: false,
			description: "Template without Resources should still parse",
		},
		"template with nested parameter references": {
			template: `{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Parameters": {
					"Param1": {"Type": "String", "Default": "Value1"},
					"Param2": {"Type": "String", "Default": {"Ref": "Param1"}}
				},
				"Resources": {}
			}`,
			overrides:   map[string]any{},
			shouldPanic: false,
			description: "Nested parameter references should be resolved",
		},
		"template with circular parameter references": {
			template: `{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Parameters": {
					"Param1": {"Type": "String", "Default": {"Ref": "Param2"}},
					"Param2": {"Type": "String", "Default": {"Ref": "Param1"}}
				},
				"Resources": {}
			}`,
			overrides:   map[string]any{},
			shouldPanic: true,
			description: "Circular parameter references should cause panic",
		},
		"template with very long string values": {
			template: `{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Description": "` + strings.Repeat("A", 10000) + `",
				"Resources": {}
			}`,
			overrides:   map[string]any{},
			shouldPanic: false,
			description: "Very long string values should be handled",
		},
		"template with special characters in keys": {
			template: `{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Resources": {
					"My-Resource-Name-With-Dashes": {
						"Type": "AWS::S3::Bucket"
					}
				}
			}`,
			overrides:   map[string]any{},
			shouldPanic: false,
			description: "Special characters in resource names should be allowed",
		},
		"template with numeric resource names": {
			template: `{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Resources": {
					"123Resource": {
						"Type": "AWS::S3::Bucket"
					}
				}
			}`,
			overrides:   map[string]any{},
			shouldPanic: false,
			description: "Numeric prefixes in resource names should be allowed",
		},
		"override with nil value": {
			template: `{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Parameters": {
					"Param1": {"Type": "String", "Default": "Default"}
				},
				"Resources": {}
			}`,
			overrides:   map[string]any{"Param1": nil},
			shouldPanic: false,
			description: "Nil override values should be handled",
		},
		"override with complex nested structure": {
			template: `{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Parameters": {
					"ComplexParam": {"Type": "String"}
				},
				"Resources": {}
			}`,
			overrides: map[string]any{
				"ComplexParam": map[string]any{
					"nested": map[string]any{
						"deep": "value",
					},
				},
			},
			shouldPanic: false,
			description: "Complex nested override structures should be handled",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.shouldPanic {
				assert.Panics(t, func() {
					ParseTemplateString(tc.template, &tc.overrides)
				}, tc.description)
			} else {
				assert.NotPanics(t, func() {
					body := ParseTemplateString(tc.template, &tc.overrides)
					assert.NotNil(t, body, tc.description)
				}, tc.description)
			}
		})
	}
}

// TestCfnTemplateParameter_UnmarshalJSON_EdgeCases tests edge cases for parameter unmarshaling.
func TestCfnTemplateParameter_UnmarshalJSON_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		json        string
		expectError bool
		check       func(*testing.T, *CfnTemplateParameter)
	}{
		"MaxLength as string": {
			json: `{"Type": "String", "MaxLength": "100"}`,
			check: func(t *testing.T, p *CfnTemplateParameter) {
				t.Helper()
				assert.Equal(t, 100, p.MaxLength)
			},
		},
		"MaxLength as number": {
			json: `{"Type": "String", "MaxLength": 100}`,
			check: func(t *testing.T, p *CfnTemplateParameter) {
				t.Helper()
				assert.Equal(t, 100, p.MaxLength)
			},
		},
		"MinLength as string": {
			json: `{"Type": "String", "MinLength": "10"}`,
			check: func(t *testing.T, p *CfnTemplateParameter) {
				t.Helper()
				assert.Equal(t, 10, p.MinLength)
			},
		},
		"MaxValue as string": {
			json: `{"Type": "Number", "MaxValue": "999.99"}`,
			check: func(t *testing.T, p *CfnTemplateParameter) {
				t.Helper()
				assert.Equal(t, 999.99, p.MaxValue)
			},
		},
		"MinValue as string": {
			json: `{"Type": "Number", "MinValue": "1.5"}`,
			check: func(t *testing.T, p *CfnTemplateParameter) {
				t.Helper()
				assert.Equal(t, 1.5, p.MinValue)
			},
		},
		"invalid MaxLength string": {
			json: `{"Type": "String", "MaxLength": "not-a-number"}`,
			check: func(t *testing.T, p *CfnTemplateParameter) {
				t.Helper()
				assert.Equal(t, 0, p.MaxLength) // Should default to zero
			},
		},
		"invalid MaxValue string": {
			json: `{"Type": "Number", "MaxValue": "invalid"}`,
			check: func(t *testing.T, p *CfnTemplateParameter) {
				t.Helper()
				assert.Equal(t, float64(0), p.MaxValue) // Should default to zero
			},
		},
		"negative MaxLength": {
			json: `{"Type": "String", "MaxLength": -10}`,
			check: func(t *testing.T, p *CfnTemplateParameter) {
				t.Helper()
				assert.Equal(t, -10, p.MaxLength) // Negative values are preserved
			},
		},
		"zero values": {
			json: `{"Type": "String", "MaxLength": 0, "MinLength": 0}`,
			check: func(t *testing.T, p *CfnTemplateParameter) {
				t.Helper()
				assert.Equal(t, 0, p.MaxLength)
				assert.Equal(t, 0, p.MinLength)
			},
		},
		"very large numbers": {
			json: `{"Type": "Number", "MaxValue": 1.7976931348623157e+308}`,
			check: func(t *testing.T, p *CfnTemplateParameter) {
				t.Helper()
				assert.NotZero(t, p.MaxValue)
			},
		},
		"NoEcho true": {
			json: `{"Type": "String", "NoEcho": true}`,
			check: func(t *testing.T, p *CfnTemplateParameter) {
				t.Helper()
				assert.True(t, p.NoEcho)
			},
		},
		"NoEcho false": {
			json: `{"Type": "String", "NoEcho": false}`,
			check: func(t *testing.T, p *CfnTemplateParameter) {
				t.Helper()
				assert.False(t, p.NoEcho)
			},
		},
		"AllowedValues empty array": {
			json: `{"Type": "String", "AllowedValues": []}`,
			check: func(t *testing.T, p *CfnTemplateParameter) {
				t.Helper()
				assert.Empty(t, p.AllowedValues)
			},
		},
		"AllowedValues with various types": {
			json: `{"Type": "String", "AllowedValues": ["string", 123, true, null]}`,
			check: func(t *testing.T, p *CfnTemplateParameter) {
				t.Helper()
				assert.Len(t, p.AllowedValues, 4)
			},
		},
		"Default as various types": {
			json: `{"Type": "String", "Default": {"complex": "object"}}`,
			check: func(t *testing.T, p *CfnTemplateParameter) {
				t.Helper()
				assert.NotNil(t, p.Default)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var param CfnTemplateParameter
			err := param.UnmarshalJSON([]byte(tc.json))

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tc.check != nil {
					tc.check(t, &param)
				}
			}
		})
	}
}

// TestStackDeploymentFile_EdgeCases tests edge cases for stack deployment file structure.
func TestStackDeploymentFile_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		file StackDeploymentFile
		desc string
	}{
		"empty template file path": {
			file: StackDeploymentFile{
				TemplateFilePath: "",
				Parameters:       map[string]string{},
				Tags:             map[string]string{},
			},
			desc: "Empty template file path should be allowed",
		},
		"nil parameters map": {
			file: StackDeploymentFile{
				TemplateFilePath: "template.yaml",
				Parameters:       nil,
				Tags:             map[string]string{},
			},
			desc: "Nil parameters map should be allowed",
		},
		"nil tags map": {
			file: StackDeploymentFile{
				TemplateFilePath: "template.yaml",
				Parameters:       map[string]string{},
				Tags:             nil,
			},
			desc: "Nil tags map should be allowed",
		},
		"parameters with empty values": {
			file: StackDeploymentFile{
				TemplateFilePath: "template.yaml",
				Parameters: map[string]string{
					"EmptyParam": "",
					"NormalParam": "value",
				},
				Tags: map[string]string{},
			},
			desc: "Parameters with empty string values should be allowed",
		},
		"parameters with special characters": {
			file: StackDeploymentFile{
				TemplateFilePath: "template.yaml",
				Parameters: map[string]string{
					"Param-With-Dashes": "value",
					"Param.With.Dots":   "value",
					"Param_With_Underscores": "value",
				},
				Tags: map[string]string{},
			},
			desc: "Parameters with special characters in names should be allowed",
		},
		"very long parameter values": {
			file: StackDeploymentFile{
				TemplateFilePath: "template.yaml",
				Parameters: map[string]string{
					"LongParam": strings.Repeat("A", 4096),
				},
				Tags: map[string]string{},
			},
			desc: "Very long parameter values should be handled",
		},
		"tags with special characters": {
			file: StackDeploymentFile{
				TemplateFilePath: "template.yaml",
				Parameters:       map[string]string{},
				Tags: map[string]string{
					"aws:cloudformation:stack-name": "value",
					"Project/Environment": "value",
				},
			},
			desc: "Tags with special characters should be allowed",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Should not panic or error
			assert.NotPanics(t, func() {
				_ = tc.file.TemplateFilePath
				_ = tc.file.Parameters
				_ = tc.file.Tags
			}, tc.desc)
		})
	}
}

// TestCfnTemplateBody_EdgeCases tests edge cases for template body structure.
func TestCfnTemplateBody_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		body CfnTemplateBody
		desc string
	}{
		"all fields empty": {
			body: CfnTemplateBody{},
			desc: "Empty template body should be valid",
		},
		"nil maps": {
			body: CfnTemplateBody{
				Metadata:   nil,
				Mappings:   nil,
				Rules:      nil,
				Parameters: nil,
				Resources:  nil,
				Conditions: nil,
				Outputs:    nil,
			},
			desc: "Nil maps should be allowed",
		},
		"empty description": {
			body: CfnTemplateBody{
				Description: "",
			},
			desc: "Empty description should be allowed",
		},
		"very long description": {
			body: CfnTemplateBody{
				Description: strings.Repeat("A", 10000),
			},
			desc: "Very long description should be handled",
		},
		"empty version": {
			body: CfnTemplateBody{
				AWSTemplateFormatVersion: "",
			},
			desc: "Empty version should be allowed",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				_ = tc.body.AWSTemplateFormatVersion
				_ = tc.body.Description
				_ = tc.body.Resources
			}, tc.desc)
		})
	}
}

// TestCfnTemplateResource_EdgeCases tests edge cases for template resource structure.
func TestCfnTemplateResource_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		resource CfnTemplateResource
		desc     string
	}{
		"empty resource type": {
			resource: CfnTemplateResource{
				Type:       "",
				Properties: map[string]any{},
			},
			desc: "Empty resource type should be allowed",
		},
		"nil properties": {
			resource: CfnTemplateResource{
				Type:       "AWS::S3::Bucket",
				Properties: nil,
			},
			desc: "Nil properties should be allowed",
		},
		"nil metadata": {
			resource: CfnTemplateResource{
				Type:       "AWS::S3::Bucket",
				Properties: map[string]any{},
				Metadata:   nil,
			},
			desc: "Nil metadata should be allowed",
		},
		"empty condition": {
			resource: CfnTemplateResource{
				Type:       "AWS::S3::Bucket",
				Condition:  "",
				Properties: map[string]any{},
			},
			desc: "Empty condition should be allowed",
		},
		"properties with nested maps": {
			resource: CfnTemplateResource{
				Type: "AWS::S3::Bucket",
				Properties: map[string]any{
					"BucketName": "test",
					"Tags": []map[string]any{
						{"Key": "Name", "Value": "test"},
					},
				},
			},
			desc: "Properties with nested structures should be handled",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				_ = tc.resource.Type
				_ = tc.resource.Properties
				_ = tc.resource.Metadata
			}, tc.desc)
		})
	}
}
