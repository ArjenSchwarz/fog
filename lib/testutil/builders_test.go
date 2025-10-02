package testutil

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMockCFNClient(t *testing.T) {
	t.Helper()

	client := NewMockCFNClient()

	assert.NotNil(t, client)
	assert.NotNil(t, client.Stacks)
	assert.NotNil(t, client.StackEvents)
	assert.NotNil(t, client.StackResources)
	assert.NotNil(t, client.Changesets)
	assert.Empty(t, client.Stacks)
	assert.Empty(t, client.StackEvents)
	assert.Empty(t, client.StackResources)
	assert.Empty(t, client.Changesets)
	assert.Nil(t, client.Error)
}

func TestMockCFNClient_WithStack(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		setup func() *MockCFNClient
		stack *types.Stack
		want  map[string]*types.Stack
	}{
		"add single stack": {
			setup: func() *MockCFNClient {
				return NewMockCFNClient()
			},
			stack: &types.Stack{
				StackName:   aws.String("test-stack"),
				StackStatus: types.StackStatusCreateComplete,
			},
			want: map[string]*types.Stack{
				"test-stack": {
					StackName:   aws.String("test-stack"),
					StackStatus: types.StackStatusCreateComplete,
				},
			},
		},
		"add multiple stacks": {
			setup: func() *MockCFNClient {
				client := NewMockCFNClient()
				client.WithStack(&types.Stack{
					StackName:   aws.String("stack-1"),
					StackStatus: types.StackStatusCreateComplete,
				})
				return client
			},
			stack: &types.Stack{
				StackName:   aws.String("stack-2"),
				StackStatus: types.StackStatusUpdateComplete,
			},
			want: map[string]*types.Stack{
				"stack-1": {
					StackName:   aws.String("stack-1"),
					StackStatus: types.StackStatusCreateComplete,
				},
				"stack-2": {
					StackName:   aws.String("stack-2"),
					StackStatus: types.StackStatusUpdateComplete,
				},
			},
		},
		"override existing stack": {
			setup: func() *MockCFNClient {
				client := NewMockCFNClient()
				client.WithStack(&types.Stack{
					StackName:   aws.String("test-stack"),
					StackStatus: types.StackStatusCreateComplete,
				})
				return client
			},
			stack: &types.Stack{
				StackName:   aws.String("test-stack"),
				StackStatus: types.StackStatusUpdateComplete,
				Description: aws.String("Updated stack"),
			},
			want: map[string]*types.Stack{
				"test-stack": {
					StackName:   aws.String("test-stack"),
					StackStatus: types.StackStatusUpdateComplete,
					Description: aws.String("Updated stack"),
				},
			},
		},
	}

	for name, tc := range tests {
		tc := tc // capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := tc.setup()
			result := client.WithStack(tc.stack)

			// Verify fluent interface returns same instance
			assert.Equal(t, client, result)

			// Verify stacks are correctly stored
			assert.Equal(t, tc.want, client.Stacks)
		})
	}
}

func TestMockCFNClient_WithError(t *testing.T) {
	t.Helper()

	expectedErr := errors.New("test error")
	client := NewMockCFNClient()
	result := client.WithError(expectedErr)

	// Verify fluent interface
	assert.Equal(t, client, result)
	assert.Equal(t, expectedErr, client.Error)
}

func TestMockCFNClient_WithStackEvents(t *testing.T) {
	t.Helper()

	event1 := types.StackEvent{
		EventId:        aws.String("event-1"),
		ResourceStatus: types.ResourceStatusCreateInProgress,
	}
	event2 := types.StackEvent{
		EventId:        aws.String("event-2"),
		ResourceStatus: types.ResourceStatusCreateComplete,
	}

	client := NewMockCFNClient()
	result := client.WithStackEvents(event1, event2)

	// Verify fluent interface
	assert.Equal(t, client, result)
	assert.Len(t, client.StackEvents, 2)
	assert.Equal(t, event1, client.StackEvents[0])
	assert.Equal(t, event2, client.StackEvents[1])
}

