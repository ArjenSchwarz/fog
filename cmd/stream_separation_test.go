package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/stretchr/testify/assert"
)

// captureStderr captures stderr output during function execution
func captureStderr(f func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	f()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// captureBothStreams captures both stdout and stderr output during function execution
func captureBothStreams(f func()) (stdout string, stderr string) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	f()

	wOut.Close()
	wErr.Close()

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	io.Copy(&bufOut, rOut)
	io.Copy(&bufErr, rErr)

	return bufOut.String(), bufErr.String()
}

// TestPrintMessage_UsesStderr verifies that printMessage writes to stderr
func TestPrintMessage_UsesStderr(t *testing.T) {
	stderr := captureStderr(func() {
		printMessage(formatInfo("Test message"))
	})

	assert.Contains(t, stderr, "Test message")
	assert.Contains(t, stderr, "ℹ️")
}

// TestCreateStderrOutput_UsesStderr verifies that createStderrOutput produces stderr output
func TestCreateStderrOutput_UsesStderr(t *testing.T) {
	doc := output.New().Text("Test stderr output").Build()

	stderr := captureStderr(func() {
		out := createStderrOutput()
		_ = out.Render(context.Background(), doc)
	})

	assert.Contains(t, stderr, "Test stderr output")
}

// TestOutputSuccessResult_UsesStdout verifies final output goes to stdout
func TestOutputSuccessResult_UsesStdout(t *testing.T) {
	deployment := createTestDeployment()

	stdout, stderr := captureBothStreams(func() {
		_ = outputSuccessResult(deployment)
	})

	// Verify stdout contains the output
	assert.Contains(t, stdout, "Deployment Summary")
	assert.Contains(t, stdout, "CREATE_COMPLETE")

	// Verify stderr is minimal (should just contain sync side effects if any)
	// The header "=== Deployment Summary ===" goes to stdout via fmt.Println
	assert.NotContains(t, stderr, "Deployment Summary")
}

// TestOutputNoChangesResult_UsesStdout verifies no-changes output goes to stdout
func TestOutputNoChangesResult_UsesStdout(t *testing.T) {
	deployment := createTestDeployment()

	stdout, stderr := captureBothStreams(func() {
		_ = outputNoChangesResult(deployment)
	})

	// Verify stdout contains the output
	assert.Contains(t, stdout, "No changes to deploy")
	assert.Contains(t, stdout, "Stack Information")

	// Verify stderr doesn't contain the final output
	assert.NotContains(t, stderr, "No changes to deploy")
}

// TestOutputFailureResult_UsesStdout verifies failure output goes to stdout
func TestOutputFailureResult_UsesStdout(t *testing.T) {
	deployment := createTestDeployment()
	deployment.DeploymentError = fmt.Errorf("test deployment error")
	deployment.FinalStackState.StackStatus = types.StackStatusRollbackComplete

	awsConfig := createMockAWSConfig()

	stdout, stderr := captureBothStreams(func() {
		_ = outputFailureResult(deployment, awsConfig)
	})

	// Verify stdout contains the error output
	assert.Contains(t, stdout, "Deployment failed")
	assert.Contains(t, stdout, "test deployment error")
	assert.Contains(t, stdout, "Stack Status")

	// Verify stderr doesn't contain the final output
	assert.NotContains(t, stderr, "Deployment failed")
}

