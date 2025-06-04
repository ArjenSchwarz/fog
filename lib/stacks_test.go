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
