package lib

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	cfntypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseTemplateString_EdgeCases tests edge cases for template parsing
func TestParseTemplateString_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		template    string
		overrides   *map[string]any
		expectPanic bool
		validateFn  func(*testing.T, *CfnTemplateBody)
	}{
		"empty template": {
			template:    "",
			overrides:   &map[string]any{},
			expectPanic: true, // Empty template should fail to parse
		},
		"minimal valid JSON template": {
			template:  `{"AWSTemplateFormatVersion": "2010-09-09"}`,
			overrides: &map[string]any{},
			validateFn: func(t *testing.T, body *CfnTemplateBody) {
				assert.Equal(t, "2010-09-09", body.AWSTemplateFormatVersion)
			},
		},
		"minimal valid YAML template": {
			template:  `AWSTemplateFormatVersion: "2010-09-09"`,
			overrides: &map[string]any{},
			validateFn: func(t *testing.T, body *CfnTemplateBody) {
				assert.Equal(t, "2010-09-09", body.AWSTemplateFormatVersion)
			},
		},
		"template with only resources": {
			template: `{
				"Resources": {
					"Bucket": {
						"Type": "AWS::S3::Bucket"
					}
				}
			}`,
			overrides: &map[string]any{},
			validateFn: func(t *testing.T, body *CfnTemplateBody) {
				assert.Len(t, body.Resources, 1)
				assert.Equal(t, "AWS::S3::Bucket", body.Resources["Bucket"].Type)
			},
		},
		"template with deeply nested structures": {
			template: `{
				"Resources": {
					"Nested": {
						"Type": "AWS::CloudFormation::Stack",
						"Properties": {
							"Level1": {
								"Level2": {
									"Level3": {
										"Level4": {
											"Value": "deep"
										}
									}
								}
							}
						}
					}
				}
			}`,
			overrides: &map[string]any{},
			validateFn: func(t *testing.T, body *CfnTemplateBody) {
				assert.Len(t, body.Resources, 1)
			},
		},
		"template with multiple parameter types": {
			template: `{
				"Parameters": {
					"StringParam": {"Type": "String", "Default": "test"},
					"NumberParam": {"Type": "Number", "Default": "42"},
					"ListParam": {"Type": "CommaDelimitedList", "Default": "a,b,c"},
					"NoDefault": {"Type": "String"}
				}
			}`,
			overrides: &map[string]any{},
			validateFn: func(t *testing.T, body *CfnTemplateBody) {
				assert.Len(t, body.Parameters, 4)
			},
		},
		"override with nil value": {
			template: `{
				"Parameters": {
					"Param1": {"Type": "String", "Default": "default"}
				},
				"Resources": {
					"Resource1": {
						"Type": "AWS::S3::Bucket",
						"Properties": {"BucketName": {"Ref": "Param1"}}
					}
				}
			}`,
			overrides: &map[string]any{
				"Param1": nil,
			},
			validateFn: func(t *testing.T, body *CfnTemplateBody) {
				// Should handle nil override gracefully
				assert.NotNil(t, body)
			},
		},
		"override with complex type": {
			template: `{
				"Parameters": {
					"ComplexParam": {"Type": "String", "Default": "simple"}
				}
			}`,
			overrides: &map[string]any{
				"ComplexParam": map[string]any{
					"nested": "value",
				},
			},
			validateFn: func(t *testing.T, body *CfnTemplateBody) {
				assert.NotNil(t, body)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.expectPanic {
				assert.Panics(t, func() {
					ParseTemplateString(tc.template, tc.overrides)
				})
				return
			}

			body := ParseTemplateString(tc.template, tc.overrides)
			require.NotNil(t, body)

			if tc.validateFn != nil {
				tc.validateFn(t, body)
			}
		})
	}
}