func TestMockCFNClient_DescribeStacks(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		setup   func() *MockCFNClient
		input   *cloudformation.DescribeStacksInput
		want    *cloudformation.DescribeStacksOutput
		wantErr bool
		errMsg  string
	}{
		"return specific stack by name": {
			setup: func() *MockCFNClient {
				return NewMockCFNClient().WithStack(&types.Stack{
					StackName:   aws.String("test-stack"),
					StackStatus: types.StackStatusCreateComplete,
				})
			},
			input: &cloudformation.DescribeStacksInput{
				StackName: aws.String("test-stack"),
			},
			want: &cloudformation.DescribeStacksOutput{
				Stacks: []types.Stack{
					{
						StackName:   aws.String("test-stack"),
						StackStatus: types.StackStatusCreateComplete,
					},
				},
			},
		},
		"return all stacks when no name specified": {
			setup: func() *MockCFNClient {
				client := NewMockCFNClient()
				client.WithStack(&types.Stack{
					StackName: aws.String("stack-1"),
				})
				client.WithStack(&types.Stack{
					StackName: aws.String("stack-2"),
				})
				return client
			},
			input: &cloudformation.DescribeStacksInput{},
			want: &cloudformation.DescribeStacksOutput{
				Stacks: []types.Stack{
					{StackName: aws.String("stack-1")},
					{StackName: aws.String("stack-2")},
				},
			},
		},
		"return error for non-existent stack": {
			setup: func() *MockCFNClient {
				return NewMockCFNClient()
			},
			input: &cloudformation.DescribeStacksInput{
				StackName: aws.String("non-existent"),
			},
			wantErr: true,
			errMsg:  "Stack with name non-existent does not exist",
		},
		"return configured error": {
			setup: func() *MockCFNClient {
				return NewMockCFNClient().WithError(errors.New("API error"))
			},
			input: &cloudformation.DescribeStacksInput{
				StackName: aws.String("test-stack"),
			},
			wantErr: true,
			errMsg:  "API error",
		},
		"use custom function when provided": {
			setup: func() *MockCFNClient {
				client := NewMockCFNClient()
				client.DescribeStacksFn = func(ctx context.Context, input *cloudformation.DescribeStacksInput, opts ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
					return &cloudformation.DescribeStacksOutput{
						Stacks: []types.Stack{
							{
								StackName:   aws.String("custom-stack"),
								StackStatus: types.StackStatusCreateInProgress,
							},
						},
					}, nil
				}
				return client
			},
			input: &cloudformation.DescribeStacksInput{},
			want: &cloudformation.DescribeStacksOutput{
				Stacks: []types.Stack{
					{
						StackName:   aws.String("custom-stack"),
						StackStatus: types.StackStatusCreateInProgress,
					},
				},
			},
		},
	}

	for name, tc := range tests {
		tc := tc // capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := tc.setup()
			got, err := client.DescribeStacks(context.Background(), tc.input)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				return
			}

			require.NoError(t, err)

			// Sort stacks for comparison (order may vary)
			opts := []cmp.Option{
				cmpopts.SortSlices(func(a, b types.Stack) bool {
					return *a.StackName < *b.StackName
				}),
				cmpopts.IgnoreUnexported(types.Stack{}),
				cmpopts.IgnoreUnexported(cloudformation.DescribeStacksOutput{}),
				cmpopts.IgnoreFields(cloudformation.DescribeStacksOutput{}, "ResultMetadata"),
			}

			if diff := cmp.Diff(tc.want, got, opts...); diff != "" {
				t.Errorf("DescribeStacks() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMockCFNClient_DescribeStackEvents(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		setup   func() *MockCFNClient
		want    *cloudformation.DescribeStackEventsOutput
		wantErr bool
	}{
		"return configured events": {
			setup: func() *MockCFNClient {
				return NewMockCFNClient().WithStackEvents(
					types.StackEvent{
						EventId:        aws.String("event-1"),
						ResourceStatus: types.ResourceStatusCreateInProgress,
					},
					types.StackEvent{
						EventId:        aws.String("event-2"),
						ResourceStatus: types.ResourceStatusCreateComplete,
					},
				)
			},
			want: &cloudformation.DescribeStackEventsOutput{
				StackEvents: []types.StackEvent{
					{
						EventId:        aws.String("event-1"),
						ResourceStatus: types.ResourceStatusCreateInProgress,
					},
					{
						EventId:        aws.String("event-2"),
						ResourceStatus: types.ResourceStatusCreateComplete,
					},
				},
			},
		},
		"return empty when no events": {
			setup: func() *MockCFNClient {
				return NewMockCFNClient()
			},
			want: &cloudformation.DescribeStackEventsOutput{
				StackEvents: []types.StackEvent{},
			},
		},
		"return error when configured": {
			setup: func() *MockCFNClient {
				return NewMockCFNClient().WithError(errors.New("API error"))
			},
			wantErr: true,
		},
		"use custom function when provided": {
			setup: func() *MockCFNClient {
				client := NewMockCFNClient()
				client.DescribeStackEventsFn = func(ctx context.Context, input *cloudformation.DescribeStackEventsInput, opts ...func(*cloudformation.Options)) (*cloudformation.DescribeStackEventsOutput, error) {
					return &cloudformation.DescribeStackEventsOutput{
						StackEvents: []types.StackEvent{
							{EventId: aws.String("custom-event")},
						},
					}, nil
				}
				return client
			},
			want: &cloudformation.DescribeStackEventsOutput{
				StackEvents: []types.StackEvent{
					{EventId: aws.String("custom-event")},
				},
			},
		},
	}

	for name, tc := range tests {
		tc := tc // capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := tc.setup()
			got, err := client.DescribeStackEvents(context.Background(), &cloudformation.DescribeStackEventsInput{})

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestMockEC2Client(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		name string
		test func(t *testing.T)
	}{
		"NewMockEC2Client": {
			test: func(t *testing.T) {
				client := NewMockEC2Client()
				assert.NotNil(t, client)
				assert.NotNil(t, client.RouteTables)
				assert.NotNil(t, client.NetworkAcls)
				assert.Empty(t, client.RouteTables)
				assert.Empty(t, client.NetworkAcls)
			},
		},
		"WithRouteTable": {
			test: func(t *testing.T) {
				rt := ec2types.RouteTable{
					RouteTableId: aws.String("rtb-12345"),
				}

				client := NewMockEC2Client()
				result := client.WithRouteTable(rt)

				assert.Equal(t, client, result) // Fluent interface
				assert.Len(t, client.RouteTables, 1)
				assert.Equal(t, rt, client.RouteTables[0])
			},
		},
		"WithNetworkAcl": {
			test: func(t *testing.T) {
				acl := ec2types.NetworkAcl{
					NetworkAclId: aws.String("acl-12345"),
				}

				client := NewMockEC2Client()
				result := client.WithNetworkAcl(acl)

				assert.Equal(t, client, result) // Fluent interface
				assert.Len(t, client.NetworkAcls, 1)
				assert.Equal(t, acl, client.NetworkAcls[0])
			},
		},
		"WithError": {
			test: func(t *testing.T) {
				expectedErr := errors.New("EC2 error")
				client := NewMockEC2Client()
				result := client.WithError(expectedErr)

				assert.Equal(t, client, result) // Fluent interface
				assert.Equal(t, expectedErr, client.Error)
			},
		},
		"DescribeRouteTables": {
			test: func(t *testing.T) {
				rt := ec2types.RouteTable{
					RouteTableId: aws.String("rtb-12345"),
				}

				client := NewMockEC2Client().WithRouteTable(rt)
				got, err := client.DescribeRouteTables(context.Background(), &ec2.DescribeRouteTablesInput{})

				require.NoError(t, err)
				assert.Len(t, got.RouteTables, 1)
				assert.Equal(t, rt, got.RouteTables[0])
			},
		},
		"DescribeNetworkAcls": {
			test: func(t *testing.T) {
				acl := ec2types.NetworkAcl{
					NetworkAclId: aws.String("acl-12345"),
				}

				client := NewMockEC2Client().WithNetworkAcl(acl)
				got, err := client.DescribeNetworkAcls(context.Background(), &ec2.DescribeNetworkAclsInput{})

				require.NoError(t, err)
				assert.Len(t, got.NetworkAcls, 1)
				assert.Equal(t, acl, got.NetworkAcls[0])
			},
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.test(t)
		})
	}
}

