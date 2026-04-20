package lib

import (
	"sort"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// TestReverseEvents_NilTimestamps verifies that sorting a slice of StackEvents
// that contain nil Timestamp pointers does not panic (regression test for T-799).
// Before the fix, ReverseEvents.Less dereferenced the Timestamp pointers
// directly, causing a nil pointer dereference.
func TestReverseEvents_NilTimestamps(t *testing.T) {
	t.Parallel()

	t1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		events ReverseEvents
	}{
		"both timestamps nil": {
			events: ReverseEvents{
				{Timestamp: nil},
				{Timestamp: nil},
			},
		},
		"first timestamp nil": {
			events: ReverseEvents{
				{Timestamp: nil},
				{Timestamp: &t1},
			},
		},
		"second timestamp nil": {
			events: ReverseEvents{
				{Timestamp: &t1},
				{Timestamp: nil},
			},
		},
		"mixed nil and valid": {
			events: ReverseEvents{
				{Timestamp: &t2},
				{Timestamp: nil},
				{Timestamp: &t1},
				{Timestamp: nil},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("sort.Sort panicked on nil Timestamp: %v", r)
				}
			}()
			sort.Sort(tc.events)
		})
	}
}

// TestGenerateResourceEventName_NilFields verifies that generateResourceEventName
// does not panic when the StackEvent has nil ResourceType or LogicalResourceId
// pointers (regression test for T-799).
func TestGenerateResourceEventName_NilFields(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		event types.StackEvent
	}{
		"nil ResourceType": {
			event: types.StackEvent{
				ResourceType:      nil,
				LogicalResourceId: aws.String("MyBucket"),
			},
		},
		"nil LogicalResourceId": {
			event: types.StackEvent{
				ResourceType:      aws.String("AWS::S3::Bucket"),
				LogicalResourceId: nil,
			},
		},
		"both nil": {
			event: types.StackEvent{
				ResourceType:      nil,
				LogicalResourceId: nil,
			},
		},
	}

	stackEvent := StackEvent{StartDate: baseTime}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("generateResourceEventName panicked on nil field: %v", r)
				}
			}()
			got := generateResourceEventName(tc.event, stackEvent, nil, nil)
			if got == "" {
				t.Errorf("generateResourceEventName returned empty string; want a sensible fallback name")
			}
		})
	}
}

// TestProcessStackEvents_NilFields verifies that processStackEvents does not
// panic when events in the slice contain nil pointer fields (regression test
// for T-799). This covers the end-to-end path used by GetEvents in the
// report command.
func TestProcessStackEvents_NilFields(t *testing.T) {
	t.Parallel()

	t1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	events := []types.StackEvent{
		{
			// stack-level event with valid fields to start the event group
			LogicalResourceId: aws.String("my-stack"),
			ResourceType:      aws.String("AWS::CloudFormation::Stack"),
			ResourceStatus:    types.ResourceStatusCreateInProgress,
			Timestamp:         &t1,
		},
		{
			// resource event with nil ResourceType and LogicalResourceId
			LogicalResourceId: nil,
			ResourceType:      nil,
			ResourceStatus:    types.ResourceStatusCreateInProgress,
			Timestamp:         nil,
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("processStackEvents panicked on nil field: %v", r)
		}
	}()
	_ = processStackEvents(events, "my-stack")
}