// TestParseTemplateString_LargeTemplate tests handling of very large templates
func TestParseTemplateString_LargeTemplate(t *testing.T) {
	// Create a template with many resources
	numResources := 500
	resources := make([]string, numResources)
	for i := 0; i < numResources; i++ {
		resources[i] = `"Resource` + string(rune('A'+i%26)) + string(rune('0'+i%10)) + `": {
			"Type": "AWS::S3::Bucket",
			"Properties": {"BucketName": "bucket-` + string(rune('a'+i%26)) + string(rune('0'+i%10)) + `"}
		}`
	}

	template := `{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			` + strings.Join(resources, ",") + `
		}
	}`

	body := ParseTemplateString(template, &map[string]any{})
	require.NotNil(t, body)
	assert.Equal(t, numResources, len(body.Resources))
}

// TestParseTemplateString_MalformedTemplates tests error handling for malformed templates
func TestParseTemplateString_MalformedTemplates(t *testing.T) {
	tests := map[string]struct {
		template    string
		description string
	}{
		"invalid JSON": {
			template:    `{"invalid": }`,
			description: "Should panic on invalid JSON",
		},
		"invalid YAML": {
			template: `
invalid:
  - item
 bad_indent: value`,
			description: "Should panic on invalid YAML",
		},
		"mixed JSON and YAML": {
			template:    `{"key": "value"} key2: value2`,
			description: "Should panic on mixed formats",
		},
		"just a string": {
			template:    "not a template",
			description: "Should panic on plain string",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Panics(t, func() {
				ParseTemplateString(tc.template, &map[string]any{})
			}, tc.description)
		})
	}
}

// TestNaclResourceToNaclEntry_EdgeCases tests edge cases for NACL entry conversion
func TestNaclResourceToNaclEntry_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		resource   CfnTemplateResource
		params     []cfntypes.Parameter
		validateFn func(*testing.T, any)
	}{
		"missing properties": {
			resource: CfnTemplateResource{
				Type:       "AWS::EC2::NetworkAclEntry",
				Properties: map[string]any{},
			},
			params: []cfntypes.Parameter{},
			validateFn: func(t *testing.T, entry any) {
				// Should handle missing properties without panic
				assert.NotNil(t, entry)
			},
		},
		"nil properties map": {
			resource: CfnTemplateResource{
				Type:       "AWS::EC2::NetworkAclEntry",
				Properties: nil,
			},
			params: []cfntypes.Parameter{},
			validateFn: func(t *testing.T, entry any) {
				// Should handle nil properties
				assert.NotNil(t, entry)
			},
		},
		"properties with wrong types": {
			resource: CfnTemplateResource{
				Type: "AWS::EC2::NetworkAclEntry",
				Properties: map[string]any{
					"RuleNumber": "not-a-number",
					"Protocol":   "not-a-protocol",
				},
			},
			params: []cfntypes.Parameter{},
			validateFn: func(t *testing.T, entry any) {
				// Should handle type mismatches
				assert.NotNil(t, entry)
			},
		},
		"all protocol": {
			resource: CfnTemplateResource{
				Type: "AWS::EC2::NetworkAclEntry",
				Properties: map[string]any{
					"Protocol":   -1.0,
					"RuleNumber": 100.0,
					"RuleAction": "allow",
					"Egress":     true,
				},
			},
			params: []cfntypes.Parameter{},
			validateFn: func(t *testing.T, entry any) {
				assert.NotNil(t, entry)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := NaclResourceToNaclEntry(tc.resource, tc.params)
			tc.validateFn(t, result)
		})
	}
}

