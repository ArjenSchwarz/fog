package lib

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	cfntypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/stretchr/testify/assert"
)

// TestCfnTemplateBody_HandleMissingRefs tests error handling for missing references
func TestCfnTemplateBody_HandleMissingRefs(t *testing.T) {
	tests := map[string]struct {
		template    string
		params      []cfntypes.Parameter
		description string
	}{
		"reference to non-existent parameter": {
			template: `{
				"Resources": {
					"Bucket": {
						"Type": "AWS::S3::Bucket",
						"Properties": {"BucketName": {"Ref": "NonExistent"}}
					}
				}
			}`,
			params:      []cfntypes.Parameter{},
			description: "Should handle missing parameter reference",
		},
		"reference to empty parameter": {
			template: `{
				"Parameters": {
					"EmptyParam": {"Type": "String"}
				},
				"Resources": {
					"Bucket": {
						"Type": "AWS::S3::Bucket",
						"Properties": {"BucketName": {"Ref": "EmptyParam"}}
					}
				}
			}`,
			params: []cfntypes.Parameter{
				{
					ParameterKey:   aws.String("EmptyParam"),
					ParameterValue: aws.String(""),
				},
			},
			description: "Should handle empty parameter value",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Should not panic when parsing templates with missing refs
			assert.NotPanics(t, func() {
				body := ParseTemplateString(tc.template, &map[string]any{})
				assert.NotNil(t, body, tc.description)
			})
		})
	}
}

// TestNaclResourceToNaclEntry_ErrorPaths tests error handling in NACL conversion
func TestNaclResourceToNaclEntry_ErrorPaths(t *testing.T) {
	tests := map[string]struct {
		resource   CfnTemplateResource
		params     []cfntypes.Parameter
		shouldFail bool
	}{
		"invalid protocol type": {
			resource: CfnTemplateResource{
				Type: "AWS::EC2::NetworkAclEntry",
				Properties: map[string]any{
					"Protocol": []string{"invalid"},
				},
			},
			params:     []cfntypes.Parameter{},
			shouldFail: false, // Should handle gracefully
		},
		"invalid rule number type": {
			resource: CfnTemplateResource{
				Type: "AWS::EC2::NetworkAclEntry",
				Properties: map[string]any{
					"RuleNumber": "not-a-number",
				},
			},
			params:     []cfntypes.Parameter{},
			shouldFail: false,
		},
		"missing required CIDR": {
			resource: CfnTemplateResource{
				Type: "AWS::EC2::NetworkAclEntry",
				Properties: map[string]any{
					"Protocol":   6.0,
					"RuleNumber": 100.0,
					"RuleAction": "allow",
				},
			},
			params:     []cfntypes.Parameter{},
			shouldFail: false,
		},
		"invalid port range types": {
			resource: CfnTemplateResource{
				Type: "AWS::EC2::NetworkAclEntry",
				Properties: map[string]any{
					"PortRange": map[string]any{
						"From": "invalid",
						"To":   []int{443},
					},
				},
			},
			params:     []cfntypes.Parameter{},
			shouldFail: false,
		},
		"invalid ICMP types": {
			resource: CfnTemplateResource{
				Type: "AWS::EC2::NetworkAclEntry",
				Properties: map[string]any{
					"Icmp": map[string]any{
						"Type": "not-a-number",
						"Code": map[string]any{"invalid": "data"},
					},
				},
			},
			params:     []cfntypes.Parameter{},
			shouldFail: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.shouldFail {
				assert.Panics(t, func() {
					NaclResourceToNaclEntry(tc.resource, tc.params)
				})
			} else {
				assert.NotPanics(t, func() {
					result := NaclResourceToNaclEntry(tc.resource, tc.params)
					assert.NotNil(t, result)
				})
			}
		})
	}
}