func TestMockS3Client(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		name string
		test func(t *testing.T)
	}{
		"NewMockS3Client": {
			test: func(t *testing.T) {
				client := NewMockS3Client()
				assert.NotNil(t, client)
				assert.NotNil(t, client.Objects)
				assert.Empty(t, client.Objects)
			},
		},
		"WithObject": {
			test: func(t *testing.T) {
				data := []byte("test data")
				client := NewMockS3Client()
				result := client.WithObject("test-key", data)

				assert.Equal(t, client, result) // Fluent interface
				assert.Equal(t, data, client.Objects["test-key"])
			},
		},
		"WithError": {
			test: func(t *testing.T) {
				expectedErr := errors.New("S3 error")
				client := NewMockS3Client()
				result := client.WithError(expectedErr)

				assert.Equal(t, client, result) // Fluent interface
				assert.Equal(t, expectedErr, client.Error)
			},
		},
		"PutObject success": {
			test: func(t *testing.T) {
				client := NewMockS3Client()
				input := &s3.PutObjectInput{
					Key: aws.String("test-key"),
				}

				got, err := client.PutObject(context.Background(), input)

				require.NoError(t, err)
				assert.NotNil(t, got.ETag)
				assert.Contains(t, client.Objects, "test-key")
			},
		},
		"PutObject with error": {
			test: func(t *testing.T) {
				client := NewMockS3Client().WithError(errors.New("upload failed"))
				input := &s3.PutObjectInput{
					Key: aws.String("test-key"),
				}

				_, err := client.PutObject(context.Background(), input)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "upload failed")
			},
		},
		"HeadObject existing": {
			test: func(t *testing.T) {
				data := []byte("test data")
				client := NewMockS3Client().WithObject("test-key", data)
				input := &s3.HeadObjectInput{
					Key: aws.String("test-key"),
				}

				got, err := client.HeadObject(context.Background(), input)

				require.NoError(t, err)
				assert.Equal(t, int64(len(data)), *got.ContentLength)
			},
		},
		"HeadObject non-existent": {
			test: func(t *testing.T) {
				client := NewMockS3Client()
				input := &s3.HeadObjectInput{
					Key: aws.String("missing-key"),
				}

				_, err := client.HeadObject(context.Background(), input)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "does not exist")
			},
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.test(t)
		})
	}
}