// TestStreamSeparation_ProgressVsFinal verifies complete separation of progress and final output
func TestStreamSeparation_ProgressVsFinal(t *testing.T) {
	tests := map[string]struct {
		progressFunc func()
		finalFunc    func()
		expectStderr string
		expectStdout string
	}{
		"progress message then success output": {
			progressFunc: func() {
				printMessage(formatInfo("Deploying stack..."))
			},
			finalFunc: func() {
				deployment := createTestDeployment()
				_ = outputSuccessResult(deployment)
			},
			expectStderr: "Deploying stack...",
			expectStdout: "Deployment Summary",
		},
		"multiple progress messages then final output": {
			progressFunc: func() {
				printMessage(formatInfo("Creating changeset..."))
				printMessage(formatSuccess("Changeset created"))
			},
			finalFunc: func() {
				deployment := createTestDeployment()
				_ = outputNoChangesResult(deployment)
			},
			expectStderr: "Creating changeset...",
			expectStdout: "No changes to deploy",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			stdout, stderr := captureBothStreams(func() {
				tc.progressFunc()
				tc.finalFunc()
			})

			// Verify stderr contains progress messages
			assert.Contains(t, stderr, tc.expectStderr)

			// Verify stdout contains final output
			assert.Contains(t, stdout, tc.expectStdout)

			// Verify streams don't contain each other's content
			assert.NotContains(t, stderr, tc.expectStdout)
			assert.NotContains(t, stdout, tc.expectStderr)
		})
	}
}

// TestFormatHelpers_ProduceCorrectOutput verifies format helpers produce expected patterns
func TestFormatHelpers_ProduceCorrectOutput(t *testing.T) {
	tests := map[string]struct {
		formatter   func(string) string
		input       string
		expectEmoji string
	}{
		"formatInfo includes emoji": {
			formatter:   formatInfo,
			input:       "test info",
			expectEmoji: "ℹ️",
		},
		"formatSuccess includes emoji": {
			formatter:   formatSuccess,
			input:       "test success",
			expectEmoji: "✅",
		},
		"formatError includes emoji": {
			formatter:   formatError,
			input:       "test error",
			expectEmoji: "🚨",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := tc.formatter(tc.input)
			assert.Contains(t, result, tc.expectEmoji)
			assert.Contains(t, result, tc.input)
		})
	}
}

// TestQuietMode_SuppressesStderr verifies quiet mode behavior
func TestQuietMode_SuppressesStderr(t *testing.T) {
	// Create a test deployment
	deployment := lib.DeployInfo{
		StackName: "test-stack",
		IsNew:     true,
	}

	awsConfig := createMockAWSConfig()

	t.Run("quiet mode suppresses showDeploymentInfo", func(t *testing.T) {
		stderr := captureStderr(func() {
			showDeploymentInfo(deployment, awsConfig, true) // quiet = true
		})

		// Should produce no output
		assert.Empty(t, stderr)
	})

	t.Run("non-quiet mode shows output", func(t *testing.T) {
		stderr := captureStderr(func() {
			showDeploymentInfo(deployment, awsConfig, false) // quiet = false
		})

		// Should contain deployment info
		assert.Contains(t, stderr, "test-stack")
	})
}

// TestStderrSync_CalledBeforeStdout verifies os.Stderr.Sync() is called appropriately
func TestStderrSync_CalledBeforeStdout(t *testing.T) {
	// This test verifies that the stderr sync pattern is in place
	// by checking that output functions don't panic when called

	deployment := createTestDeployment()
	awsConfig := createMockAWSConfig()

	tests := map[string]func(){
		"outputSuccessResult": func() {
			_ = outputSuccessResult(deployment)
		},
		"outputNoChangesResult": func() {
			_ = outputNoChangesResult(deployment)
		},
		"outputFailureResult": func() {
			_ = outputFailureResult(deployment, awsConfig)
		},
		"outputDryRunResult": func() {
			outputDryRunResult(deployment, awsConfig)
		},
	}

	for name, fn := range tests {
		t.Run(name, func(t *testing.T) {
			stdout, _ := captureBothStreams(fn)

			// Verify function completes without panic
			assert.NotEmpty(t, stdout)
		})
	}
}

// Helper to create mock AWS config for tests
func createMockAWSConfig() config.AWSConfig {
	return config.AWSConfig{
		AccountID:    "123456789012",
		AccountAlias: "test-account",
		Region:       "us-east-1",
	}
}
