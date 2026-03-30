package lib

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mocks implementing the CloudFormation interfaces used by the stack helpers

// mockStackEventsClient supports multi-page DescribeStackEvents responses.
// Pages are keyed by NextToken ("" for the first call).
// Use pageErrors to inject errors on specific pages.
type mockStackEventsClient struct {
	pages      map[string]cloudformation.DescribeStackEventsOutput
	pageErrors map[string]error
	err        error
}

func newSinglePageMock(output cloudformation.DescribeStackEventsOutput, err error) mockStackEventsClient {
	return mockStackEventsClient{
		pages: map[string]cloudformation.DescribeStackEventsOutput{"": output},
		err:   err,
	}
}

func (m mockStackEventsClient) DescribeStackEvents(ctx context.Context, params *cloudformation.DescribeStackEventsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackEventsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	token := ""
	if params.NextToken != nil {
		token = *params.NextToken
	}
	if m.pageErrors != nil {
		if err, ok := m.pageErrors[token]; ok {
			return nil, err
		}
	}
	out := m.pages[token]
	return &out, nil
}

type mockDescribeStacksClient struct {
	output cloudformation.DescribeStacksOutput
	err    error
}

func (m mockDescribeStacksClient) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	return &m.output, m.err
}

// Helper to create string pointers in tests
func strPtr(s string) *string        { return &s }
func ptrTime(t time.Time) *time.Time { return &t }

func TestDeployInfo_GetEvents(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := map[string]struct {
		deployment *DeployInfo
		mock       mockStackEventsClient
		wantCount  int
		wantErr    bool
	}{
		"single page of events": {
			deployment: &DeployInfo{StackName: "test-stack"},
			mock: newSinglePageMock(cloudformation.DescribeStackEventsOutput{
				StackEvents: []types.StackEvent{
					{LogicalResourceId: strPtr("Res1"), Timestamp: ptrTime(now)},
					{LogicalResourceId: strPtr("Res2"), Timestamp: ptrTime(now)},
				},
			}, nil),
			wantCount: 2,
		},
		"multiple pages of events": {
			deployment: &DeployInfo{StackName: "big-stack"},
			mock: mockStackEventsClient{
				pages: map[string]cloudformation.DescribeStackEventsOutput{
					"": {
						StackEvents: []types.StackEvent{
							{LogicalResourceId: strPtr("Page1A"), Timestamp: ptrTime(now)},
							{LogicalResourceId: strPtr("Page1B"), Timestamp: ptrTime(now)},
						},
						NextToken: strPtr("token2"),
					},
					"token2": {
						StackEvents: []types.StackEvent{
							{LogicalResourceId: strPtr("Page2A"), Timestamp: ptrTime(now)},
						},
						NextToken: strPtr("token3"),
					},
					"token3": {
						StackEvents: []types.StackEvent{
							{LogicalResourceId: strPtr("Page3A"), Timestamp: ptrTime(now)},
							{LogicalResourceId: strPtr("Page3B"), Timestamp: ptrTime(now)},
						},
					},
				},
			},
			wantCount: 5,
		},
		"empty events": {
			deployment: &DeployInfo{StackName: "empty-stack"},
			mock:       newSinglePageMock(cloudformation.DescribeStackEventsOutput{}, nil),
			wantCount:  0,
		},
		"API error": {
			deployment: &DeployInfo{StackName: "bad-stack"},
			mock:       mockStackEventsClient{err: errors.New("access denied")},
			wantErr:    true,
		},
		"API error on second page returns nil": {
			deployment: &DeployInfo{StackName: "mid-error-stack"},
			mock: mockStackEventsClient{
				pages: map[string]cloudformation.DescribeStackEventsOutput{
					"": {
						StackEvents: []types.StackEvent{
							{LogicalResourceId: strPtr("Page1"), Timestamp: ptrTime(now)},
						},
						NextToken: strPtr("page2"),
					},
				},
				pageErrors: map[string]error{
					"page2": errors.New("throttling"),
				},
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.deployment.GetEvents(tc.mock)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, got, tc.wantCount)
		})
	}
}

