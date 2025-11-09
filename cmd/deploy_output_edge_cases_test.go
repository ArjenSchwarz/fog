package cmd

import (
	"testing"

	"github.com/ArjenSchwarz/fog/lib"
	"github.com/aws/aws-sdk-go-v2/aws"
	cfntypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/stretchr/testify/assert"
)

// TestOutputSuccessResult_EdgeCases tests edge cases for successful deployment output.
// These tests address Issue 4.5 from the audit report regarding missing edge case tests for deploy output.
func TestOutputSuccessResult_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		deployment  *lib.DeployInfo
		shouldPanic bool
		description string
	}{
		"nil FinalStackState": {
			deployment: &lib.DeployInfo{
				StackName:       "test-stack",
				FinalStackState: nil, // Edge case: nil final stack state
				ChangesetResponse: &cfntypes.DescribeChangeSetOutput{
					ChangeSetName: aws.String("test-changeset"),
					Changes:       []cfntypes.Change{},
				},
			},
			shouldPanic: false,
			description: "Nil FinalStackState should be handled gracefully",
		},
		"FinalStackState with nil outputs": {
			deployment: &lib.DeployInfo{
				StackName: "test-stack",
				FinalStackState: &lib.CfnStack{
					StackName: aws.String("test-stack"),
					Outputs:   nil, // Edge case: nil outputs
				},
				ChangesetResponse: &cfntypes.DescribeChangeSetOutput{
					ChangeSetName: aws.String("test-changeset"),
				},
			},
			shouldPanic: false,
			description: "Nil outputs should be handled gracefully",
		},
		"FinalStackState with empty outputs": {
			deployment: &lib.DeployInfo{
				StackName: "test-stack",
				FinalStackState: &lib.CfnStack{
					StackName: aws.String("test-stack"),
					Outputs:   []cfntypes.Output{}, // Edge case: empty outputs
				},
				ChangesetResponse: &cfntypes.DescribeChangeSetOutput{
					ChangeSetName: aws.String("test-changeset"),
				},
			},
			shouldPanic: false,
			description: "Empty outputs should be handled gracefully",
		},
		"nil ChangesetResponse": {
			deployment: &lib.DeployInfo{
				StackName: "test-stack",
				FinalStackState: &lib.CfnStack{
					StackName: aws.String("test-stack"),
				},
				ChangesetResponse: nil, // Edge case: nil changeset response
			},
			shouldPanic: false,
			description: "Nil ChangesetResponse should be handled gracefully",
		},
		"empty changeset with zero changes": {
			deployment: &lib.DeployInfo{
				StackName: "test-stack",
				FinalStackState: &lib.CfnStack{
					StackName: aws.String("test-stack"),
				},
				ChangesetResponse: &cfntypes.DescribeChangeSetOutput{
					ChangeSetName: aws.String("test-changeset"),
					Changes:       []cfntypes.Change{}, // Edge case: zero changes
				},
			},
			shouldPanic: false,
			description: "Empty changeset should be handled gracefully",
		},
		"large changeset with many changes": {
			deployment: &lib.DeployInfo{
				StackName: "test-stack",
				FinalStackState: &lib.CfnStack{
					StackName: aws.String("test-stack"),
				},
				ChangesetResponse: &cfntypes.DescribeChangeSetOutput{
					ChangeSetName: aws.String("test-changeset"),
					Changes:       generateManyChanges(150), // Edge case: large changeset
				},
			},
			shouldPanic: false,
			description: "Large changeset should be handled without performance issues",
		},
		"nil StackName in FinalStackState": {
			deployment: &lib.DeployInfo{
				StackName: "test-stack",
				FinalStackState: &lib.CfnStack{
					StackName: nil, // Edge case: nil stack name
				},
			},
			shouldPanic: false,
			description: "Nil StackName pointer should be handled",
		},
		"deployment with only empty strings": {
			deployment: &lib.DeployInfo{
				StackName: "",
				FinalStackState: &lib.CfnStack{
					StackName: aws.String(""),
				},
			},
			shouldPanic: false,
			description: "Empty string values should be handled",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.shouldPanic {
				assert.Panics(t, func() {
					outputSuccessResult(tc.deployment)
				}, tc.description)
			} else {
				assert.NotPanics(t, func() {
					// Note: This may fail but shouldn't panic
					_ = outputSuccessResult(tc.deployment)
				}, tc.description)
			}
		})
	}
}