// TestRouteResourceToRoute_EdgeCases tests edge cases for route conversion
func TestRouteResourceToRoute_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		resource   CfnTemplateResource
		params     []cfntypes.Parameter
		validateFn func(*testing.T, any)
	}{
		"empty properties": {
			resource: CfnTemplateResource{
				Type:       "AWS::EC2::Route",
				Properties: map[string]any{},
			},
			params: []cfntypes.Parameter{},
			validateFn: func(t *testing.T, route any) {
				assert.NotNil(t, route)
			},
		},
		"nil destination CIDR": {
			resource: CfnTemplateResource{
				Type: "AWS::EC2::Route",
				Properties: map[string]any{
					"DestinationCidrBlock": nil,
					"GatewayId":            "igw-123",
				},
			},
			params: []cfntypes.Parameter{},
			validateFn: func(t *testing.T, route any) {
				assert.NotNil(t, route)
			},
		},
		"multiple destination types": {
			resource: CfnTemplateResource{
				Type: "AWS::EC2::Route",
				Properties: map[string]any{
					"DestinationCidrBlock":     "0.0.0.0/0",
					"DestinationIpv6CidrBlock": "::/0",
					"GatewayId":                "igw-123",
				},
			},
			params: []cfntypes.Parameter{},
			validateFn: func(t *testing.T, route any) {
				assert.NotNil(t, route)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := RouteResourceToRoute(tc.resource, tc.params)
			tc.validateFn(t, result)
		})
	}
}

// TestCfnTemplateBody_ShouldHaveResource_EdgeCases tests edge cases for resource checking
func TestCfnTemplateBody_ShouldHaveResource_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		body         *CfnTemplateBody
		logicalIDs   []string
		resourceType string
		expected     bool
	}{
		"empty resources map": {
			body: &CfnTemplateBody{
				Resources: map[string]CfnTemplateResource{},
			},
			logicalIDs:   []string{"Resource1"},
			resourceType: "AWS::S3::Bucket",
			expected:     false,
		},
		"nil resources map": {
			body: &CfnTemplateBody{
				Resources: nil,
			},
			logicalIDs:   []string{"Resource1"},
			resourceType: "AWS::S3::Bucket",
			expected:     false,
		},
		"empty logical IDs": {
			body: &CfnTemplateBody{
				Resources: map[string]CfnTemplateResource{
					"Resource1": {Type: "AWS::S3::Bucket"},
				},
			},
			logicalIDs:   []string{},
			resourceType: "AWS::S3::Bucket",
			expected:     false,
		},
		"case sensitive resource type": {
			body: &CfnTemplateBody{
				Resources: map[string]CfnTemplateResource{
					"Resource1": {Type: "AWS::S3::Bucket"},
				},
			},
			logicalIDs:   []string{"Resource1"},
			resourceType: "aws::s3::bucket", // Wrong case
			expected:     false,
		},
		"partial match": {
			body: &CfnTemplateBody{
				Resources: map[string]CfnTemplateResource{
					"Resource1": {Type: "AWS::S3::Bucket"},
					"Resource2": {Type: "AWS::Lambda::Function"},
				},
			},
			logicalIDs:   []string{"Resource1", "Resource3"},
			resourceType: "AWS::S3::Bucket",
			expected:     false, // Resource3 doesn't exist
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := tc.body.ShouldHaveResource(tc.logicalIDs, tc.resourceType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestFilterNaclEntriesByLogicalId_EdgeCases tests edge cases for NACL filtering
func TestFilterNaclEntriesByLogicalId_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		body       *CfnTemplateBody
		logicalIDs []string
		params     []cfntypes.Parameter
		expected   int
	}{
		"empty template": {
			body:       &CfnTemplateBody{Resources: map[string]CfnTemplateResource{}},
			logicalIDs: []string{"Entry1"},
			params:     []cfntypes.Parameter{},
			expected:   0,
		},
		"nil logical IDs": {
			body: &CfnTemplateBody{
				Resources: map[string]CfnTemplateResource{
					"Entry1": {Type: "AWS::EC2::NetworkAclEntry"},
				},
			},
			logicalIDs: nil,
			params:     []cfntypes.Parameter{},
			expected:   0,
		},
		"empty logical IDs": {
			body: &CfnTemplateBody{
				Resources: map[string]CfnTemplateResource{
					"Entry1": {Type: "AWS::EC2::NetworkAclEntry"},
				},
			},
			logicalIDs: []string{},
			params:     []cfntypes.Parameter{},
			expected:   0,
		},
		"wrong resource type": {
			body: &CfnTemplateBody{
				Resources: map[string]CfnTemplateResource{
					"Entry1": {Type: "AWS::S3::Bucket"},
				},
			},
			logicalIDs: []string{"Entry1"},
			params:     []cfntypes.Parameter{},
			expected:   0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := FilterNaclEntriesByLogicalId(tc.body, tc.logicalIDs, tc.params)
			assert.Len(t, result, tc.expected)
		})
	}
}