func TestDeployInfo_GetCleanedStackName(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		stackName string
		want      string
	}{
		"no ARN in stack name": {
			stackName: "test-stack",
			want:      "test-stack",
		},
		"ARN in stack name": {
			stackName: "arn:aws:cloudformation:ap-southeast-2:12345678901:stack/test-stack/5f584530-013c-11ee-9c69-0a254d5985de",
			want:      "test-stack",
		},
		// Regression tests for T-659: malformed ARN input must not panic
		"malformed ARN without slash returns original": {
			stackName: "arn:aws:cloudformation:ap-southeast-2:123456789012",
			want:      "arn:aws:cloudformation:ap-southeast-2:123456789012",
		},
		"ARN prefix only returns original": {
			stackName: "arn:",
			want:      "arn:",
		},
		"ARN with trailing slash returns original ARN": {
			stackName: "arn:aws:cloudformation:ap-southeast-2:123456789012:stack/",
			want:      "arn:aws:cloudformation:ap-southeast-2:123456789012:stack/",
		},
		"ARN with single slash and name returns name": {
			stackName: "arn:aws:cloudformation:ap-southeast-2:123456789012:stack/my-stack",
			want:      "my-stack",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			deployment := &DeployInfo{
				StackName: tc.stackName,
			}

			got := deployment.GetCleanedStackName()
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestDeployInfo_GetExecutionTimes verifies that resource event timestamps are grouped per resource and status.
func TestDeployInfo_GetExecutionTimes(t *testing.T) {
	t.Helper()

	now := time.Now().UTC()

	tests := map[string]struct {
		deployment *DeployInfo
		events     []types.StackEvent
		want       map[string]map[string]time.Time
		wantErr    bool
	}{
		"groups events by resource and status": {
			deployment: &DeployInfo{
				Changeset: &ChangesetInfo{CreationTime: now},
				StackName: "test-stack",
			},
			events: []types.StackEvent{
				{
					LogicalResourceId: strPtr("Bucket"),
					ResourceType:      strPtr("AWS::S3::Bucket"),
					ResourceStatus:    types.ResourceStatusCreateInProgress,
					Timestamp:         ptrTime(now.Add(1 * time.Minute)),
				},
				{
					LogicalResourceId: strPtr("Bucket"),
					ResourceType:      strPtr("AWS::S3::Bucket"),
					ResourceStatus:    types.ResourceStatusCreateComplete,
					Timestamp:         ptrTime(now.Add(2 * time.Minute)),
				},
				{
					LogicalResourceId: strPtr("Role"),
					ResourceType:      strPtr("AWS::IAM::Role"),
					ResourceStatus:    types.ResourceStatusCreateInProgress,
					Timestamp:         ptrTime(now.Add(3 * time.Minute)),
				},
				{
					LogicalResourceId: strPtr("Old"),
					ResourceType:      strPtr("AWS::S3::Bucket"),
					ResourceStatus:    types.ResourceStatusCreateInProgress,
					Timestamp:         ptrTime(now.Add(-1 * time.Minute)),
				},
			},
			want: map[string]map[string]time.Time{
				"AWS  S3  Bucket (Bucket)": {
					string(types.ResourceStatusCreateInProgress): now.Add(1 * time.Minute),
					string(types.ResourceStatusCreateComplete):   now.Add(2 * time.Minute),
				},
				"AWS  IAM  Role (Role)": {
					string(types.ResourceStatusCreateInProgress): now.Add(3 * time.Minute),
				},
			},
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockSvc := newSinglePageMock(cloudformation.DescribeStackEventsOutput{StackEvents: tc.events}, nil)

			got, err := tc.deployment.GetExecutionTimes(mockSvc)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// Test_stringInSlice checks membership detection in a slice.
func Test_stringInSlice(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		needle   string
		haystack []string
		want     bool
	}{
		"string exists in slice": {
			needle:   "a",
			haystack: []string{"b", "a"},
			want:     true,
		},
		"string does not exist in slice": {
			needle:   "c",
			haystack: []string{"a", "b"},
			want:     false,
		},
		"empty slice": {
			needle:   "a",
			haystack: []string{},
			want:     false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := stringInSlice(tc.needle, tc.haystack)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestResourceEvent_GetDuration ensures the duration uses StartDate and EndDate.
func TestResourceEvent_GetDuration(t *testing.T) {
	t.Helper()

	start := time.Now()

	tests := map[string]struct {
		event ResourceEvent
		want  time.Duration
	}{
		"2 minute duration": {
			event: ResourceEvent{
				StartDate: start,
				EndDate:   start.Add(2 * time.Minute),
			},
			want: 2 * time.Minute,
		},
		"5 second duration": {
			event: ResourceEvent{
				StartDate: start,
				EndDate:   start.Add(5 * time.Second),
			},
			want: 5 * time.Second,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tc.event.GetDuration()
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestStackEvent_GetDuration ensures the duration uses StartDate and EndDate.
func TestStackEvent_GetDuration(t *testing.T) {
	t.Helper()

	start := time.Now()

	tests := map[string]struct {
		event StackEvent
		want  time.Duration
	}{
		"3 minute duration": {
			event: StackEvent{
				StartDate: start,
				EndDate:   start.Add(3 * time.Minute),
			},
			want: 3 * time.Minute,
		},
		"10 second duration": {
			event: StackEvent{
				StartDate: start,
				EndDate:   start.Add(10 * time.Second),
			},
			want: 10 * time.Second,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tc.event.GetDuration()
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestDeployInfo_ChangesetType verifies that the correct changeset type is
// returned based on whether the stack is new or existing.
func TestDeployInfo_ChangesetType(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		isNew bool
		want  types.ChangeSetType
	}{
		"new stack": {
			isNew: true,
			want:  types.ChangeSetTypeCreate,
		},
		"existing stack": {
			isNew: false,
			want:  types.ChangeSetTypeUpdate,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			deployment := &DeployInfo{IsNew: tc.isNew}

			got := deployment.ChangesetType()
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestGetStack verifies that GetStack correctly retrieves stack information
// using the mocked CloudFormation client.
func TestGetStack(t *testing.T) {
	t.Parallel()

	stackName := "test-stack"
	expectedStack := types.Stack{
		StackName: strPtr(stackName),
		StackId:   strPtr("arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123"),
	}

	tests := map[string]struct {
		client    mockDescribeStacksClient
		stackName *string
		want      types.Stack
		wantErr   bool
		errMsg    string
	}{
		"successful retrieval": {
			client: mockDescribeStacksClient{
				output: cloudformation.DescribeStacksOutput{Stacks: []types.Stack{expectedStack}},
			},
			stackName: &stackName,
			want:      expectedStack,
			wantErr:   false,
		},
		"stack not found error": {
			client: mockDescribeStacksClient{
				err: errors.New("stack not found"),
			},
			stackName: &stackName,
			wantErr:   true,
			errMsg:    "stack not found",
		},
		"empty stacks response": {
			client: mockDescribeStacksClient{
				output: cloudformation.DescribeStacksOutput{Stacks: []types.Stack{}},
			},
			stackName: &stackName,
			wantErr:   true,
			errMsg:    "no stacks found",
		},
		"multiple stacks response": {
			client: mockDescribeStacksClient{
				output: cloudformation.DescribeStacksOutput{Stacks: []types.Stack{
					{StackName: strPtr("stack-a")},
					{StackName: strPtr("stack-b")},
				}},
			},
			stackName: &stackName,
			wantErr:   true,
			errMsg:    "expected exactly one stack",
		},
		"nil stack name": {
			client:    mockDescribeStacksClient{},
			stackName: nil,
			wantErr:   true,
			errMsg:    "stack name must not be nil",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := GetStack(tc.stackName, tc.client)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				return
			}

			require.NoError(t, err)

			opts := []cmp.Option{
				cmpopts.IgnoreUnexported(types.Stack{}),
			}

			if diff := cmp.Diff(tc.want, got, opts...); diff != "" {
				t.Errorf("GetStack() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestParseParameterString verifies that JSON parameter strings are correctly
// parsed into CloudFormation parameter structures.
func TestParseParameterString(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		input   string
		want    []types.Parameter
		wantErr bool
	}{
		"valid parameters": {
			input: `[{"ParameterKey":"Key1","ParameterValue":"Value1"},{"ParameterKey":"Key2","ParameterValue":"Value2"}]`,
			want: []types.Parameter{
				{ParameterKey: strPtr("Key1"), ParameterValue: strPtr("Value1")},
				{ParameterKey: strPtr("Key2"), ParameterValue: strPtr("Value2")},
			},
			wantErr: false,
		},
		"empty array": {
			input:   `[]`,
			want:    []types.Parameter{},
			wantErr: false,
		},
		"invalid JSON": {
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseParameterString(tc.input)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			opts := []cmp.Option{
				cmpopts.IgnoreUnexported(types.Parameter{}),
			}

			if diff := cmp.Diff(tc.want, got, opts...); diff != "" {
				t.Errorf("ParseParameterString() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestParseTagString verifies that JSON tag strings are correctly parsed
// into CloudFormation tag structures.
func TestParseTagString(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		input   string
		want    []types.Tag
		wantErr bool
	}{
		"valid tags": {
			input: `[{"Key":"Environment","Value":"Production"},{"Key":"Owner","Value":"TeamA"}]`,
			want: []types.Tag{
				{Key: strPtr("Environment"), Value: strPtr("Production")},
				{Key: strPtr("Owner"), Value: strPtr("TeamA")},
			},
			wantErr: false,
		},
		"empty array": {
			input:   `[]`,
			want:    []types.Tag{},
			wantErr: false,
		},
		"invalid JSON": {
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseTagString(tc.input)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			opts := []cmp.Option{
				cmpopts.IgnoreUnexported(types.Tag{}),
			}

			if diff := cmp.Diff(tc.want, got, opts...); diff != "" {
				t.Errorf("ParseTagString() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestParseDeploymentFile verifies that deployment files in both JSON and YAML
// formats are correctly parsed.
func TestParseDeploymentFile(t *testing.T) {
	jsonInput := `{"template-file-path":"templates/test-stack.yaml","parameters":{"Key1":"Value1"}}`
	yamlInput := `template-file-path: templates/test-stack.yaml
parameters:
  Key1: Value1`
	yamlFlowInput := " \n{template-file-path: templates/test-stack.yaml, parameters: {Key1: Value1}}"

	tests := map[string]struct {
		input                  string
		wantTemplateFilePath   string
		wantParametersCount    int
		wantParameterKey1Value string
		wantErr                bool
		wantErrContains        string
	}{
		"valid JSON": {
			input:                  jsonInput,
			wantTemplateFilePath:   "templates/test-stack.yaml",
			wantParametersCount:    1,
			wantParameterKey1Value: "Value1",
			wantErr:                false,
		},
		"valid YAML": {
			input:                  yamlInput,
			wantTemplateFilePath:   "templates/test-stack.yaml",
			wantParametersCount:    1,
			wantParameterKey1Value: "Value1",
			wantErr:                false,
		},
		"valid YAML flow mapping with leading whitespace": {
			input:                  yamlFlowInput,
			wantTemplateFilePath:   "templates/test-stack.yaml",
			wantParametersCount:    1,
			wantParameterKey1Value: "Value1",
			wantErr:                false,
		},
		"invalid JSON": {
			input:   `{invalid`,
			wantErr: true,
		},
		"json array input": {
			input:   `[]`,
			wantErr: true,
		},
		"empty input": {
			input:           ``,
			wantErr:         true,
			wantErrContains: "empty or whitespace",
		},
		"whitespace only input": {
			input:           " \n\t ",
			wantErr:         true,
			wantErrContains: "empty or whitespace",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseDeploymentFile(tc.input)

			if tc.wantErr {
				require.Error(t, err)
				if tc.wantErrContains != "" {
					assert.Contains(t, err.Error(), tc.wantErrContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantTemplateFilePath, got.TemplateFilePath)
			assert.Len(t, got.Parameters, tc.wantParametersCount)

			if tc.wantParametersCount > 0 {
				val, ok := got.Parameters["Key1"]
				require.True(t, ok, "Parameters should contain Key1")
				assert.Equal(t, tc.wantParameterKey1Value, val)
			}
		})
	}
}

// TestGetParametersMap tests converting CloudFormation parameters to a map
func TestGetParametersMap(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		params []types.Parameter
		want   map[string]any
	}{
		"empty parameters": {
			params: []types.Parameter{},
			want:   map[string]any{},
		},
		"single parameter": {
			params: []types.Parameter{
				{ParameterKey: strPtr("Key1"), ParameterValue: strPtr("Value1")},
			},
			want: map[string]any{
				"Key1": "Value1",
			},
		},
		"multiple parameters": {
			params: []types.Parameter{
				{ParameterKey: strPtr("Key1"), ParameterValue: strPtr("Value1")},
				{ParameterKey: strPtr("Key2"), ParameterValue: strPtr("Value2")},
				{ParameterKey: strPtr("Key3"), ParameterValue: strPtr("Value3")},
			},
			want: map[string]any{
				"Key1": "Value1",
				"Key2": "Value2",
				"Key3": "Value3",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GetParametersMap(tc.params)

			opts := []cmp.Option{
				cmpopts.IgnoreUnexported(types.Parameter{}),
			}

			if diff := cmp.Diff(tc.want, *got, opts...); diff != "" {
				t.Errorf("GetParametersMap() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestReverseEvents tests the sorting interface for stack events
func TestReverseEvents(t *testing.T) {
	t.Helper()

	time1 := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	time2 := time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC)
	time3 := time.Date(2023, 1, 3, 12, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		events   ReverseEvents
		wantLen  int
		testLess func(t *testing.T, events ReverseEvents)
		testSwap func(t *testing.T, events ReverseEvents)
	}{
		"sorts events by timestamp": {
			events: ReverseEvents{
				{Timestamp: &time2},
				{Timestamp: &time1},
				{Timestamp: &time3},
			},
			wantLen: 3,
			testLess: func(t *testing.T, events ReverseEvents) {
				t.Helper()
				assert.True(t, events.Less(1, 0), "time1 < time2")
				assert.False(t, events.Less(0, 1), "time2 > time1")
			},
			testSwap: func(t *testing.T, events ReverseEvents) {
				t.Helper()
				events.Swap(0, 1)
				assert.Equal(t, &time1, events[0].Timestamp)
				assert.Equal(t, &time2, events[1].Timestamp)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Test Len
			assert.Equal(t, tc.wantLen, tc.events.Len())

			// Test Less
			if tc.testLess != nil {
				tc.testLess(t, tc.events)
			}

			// Test Swap (needs a fresh copy as it mutates)
			if tc.testSwap != nil {
				eventsCopy := make(ReverseEvents, len(tc.events))
				copy(eventsCopy, tc.events)
				tc.testSwap(t, eventsCopy)
			}
		})
	}
}

// TestCfnStack_GetEventSummaries verifies that GetEventSummaries uses pagination
// to retrieve all stack events via fetchAllStackEvents.
func TestCfnStack_GetEventSummaries(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := map[string]struct {
		stack     CfnStack
		mock      mockStackEventsClient
		wantCount int
		wantErr   bool
	}{
		"single page of events": {
			stack: CfnStack{Id: "stack-id-1", Name: "test-stack"},
			mock: newSinglePageMock(cloudformation.DescribeStackEventsOutput{
				StackEvents: []types.StackEvent{
					{LogicalResourceId: strPtr("Res1"), Timestamp: ptrTime(now)},
					{LogicalResourceId: strPtr("Res2"), Timestamp: ptrTime(now.Add(1 * time.Second))},
				},
			}, nil),
			wantCount: 2,
		},
		"multiple pages of events are all returned": {
			stack: CfnStack{Id: "stack-id-2", Name: "big-stack"},
			mock: mockStackEventsClient{
				pages: map[string]cloudformation.DescribeStackEventsOutput{
					"": {
						StackEvents: []types.StackEvent{
							{LogicalResourceId: strPtr("Page1A"), Timestamp: ptrTime(now)},
							{LogicalResourceId: strPtr("Page1B"), Timestamp: ptrTime(now.Add(1 * time.Second))},
						},
						NextToken: strPtr("token2"),
					},
					"token2": {
						StackEvents: []types.StackEvent{
							{LogicalResourceId: strPtr("Page2A"), Timestamp: ptrTime(now.Add(2 * time.Second))},
						},
						NextToken: strPtr("token3"),
					},
					"token3": {
						StackEvents: []types.StackEvent{
							{LogicalResourceId: strPtr("Page3A"), Timestamp: ptrTime(now.Add(3 * time.Second))},
							{LogicalResourceId: strPtr("Page3B"), Timestamp: ptrTime(now.Add(4 * time.Second))},
						},
					},
				},
			},
			wantCount: 5,
		},
		"empty events": {
			stack:     CfnStack{Id: "stack-id-3", Name: "empty-stack"},
			mock:      newSinglePageMock(cloudformation.DescribeStackEventsOutput{}, nil),
			wantCount: 0,
		},
		"API error": {
			stack:   CfnStack{Id: "stack-id-4", Name: "bad-stack"},
			mock:    mockStackEventsClient{err: errors.New("access denied")},
			wantErr: true,
		},
		"API error on second page": {
			stack: CfnStack{Id: "stack-id-5", Name: "mid-error-stack"},
			mock: mockStackEventsClient{
				pages: map[string]cloudformation.DescribeStackEventsOutput{
					"": {
						StackEvents: []types.StackEvent{
							{LogicalResourceId: strPtr("Page1"), Timestamp: ptrTime(now)},
						},
						NextToken: strPtr("page2"),
					},
				},
				pageErrors: map[string]error{
					"page2": errors.New("throttling"),
				},
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.stack.GetEventSummaries(tc.mock)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, got, tc.wantCount)
		})
	}
}

// TestSortStacks tests the sorting interface for CfnStack
func TestSortStacks(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		stacks   SortStacks
		wantLen  int
		testLess func(t *testing.T, stacks SortStacks)
		testSwap func(t *testing.T, stacks SortStacks)
	}{
		"sorts stacks alphabetically": {
			stacks: SortStacks{
				{Name: "zebra-stack"},
				{Name: "alpha-stack"},
				{Name: "middle-stack"},
			},
			wantLen: 3,
			testLess: func(t *testing.T, stacks SortStacks) {
				t.Helper()
				assert.True(t, stacks.Less(1, 0), "alpha < zebra")
				assert.False(t, stacks.Less(0, 1), "zebra > alpha")
				assert.True(t, stacks.Less(1, 2), "alpha < middle")
			},
			testSwap: func(t *testing.T, stacks SortStacks) {
				t.Helper()
				stacks.Swap(0, 1)
				assert.Equal(t, "alpha-stack", stacks[0].Name)
				assert.Equal(t, "zebra-stack", stacks[1].Name)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Test Len
			assert.Equal(t, tc.wantLen, tc.stacks.Len())

			// Test Less
			if tc.testLess != nil {
				tc.testLess(t, tc.stacks)
			}

			// Test Swap (needs a fresh copy as it mutates)
			if tc.testSwap != nil {
				stacksCopy := make(SortStacks, len(tc.stacks))
				copy(stacksCopy, tc.stacks)
				tc.testSwap(t, stacksCopy)
			}
		})
	}
}