func TestStackBuilder(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		name  string
		setup func() *types.Stack
		check func(t *testing.T, stack *types.Stack)
	}{
		"basic stack creation": {
			setup: func() *types.Stack {
				return NewStackBuilder("test-stack").Build()
			},
			check: func(t *testing.T, stack *types.Stack) {
				assert.Equal(t, "test-stack", *stack.StackName)
				assert.Equal(t, types.StackStatusCreateComplete, stack.StackStatus)
				assert.NotNil(t, stack.CreationTime)
				assert.Contains(t, *stack.StackId, "test-stack")
			},
		},
		"with custom status": {
			setup: func() *types.Stack {
				return NewStackBuilder("test-stack").
					WithStatus(types.StackStatusUpdateInProgress).
					Build()
			},
			check: func(t *testing.T, stack *types.Stack) {
				assert.Equal(t, types.StackStatusUpdateInProgress, stack.StackStatus)
			},
		},
		"with parameters": {
			setup: func() *types.Stack {
				return NewStackBuilder("test-stack").
					WithParameter("Environment", "Production").
					WithParameter("Version", "1.0.0").
					Build()
			},
			check: func(t *testing.T, stack *types.Stack) {
				assert.Len(t, stack.Parameters, 2)

				params := make(map[string]string)
				for _, p := range stack.Parameters {
					params[*p.ParameterKey] = *p.ParameterValue
				}

				assert.Equal(t, "Production", params["Environment"])
				assert.Equal(t, "1.0.0", params["Version"])
			},
		},
		"with outputs": {
			setup: func() *types.Stack {
				return NewStackBuilder("test-stack").
					WithOutput("BucketName", "my-bucket").
					WithOutput("Region", "us-west-2").
					Build()
			},
			check: func(t *testing.T, stack *types.Stack) {
				assert.Len(t, stack.Outputs, 2)

				outputs := make(map[string]string)
				for _, o := range stack.Outputs {
					outputs[*o.OutputKey] = *o.OutputValue
				}

				assert.Equal(t, "my-bucket", outputs["BucketName"])
				assert.Equal(t, "us-west-2", outputs["Region"])
			},
		},
		"with tags": {
			setup: func() *types.Stack {
				return NewStackBuilder("test-stack").
					WithTag("Team", "Platform").
					WithTag("Environment", "Dev").
					Build()
			},
			check: func(t *testing.T, stack *types.Stack) {
				assert.Len(t, stack.Tags, 2)

				tags := make(map[string]string)
				for _, tag := range stack.Tags {
					tags[*tag.Key] = *tag.Value
				}

				assert.Equal(t, "Platform", tags["Team"])
				assert.Equal(t, "Dev", tags["Environment"])
			},
		},
		"with capabilities": {
			setup: func() *types.Stack {
				return NewStackBuilder("test-stack").
					WithCapability(types.CapabilityCapabilityIam).
					WithCapability(types.CapabilityCapabilityNamedIam).
					Build()
			},
			check: func(t *testing.T, stack *types.Stack) {
				assert.Len(t, stack.Capabilities, 2)
				assert.Contains(t, stack.Capabilities, types.CapabilityCapabilityIam)
				assert.Contains(t, stack.Capabilities, types.CapabilityCapabilityNamedIam)
			},
		},
		"with description": {
			setup: func() *types.Stack {
				return NewStackBuilder("test-stack").
					WithDescription("Test stack for unit testing").
					Build()
			},
			check: func(t *testing.T, stack *types.Stack) {
				require.NotNil(t, stack.Description)
				assert.Equal(t, "Test stack for unit testing", *stack.Description)
			},
		},
		"with drift status": {
			setup: func() *types.Stack {
				return NewStackBuilder("test-stack").
					WithDriftStatus(types.StackDriftStatusDrifted).
					Build()
			},
			check: func(t *testing.T, stack *types.Stack) {
				require.NotNil(t, stack.DriftInformation)
				assert.Equal(t, types.StackDriftStatusDrifted, stack.DriftInformation.StackDriftStatus)
			},
		},
		"complex stack with all options": {
			setup: func() *types.Stack {
				return NewStackBuilder("complex-stack").
					WithStatus(types.StackStatusUpdateComplete).
					WithParameter("Env", "Prod").
					WithOutput("URL", "https://example.com").
					WithTag("Owner", "TeamA").
					WithCapability(types.CapabilityCapabilityIam).
					WithDescription("Complex test stack").
					WithDriftStatus(types.StackDriftStatusInSync).
					Build()
			},
			check: func(t *testing.T, stack *types.Stack) {
				assert.Equal(t, "complex-stack", *stack.StackName)
				assert.Equal(t, types.StackStatusUpdateComplete, stack.StackStatus)
				assert.Len(t, stack.Parameters, 1)
				assert.Len(t, stack.Outputs, 1)
				assert.Len(t, stack.Tags, 1)
				assert.Len(t, stack.Capabilities, 1)
				assert.NotNil(t, stack.Description)
				assert.NotNil(t, stack.DriftInformation)
			},
		},
	}

	for name, tc := range tests {
		tc := tc // capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stack := tc.setup()
			tc.check(t, stack)
		})
	}
}