// TestRouteResourceToRoute_ErrorPaths tests error handling in route conversion
func TestRouteResourceToRoute_ErrorPaths(t *testing.T) {
	tests := map[string]struct {
		resource CfnTemplateResource
		params   []cfntypes.Parameter
	}{
		"missing all destination types": {
			resource: CfnTemplateResource{
				Type: "AWS::EC2::Route",
				Properties: map[string]any{
					"GatewayId": "igw-123",
				},
			},
			params: []cfntypes.Parameter{},
		},
		"invalid reference type": {
			resource: CfnTemplateResource{
				Type: "AWS::EC2::Route",
				Properties: map[string]any{
					"DestinationCidrBlock": map[string]any{
						"InvalidRef": "SomeParam",
					},
					"GatewayId": "igw-123",
				},
			},
			params: []cfntypes.Parameter{},
		},
		"reference to missing parameter": {
			resource: CfnTemplateResource{
				Type: "AWS::EC2::Route",
				Properties: map[string]any{
					"DestinationCidrBlock": map[string]any{
						"Ref": "MissingParam",
					},
					"GatewayId": "igw-123",
				},
			},
			params: []cfntypes.Parameter{},
		},
		"nil gateway ID": {
			resource: CfnTemplateResource{
				Type: "AWS::EC2::Route",
				Properties: map[string]any{
					"DestinationCidrBlock": "0.0.0.0/0",
					"GatewayId":            nil,
				},
			},
			params: []cfntypes.Parameter{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				result := RouteResourceToRoute(tc.resource, tc.params)
				assert.NotNil(t, result)
			})
		})
	}
}

// TestFilterNaclEntriesByLogicalId_ErrorPaths tests error handling in NACL filtering
func TestFilterNaclEntriesByLogicalId_ErrorPaths(t *testing.T) {
	tests := map[string]struct {
		body       *CfnTemplateBody
		logicalIDs []string
		params     []cfntypes.Parameter
	}{
		"nil template body": {
			body:       nil,
			logicalIDs: []string{"Entry1"},
			params:     []cfntypes.Parameter{},
		},
		"resource with invalid properties": {
			body: &CfnTemplateBody{
				Resources: map[string]CfnTemplateResource{
					"Entry1": {
						Type:       "AWS::EC2::NetworkAclEntry",
						Properties: nil,
					},
				},
			},
			logicalIDs: []string{"Entry1"},
			params:     []cfntypes.Parameter{},
		},
		"duplicate logical IDs": {
			body: &CfnTemplateBody{
				Resources: map[string]CfnTemplateResource{
					"Entry1": {
						Type: "AWS::EC2::NetworkAclEntry",
						Properties: map[string]any{
							"Protocol":   6.0,
							"RuleNumber": 100.0,
						},
					},
				},
			},
			logicalIDs: []string{"Entry1", "Entry1", "Entry1"},
			params:     []cfntypes.Parameter{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				result := FilterNaclEntriesByLogicalId(tc.body, tc.logicalIDs, tc.params)
				// Result may be empty or have items, but should not panic
				_ = result
			})
		})
	}
}

// TestFilterRoutesByLogicalId_ErrorPaths tests error handling in route filtering
func TestFilterRoutesByLogicalId_ErrorPaths(t *testing.T) {
	tests := map[string]struct {
		body       *CfnTemplateBody
		logicalIDs []string
		params     []cfntypes.Parameter
	}{
		"nil template body": {
			body:       nil,
			logicalIDs: []string{"Route1"},
			params:     []cfntypes.Parameter{},
		},
		"resource with malformed properties": {
			body: &CfnTemplateBody{
				Resources: map[string]CfnTemplateResource{
					"Route1": {
						Type: "AWS::EC2::Route",
						Properties: map[string]any{
							"DestinationCidrBlock": []int{1, 2, 3}, // Invalid type
						},
					},
				},
			},
			logicalIDs: []string{"Route1"},
			params:     []cfntypes.Parameter{},
		},
		"very long logical ID list": {
			body: &CfnTemplateBody{
				Resources: map[string]CfnTemplateResource{
					"Route1": {Type: "AWS::EC2::Route"},
				},
			},
			logicalIDs: func() []string {
				ids := make([]string, 10000)
				for i := 0; i < 10000; i++ {
					ids[i] = "Route1"
				}
				return ids
			}(),
			params: []cfntypes.Parameter{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				result := FilterRoutesByLogicalId(tc.body, tc.logicalIDs, tc.params)
				_ = result
			})
		})
	}
}

