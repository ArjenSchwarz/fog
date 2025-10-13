package lib

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cfntypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGetTemplateClient implements CloudFormationGetTemplateAPI
type mockGetTemplateClient struct {
	getTemplateFn func(context.Context, *cloudformation.GetTemplateInput, ...func(*cloudformation.Options)) (*cloudformation.GetTemplateOutput, error)
}

func (m *mockGetTemplateClient) GetTemplate(ctx context.Context, params *cloudformation.GetTemplateInput, optFns ...func(*cloudformation.Options)) (*cloudformation.GetTemplateOutput, error) {
	if m.getTemplateFn != nil {
		return m.getTemplateFn(ctx, params, optFns...)
	}
	return &cloudformation.GetTemplateOutput{}, nil
}

func TestGetTemplateBody(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		stackName  string
		parameters *map[string]any
		setupMock  func() *mockGetTemplateClient
		want       CfnTemplateBody
		wantPanic  bool
	}{
		"retrieves JSON template successfully": {
			stackName:  "test-stack",
			parameters: nil,
			setupMock: func() *mockGetTemplateClient {
				templateBody := `{
					"AWSTemplateFormatVersion": "2010-09-09",
					"Description": "Test template",
					"Resources": {
						"MyBucket": {
							"Type": "AWS::S3::Bucket",
							"Properties": {
								"BucketName": "test-bucket"
							}
						}
					}
				}`
				return &mockGetTemplateClient{
					getTemplateFn: func(ctx context.Context, params *cloudformation.GetTemplateInput, optFns ...func(*cloudformation.Options)) (*cloudformation.GetTemplateOutput, error) {
						return &cloudformation.GetTemplateOutput{
							TemplateBody: &templateBody,
						}, nil
					},
				}
			},
			want: CfnTemplateBody{
				AWSTemplateFormatVersion: "2010-09-09",
				Description:              "Test template",
				Resources: map[string]CfnTemplateResource{
					"MyBucket": {
						Type: "AWS::S3::Bucket",
						Properties: map[string]any{
							"BucketName": "test-bucket",
						},
					},
				},
			},
		},
		"retrieves YAML template successfully": {
			stackName:  "yaml-stack",
			parameters: nil,
			setupMock: func() *mockGetTemplateClient {
				templateBody := `AWSTemplateFormatVersion: "2010-09-09"
Description: YAML test template
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: yaml-test-bucket`
				return &mockGetTemplateClient{
					getTemplateFn: func(ctx context.Context, params *cloudformation.GetTemplateInput, optFns ...func(*cloudformation.Options)) (*cloudformation.GetTemplateOutput, error) {
						return &cloudformation.GetTemplateOutput{
							TemplateBody: &templateBody,
						}, nil
					},
				}
			},
			want: CfnTemplateBody{
				AWSTemplateFormatVersion: "2010-09-09",
				Description:              "YAML test template",
				Resources: map[string]CfnTemplateResource{
					"MyBucket": {
						Type: "AWS::S3::Bucket",
						Properties: map[string]any{
							"BucketName": "yaml-test-bucket",
						},
					},
				},
			},
		},
		"template with parameters": {
			stackName: "param-stack",
			parameters: &map[string]any{
				"BucketNameParam": "overridden-bucket",
			},
			setupMock: func() *mockGetTemplateClient {
				templateBody := `{
					"AWSTemplateFormatVersion": "2010-09-09",
					"Parameters": {
						"BucketNameParam": {
							"Type": "String",
							"Default": "default-bucket"
						}
					},
					"Resources": {
						"MyBucket": {
							"Type": "AWS::S3::Bucket",
							"Properties": {
								"BucketName": {"Ref": "BucketNameParam"}
							}
						}
					}
				}`
				return &mockGetTemplateClient{
					getTemplateFn: func(ctx context.Context, params *cloudformation.GetTemplateInput, optFns ...func(*cloudformation.Options)) (*cloudformation.GetTemplateOutput, error) {
						return &cloudformation.GetTemplateOutput{
							TemplateBody: &templateBody,
						}, nil
					},
				}
			},
			want: CfnTemplateBody{
				AWSTemplateFormatVersion: "2010-09-09",
				Parameters: map[string]CfnTemplateParameter{
					"BucketNameParam": {
						Type:    "String",
						Default: "default-bucket",
					},
				},
				Resources: map[string]CfnTemplateResource{
					"MyBucket": {
						Type: "AWS::S3::Bucket",
						Properties: map[string]any{
							"BucketName": "overridden-bucket",
						},
					},
				},
			},
		},
		"API error triggers panic": {
			stackName:  "error-stack",
			parameters: nil,
			setupMock: func() *mockGetTemplateClient {
				return &mockGetTemplateClient{
					getTemplateFn: func(ctx context.Context, params *cloudformation.GetTemplateInput, optFns ...func(*cloudformation.Options)) (*cloudformation.GetTemplateOutput, error) {
						return nil, errors.New("API error")
					},
				}
			},
			wantPanic: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockClient := tc.setupMock()

			if tc.wantPanic {
				assert.Panics(t, func() {
					GetTemplateBody(&tc.stackName, tc.parameters, mockClient)
				}, "Expected GetTemplateBody to panic on API error")
				return
			}

			got := GetTemplateBody(&tc.stackName, tc.parameters, mockClient)

			// Compare key fields
			assert.Equal(t, tc.want.AWSTemplateFormatVersion, got.AWSTemplateFormatVersion)
			assert.Equal(t, tc.want.Description, got.Description)

			// Verify resource count
			require.Len(t, got.Resources, len(tc.want.Resources))

			// Compare resources
			for key, wantResource := range tc.want.Resources {
				gotResource, ok := got.Resources[key]
				require.True(t, ok, "Expected resource %s not found", key)
				assert.Equal(t, wantResource.Type, gotResource.Type)

				// Compare properties if they exist
				if wantResource.Properties != nil {
					for propKey, wantProp := range wantResource.Properties {
						gotProp, ok := gotResource.Properties[propKey]
						require.True(t, ok, "Expected property %s not found in resource %s", propKey, key)
						assert.Equal(t, wantProp, gotProp)
					}
				}
			}
		})
	}
}

