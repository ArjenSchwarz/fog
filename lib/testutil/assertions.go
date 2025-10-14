package testutil

import (
	"slices"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// AssertError checks that an error occurred and optionally contains a specific message
func AssertError(t *testing.T, err error, expectedMsg string) {
	t.Helper()

	if err == nil {
		t.Fatal("Expected error but got nil")
	}

	if expectedMsg != "" && !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Error message mismatch\nGot: %v\nExpected to contain: %s", err, expectedMsg)
	}
}

// AssertNoError checks that no error occurred
func AssertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}
}

// AssertAWSError checks that an error contains the expected AWS error code or message
func AssertAWSError(t *testing.T, err error, expectedCode string) {
	t.Helper()

	if err == nil {
		t.Fatal("Expected AWS error but got nil")
	}

	// AWS SDK v2 errors don't have ErrorCode() method like v1
	// We'll check the error message instead
	if expectedCode != "" && !strings.Contains(err.Error(), expectedCode) {
		t.Errorf("AWS error code mismatch\nGot: %v\nExpected code: %s", err, expectedCode)
	}
}

// AssertStackStatus checks that a stack has the expected status
func AssertStackStatus(t *testing.T, stack *types.Stack, expectedStatus types.StackStatus) {
	t.Helper()

	if stack == nil {
		t.Fatal("Stack is nil")
	}

	if stack.StackStatus != expectedStatus {
		t.Errorf("Stack status mismatch\nGot: %v\nExpected: %v", stack.StackStatus, expectedStatus)
	}
}

// AssertStackExists checks that a stack exists in the list
func AssertStackExists(t *testing.T, stacks []types.Stack, stackName string) {
	t.Helper()

	for _, stack := range stacks {
		if stack.StackName != nil && *stack.StackName == stackName {
			return
		}
	}

	t.Errorf("Stack %s not found in list", stackName)
}

// AssertStackNotExists checks that a stack does not exist in the list
func AssertStackNotExists(t *testing.T, stacks []types.Stack, stackName string) {
	t.Helper()

	for _, stack := range stacks {
		if stack.StackName != nil && *stack.StackName == stackName {
			t.Errorf("Stack %s should not exist but was found", stackName)
			return
		}
	}
}

// AssertStackParameter checks that a stack has a parameter with the expected value
func AssertStackParameter(t *testing.T, stack *types.Stack, key, expectedValue string) {
	t.Helper()

	if stack == nil {
		t.Fatal("Stack is nil")
	}

	for _, param := range stack.Parameters {
		if param.ParameterKey != nil && *param.ParameterKey == key {
			if param.ParameterValue != nil && *param.ParameterValue == expectedValue {
				return
			}
			t.Errorf("Parameter %s value mismatch\nGot: %v\nExpected: %s",
				key,
				*param.ParameterValue,
				expectedValue)
			return
		}
	}

	t.Errorf("Parameter %s not found in stack", key)
}

// AssertStackOutput checks that a stack has an output with the expected value
func AssertStackOutput(t *testing.T, stack *types.Stack, key, expectedValue string) {
	t.Helper()

	if stack == nil {
		t.Fatal("Stack is nil")
	}

	for _, output := range stack.Outputs {
		if output.OutputKey != nil && *output.OutputKey == key {
			if output.OutputValue != nil && *output.OutputValue == expectedValue {
				return
			}
			t.Errorf("Output %s value mismatch\nGot: %v\nExpected: %s",
				key,
				*output.OutputValue,
				expectedValue)
			return
		}
	}

	t.Errorf("Output %s not found in stack", key)
}

// AssertStackTag checks that a stack has a tag with the expected value
func AssertStackTag(t *testing.T, stack *types.Stack, key, expectedValue string) {
	t.Helper()

	if stack == nil {
		t.Fatal("Stack is nil")
	}

	for _, tag := range stack.Tags {
		if tag.Key != nil && *tag.Key == key {
			if tag.Value != nil && *tag.Value == expectedValue {
				return
			}
			t.Errorf("Tag %s value mismatch\nGot: %v\nExpected: %s",
				key,
				*tag.Value,
				expectedValue)
			return
		}
	}

	t.Errorf("Tag %s not found in stack", key)
}

// AssertChangesetStatus checks that a changeset has the expected status
func AssertChangesetStatus(t *testing.T, changeset *cloudformation.DescribeChangeSetOutput, expectedStatus types.ChangeSetStatus) {
	t.Helper()

	if changeset == nil {
		t.Fatal("Changeset is nil")
	}

	if changeset.Status != expectedStatus {
		t.Errorf("Changeset status mismatch\nGot: %v\nExpected: %v", changeset.Status, expectedStatus)
	}
}

