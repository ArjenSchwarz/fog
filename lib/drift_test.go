package lib

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDriftDetectionClient implements CloudFormationDetectStackDriftAPI
type mockDriftDetectionClient struct {
	detectStackDriftFn func(context.Context, *cloudformation.DetectStackDriftInput, ...func(*cloudformation.Options)) (*cloudformation.DetectStackDriftOutput, error)
}

func (m *mockDriftDetectionClient) DetectStackDrift(ctx context.Context, params *cloudformation.DetectStackDriftInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DetectStackDriftOutput, error) {
	if m.detectStackDriftFn != nil {
		return m.detectStackDriftFn(ctx, params, optFns...)
	}
	return &cloudformation.DetectStackDriftOutput{}, nil
}

// mockDriftStatusClient implements CloudFormationDescribeStackDriftDetectionStatusAPI
type mockDriftStatusClient struct {
	describeStatusFn func(context.Context, *cloudformation.DescribeStackDriftDetectionStatusInput, ...func(*cloudformation.Options)) (*cloudformation.DescribeStackDriftDetectionStatusOutput, error)
	callCount        int
}

func (m *mockDriftStatusClient) DescribeStackDriftDetectionStatus(ctx context.Context, params *cloudformation.DescribeStackDriftDetectionStatusInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackDriftDetectionStatusOutput, error) {
	m.callCount++
	if m.describeStatusFn != nil {
		return m.describeStatusFn(ctx, params, optFns...)
	}
	return &cloudformation.DescribeStackDriftDetectionStatusOutput{}, nil
}

// mockResourceDriftsClient implements CloudFormationDescribeStackResourceDriftsAPI
type mockResourceDriftsClient struct {
	describeResourceDriftsFn func(context.Context, *cloudformation.DescribeStackResourceDriftsInput, ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourceDriftsOutput, error)
	callCount                int
}

func (m *mockResourceDriftsClient) DescribeStackResourceDrifts(ctx context.Context, params *cloudformation.DescribeStackResourceDriftsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourceDriftsOutput, error) {
	m.callCount++
	if m.describeResourceDriftsFn != nil {
		return m.describeResourceDriftsFn(ctx, params, optFns...)
	}
	return &cloudformation.DescribeStackResourceDriftsOutput{}, nil
}

// mockStacksAndResourcesClient implements both CloudFormationDescribeStacksAPI and CloudFormationDescribeStackResourcesAPI
type mockStacksAndResourcesClient struct {
	describeStacksFn         func(context.Context, *cloudformation.DescribeStacksInput, ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
	describeStackResourcesFn func(context.Context, *cloudformation.DescribeStackResourcesInput, ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error)
}

func (m *mockStacksAndResourcesClient) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	if m.describeStacksFn != nil {
		return m.describeStacksFn(ctx, params, optFns...)
	}
	return &cloudformation.DescribeStacksOutput{}, nil
}

func (m *mockStacksAndResourcesClient) DescribeStackResources(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error) {
	if m.describeStackResourcesFn != nil {
		return m.describeStackResourcesFn(ctx, params, optFns...)
	}
	return &cloudformation.DescribeStackResourcesOutput{}, nil
}

func TestStartDriftDetection(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		stackName string
		setupMock func() *mockDriftDetectionClient
		want      string
		wantPanic bool
	}{
		"successful drift detection": {
			stackName: "test-stack",
			setupMock: func() *mockDriftDetectionClient {
				driftID := "drift-detection-id-123"
				return &mockDriftDetectionClient{
					detectStackDriftFn: func(ctx context.Context, params *cloudformation.DetectStackDriftInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DetectStackDriftOutput, error) {
						return &cloudformation.DetectStackDriftOutput{
							StackDriftDetectionId: &driftID,
						}, nil
					},
				}
			},
			want: "drift-detection-id-123",
		},
		"API error triggers panic": {
			stackName: "test-stack",
			setupMock: func() *mockDriftDetectionClient {
				return &mockDriftDetectionClient{
					detectStackDriftFn: func(ctx context.Context, params *cloudformation.DetectStackDriftInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DetectStackDriftOutput, error) {
						return nil, errors.New("API error")
					},
				}
			},
			wantPanic: true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockClient := tc.setupMock()

			if tc.wantPanic {
				assert.Panics(t, func() {
					StartDriftDetection(&tc.stackName, mockClient)
				}, "Expected StartDriftDetection to panic on API error")
				return
			}

			got := StartDriftDetection(&tc.stackName, mockClient)
			assert.Equal(t, tc.want, *got)
		})
	}
}

