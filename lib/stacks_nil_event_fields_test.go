package lib

import (
	"sort"
	"strings"
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
		event           types.StackEvent
		wantContainsRT  string
		wantContainsLID string
	}{
		"nil ResourceType": {
			event: types.StackEvent{
				ResourceType:      nil,
				LogicalResourceId: aws.String("MyBucket"),
			},
			wantContainsLID: "MyBucket",
		},
		"nil LogicalResourceId": {
			event: types.StackEvent{
				ResourceType:      aws.String("AWS::S3::Bucket"),
				LogicalResourceId: nil,
			},
			// ResourceType runs through slug.Make (lowercased, "::" -> "-").
			wantContainsRT: "aws-s3-bucket",
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
			// The fallback format is "<ResourceType>-<LogicalResourceId>-<StartDate RFC3339>".
			// Missing pointer fields become empty strings, so the timestamp suffix
			// must always be present to make the name unique per event group.
			wantSuffix := baseTime.Format(time.RFC3339)
			if !strings.HasSuffix(got, wantSuffix) {
				t.Errorf("generateResourceEventName = %q; want suffix %q", got, wantSuffix)
			}
			if tc.wantContainsRT != "" && !strings.Contains(got, tc.wantContainsRT) {
				t.Errorf("generateResourceEventName = %q; want to contain ResourceType %q", got, tc.wantContainsRT)
			}
			if tc.wantContainsLID != "" && !strings.Contains(got, tc.wantContainsLID) {
				t.Errorf("generateResourceEventName = %q; want to contain LogicalResourceId %q", got, tc.wantContainsLID)
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
	t2 := time.Date(2024, 1, 1, 12, 5, 0, 0, time.UTC)

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
		{
			// stack-level completion event to finalize the group so we can
			// assert the group was produced without dropping events.
			LogicalResourceId: aws.String("my-stack"),
			ResourceType:      aws.String("AWS::CloudFormation::Stack"),
			ResourceStatus:    types.ResourceStatusCreateComplete,
			Timestamp:         &t2,
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("processStackEvents panicked on nil field: %v", r)
		}
	}()
	result := processStackEvents(events, "my-stack")
	// The sequence contains a start + completion pair, so the result must
	// contain at least one finalized event group. Without this check the test
	// would only guard against a panic, not against the function silently
	// dropping all events.
	if len(result) == 0 {
		t.Errorf("processStackEvents returned empty result; want at least one finalized event group")
	}
}