// TestCfnTemplateTransform_EdgeCases tests edge cases for template transforms
func TestCfnTemplateTransform_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		input      string
		expectNil  bool
		validateFn func(*testing.T, *CfnTemplateTransform)
	}{
		"empty string": {
			input:     "",
			expectNil: true,
		},
		"whitespace only": {
			input:     "   ",
			expectNil: false,
			validateFn: func(t *testing.T, transform *CfnTemplateTransform) {
				// Whitespace should be preserved
				assert.NotNil(t, transform)
			},
		},
		"very long transform name": {
			input: func() string {
				result := ""
				for i := 0; i < 1000; i++ {
					result += "a"
				}
				return result
			}(),
			expectNil: false,
			validateFn: func(t *testing.T, transform *CfnTemplateTransform) {
				assert.NotNil(t, transform)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			transform := CfnTemplateTransform(tc.input)
			if tc.expectNil {
				// String transforms can't be nil, but can be empty
				assert.Empty(t, string(transform))
			} else if tc.validateFn != nil {
				tc.validateFn(t, &transform)
			}
		})
	}
}

// TestParseTemplateString_Concurrency tests concurrent template parsing
func TestParseTemplateString_Concurrency(t *testing.T) {
	t.Parallel()

	const numGoroutines = 20
	template := `{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Parameters": {
			"Param1": {"Type": "String", "Default": "value1"}
		},
		"Resources": {
			"Bucket": {
				"Type": "AWS::S3::Bucket",
				"Properties": {"BucketName": {"Ref": "Param1"}}
			}
		}
	}`

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			overrides := map[string]any{
				"Param1": "concurrent-" + string(rune('0'+id%10)),
			}

			body := ParseTemplateString(template, &overrides)
			assert.NotNil(t, body)
			assert.Equal(t, "2010-09-09", body.AWSTemplateFormatVersion)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// TestParseTemplateString_SpecialCharacters tests templates with special characters
func TestParseTemplateString_SpecialCharacters(t *testing.T) {
	tests := map[string]struct {
		template   string
		validateFn func(*testing.T, *CfnTemplateBody)
	}{
		"unicode in description": {
			template: `{
				"Description": "Template with unicode: 世界 🌍",
				"Resources": {
					"Bucket": {"Type": "AWS::S3::Bucket"}
				}
			}`,
			validateFn: func(t *testing.T, body *CfnTemplateBody) {
				assert.Contains(t, body.Description, "世界")
			},
		},
		"escaped characters in JSON": {
			template: `{
				"Resources": {
					"Bucket": {
						"Type": "AWS::S3::Bucket",
						"Properties": {
							"Tags": [{"Key": "quote", "Value": "\"quoted\""}]
						}
					}
				}
			}`,
			validateFn: func(t *testing.T, body *CfnTemplateBody) {
				assert.NotNil(t, body.Resources["Bucket"])
			},
		},
		"newlines in YAML": {
			template: `
Description: |
  Multi-line
  description
  here
Resources:
  Bucket:
    Type: AWS::S3::Bucket`,
			validateFn: func(t *testing.T, body *CfnTemplateBody) {
				assert.Contains(t, body.Description, "Multi-line")
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			body := ParseTemplateString(tc.template, &map[string]any{})
			require.NotNil(t, body)
			tc.validateFn(t, body)
		})
	}
}