// AssertChangesetHasChanges checks that a changeset contains changes
func AssertChangesetHasChanges(t *testing.T, changeset *cloudformation.DescribeChangeSetOutput) {
	t.Helper()

	if changeset == nil {
		t.Fatal("Changeset is nil")
	}

	if len(changeset.Changes) == 0 {
		t.Error("Expected changeset to have changes but found none")
	}
}

// AssertChangesetNoChanges checks that a changeset contains no changes
func AssertChangesetNoChanges(t *testing.T, changeset *cloudformation.DescribeChangeSetOutput) {
	t.Helper()

	if changeset == nil {
		t.Fatal("Changeset is nil")
	}

	if len(changeset.Changes) > 0 {
		t.Errorf("Expected changeset to have no changes but found %d", len(changeset.Changes))
	}
}

// AssertResourceStatus checks that a resource has the expected status
func AssertResourceStatus(t *testing.T, resource types.StackResource, expectedStatus types.ResourceStatus) {
	t.Helper()

	if resource.ResourceStatus != expectedStatus {
		t.Errorf("Resource status mismatch\nGot: %v\nExpected: %v", resource.ResourceStatus, expectedStatus)
	}
}

// AssertEqual checks that two values are equal using cmp.Diff
func AssertEqual(t *testing.T, got, want any, opts ...cmp.Option) {
	t.Helper()

	if diff := cmp.Diff(want, got, opts...); diff != "" {
		t.Errorf("Mismatch (-want +got):\n%s", diff)
	}
}

// AssertNotEqual checks that two values are not equal
func AssertNotEqual(t *testing.T, got, want any) {
	t.Helper()

	if cmp.Equal(want, got) {
		t.Errorf("Values should not be equal but are: %v", got)
	}
}

// AssertContains checks that a string contains a substring
func AssertContains(t *testing.T, s, substr string) {
	t.Helper()

	if !strings.Contains(s, substr) {
		t.Errorf("String does not contain expected substring\nString: %s\nExpected substring: %s", s, substr)
	}
}

// AssertNotContains checks that a string does not contain a substring
func AssertNotContains(t *testing.T, s, substr string) {
	t.Helper()

	if strings.Contains(s, substr) {
		t.Errorf("String contains unexpected substring\nString: %s\nUnexpected substring: %s", s, substr)
	}
}

// AssertSliceContains checks that a slice contains a specific element
func AssertSliceContains(t *testing.T, slice []string, element string) {
	t.Helper()

	if slices.Contains(slice, element) {
		return
	}

	t.Errorf("Slice does not contain element %s\nSlice: %v", element, slice)
}

// AssertSliceNotContains checks that a slice does not contain a specific element
func AssertSliceNotContains(t *testing.T, slice []string, element string) {
	t.Helper()

	if slices.Contains(slice, element) {
		t.Errorf("Slice should not contain element %s\nSlice: %v", element, slice)
		return
	}
}

// AssertLen checks that a slice or map has the expected length
func AssertLen(t *testing.T, collection any, expectedLen int) {
	t.Helper()

	actualLen := -1

	switch v := collection.(type) {
	case []string:
		actualLen = len(v)
	case []types.Stack:
		actualLen = len(v)
	case []types.StackEvent:
		actualLen = len(v)
	case []types.StackResource:
		actualLen = len(v)
	case []types.Change:
		actualLen = len(v)
	case map[string]string:
		actualLen = len(v)
	case map[string]any:
		actualLen = len(v)
	default:
		t.Fatalf("AssertLen: unsupported type %T", collection)
	}

	if actualLen != expectedLen {
		t.Errorf("Length mismatch\nGot: %d\nExpected: %d", actualLen, expectedLen)
	}
}

// AssertNotNil checks that a value is not nil
func AssertNotNil(t *testing.T, value any, name string) {
	t.Helper()

	if value == nil {
		t.Errorf("%s should not be nil", name)
	}
}

// AssertNil checks that a value is nil
func AssertNil(t *testing.T, value any, name string) {
	t.Helper()

	if value != nil {
		t.Errorf("%s should be nil but got: %v", name, value)
	}
}

// Common cmp options for comparing AWS types

// StackComparer provides cmp options for comparing Stack types
var StackComparer = cmp.Options{
	cmpopts.IgnoreFields(types.Stack{}, "CreationTime", "LastUpdatedTime"),
	cmpopts.IgnoreUnexported(types.Stack{}),
}

// ChangesetComparer provides cmp options for comparing Changeset types
var ChangesetComparer = cmp.Options{
	cmpopts.IgnoreFields(cloudformation.DescribeChangeSetOutput{}, "CreationTime"),
	cmpopts.IgnoreUnexported(cloudformation.DescribeChangeSetOutput{}),
}

// EventComparer provides cmp options for comparing StackEvent types
var EventComparer = cmp.Options{
	cmpopts.IgnoreFields(types.StackEvent{}, "Timestamp"),
	cmpopts.IgnoreUnexported(types.StackEvent{}),
}