func TestWaitForDriftDetectionToFinish(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		driftDetectionID string
		setupMock        func() *mockDriftStatusClient
		want             types.StackDriftDetectionStatus
		wantMinCalls     int
		wantPanic        bool
	}{
		"completes immediately": {
			driftDetectionID: "drift-id-123",
			setupMock: func() *mockDriftStatusClient {
				return &mockDriftStatusClient{
					describeStatusFn: func(ctx context.Context, params *cloudformation.DescribeStackDriftDetectionStatusInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackDriftDetectionStatusOutput, error) {
						return &cloudformation.DescribeStackDriftDetectionStatusOutput{
							DetectionStatus: types.StackDriftDetectionStatusDetectionComplete,
						}, nil
					},
				}
			},
			want:         types.StackDriftDetectionStatusDetectionComplete,
			wantMinCalls: 1,
		},
		"completes after one in-progress status": {
			driftDetectionID: "drift-id-456",
			setupMock: func() *mockDriftStatusClient {
				mock := &mockDriftStatusClient{}
				mock.describeStatusFn = func(ctx context.Context, params *cloudformation.DescribeStackDriftDetectionStatusInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackDriftDetectionStatusOutput, error) {
					if mock.callCount == 1 {
						return &cloudformation.DescribeStackDriftDetectionStatusOutput{
							DetectionStatus: types.StackDriftDetectionStatusDetectionInProgress,
						}, nil
					}
					return &cloudformation.DescribeStackDriftDetectionStatusOutput{
						DetectionStatus: types.StackDriftDetectionStatusDetectionComplete,
					}, nil
				}
				return mock
			},
			want:         types.StackDriftDetectionStatusDetectionComplete,
			wantMinCalls: 2,
		},
		"detection failed status": {
			driftDetectionID: "drift-id-789",
			setupMock: func() *mockDriftStatusClient {
				return &mockDriftStatusClient{
					describeStatusFn: func(ctx context.Context, params *cloudformation.DescribeStackDriftDetectionStatusInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackDriftDetectionStatusOutput, error) {
						return &cloudformation.DescribeStackDriftDetectionStatusOutput{
							DetectionStatus: types.StackDriftDetectionStatusDetectionFailed,
						}, nil
					},
				}
			},
			want:         types.StackDriftDetectionStatusDetectionFailed,
			wantMinCalls: 1,
		},
		"API error triggers panic": {
			driftDetectionID: "drift-id-error",
			setupMock: func() *mockDriftStatusClient {
				return &mockDriftStatusClient{
					describeStatusFn: func(ctx context.Context, params *cloudformation.DescribeStackDriftDetectionStatusInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackDriftDetectionStatusOutput, error) {
						return nil, errors.New("API error")
					},
				}
			},
			wantPanic: true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// Cannot run in parallel due to sleep timing dependencies
			mockClient := tc.setupMock()

			if tc.wantPanic {
				assert.Panics(t, func() {
					WaitForDriftDetectionToFinish(&tc.driftDetectionID, mockClient)
				}, "Expected WaitForDriftDetectionToFinish to panic on API error")
				return
			}

			got := WaitForDriftDetectionToFinish(&tc.driftDetectionID, mockClient)

			assert.Equal(t, tc.want, got)
			assert.GreaterOrEqual(t, mockClient.callCount, tc.wantMinCalls,
				"Expected at least %d API calls, got %d", tc.wantMinCalls, mockClient.callCount)
		})
	}
}

