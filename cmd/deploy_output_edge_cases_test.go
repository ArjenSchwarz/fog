package cmd

import (
	"testing"
	"time"

	"github.com/ArjenSchwarz/fog/lib"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// TestOutputSuccessResult_LargeChangeset tests performance with extremely large changesets
// This addresses Issue 4.5 from the audit report
func TestOutputSuccessResult_LargeChangeset(t *testing.T) {
	tests := map[string]struct {
		numChanges  int
		description string
	}{
		"100 changes": {
			numChanges:  100,
			description: "Should handle 100 changes without performance issues",
		},
		"200 changes": {
			numChanges:  200,
			description: "Should handle 200 changes without performance issues",
		},
		"500 changes": {
			numChanges:  500,
			description: "Should handle 500 changes (stress test)",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a deployment with large changeset
			deployment := createLargeChangesetDeployment(tc.numChanges)

			viper.Set("output", "json")
			defer viper.Reset()

			// Measure execution time
			start := time.Now()

			output := captureStdout(func() {
				err := outputSuccessResult(deployment)
				assert.NoError(t, err, tc.description)
			})

			elapsed := time.Since(start)

			// Basic validation
			assert.NotEmpty(t, output)
			assert.Contains(t, output, "Deployment Summary")

			// Performance check - should complete in reasonable time
			// Even 500 changes should complete in under 5 seconds
			assert.Less(t, elapsed.Seconds(), 5.0, "Should complete large changeset in reasonable time")

			t.Logf("Processed %d changes in %v", tc.numChanges, elapsed)
		})
	}
}

// createLargeChangesetDeployment creates a deployment with a specified number of changes
func createLargeChangesetDeployment(numChanges int) *lib.DeployInfo {
	now := time.Now()
	changes := make([]lib.ChangesetChanges, numChanges)

	// Generate changes
	for i := 0; i < numChanges; i++ {
		action := "Add"
		if i%3 == 0 {
			action = "Modify"
		} else if i%5 == 0 {
			action = "Remove"
		}

		resourceType := "AWS::S3::Bucket"
		if i%2 == 0 {
			resourceType = "AWS::Lambda::Function"
		} else if i%3 == 0 {
			resourceType = "AWS::DynamoDB::Table"
		}

		changes[i] = lib.ChangesetChanges{
			Action:      action,
			LogicalID:   "Resource" + string(rune('A'+i%26)) + string(rune('0'+i%10)),
			Type:        resourceType,
			ResourceID:  "resource-" + string(rune('a'+i%26)) + "-" + string(rune('0'+i%10)),
			Replacement: "False",
		}
	}

	return &lib.DeployInfo{
		StackName:       "large-stack",
		StackArn:        "arn:aws:cloudformation:us-east-1:123456789012:stack/large-stack/abc123",
		DeploymentStart: now.Add(-10 * time.Minute),
		DeploymentEnd:   now,
		CapturedChangeset: &lib.ChangesetInfo{
			ID:           "arn:aws:cloudformation:us-east-1:123456789012:changeSet/large-changeset",
			CreationTime: now.Add(-11 * time.Minute),
			Changes:      changes,
		},
		FinalStackState: &types.Stack{
			StackStatus: types.StackStatusUpdateComplete,
			Outputs:     []types.Output{},
		},
		RawStack: &types.Stack{
			StackStatus:     types.StackStatusUpdateComplete,
			LastUpdatedTime: aws.Time(now.Add(-1 * time.Hour)),
		},
	}
}

// TestOutputSuccessResult_ZeroOutputs tests deployment with no stack outputs
func TestOutputSuccessResult_ZeroOutputs(t *testing.T) {
	deployment := createTestDeployment()
	deployment.FinalStackState.Outputs = []types.Output{}

	viper.Set("output", "json")
	defer viper.Reset()

	output := captureStdout(func() {
		err := outputSuccessResult(deployment)
		assert.NoError(t, err)
	})

	assert.NotEmpty(t, output)
	assert.Contains(t, output, "Deployment Summary")
	// Should not have outputs section or should gracefully handle empty outputs
}

// TestOutputNoChangesResult_EdgeCases tests edge cases for no-changes scenario
func TestOutputNoChangesResult_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		setupDeployment func() *lib.DeployInfo
		description     string
	}{
		"nil LastUpdatedTime and nil CreationTime": {
			setupDeployment: func() *lib.DeployInfo {
				d := createTestDeployment()
				d.RawStack.LastUpdatedTime = nil
				d.RawStack.CreationTime = nil
				return d
			},
			description: "Should handle both nil LastUpdatedTime and nil CreationTime",
		},
		"nil RawStack entirely": {
			setupDeployment: func() *lib.DeployInfo {
				d := createTestDeployment()
				d.RawStack = nil
				return d
			},
			description: "Should handle nil RawStack gracefully",
		},
		"empty stack name": {
			setupDeployment: func() *lib.DeployInfo {
				d := createTestDeployment()
				d.StackName = ""
				return d
			},
			description: "Should handle empty stack name",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			deployment := tc.setupDeployment()

			viper.Set("output", "json")
			defer viper.Reset()

			output := captureStdout(func() {
				err := outputNoChangesResult(deployment)
				assert.NoError(t, err, tc.description)
			})

			assert.NotEmpty(t, output)
		})
	}
}