func TestStackEventBuilder(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		name  string
		setup func() types.StackEvent
		check func(t *testing.T, event types.StackEvent)
	}{
		"basic event creation": {
			setup: func() types.StackEvent {
				return NewStackEventBuilder("test-stack", "MyResource").Build()
			},
			check: func(t *testing.T, event types.StackEvent) {
				assert.Equal(t, "test-stack", *event.StackName)
				assert.Equal(t, "MyResource", *event.LogicalResourceId)
				assert.Equal(t, "physical-MyResource", *event.PhysicalResourceId)
				assert.Equal(t, "AWS::S3::Bucket", *event.ResourceType)
				assert.Equal(t, types.ResourceStatusCreateComplete, event.ResourceStatus)
				assert.NotNil(t, event.EventId)
				assert.NotNil(t, event.Timestamp)
			},
		},
		"with custom status": {
			setup: func() types.StackEvent {
				return NewStackEventBuilder("test-stack", "MyResource").
					WithStatus(types.ResourceStatusDeleteInProgress).
					Build()
			},
			check: func(t *testing.T, event types.StackEvent) {
				assert.Equal(t, types.ResourceStatusDeleteInProgress, event.ResourceStatus)
			},
		},
		"with resource type": {
			setup: func() types.StackEvent {
				return NewStackEventBuilder("test-stack", "MyFunction").
					WithResourceType("AWS::Lambda::Function").
					Build()
			},
			check: func(t *testing.T, event types.StackEvent) {
				assert.Equal(t, "AWS::Lambda::Function", *event.ResourceType)
			},
		},
		"with status reason": {
			setup: func() types.StackEvent {
				return NewStackEventBuilder("test-stack", "MyResource").
					WithStatusReason("Resource creation failed due to invalid configuration").
					Build()
			},
			check: func(t *testing.T, event types.StackEvent) {
				require.NotNil(t, event.ResourceStatusReason)
				assert.Contains(t, *event.ResourceStatusReason, "invalid configuration")
			},
		},
		"with custom timestamp": {
			setup: func() types.StackEvent {
				customTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
				return NewStackEventBuilder("test-stack", "MyResource").
					WithTimestamp(customTime).
					Build()
			},
			check: func(t *testing.T, event types.StackEvent) {
				expectedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
				assert.Equal(t, expectedTime, *event.Timestamp)
			},
		},
		"complex event with all options": {
			setup: func() types.StackEvent {
				customTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
				return NewStackEventBuilder("prod-stack", "Database").
					WithStatus(types.ResourceStatusUpdateFailed).
					WithResourceType("AWS::RDS::DBInstance").
					WithStatusReason("Update failed: insufficient permissions").
					WithTimestamp(customTime).
					Build()
			},
			check: func(t *testing.T, event types.StackEvent) {
				assert.Equal(t, "prod-stack", *event.StackName)
				assert.Equal(t, "Database", *event.LogicalResourceId)
				assert.Equal(t, types.ResourceStatusUpdateFailed, event.ResourceStatus)
				assert.Equal(t, "AWS::RDS::DBInstance", *event.ResourceType)
				assert.Contains(t, *event.ResourceStatusReason, "insufficient permissions")

				expectedTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
				assert.Equal(t, expectedTime, *event.Timestamp)
			},
		},
	}

	for name, tc := range tests {
		tc := tc // capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			event := tc.setup()
			tc.check(t, event)
		})
	}
}

