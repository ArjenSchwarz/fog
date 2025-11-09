package lib

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// TestIsStackLevelEvent tests identification of stack-level events
func TestIsStackLevelEvent(t *testing.T) {
	tests := []struct {
		name      string
		event     types.StackEvent
		stackName string
		want      bool
	}{
		{
			name: "stack-level event",
			event: types.StackEvent{
				LogicalResourceId: aws.String("my-stack"),
				ResourceType:      aws.String("AWS::CloudFormation::Stack"),
			},
			stackName: "my-stack",
			want:      true,
		},
		{
			name: "resource-level event",
			event: types.StackEvent{
				LogicalResourceId: aws.String("MyBucket"),
				ResourceType:      aws.String("AWS::S3::Bucket"),
			},
			stackName: "my-stack",
			want:      false,
		},
		{
			name: "wrong stack name",
			event: types.StackEvent{
				LogicalResourceId: aws.String("other-stack"),
				ResourceType:      aws.String("AWS::CloudFormation::Stack"),
			},
			stackName: "my-stack",
			want:      false,
		},
		{
			name: "nil pointers",
			event: types.StackEvent{
				LogicalResourceId: nil,
				ResourceType:      nil,
			},
			stackName: "my-stack",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isStackLevelEvent(tt.event, tt.stackName)
			if got != tt.want {
				t.Errorf("isStackLevelEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsEventStart tests event start detection
func TestIsEventStart(t *testing.T) {
	tests := []struct {
		name      string
		eventName string
		want      bool
	}{
		{
			name:      "empty event name",
			eventName: "",
			want:      true,
		},
		{
			name:      "COMPLETE suffix",
			eventName: "CREATE_COMPLETE",
			want:      true,
		},
		{
			name:      "FAILED suffix",
			eventName: "UPDATE_FAILED",
			want:      true,
		},
		{
			name:      "IN_PROGRESS status",
			eventName: "CREATE_IN_PROGRESS",
			want:      false,
		},
		{
			name:      "other status",
			eventName: "REVIEW_IN_PROGRESS",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEventStart(tt.eventName)
			if got != tt.want {
				t.Errorf("isEventStart() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDetermineStackEventType tests stack event type determination
func TestDetermineStackEventType(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{
			name:   "CREATE_IN_PROGRESS",
			status: "CREATE_IN_PROGRESS",
			want:   "Create",
		},
		{
			name:   "REVIEW_IN_PROGRESS",
			status: "REVIEW_IN_PROGRESS",
			want:   "Create",
		},
		{
			name:   "UPDATE_IN_PROGRESS",
			status: "UPDATE_IN_PROGRESS",
			want:   "Update",
		},
		{
			name:   "DELETE_IN_PROGRESS",
			status: "DELETE_IN_PROGRESS",
			want:   "Delete",
		},
		{
			name:   "IMPORT_IN_PROGRESS",
			status: "IMPORT_IN_PROGRESS",
			want:   "Import",
		},
		{
			name:   "unknown status",
			status: "UNKNOWN_STATUS",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineStackEventType(tt.status)
			if got != tt.want {
				t.Errorf("determineStackEventType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDetermineResourceEventType tests resource event type determination
func TestDetermineResourceEventType(t *testing.T) {
	tests := []struct {
		name             string
		status           string
		resourceName     string
		wantType         string
		wantExpectedEnd  string
	}{
		{
			name:            "CREATE status",
			status:          "CREATE_IN_PROGRESS",
			resourceName:    "resource-1",
			wantType:        "Add",
			wantExpectedEnd: "CREATE_COMPLETE",
		},
		{
			name:            "UPDATE status",
			status:          "UPDATE_IN_PROGRESS",
			resourceName:    "resource-1",
			wantType:        "Modify",
			wantExpectedEnd: "UPDATE_COMPLETE",
		},
		{
			name:            "DELETE normal",
			status:          "DELETE_IN_PROGRESS",
			resourceName:    "resource-1",
			wantType:        "Remove",
			wantExpectedEnd: "DELETE_COMPLETE",
		},
		{
			name:            "DELETE replacement",
			status:          "DELETE_IN_PROGRESS",
			resourceName:    "resource-1-replacement",
			wantType:        "Cleanup",
			wantExpectedEnd: "DELETE_COMPLETE",
		},
		{
			name:            "DELETE cleanup",
			status:          "DELETE_IN_PROGRESS",
			resourceName:    "resource-1-cleanup",
			wantType:        "Cleanup",
			wantExpectedEnd: "DELETE_COMPLETE",
		},
		{
			name:            "unknown status",
			status:          "UNKNOWN",
			resourceName:    "resource-1",
			wantType:        "",
			wantExpectedEnd: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotEnd := determineResourceEventType(tt.status, tt.resourceName)
			if gotType != tt.wantType {
				t.Errorf("determineResourceEventType() type = %v, want %v", gotType, tt.wantType)
			}
			if gotEnd != tt.wantExpectedEnd {
				t.Errorf("determineResourceEventType() expectedEnd = %v, want %v", gotEnd, tt.wantExpectedEnd)
			}
		})
	}
}

// TestUpdateResourceId tests resource ID update logic
func TestUpdateResourceId(t *testing.T) {
	tests := []struct {
		name      string
		currentId string
		newId     string
		want      string
	}{
		{
			name:      "both empty",
			currentId: "",
			newId:     "",
			want:      "",
		},
		{
			name:      "current empty, new provided",
			currentId: "",
			newId:     "resource-123",
			want:      "resource-123",
		},
		{
			name:      "same ID",
			currentId: "resource-123",
			newId:     "resource-123",
			want:      "resource-123",
		},
		{
			name:      "different IDs - replacement",
			currentId: "resource-old",
			newId:     "resource-new",
			want:      "resource-old => resource-new",
		},
		{
			name:      "current contains new",
			currentId: "resource-old => resource-123",
			newId:     "resource-123",
			want:      "resource-old => resource-123",
		},
		{
			name:      "new empty",
			currentId: "resource-123",
			newId:     "",
			want:      "resource-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateResourceId(tt.currentId, tt.newId)
			if got != tt.want {
				t.Errorf("updateResourceId() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCreateNewStackEvent tests creation of new stack events
func TestCreateNewStackEvent(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		event     types.StackEvent
		wantType  string
		wantDate  time.Time
	}{
		{
			name: "CREATE_IN_PROGRESS with timestamp",
			event: types.StackEvent{
				Timestamp:      &now,
				ResourceStatus: types.StackStatusCreateInProgress,
			},
			wantType: "Create",
			wantDate: now,
		},
		{
			name: "UPDATE_IN_PROGRESS with timestamp",
			event: types.StackEvent{
				Timestamp:      &now,
				ResourceStatus: types.StackStatusUpdateInProgress,
			},
			wantType: "Update",
			wantDate: now,
		},
		{
			name: "nil timestamp",
			event: types.StackEvent{
				Timestamp:      nil,
				ResourceStatus: types.StackStatusCreateInProgress,
			},
			wantType: "Create",
			wantDate: time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createNewStackEvent(tt.event)
			if got.Type != tt.wantType {
				t.Errorf("createNewStackEvent().Type = %v, want %v", got.Type, tt.wantType)
			}
			if !got.StartDate.Equal(tt.wantDate) {
				t.Errorf("createNewStackEvent().StartDate = %v, want %v", got.StartDate, tt.wantDate)
			}
			if got.Milestones == nil {
				t.Error("createNewStackEvent().Milestones = nil, want non-nil map")
			}
		})
	}
}

// TestCreateNewResourceEvent tests creation of new resource events
func TestCreateNewResourceEvent(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name         string
		event        types.StackEvent
		stackName    string
		resourceName string
		wantType     string
	}{
		{
			name: "CREATE event with timestamp",
			event: types.StackEvent{
				Timestamp:         &now,
				ResourceStatus:    types.ResourceStatusCreateInProgress,
				ResourceType:      aws.String("AWS::S3::Bucket"),
				PhysicalResourceId: aws.String("my-bucket"),
				LogicalResourceId: aws.String("MyBucket"),
			},
			stackName:    "my-stack",
			resourceName: "resource-1",
			wantType:     "Add",
		},
		{
			name: "nil timestamp",
			event: types.StackEvent{
				Timestamp:         nil,
				ResourceStatus:    types.ResourceStatusCreateInProgress,
				ResourceType:      aws.String("AWS::S3::Bucket"),
				PhysicalResourceId: nil,
				LogicalResourceId: aws.String("MyBucket"),
			},
			stackName:    "my-stack",
			resourceName: "resource-1",
			wantType:     "Add",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createNewResourceEvent(tt.event, tt.stackName, tt.resourceName)
			if got.EventType != tt.wantType {
				t.Errorf("createNewResourceEvent().EventType = %v, want %v", got.EventType, tt.wantType)
			}
			if got.Resource.StackName != tt.stackName {
				t.Errorf("createNewResourceEvent().Resource.StackName = %v, want %v", got.Resource.StackName, tt.stackName)
			}
			if tt.event.Timestamp != nil && !got.StartDate.Equal(*tt.event.Timestamp) {
				t.Errorf("createNewResourceEvent().StartDate = %v, want %v", got.StartDate, *tt.event.Timestamp)
			}
			if tt.event.Timestamp == nil && !got.StartDate.IsZero() {
				t.Errorf("createNewResourceEvent().StartDate = %v, want zero time", got.StartDate)
			}
			if len(got.RawInfo) != 1 {
				t.Errorf("createNewResourceEvent().RawInfo length = %d, want 1", len(got.RawInfo))
			}
		})
	}
}

// TestUpdateExistingResourceEvent tests updating existing resource events
func TestUpdateExistingResourceEvent(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Minute)

	initialEvent := ResourceEvent{
		StartDate:   earlier,
		StartStatus: "CREATE_IN_PROGRESS",
		EndDate:     earlier,
		EndStatus:   "CREATE_IN_PROGRESS",
		Resource: CfnResource{
			ResourceID: "resource-old",
		},
		RawInfo: []types.StackEvent{},
	}

	tests := []struct {
		name              string
		initial           ResourceEvent
		event             types.StackEvent
		resourceName      string
		expectEndDate     time.Time
		expectStatus      string
		expectFinished    bool
		expectFailed      bool
		expectResourceId  string
		expectRawInfoLen  int
	}{
		{
			name:    "COMPLETE event",
			initial: initialEvent,
			event: types.StackEvent{
				Timestamp:           &now,
				ResourceStatus:      types.ResourceStatusCreateComplete,
				PhysicalResourceId:  aws.String("resource-new"),
				ResourceStatusReason: nil,
			},
			resourceName:     "test-resource",
			expectEndDate:    now,
			expectStatus:     "CREATE_COMPLETE",
			expectFinished:   true,
			expectFailed:     false,
			expectResourceId: "resource-old => resource-new",
			expectRawInfoLen: 1,
		},
		{
			name:    "FAILED event",
			initial: initialEvent,
			event: types.StackEvent{
				Timestamp:           &now,
				ResourceStatus:      types.ResourceStatusCreateFailed,
				PhysicalResourceId:  aws.String("resource-new"),
				ResourceStatusReason: aws.String("Test failure reason"),
			},
			resourceName:     "test-resource",
			expectEndDate:    now,
			expectStatus:     "CREATE_FAILED",
			expectFinished:   false,
			expectFailed:     true,
			expectResourceId: "resource-old => resource-new",
			expectRawInfoLen: 1,
		},
		{
			name:    "nil timestamp",
			initial: initialEvent,
			event: types.StackEvent{
				Timestamp:          nil,
				ResourceStatus:     types.ResourceStatusCreateComplete,
				PhysicalResourceId: aws.String("resource-new"),
			},
			resourceName:     "test-resource",
			expectEndDate:    earlier,
			expectStatus:     "CREATE_COMPLETE",
			expectFinished:   true,
			expectFailed:     false,
			expectResourceId: "resource-old => resource-new",
			expectRawInfoLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finishedEvents := []string{}
			failedEvents := []string{}

			got, gotFinished, gotFailed := updateExistingResourceEvent(
				tt.initial, tt.event, tt.resourceName, finishedEvents, failedEvents)

			if !got.EndDate.Equal(tt.expectEndDate) {
				t.Errorf("updateExistingResourceEvent().EndDate = %v, want %v", got.EndDate, tt.expectEndDate)
			}
			if got.EndStatus != tt.expectStatus {
				t.Errorf("updateExistingResourceEvent().EndStatus = %v, want %v", got.EndStatus, tt.expectStatus)
			}
			if tt.expectFinished && len(gotFinished) == 0 {
				t.Error("expected finished events to be updated")
			}
			if tt.expectFailed && len(gotFailed) == 0 {
				t.Error("expected failed events to be updated")
			}
			if got.Resource.ResourceID != tt.expectResourceId {
				t.Errorf("updateExistingResourceEvent().Resource.ResourceID = %v, want %v",
					got.Resource.ResourceID, tt.expectResourceId)
			}
			if len(got.RawInfo) != tt.expectRawInfoLen {
				t.Errorf("updateExistingResourceEvent().RawInfo length = %d, want %d",
					len(got.RawInfo), tt.expectRawInfoLen)
			}
		})
	}
}

// TestGenerateResourceEventName tests resource event name generation
func TestGenerateResourceEventName(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		event          types.StackEvent
		stackEvent     StackEvent
		finishedEvents []string
		failedEvents   []string
		wantSuffix     string
	}{
		{
			name: "basic name",
			event: types.StackEvent{
				ResourceType:      aws.String("AWS::S3::Bucket"),
				LogicalResourceId: aws.String("MyBucket"),
			},
			stackEvent: StackEvent{
				StartDate: baseTime,
			},
			finishedEvents: []string{},
			failedEvents:   []string{},
			wantSuffix:     "",
		},
		{
			name: "replacement suffix",
			event: types.StackEvent{
				ResourceType:      aws.String("AWS::S3::Bucket"),
				LogicalResourceId: aws.String("MyBucket"),
			},
			stackEvent: StackEvent{
				StartDate: baseTime,
			},
			finishedEvents: []string{"aws-s3-bucket-MyBucket-2024-01-01T12:00:00Z"},
			failedEvents:   []string{},
			wantSuffix:     "-replacement",
		},
		{
			name: "cleanup suffix",
			event: types.StackEvent{
				ResourceType:      aws.String("AWS::S3::Bucket"),
				LogicalResourceId: aws.String("MyBucket"),
			},
			stackEvent: StackEvent{
				StartDate: baseTime,
			},
			finishedEvents: []string{},
			failedEvents:   []string{"aws-s3-bucket-MyBucket-2024-01-01T12:00:00Z"},
			wantSuffix:     "-cleanup",
		},
		{
			name: "both suffixes - replacement takes precedence",
			event: types.StackEvent{
				ResourceType:      aws.String("AWS::S3::Bucket"),
				LogicalResourceId: aws.String("MyBucket"),
			},
			stackEvent: StackEvent{
				StartDate: baseTime,
			},
			finishedEvents: []string{"aws-s3-bucket-MyBucket-2024-01-01T12:00:00Z"},
			failedEvents:   []string{"aws-s3-bucket-MyBucket-2024-01-01T12:00:00Z"},
			wantSuffix:     "-replacement",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateResourceEventName(tt.event, tt.stackEvent, tt.finishedEvents, tt.failedEvents)
			if tt.wantSuffix != "" {
				if !endsWith(got, tt.wantSuffix) {
					t.Errorf("generateResourceEventName() = %v, want suffix %v", got, tt.wantSuffix)
				}
			}
			// Verify the base name contains the slugified resource type
			if !contains(got, "aws-s3-bucket") {
				t.Errorf("generateResourceEventName() = %v, want to contain 'aws-s3-bucket'", got)
			}
		})
	}
}

// Helper functions for string operations
func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && hasSubstring(s, substr))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