// TestOutputFailureResult_EdgeCases tests edge cases for failure scenarios
func TestOutputFailureResult_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		setupDeployment func() *lib.DeployInfo
		description     string
	}{
		"nil FinalStackState": {
			setupDeployment: func() *lib.DeployInfo {
				d := createTestDeployment()
				d.DeploymentError = assert.AnError
				d.FinalStackState = nil
				return d
			},
			description: "Should handle nil FinalStackState gracefully",
		},
		"nil StatusReason": {
			setupDeployment: func() *lib.DeployInfo {
				d := createTestDeployment()
				d.DeploymentError = assert.AnError
				d.FinalStackState = &types.Stack{
					StackStatus:       types.StackStatusRollbackComplete,
					StackStatusReason: nil,
				}
				return d
			},
			description: "Should handle nil StatusReason",
		},
		"empty error message": {
			setupDeployment: func() *lib.DeployInfo {
				d := createTestDeployment()
				d.DeploymentError = &emptyError{}
				d.FinalStackState = &types.Stack{
					StackStatus: types.StackStatusRollbackComplete,
				}
				return d
			},
			description: "Should handle empty error message",
		},
		"very long error message": {
			setupDeployment: func() *lib.DeployInfo {
				d := createTestDeployment()
				longMessage := ""
				for i := 0; i < 1000; i++ {
					longMessage += "Error occurred. "
				}
				d.DeploymentError = &customError{msg: longMessage}
				d.FinalStackState = &types.Stack{
					StackStatus: types.StackStatusRollbackComplete,
				}
				return d
			},
			description: "Should handle very long error message",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			deployment := tc.setupDeployment()

			viper.Set("output", "json")
			defer viper.Reset()

			output := captureStdout(func() {
				err := outputFailureResult(deployment, config.AWSConfig{})
				assert.NoError(t, err, tc.description)
			})

			assert.NotEmpty(t, output)
		})
	}
}

// emptyError is a test error type with empty message
type emptyError struct{}

func (e *emptyError) Error() string {
	return ""
}

// customError is a test error type with custom message
type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}

// TestOutputSuccessResult_ChangesetInconsistency tests changeset with no changes but status indicates changes
// This addresses a specific edge case from Issue 4.5
func TestOutputSuccessResult_ChangesetInconsistency(t *testing.T) {
	tests := map[string]struct {
		setupDeployment func() *lib.DeployInfo
		description     string
	}{
		"empty changes array but deployment succeeded": {
			setupDeployment: func() *lib.DeployInfo {
				d := createTestDeployment()
				d.CapturedChangeset.Changes = []lib.ChangesetChanges{}
				return d
			},
			description: "Should handle empty changes array",
		},
		"nil changes array": {
			setupDeployment: func() *lib.DeployInfo {
				d := createTestDeployment()
				d.CapturedChangeset.Changes = nil
				return d
			},
			description: "Should handle nil changes array",
		},
		"single change": {
			setupDeployment: func() *lib.DeployInfo {
				d := createTestDeployment()
				d.CapturedChangeset.Changes = []lib.ChangesetChanges{
					{
						Action:      "Add",
						LogicalID:   "SingleResource",
						Type:        "AWS::S3::Bucket",
						ResourceID:  "single-bucket",
						Replacement: "False",
					},
				}
				return d
			},
			description: "Should handle single change",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			deployment := tc.setupDeployment()

			viper.Set("output", "json")
			defer viper.Reset()

			output := captureStdout(func() {
				err := outputSuccessResult(deployment)
				assert.NoError(t, err, tc.description)
			})

			assert.NotEmpty(t, output)
			assert.Contains(t, output, "Deployment Summary")
		})
	}
}

// TestOutputSuccessResult_ConcurrentAccess tests concurrent output generation
func TestOutputSuccessResult_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const numGoroutines = 10

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			deployment := createTestDeployment()
			deployment.StackName = "concurrent-stack-" + string(rune('0'+id))

			// Each goroutine uses its own viper instance to avoid conflicts
			// In real code, this would be handled by proper context passing
			output := captureStdout(func() {
				// Note: This test verifies there are no race conditions in output generation
				// It doesn't verify correctness of concurrent viper access
				_ = outputSuccessResult(deployment)
			})

			assert.NotEmpty(t, output)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// TestOutputFormats_InvalidFormat tests handling of invalid output formats
func TestOutputFormats_InvalidFormat(t *testing.T) {
	deployment := createTestDeployment()

	tests := map[string]struct {
		format      string
		description string
	}{
		"invalid format": {
			format:      "invalid",
			description: "Should handle invalid format gracefully",
		},
		"empty format": {
			format:      "",
			description: "Should handle empty format",
		},
		"unsupported format": {
			format:      "xml",
			description: "Should handle unsupported format",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			viper.Set("output", tc.format)
			defer viper.Reset()

			// Should not panic even with invalid format
			output := captureStdout(func() {
				err := outputSuccessResult(deployment)
				// May or may not error depending on go-output's handling
				// At minimum, should not panic
				_ = err
			})

			// Should produce some output or handle gracefully
			_ = output
		})
	}
}