// TestOutputFailureResult_EdgeCases tests edge cases for failed deployment output.
func TestOutputFailureResult_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		deployment  *lib.DeployInfo
		err         error
		shouldPanic bool
		description string
	}{
		"nil FinalStackState with error": {
			deployment: &lib.DeployInfo{
				StackName:       "test-stack",
				FinalStackState: nil,
			},
			err:         assert.AnError,
			shouldPanic: false,
			description: "Nil FinalStackState with error should be handled",
		},
		"nil error parameter": {
			deployment: &lib.DeployInfo{
				StackName: "test-stack",
				FinalStackState: &lib.CfnStack{
					StackName: aws.String("test-stack"),
				},
			},
			err:         nil, // Edge case: nil error
			shouldPanic: false,
			description: "Nil error should be handled gracefully",
		},
		"empty error message": {
			deployment: &lib.DeployInfo{
				StackName: "test-stack",
			},
			err:         &emptyError{},
			shouldPanic: false,
			description: "Empty error message should be handled",
		},
		"very long error message": {
			deployment: &lib.DeployInfo{
				StackName: "test-stack",
			},
			err:         &longError{msg: generateLongString(10000)},
			shouldPanic: false,
			description: "Very long error messages should be handled",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.shouldPanic {
				assert.Panics(t, func() {
					outputFailureResult(tc.deployment, tc.err)
				}, tc.description)
			} else {
				assert.NotPanics(t, func() {
					_ = outputFailureResult(tc.deployment, tc.err)
				}, tc.description)
			}
		})
	}
}

// TestOutputNoChangesResult_EdgeCases tests edge cases for no-changes deployment output.
func TestOutputNoChangesResult_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		deployment  *lib.DeployInfo
		shouldPanic bool
		description string
	}{
		"nil RawStack": {
			deployment: &lib.DeployInfo{
				StackName: "test-stack",
				RawStack:  nil, // Edge case: nil RawStack
			},
			shouldPanic: false,
			description: "Nil RawStack should be handled gracefully",
		},
		"empty RawStack": {
			deployment: &lib.DeployInfo{
				StackName: "test-stack",
				RawStack:  &cfntypes.Stack{},
			},
			shouldPanic: false,
			description: "Empty RawStack should be handled",
		},
		"RawStack with nil StackName": {
			deployment: &lib.DeployInfo{
				StackName: "test-stack",
				RawStack: &cfntypes.Stack{
					StackName: nil,
				},
			},
			shouldPanic: false,
			description: "Nil StackName in RawStack should be handled",
		},
		"RawStack with empty outputs": {
			deployment: &lib.DeployInfo{
				StackName: "test-stack",
				RawStack: &cfntypes.Stack{
					StackName: aws.String("test-stack"),
					Outputs:   []cfntypes.Output{},
				},
			},
			shouldPanic: false,
			description: "Empty outputs in RawStack should be handled",
		},
		"nil deployment": {
			deployment:  nil,
			shouldPanic: true, // Will panic due to nil dereference
			description: "Nil deployment should cause panic",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.shouldPanic {
				assert.Panics(t, func() {
					outputNoChangesResult(tc.deployment)
				}, tc.description)
			} else {
				assert.NotPanics(t, func() {
					_ = outputNoChangesResult(tc.deployment)
				}, tc.description)
			}
		})
	}
}

// Helper functions for test data generation

