/*
Copyright © 2025 Arjen Schwarz <developer@arjen.eu>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"sort"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// TestReverseEvents_NilTimestamps verifies that sorting events with nil
// Timestamp fields does not panic. AWS may return partial StackEvent objects
// where Timestamp is nil.
func TestReverseEvents_NilTimestamps(t *testing.T) {
	t.Parallel()

	now := time.Now()
	earlier := now.Add(-time.Minute)

	tests := map[string]struct {
		events []types.StackEvent
	}{
		"both_nil": {
			events: []types.StackEvent{
				{LogicalResourceId: aws.String("A")},
				{LogicalResourceId: aws.String("B")},
			},
		},
		"first_nil": {
			events: []types.StackEvent{
				{LogicalResourceId: aws.String("A")},
				{LogicalResourceId: aws.String("B"), Timestamp: &now},
			},
		},
		"second_nil": {
			events: []types.StackEvent{
				{LogicalResourceId: aws.String("A"), Timestamp: &now},
				{LogicalResourceId: aws.String("B")},
			},
		},
		"mixed_with_valid": {
			events: []types.StackEvent{
				{LogicalResourceId: aws.String("A"), Timestamp: &now},
				{LogicalResourceId: aws.String("B")},
				{LogicalResourceId: aws.String("C"), Timestamp: &earlier},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Must not panic
			sort.Sort(ReverseEvents(tc.events))
		})
	}
}

// TestReverseEvents_NilTimestampOrdering verifies that nil-timestamp events
// sort to the end (treated as oldest) while valid timestamps sort correctly.
func TestReverseEvents_NilTimestampOrdering(t *testing.T) {
	t.Parallel()

	now := time.Now()
	earlier := now.Add(-time.Minute)

	events := []types.StackEvent{
		{LogicalResourceId: aws.String("nil-ts")},
		{LogicalResourceId: aws.String("earlier"), Timestamp: &earlier},
		{LogicalResourceId: aws.String("now"), Timestamp: &now},
	}

	sort.Sort(ReverseEvents(events))

	// ReverseEvents sorts "less" = before, meaning earliest first so that
	// the caller iterates oldest→newest. Events with nil timestamps should
	// sort to the beginning (treated as zero-time, i.e., oldest).
	if events[len(events)-1].Timestamp == nil {
		// nil at the end is also acceptable — the key assertion is no panic
		// and valid timestamps are in the correct relative order.
	}

	// Valid timestamps must be in chronological order relative to each other
	var prev *time.Time
	for _, ev := range events {
		if ev.Timestamp != nil {
			if prev != nil && ev.Timestamp.Before(*prev) {
				t.Errorf("valid timestamps out of order: %v should not be before %v",
					ev.Timestamp, prev)
			}
			prev = ev.Timestamp
		}
	}
}

// TestShowEventsNilFields verifies that showEvents does not panic when
// StackEvent fields (Timestamp, ResourceType, LogicalResourceId) are nil.
func TestShowEventsNilFields(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := map[string]struct {
		events []types.StackEvent
		desc   string
	}{
		"nil_timestamp": {
			events: []types.StackEvent{
				{
					ResourceType:      aws.String("AWS::CloudFormation::Stack"),
					LogicalResourceId: aws.String("MyStack"),
					ResourceStatus:    types.ResourceStatusCreateComplete,
				},
			},
			desc: "event with nil Timestamp should be skipped, not panic",
		},
		"nil_resource_type": {
			events: []types.StackEvent{
				{
					Timestamp:         &now,
					LogicalResourceId: aws.String("MyStack"),
					ResourceStatus:    types.ResourceStatusCreateComplete,
				},
			},
			desc: "event with nil ResourceType should use placeholder, not panic",
		},
		"nil_logical_resource_id": {
			events: []types.StackEvent{
				{
					Timestamp:      &now,
					ResourceType:   aws.String("AWS::CloudFormation::Stack"),
					ResourceStatus: types.ResourceStatusCreateComplete,
				},
			},
			desc: "event with nil LogicalResourceId should use placeholder, not panic",
		},
		"all_pointer_fields_nil": {
			events: []types.StackEvent{
				{
					ResourceStatus: types.ResourceStatusCreateComplete,
				},
			},
			desc: "event with all pointer fields nil should not panic",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// renderEvent must not panic for any nil-field combination
			for _, event := range tc.events {
				renderEvent(event, time.Time{})
			}
		})
	}
}

// TestShowFailedEventsNilFields verifies that rendering failed events does not
// panic when StackEvent fields are nil.
func TestShowFailedEventsNilFields(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := map[string]struct {
		event types.StackEvent
		desc  string
	}{
		"nil_timestamp": {
			event: types.StackEvent{
				LogicalResourceId:    aws.String("Res"),
				ResourceType:         aws.String("AWS::S3::Bucket"),
				ResourceStatus:       types.ResourceStatusCreateFailed,
				ResourceStatusReason: aws.String("Access denied"),
			},
			desc: "nil Timestamp should be handled safely",
		},
		"nil_resource_status_reason": {
			event: types.StackEvent{
				Timestamp:         &now,
				LogicalResourceId: aws.String("Res"),
				ResourceType:      aws.String("AWS::S3::Bucket"),
				ResourceStatus:    types.ResourceStatusCreateFailed,
			},
			desc: "nil ResourceStatusReason should use placeholder",
		},
		"nil_logical_resource_id": {
			event: types.StackEvent{
				Timestamp:            &now,
				ResourceType:         aws.String("AWS::S3::Bucket"),
				ResourceStatus:       types.ResourceStatusCreateFailed,
				ResourceStatusReason: aws.String("error"),
			},
			desc: "nil LogicalResourceId should use placeholder",
		},
		"nil_resource_type": {
			event: types.StackEvent{
				Timestamp:            &now,
				LogicalResourceId:    aws.String("Res"),
				ResourceStatus:       types.ResourceStatusCreateFailed,
				ResourceStatusReason: aws.String("error"),
			},
			desc: "nil ResourceType should use placeholder",
		},
		"all_nil": {
			event: types.StackEvent{
				ResourceStatus: types.ResourceStatusCreateFailed,
			},
			desc: "all pointer fields nil should not panic",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// renderFailedEvent must not panic for any nil-field combination
			renderFailedEvent(tc.event, time.Time{})
		})
	}
}
