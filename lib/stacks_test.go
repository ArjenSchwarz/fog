package lib

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// Mocks implementing the CloudFormation interfaces used by the stack helpers

type mockStackEventsClient struct {
	output cloudformation.DescribeStackEventsOutput
	err    error
}

func (m mockStackEventsClient) DescribeStackEvents(ctx context.Context, params *cloudformation.DescribeStackEventsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackEventsOutput, error) {
	return &m.output, m.err
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

func TestDeployInfo_GetCleanedStackName(t *testing.T) {
	type fields struct {
		Changeset            *ChangesetInfo
		ChangesetName        string
		IsDryRun             bool
		IsNew                bool
		Parameters           []types.Parameter
		PrechecksFailed      bool
		RawStack             *types.Stack
		StackArn             string
		StackName            string
		Tags                 []types.Tag
		Template             string
		TemplateLocalPath    string
		TemplateName         string
		TemplateRelativePath string
		TemplateUrl          string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"No arn in the stackname", fields{StackName: "test-stack"}, "test-stack"},
		{"Arn in the stackname", fields{StackName: "arn:aws:cloudformation:ap-southeast-2:12345678901:stack/test-stack/5f584530-013c-11ee-9c69-0a254d5985de"}, "test-stack"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployment := &DeployInfo{
				Changeset:            tt.fields.Changeset,
				ChangesetName:        tt.fields.ChangesetName,
				IsDryRun:             tt.fields.IsDryRun,
				IsNew:                tt.fields.IsNew,
				Parameters:           tt.fields.Parameters,
				PrechecksFailed:      tt.fields.PrechecksFailed,
				RawStack:             tt.fields.RawStack,
				StackArn:             tt.fields.StackArn,
				StackName:            tt.fields.StackName,
				Tags:                 tt.fields.Tags,
				Template:             tt.fields.Template,
				TemplateLocalPath:    tt.fields.TemplateLocalPath,
				TemplateName:         tt.fields.TemplateName,
				TemplateRelativePath: tt.fields.TemplateRelativePath,
				TemplateUrl:          tt.fields.TemplateUrl,
			}
			if got := deployment.GetCleanedStackName(); got != tt.want {
				t.Errorf("DeployInfo.GetCleanedStackName() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDeployInfo_GetExecutionTimes verifies that resource event timestamps are grouped per resource and status.
func TestDeployInfo_GetExecutionTimes(t *testing.T) {
	now := time.Now().UTC()
	deployment := &DeployInfo{
		Changeset: &ChangesetInfo{CreationTime: now},
		StackName: "test-stack",
	}

	events := []types.StackEvent{
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
	}

	mockSvc := mockStackEventsClient{output: cloudformation.DescribeStackEventsOutput{StackEvents: events}}

	got, err := deployment.GetExecutionTimes(mockSvc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]map[string]time.Time{
		"AWS  S3  Bucket (Bucket)": {
			string(types.ResourceStatusCreateInProgress): now.Add(1 * time.Minute),
			string(types.ResourceStatusCreateComplete):   now.Add(2 * time.Minute),
		},
		"AWS  IAM  Role (Role)": {
			string(types.ResourceStatusCreateInProgress): now.Add(3 * time.Minute),
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("DeployInfo.GetExecutionTimes() = %v, want %v", got, want)
	}
}

// Test_stringInSlice checks membership detection in a slice.
func Test_stringInSlice(t *testing.T) {
	if !stringInSlice("a", []string{"b", "a"}) {
		t.Errorf("expected true for existing string")
	}
	if stringInSlice("c", []string{"a", "b"}) {
		t.Errorf("expected false for missing string")
	}
}

// TestResourceEvent_GetDuration ensures the duration uses StartDate and EndDate.
func TestResourceEvent_GetDuration(t *testing.T) {
	start := time.Now()
	end := start.Add(2 * time.Minute)
	re := ResourceEvent{StartDate: start, EndDate: end}
	if dur := re.GetDuration(); dur != 2*time.Minute {
		t.Errorf("GetDuration = %v, want %v", dur, 2*time.Minute)
	}
}

// TestStackEvent_GetDuration ensures the duration uses StartDate and EndDate.
func TestStackEvent_GetDuration(t *testing.T) {
	start := time.Now()
	end := start.Add(3 * time.Minute)
	se := StackEvent{StartDate: start, EndDate: end}
	if dur := se.GetDuration(); dur != 3*time.Minute {
		t.Errorf("GetDuration = %v, want %v", dur, 3*time.Minute)
	}
}

// TestDeployInfo_StatusChecks verifies readiness, ongoing state, and new stack detection.
func TestDeployInfo_StatusChecks(t *testing.T) {
	// IsReadyForUpdate
	readyClient := mockDescribeStacksClient{output: cloudformation.DescribeStacksOutput{Stacks: []types.Stack{{
		StackName:   strPtr("stack"),
		StackId:     strPtr("stack"),
		StackStatus: types.StackStatusCreateComplete,
	}}}}
	dep := DeployInfo{StackName: "stack", StackArn: "stack"}
	if ready, status := dep.IsReadyForUpdate(readyClient); !ready || status != string(types.StackStatusCreateComplete) {
		t.Errorf("IsReadyForUpdate unexpected result: %v %v", ready, status)
	}

	notReadyClient := mockDescribeStacksClient{output: cloudformation.DescribeStacksOutput{Stacks: []types.Stack{{
		StackName:   strPtr("stack"),
		StackId:     strPtr("stack"),
		StackStatus: types.StackStatusUpdateInProgress,
	}}}}
	if ready, _ := dep.IsReadyForUpdate(notReadyClient); ready {
		t.Errorf("IsReadyForUpdate should be false for in progress status")
	}

	// IsOngoing
	if !dep.IsOngoing(notReadyClient) {
		t.Errorf("IsOngoing expected true for in progress status")
	}
	if dep.IsOngoing(readyClient) {
		t.Errorf("IsOngoing expected false for completed status")
	}

	// IsNewStack
	errorClient := mockDescribeStacksClient{err: errors.New("not found")}
	if !dep.IsNewStack(errorClient) {
		t.Errorf("IsNewStack should be true when stack does not exist")
	}

	reviewClient := mockDescribeStacksClient{output: cloudformation.DescribeStacksOutput{Stacks: []types.Stack{{
		StackName:   strPtr("stack"),
		StackId:     strPtr("stack"),
		StackStatus: types.StackStatusReviewInProgress,
	}}}}
	if !dep.IsNewStack(reviewClient) {
		t.Errorf("IsNewStack should be true for review in progress status")
	}

	if dep.IsNewStack(readyClient) {
		t.Errorf("IsNewStack should be false for existing completed stack")
	}
}

// TestDeployInfo_ChangesetType verifies that the correct changeset type is
// returned based on whether the stack is new or existing.
func TestDeployInfo_ChangesetType(t *testing.T) {
	tests := []struct {
		name  string
		isNew bool
		want  types.ChangeSetType
	}{
		{
			name:  "New stack",
			isNew: true,
			want:  types.ChangeSetTypeCreate,
		},
		{
			name:  "Existing stack",
			isNew: false,
			want:  types.ChangeSetTypeUpdate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep := &DeployInfo{IsNew: tt.isNew}
			if got := dep.ChangesetType(); got != tt.want {
				t.Errorf("ChangesetType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetStack verifies that GetStack correctly retrieves stack information
// using the mocked CloudFormation client.
func TestGetStack(t *testing.T) {
	stackName := "test-stack"
	expectedStack := types.Stack{
		StackName: strPtr(stackName),
		StackId:   strPtr("arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123"),
	}

	tests := []struct {
		name    string
		client  mockDescribeStacksClient
		want    types.Stack
		wantErr bool
	}{
		{
			name: "Success",
			client: mockDescribeStacksClient{
				output: cloudformation.DescribeStacksOutput{Stacks: []types.Stack{expectedStack}},
			},
			want:    expectedStack,
			wantErr: false,
		},
		{
			name: "Error",
			client: mockDescribeStacksClient{
				err: errors.New("stack not found"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetStack(&stackName, tt.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStack() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetStack() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseParameterString verifies that JSON parameter strings are correctly
// parsed into CloudFormation parameter structures.
func TestParseParameterString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []types.Parameter
		wantErr bool
	}{
		{
			name:  "Valid parameters",
			input: `[{"ParameterKey":"Key1","ParameterValue":"Value1"},{"ParameterKey":"Key2","ParameterValue":"Value2"}]`,
			want: []types.Parameter{
				{ParameterKey: strPtr("Key1"), ParameterValue: strPtr("Value1")},
				{ParameterKey: strPtr("Key2"), ParameterValue: strPtr("Value2")},
			},
			wantErr: false,
		},
		{
			name:    "Empty array",
			input:   `[]`,
			want:    []types.Parameter{},
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseParameterString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseParameterString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseParameterString() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseTagString verifies that JSON tag strings are correctly parsed
// into CloudFormation tag structures.
func TestParseTagString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []types.Tag
		wantErr bool
	}{
		{
			name:  "Valid tags",
			input: `[{"Key":"Environment","Value":"Production"},{"Key":"Owner","Value":"TeamA"}]`,
			want: []types.Tag{
				{Key: strPtr("Environment"), Value: strPtr("Production")},
				{Key: strPtr("Owner"), Value: strPtr("TeamA")},
			},
			wantErr: false,
		},
		{
			name:    "Empty array",
			input:   `[]`,
			want:    []types.Tag{},
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTagString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTagString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseTagString() = %v, want %v", got, tt.want)
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

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "Valid JSON",
			input:   jsonInput,
			wantErr: false,
		},
		{
			name:    "Valid YAML",
			input:   yamlInput,
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			input:   `{invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDeploymentFile(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDeploymentFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.TemplateFilePath != "templates/test-stack.yaml" {
					t.Errorf("ParseDeploymentFile() TemplateFilePath = %v, want templates/test-stack.yaml", got.TemplateFilePath)
				}
				if len(got.Parameters) != 1 {
					t.Errorf("ParseDeploymentFile() Parameters length = %v, want 1", len(got.Parameters))
				}
				if val, ok := got.Parameters["Key1"]; !ok || val != "Value1" {
					t.Errorf("ParseDeploymentFile() Parameters[Key1] = %v, want Value1", val)
				}
			}
		})
	}
}

// TestGetParametersMap tests converting CloudFormation parameters to a map
func TestGetParametersMap(t *testing.T) {
	tests := []struct {
		name   string
		params []types.Parameter
		want   map[string]interface{}
	}{
		{
			name:   "empty parameters",
			params: []types.Parameter{},
			want:   map[string]interface{}{},
		},
		{
			name: "single parameter",
			params: []types.Parameter{
				{ParameterKey: strPtr("Key1"), ParameterValue: strPtr("Value1")},
			},
			want: map[string]interface{}{
				"Key1": "Value1",
			},
		},
		{
			name: "multiple parameters",
			params: []types.Parameter{
				{ParameterKey: strPtr("Key1"), ParameterValue: strPtr("Value1")},
				{ParameterKey: strPtr("Key2"), ParameterValue: strPtr("Value2")},
				{ParameterKey: strPtr("Key3"), ParameterValue: strPtr("Value3")},
			},
			want: map[string]interface{}{
				"Key1": "Value1",
				"Key2": "Value2",
				"Key3": "Value3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetParametersMap(tt.params)
			if !reflect.DeepEqual(*got, tt.want) {
				t.Errorf("GetParametersMap() = %v, want %v", *got, tt.want)
			}
		})
	}
}

// TestReverseEvents tests the sorting interface for stack events
func TestReverseEvents(t *testing.T) {
	time1 := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	time2 := time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC)
	time3 := time.Date(2023, 1, 3, 12, 0, 0, 0, time.UTC)

	events := ReverseEvents{
		{Timestamp: &time2},
		{Timestamp: &time1},
		{Timestamp: &time3},
	}

	// Test Len
	if events.Len() != 3 {
		t.Errorf("ReverseEvents.Len() = %d, want 3", events.Len())
	}

	// Test Less - should sort by timestamp (earlier is "less")
	if !events.Less(1, 0) { // time1 < time2
		t.Errorf("ReverseEvents.Less(1, 0) should be true (time1 < time2)")
	}
	if events.Less(0, 1) { // time2 > time1
		t.Errorf("ReverseEvents.Less(0, 1) should be false (time2 > time1)")
	}

	// Test Swap
	events.Swap(0, 1)
	if events[0].Timestamp != &time1 || events[1].Timestamp != &time2 {
		t.Errorf("ReverseEvents.Swap() did not swap correctly")
	}
}

// TestSortStacks tests the sorting interface for CfnStack
func TestSortStacks(t *testing.T) {
	stacks := SortStacks{
		{Name: "zebra-stack"},
		{Name: "alpha-stack"},
		{Name: "middle-stack"},
	}

	// Test Len
	if stacks.Len() != 3 {
		t.Errorf("SortStacks.Len() = %d, want 3", stacks.Len())
	}

	// Test Less - should sort alphabetically
	if !stacks.Less(1, 0) { // "alpha" < "zebra"
		t.Errorf("SortStacks.Less(1, 0) should be true (alpha < zebra)")
	}
	if stacks.Less(0, 1) { // "zebra" > "alpha"
		t.Errorf("SortStacks.Less(0, 1) should be false (zebra > alpha)")
	}
	if !stacks.Less(1, 2) { // "alpha" < "middle"
		t.Errorf("SortStacks.Less(1, 2) should be true (alpha < middle)")
	}

	// Test Swap
	stacks.Swap(0, 1)
	if stacks[0].Name != "alpha-stack" || stacks[1].Name != "zebra-stack" {
		t.Errorf("SortStacks.Swap() did not swap correctly")
	}
}