// generateManyChanges creates a slice of changes for testing large changesets.
func generateManyChanges(count int) []cfntypes.Change {
	changes := make([]cfntypes.Change, count)
	for i := 0; i < count; i++ {
		changes[i] = cfntypes.Change{
			Type: cfntypes.ChangeTypeResource,
			ResourceChange: &cfntypes.ResourceChange{
				Action:            cfntypes.ChangeActionAdd,
				LogicalResourceId: aws.String("Resource" + string(rune(i))),
				ResourceType:      aws.String("AWS::S3::Bucket"),
			},
		}
	}
	return changes
}

// generateLongString creates a very long string for testing.
func generateLongString(length int) string {
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = 'A'
	}
	return string(result)
}

// emptyError is a test error with an empty message.
type emptyError struct{}

func (e *emptyError) Error() string {
	return ""
}

// longError is a test error with a long message.
type longError struct {
	msg string
}

func (e *longError) Error() string {
	return e.msg
}

// TestDeploymentOutput_ConcurrentAccess tests that output functions handle concurrent access safely.
// This addresses potential race conditions mentioned in the audit report.
func TestDeploymentOutput_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	deployment := &lib.DeployInfo{
		StackName: "test-stack",
		FinalStackState: &lib.CfnStack{
			StackName: aws.String("test-stack"),
		},
		ChangesetResponse: &cfntypes.DescribeChangeSetOutput{
			ChangeSetName: aws.String("test-changeset"),
		},
	}

	// Test that multiple concurrent calls don't cause race conditions
	t.Run("concurrent success outputs", func(t *testing.T) {
		t.Parallel()

		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				defer func() {
					done <- true
				}()
				assert.NotPanics(t, func() {
					_ = outputSuccessResult(deployment)
				})
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// TestDeploymentOutput_MemoryUsage tests memory handling with large data structures.
func TestDeploymentOutput_MemoryUsage(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		deployment *lib.DeployInfo
		desc       string
	}{
		"large number of outputs": {
			deployment: &lib.DeployInfo{
				StackName: "test-stack",
				FinalStackState: &lib.CfnStack{
					StackName: aws.String("test-stack"),
					Outputs:   generateManyOutputs(200),
				},
			},
			desc: "Should handle large number of outputs efficiently",
		},
		"large number of parameters": {
			deployment: &lib.DeployInfo{
				StackName: "test-stack",
				FinalStackState: &lib.CfnStack{
					StackName:  aws.String("test-stack"),
					Parameters: generateManyParameters(200),
				},
			},
			desc: "Should handle large number of parameters efficiently",
		},
		"very long output values": {
			deployment: &lib.DeployInfo{
				StackName: "test-stack",
				FinalStackState: &lib.CfnStack{
					StackName: aws.String("test-stack"),
					Outputs: []cfntypes.Output{
						{
							OutputKey:   aws.String("LongOutput"),
							OutputValue: aws.String(generateLongString(10000)),
						},
					},
				},
			},
			desc: "Should handle very long output values",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				_ = outputSuccessResult(tc.deployment)
			}, tc.desc)
		})
	}
}

// generateManyOutputs creates many outputs for testing.
func generateManyOutputs(count int) []cfntypes.Output {
	outputs := make([]cfntypes.Output, count)
	for i := 0; i < count; i++ {
		outputs[i] = cfntypes.Output{
			OutputKey:   aws.String("Output" + string(rune(i))),
			OutputValue: aws.String("Value" + string(rune(i))),
		}
	}
	return outputs
}

// generateManyParameters creates many parameters for testing.
func generateManyParameters(count int) []cfntypes.Parameter {
	params := make([]cfntypes.Parameter, count)
	for i := 0; i < count; i++ {
		params[i] = cfntypes.Parameter{
			ParameterKey:   aws.String("Param" + string(rune(i))),
			ParameterValue: aws.String("Value" + string(rune(i))),
		}
	}
	return params
}