// TestStringPointer tests the stringPointer helper function used in template processing
func TestStringPointer(t *testing.T) {
	t.Helper()

	params := []cfntypes.Parameter{
		{ParameterKey: aws.String("TestParam"), ParameterValue: aws.String("param-value")},
		{ParameterKey: aws.String("ResolvedParam"), ResolvedValue: aws.String("resolved-value")},
	}

	logicalToPhysical := map[string]string{
		"LogicalResource": "physical-resource-id",
	}

	tests := map[string]struct {
		array map[string]any
		value string
		want  *string
	}{
		"string value": {
			array: map[string]any{
				"TestKey": "direct-value",
			},
			value: "TestKey",
			want:  aws.String("direct-value"),
		},
		"ref with logical to physical mapping": {
			array: map[string]any{
				"TestKey": "REF: LogicalResource",
			},
			value: "TestKey",
			want:  aws.String("physical-resource-id"),
		},
		"ref without logical to physical mapping": {
			array: map[string]any{
				"TestKey": "REF: UnknownResource",
			},
			value: "TestKey",
			want:  aws.String("REF: UnknownResource"),
		},
		"map ref with parameter value": {
			array: map[string]any{
				"TestKey": map[string]any{"Ref": "TestParam"},
			},
			value: "TestKey",
			want:  aws.String("param-value"),
		},
		"map ref with resolved value": {
			array: map[string]any{
				"TestKey": map[string]any{"Ref": "ResolvedParam"},
			},
			value: "TestKey",
			want:  aws.String("resolved-value"),
		},
		"missing key": {
			array: map[string]any{
				"OtherKey": "value",
			},
			value: "TestKey",
			want:  nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := stringPointer(tc.array, params, logicalToPhysical, tc.value)

			if tc.want == nil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, *tc.want, *got)
			}
		})
	}
}

// TestCfnTemplateTransform_MarshalJSON tests that CfnTemplateTransform can be marshaled back to JSON
func TestCfnTemplateTransform_MarshalJSON(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		transform CfnTemplateTransform
		wantJSON  string
	}{
		"string transform": {
			transform: CfnTemplateTransform{String: aws.String("AWS::Serverless-2016-10-31")},
			wantJSON:  `"AWS::Serverless-2016-10-31"`,
		},
		"array transform": {
			transform: CfnTemplateTransform{StringArray: &[]string{"AWS::Serverless-2016-10-31", "AWS::Include"}},
			wantJSON:  `["AWS::Serverless-2016-10-31","AWS::Include"]`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Unmarshal the expected JSON to compare structures
			var want any
			var got any

			err := json.Unmarshal([]byte(tc.wantJSON), &want)
			require.NoError(t, err)

			// Get the value from the transform
			gotValue := tc.transform.Value()

			// Convert to JSON and back to compare
			gotJSON, err := json.Marshal(gotValue)
			require.NoError(t, err)

			err = json.Unmarshal(gotJSON, &got)
			require.NoError(t, err)

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("Transform value mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestParseTemplateString_ComplexScenarios tests more complex template parsing scenarios
func TestParseTemplateString_ComplexScenarios(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		template   string
		parameters *map[string]any
		wantError  bool
	}{
		"template with conditions": {
			template: `{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Conditions": {
					"CreateProdResources": {"Fn::Equals": [{"Ref": "EnvType"}, "prod"]}
				},
				"Resources": {
					"MountPoint": {
						"Type": "AWS::EC2::VolumeAttachment",
						"Condition": "CreateProdResources",
						"Properties": {
							"InstanceId": {"Ref": "Instance"},
							"VolumeId": {"Ref": "Volume"},
							"Device": "/dev/sdh"
						}
					}
				}
			}`,
			parameters: &map[string]any{
				"EnvType": "prod",
			},
		},
		"template with outputs": {
			template: `{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Resources": {
					"MyBucket": {
						"Type": "AWS::S3::Bucket"
					}
				},
				"Outputs": {
					"BucketName": {
						"Value": {"Ref": "MyBucket"},
						"Description": "Name of S3 bucket"
					}
				}
			}`,
		},
		"template with metadata": {
			template: `{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Metadata": {
					"AWS::CloudFormation::Interface": {
						"ParameterGroups": [
							{
								"Label": {"default": "Network Configuration"},
								"Parameters": ["VpcId", "SubnetIds"]
							}
						]
					}
				},
				"Resources": {
					"MyBucket": {
						"Type": "AWS::S3::Bucket"
					}
				}
			}`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.wantError {
				assert.Panics(t, func() {
					ParseTemplateString(tc.template, tc.parameters)
				})
				return
			}

			body := ParseTemplateString(tc.template, tc.parameters)
			assert.Equal(t, "2010-09-09", body.AWSTemplateFormatVersion)
		})
	}
}