// TestMockCFNClient_ErrorInjection tests error injection capabilities
func TestMockCFNClient_ErrorInjection(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		name      string
		setupMock func() *MockCFNClient
		operation func(*MockCFNClient) error
		wantErr   string
	}{
		"inject error for DescribeStacks": {
			setupMock: func() *MockCFNClient {
				return NewMockCFNClient().WithError(errors.New("throttling error"))
			},
			operation: func(client *MockCFNClient) error {
				_, err := client.DescribeStacks(context.Background(), &cloudformation.DescribeStacksInput{})
				return err
			},
			wantErr: "throttling error",
		},
		"inject error for DescribeStackEvents": {
			setupMock: func() *MockCFNClient {
				return NewMockCFNClient().WithError(errors.New("access denied"))
			},
			operation: func(client *MockCFNClient) error {
				_, err := client.DescribeStackEvents(context.Background(), &cloudformation.DescribeStackEventsInput{})
				return err
			},
			wantErr: "access denied",
		},
		"inject error for DescribeStackResources": {
			setupMock: func() *MockCFNClient {
				return NewMockCFNClient().WithError(errors.New("stack not found"))
			},
			operation: func(client *MockCFNClient) error {
				_, err := client.DescribeStackResources(context.Background(), &cloudformation.DescribeStackResourcesInput{})
				return err
			},
			wantErr: "stack not found",
		},
	}

	for name, tc := range tests {
		tc := tc // capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := tc.setupMock()
			err := tc.operation(client)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

// TestMockCFNClient_ConcurrentAccess tests thread safety of mock client
func TestMockCFNClient_ConcurrentAccess(t *testing.T) {
	t.Helper()

	client := NewMockCFNClient()

	// Add initial stacks
	for i := 0; i < 10; i++ {
		stack := &types.Stack{
			StackName:   aws.String(string(rune('A' + i))),
			StackStatus: types.StackStatusCreateComplete,
		}
		client.WithStack(stack)
	}

	// Concurrent reads should not cause race conditions
	t.Run("concurrent reads", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			go func() {
				_, _ = client.DescribeStacks(context.Background(), &cloudformation.DescribeStacksInput{})
			}()
		}
	})

	// Note: Concurrent writes would require mutex protection in production code
	// This test demonstrates current behavior - production code should add synchronization
}