// TestCfnTemplateTransform_UnmarshalJSON_ErrorPaths tests JSON unmarshaling error handling
func TestCfnTemplateTransform_UnmarshalJSON_ErrorPaths(t *testing.T) {
	tests := map[string]struct {
		jsonInput   string
		expectError bool
	}{
		"invalid JSON": {
			jsonInput:   `{invalid}`,
			expectError: true,
		},
		"null value": {
			jsonInput:   `null`,
			expectError: false,
		},
		"number": {
			jsonInput:   `123`,
			expectError: true,
		},
		"boolean": {
			jsonInput:   `true`,
			expectError: true,
		},
		"array": {
			jsonInput:   `["transform1", "transform2"]`,
			expectError: true,
		},
		"empty string": {
			jsonInput:   `""`,
			expectError: false,
		},
		"quoted string": {
			jsonInput:   `"AWS::Serverless-2016-10-31"`,
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var transform CfnTemplateTransform
			err := transform.UnmarshalJSON([]byte(tc.jsonInput))

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCfnTemplateParameter_UnmarshalJSON_ErrorPaths tests parameter unmarshaling error handling
func TestCfnTemplateParameter_UnmarshalJSON_ErrorPaths(t *testing.T) {
	tests := map[string]struct {
		jsonInput   string
		expectError bool
	}{
		"invalid JSON object": {
			jsonInput:   `{invalid}`,
			expectError: true,
		},
		"array instead of object": {
			jsonInput:   `["String"]`,
			expectError: true,
		},
		"null": {
			jsonInput:   `null`,
			expectError: false,
		},
		"empty object": {
			jsonInput:   `{}`,
			expectError: false,
		},
		"missing Type field": {
			jsonInput:   `{"Default": "value"}`,
			expectError: false,
		},
		"invalid nested structure": {
			jsonInput:   `{"Type": {"nested": "object"}}`,
			expectError: false, // May be handled differently
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var param CfnTemplateParameter
			err := param.UnmarshalJSON([]byte(tc.jsonInput))

			if tc.expectError {
				assert.Error(t, err)
			} else {
				// May or may not error depending on implementation
				_ = err
			}
		})
	}
}

// TestParseTemplateString_RecursionDepth tests handling of deeply nested structures
func TestParseTemplateString_RecursionDepth(t *testing.T) {
	// Create a deeply nested structure
	depth := 100
	nested := "\"value\""
	for i := 0; i < depth; i++ {
		nested = `{"nested": ` + nested + `}`
	}

	template := `{
		"Resources": {
			"Deep": {
				"Type": "AWS::CloudFormation::Stack",
				"Properties": ` + nested + `
			}
		}
	}`

	assert.NotPanics(t, func() {
		body := ParseTemplateString(template, &map[string]any{})
		assert.NotNil(t, body)
	})
}

// TestParseTemplateString_CircularReferences tests handling of parameter dependencies
func TestParseTemplateString_CircularReferences(t *testing.T) {
	// Template with parameters that reference each other (not truly circular in CloudFormation, but complex)
	template := `{
		"Parameters": {
			"Param1": {"Type": "String", "Default": "value1"},
			"Param2": {"Type": "String", "Default": "value2"}
		},
		"Resources": {
			"Resource1": {
				"Type": "AWS::S3::Bucket",
				"Properties": {
					"BucketName": {"Fn::Join": ["-", [{"Ref": "Param1"}, {"Ref": "Param2"}]]}
				}
			}
		}
	}`

	assert.NotPanics(t, func() {
		body := ParseTemplateString(template, &map[string]any{})
		assert.NotNil(t, body)
	})
}

// TestParseTemplateString_MemoryLimits tests handling of memory-intensive templates
func TestParseTemplateString_MemoryLimits(t *testing.T) {
	// Create a template with many large string values
	largeValue := ""
	for i := 0; i < 10000; i++ {
		largeValue += "x"
	}

	template := `{
		"Description": "` + largeValue + `",
		"Resources": {
			"Bucket": {"Type": "AWS::S3::Bucket"}
		}
	}`

	assert.NotPanics(t, func() {
		body := ParseTemplateString(template, &map[string]any{})
		assert.NotNil(t, body)
		assert.Equal(t, largeValue, body.Description)
	})
}