func TestGetDefaultStackDrift(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		stackName string
		setupMock func() *mockResourceDriftsClient
		want      []types.StackResourceDrift
		wantPanic bool
	}{
		"no drifts": {
			stackName: "test-stack",
			setupMock: func() *mockResourceDriftsClient {
				return &mockResourceDriftsClient{
					describeResourceDriftsFn: func(ctx context.Context, params *cloudformation.DescribeStackResourceDriftsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourceDriftsOutput, error) {
						return &cloudformation.DescribeStackResourceDriftsOutput{
							StackResourceDrifts: []types.StackResourceDrift{},
						}, nil
					},
				}
			},
			want: nil,
		},
		"single page of drifts": {
			stackName: "test-stack",
			setupMock: func() *mockResourceDriftsClient {
				return &mockResourceDriftsClient{
					describeResourceDriftsFn: func(ctx context.Context, params *cloudformation.DescribeStackResourceDriftsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourceDriftsOutput, error) {
						return &cloudformation.DescribeStackResourceDriftsOutput{
							StackResourceDrifts: []types.StackResourceDrift{
								{
									LogicalResourceId:        aws.String("Resource1"),
									PhysicalResourceId:       aws.String("physical-id-1"),
									ResourceType:             aws.String("AWS::S3::Bucket"),
									StackResourceDriftStatus: types.StackResourceDriftStatusModified,
									Timestamp:                aws.Time(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
								},
								{
									LogicalResourceId:        aws.String("Resource2"),
									PhysicalResourceId:       aws.String("physical-id-2"),
									ResourceType:             aws.String("AWS::IAM::Role"),
									StackResourceDriftStatus: types.StackResourceDriftStatusInSync,
									Timestamp:                aws.Time(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
								},
							},
						}, nil
					},
				}
			},
			want: []types.StackResourceDrift{
				{
					LogicalResourceId:        aws.String("Resource1"),
					PhysicalResourceId:       aws.String("physical-id-1"),
					ResourceType:             aws.String("AWS::S3::Bucket"),
					StackResourceDriftStatus: types.StackResourceDriftStatusModified,
					Timestamp:                aws.Time(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
				{
					LogicalResourceId:        aws.String("Resource2"),
					PhysicalResourceId:       aws.String("physical-id-2"),
					ResourceType:             aws.String("AWS::IAM::Role"),
					StackResourceDriftStatus: types.StackResourceDriftStatusInSync,
					Timestamp:                aws.Time(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
			},
		},
		"multiple pages of drifts": {
			stackName: "test-stack",
			setupMock: func() *mockResourceDriftsClient {
				mock := &mockResourceDriftsClient{}
				mock.describeResourceDriftsFn = func(ctx context.Context, params *cloudformation.DescribeStackResourceDriftsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourceDriftsOutput, error) {
					if mock.callCount == 1 {
						nextToken := "token-1"
						return &cloudformation.DescribeStackResourceDriftsOutput{
							StackResourceDrifts: []types.StackResourceDrift{
								{
									LogicalResourceId:        aws.String("Resource1"),
									PhysicalResourceId:       aws.String("physical-id-1"),
									ResourceType:             aws.String("AWS::S3::Bucket"),
									StackResourceDriftStatus: types.StackResourceDriftStatusModified,
									Timestamp:                aws.Time(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
								},
							},
							NextToken: &nextToken,
						}, nil
					}
					return &cloudformation.DescribeStackResourceDriftsOutput{
						StackResourceDrifts: []types.StackResourceDrift{
							{
								LogicalResourceId:        aws.String("Resource2"),
								PhysicalResourceId:       aws.String("physical-id-2"),
								ResourceType:             aws.String("AWS::IAM::Role"),
								StackResourceDriftStatus: types.StackResourceDriftStatusInSync,
								Timestamp:                aws.Time(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
							},
						},
					}, nil
				}
				return mock
			},
			want: []types.StackResourceDrift{
				{
					LogicalResourceId:        aws.String("Resource1"),
					PhysicalResourceId:       aws.String("physical-id-1"),
					ResourceType:             aws.String("AWS::S3::Bucket"),
					StackResourceDriftStatus: types.StackResourceDriftStatusModified,
					Timestamp:                aws.Time(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
				{
					LogicalResourceId:        aws.String("Resource2"),
					PhysicalResourceId:       aws.String("physical-id-2"),
					ResourceType:             aws.String("AWS::IAM::Role"),
					StackResourceDriftStatus: types.StackResourceDriftStatusInSync,
					Timestamp:                aws.Time(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
			},
		},
		"API error triggers panic": {
			stackName: "test-stack",
			setupMock: func() *mockResourceDriftsClient {
				return &mockResourceDriftsClient{
					describeResourceDriftsFn: func(ctx context.Context, params *cloudformation.DescribeStackResourceDriftsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourceDriftsOutput, error) {
						return nil, errors.New("API error")
					},
				}
			},
			wantPanic: true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockClient := tc.setupMock()

			if tc.wantPanic {
				assert.Panics(t, func() {
					GetDefaultStackDrift(&tc.stackName, mockClient)
				}, "Expected GetDefaultStackDrift to panic on API error")
				return
			}

			got := GetDefaultStackDrift(&tc.stackName, mockClient)

			opts := []cmp.Option{
				cmpopts.IgnoreUnexported(types.StackResourceDrift{}),
			}

			if diff := cmp.Diff(tc.want, got, opts...); diff != "" {
				t.Errorf("GetDefaultStackDrift() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetUncheckedStackResources(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		stackName        string
		checkedResources []string
		setupMock        func() *mockStacksAndResourcesClient
		want             []CfnResource
	}{
		"no checked resources": {
			stackName:        "test-stack",
			checkedResources: []string{},
			setupMock: func() *mockStacksAndResourcesClient {
				return &mockStacksAndResourcesClient{
					describeStacksFn: func(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
						return &cloudformation.DescribeStacksOutput{
							Stacks: []types.Stack{
								{
									StackName: aws.String("test-stack"),
									StackId:   aws.String("arn:aws:cloudformation:us-west-2:123456789012:stack/test-stack/abc"),
								},
							},
						}, nil
					},
					describeStackResourcesFn: func(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error) {
						return &cloudformation.DescribeStackResourcesOutput{
							StackResources: []types.StackResource{
								{
									LogicalResourceId:  aws.String("Resource1"),
									PhysicalResourceId: aws.String("physical-id-1"),
									ResourceType:       aws.String("AWS::S3::Bucket"),
									StackName:          aws.String("test-stack"),
								},
								{
									LogicalResourceId:  aws.String("Resource2"),
									PhysicalResourceId: aws.String("physical-id-2"),
									ResourceType:       aws.String("AWS::IAM::Role"),
									StackName:          aws.String("test-stack"),
								},
							},
						}, nil
					},
				}
			},
			want: []CfnResource{
				{
					LogicalID:  "Resource1",
					ResourceID: "physical-id-1",
					Type:       "AWS::S3::Bucket",
					StackName:  "test-stack",
				},
				{
					LogicalID:  "Resource2",
					ResourceID: "physical-id-2",
					Type:       "AWS::IAM::Role",
					StackName:  "test-stack",
				},
			},
		},
		"some checked resources": {
			stackName:        "test-stack",
			checkedResources: []string{"Resource1"},
			setupMock: func() *mockStacksAndResourcesClient {
				return &mockStacksAndResourcesClient{
					describeStacksFn: func(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
						return &cloudformation.DescribeStacksOutput{
							Stacks: []types.Stack{
								{
									StackName: aws.String("test-stack"),
									StackId:   aws.String("arn:aws:cloudformation:us-west-2:123456789012:stack/test-stack/abc"),
								},
							},
						}, nil
					},
					describeStackResourcesFn: func(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error) {
						return &cloudformation.DescribeStackResourcesOutput{
							StackResources: []types.StackResource{
								{
									LogicalResourceId:  aws.String("Resource1"),
									PhysicalResourceId: aws.String("physical-id-1"),
									ResourceType:       aws.String("AWS::S3::Bucket"),
									StackName:          aws.String("test-stack"),
								},
								{
									LogicalResourceId:  aws.String("Resource2"),
									PhysicalResourceId: aws.String("physical-id-2"),
									ResourceType:       aws.String("AWS::IAM::Role"),
									StackName:          aws.String("test-stack"),
								},
							},
						}, nil
					},
				}
			},
			want: []CfnResource{
				{
					LogicalID:  "Resource2",
					ResourceID: "physical-id-2",
					Type:       "AWS::IAM::Role",
					StackName:  "test-stack",
				},
			},
		},
		"all resources checked": {
			stackName:        "test-stack",
			checkedResources: []string{"Resource1", "Resource2"},
			setupMock: func() *mockStacksAndResourcesClient {
				return &mockStacksAndResourcesClient{
					describeStacksFn: func(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
						return &cloudformation.DescribeStacksOutput{
							Stacks: []types.Stack{
								{
									StackName: aws.String("test-stack"),
									StackId:   aws.String("arn:aws:cloudformation:us-west-2:123456789012:stack/test-stack/abc"),
								},
							},
						}, nil
					},
					describeStackResourcesFn: func(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error) {
						return &cloudformation.DescribeStackResourcesOutput{
							StackResources: []types.StackResource{
								{
									LogicalResourceId:  aws.String("Resource1"),
									PhysicalResourceId: aws.String("physical-id-1"),
									ResourceType:       aws.String("AWS::S3::Bucket"),
									StackName:          aws.String("test-stack"),
								},
								{
									LogicalResourceId:  aws.String("Resource2"),
									PhysicalResourceId: aws.String("physical-id-2"),
									ResourceType:       aws.String("AWS::IAM::Role"),
									StackName:          aws.String("test-stack"),
								},
							},
						}, nil
					},
				}
			},
			want: []CfnResource{},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockClient := tc.setupMock()
			got := GetUncheckedStackResources(&tc.stackName, tc.checkedResources, mockClient)

			require.Len(t, got, len(tc.want), "Expected %d unchecked resources", len(tc.want))

			for i := range tc.want {
				assert.Equal(t, tc.want[i].LogicalID, got[i].LogicalID, "Resource[%d].LogicalID", i)
				assert.Equal(t, tc.want[i].ResourceID, got[i].ResourceID, "Resource[%d].ResourceID", i)
				assert.Equal(t, tc.want[i].Type, got[i].Type, "Resource[%d].Type", i)
				assert.Equal(t, tc.want[i].StackName, got[i].StackName, "Resource[%d].StackName", i)
			}
		})
	}
}
