package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/ArjenSchwarz/fog/lib/testutil"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/spf13/viper"
)

// TestDeployTemplate_PrecheckAbortWritesFailureLog is a regression test for
// T-684: when runPrechecks returns abort=true the deployTemplate function must
// call deploymentLog.Failed before exiting. Previously os.Exit(1) was called
// immediately, skipping the failure log write.
func TestDeployTemplate_PrecheckAbortWritesFailureLog(t *testing.T) {
	tests := map[string]struct {
		precheckCommands     []string
		stopOnFailedPrecheck bool
		description          string
	}{
		"failed precheck with stop flag writes failure log": {
			precheckCommands:     []string{"sh -c 'exit 1'"},
			stopOnFailedPrecheck: true,
			description:          "When prechecks fail and stop-on-failed-prechecks is true, the deployment failure log must be written",
		},
		"execution error writes failure log": {
			precheckCommands:     []string{"nonexistent-cmd-t684 $TEMPLATEPATH"},
			stopOnFailedPrecheck: false,
			description:          "When a precheck command cannot be found, the deployment failure log must be written",
		},
		"execution error with stop flag writes failure log": {
			precheckCommands:     []string{"nonexistent-cmd-t684 $TEMPLATEPATH"},
			stopOnFailedPrecheck: true,
			description:          "When a precheck command cannot be found and stop flag is set, the deployment failure log must be written",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a temporary log file to capture deployment log writes
			logFile, err := os.CreateTemp(t.TempDir(), "deploy-log-*.json")
			if err != nil {
				t.Fatalf("failed to create temp log file: %v", err)
			}
			logFile.Close()

			// Save and restore global state
			origLoadAWSConfig := loadAWSConfig
			origGetCfnClient := getCfnClient
			origOsExit := osExitFunc
			origFlags := deployFlags
			defer func() {
				loadAWSConfig = origLoadAWSConfig
				getCfnClient = origGetCfnClient
				osExitFunc = origOsExit
				deployFlags = origFlags
				viper.Reset()
			}()

			// Configure viper for this test
			viper.Set("templates.prechecks", tc.precheckCommands)
			viper.Set("templates.stop-on-failed-prechecks", tc.stopOnFailedPrecheck)
			viper.Set("templates.directory", "../examples/templates")
			viper.Set("changeset.name-format", "changeset-$TIMESTAMP")
			viper.Set("logging.enabled", true)
			viper.Set("logging.filename", logFile.Name())

			// Use a mock client that reports the stack as new (error = not found)
			// so prepareDeployment skips validateStackReadiness
			mockClient := testutil.NewMockCFNClient()
			mockClient.WithError(errors.New("stack not found"))

			deployFlags = DeployFlags{
				StackName:      "test-stack-t684",
				Template:       "../examples/templates/basicvpc.yaml",
				NonInteractive: true,
			}

			loadAWSConfig = func(_ context.Context, _ config.Config) (config.AWSConfig, error) {
				return config.AWSConfig{
					AccountID: "123456789012",
					Region:    "us-east-1",
				}, nil
			}
			getCfnClient = func(_ config.AWSConfig) lib.CloudFormationDescribeStacksAPI {
				return mockClient
			}

			// Track whether os.Exit was called (prevents actual exit)
			exitCalled := false
			exitCode := 0
			osExitFunc = func(code int) {
				exitCalled = true
				exitCode = code
				// Use panic to stop execution flow without actually exiting
				panic("osExit called")
			}

			// Run deployTemplate, catching the panic from our osExit stub
			func() {
				defer func() {
					if r := recover(); r != nil {
						if r != "osExit called" {
							t.Fatalf("unexpected panic: %v", r)
						}
					}
				}()
				deployTemplate(nil, nil)
			}()

			if !exitCalled {
				t.Fatal("expected os.Exit to be called for precheck abort, but it was not")
			}
			if exitCode != 1 {
				t.Errorf("expected exit code 1, got %d", exitCode)
			}

			// Read the log file and verify a FAILED entry was written
			logData, err := os.ReadFile(logFile.Name())
			if err != nil {
				t.Fatalf("failed to read log file: %v", err)
			}

			logContent := strings.TrimSpace(string(logData))
			if logContent == "" {
				t.Fatal("deployment failure log was not written — this is the T-684 bug: " +
					"the precheck abort path skips deploymentLog.Failed()")
			}

			// Parse the log entry and verify it has FAILED status
			var logEntry lib.DeploymentLog
			if err := json.Unmarshal([]byte(logContent), &logEntry); err != nil {
				t.Fatalf("failed to parse deployment log: %v", err)
			}

			if logEntry.Status != lib.DeploymentLogStatusFailed {
				t.Errorf("expected deployment log status %q, got %q",
					lib.DeploymentLogStatusFailed, logEntry.Status)
			}

			if logEntry.PreChecks != lib.DeploymentLogPreChecksFailed {
				t.Errorf("expected precheck status %q, got %q",
					lib.DeploymentLogPreChecksFailed, logEntry.PreChecks)
			}
		})
	}
}

// TestDeployTemplate_SuccessfulPrecheckNoAbort verifies that when prechecks
// pass, the deploy flow continues past the abort check (no spurious exit).
func TestDeployTemplate_SuccessfulPrecheckNoAbort(t *testing.T) {
	// This test ensures the fix doesn't accidentally abort on successful prechecks.

	logFile, err := os.CreateTemp(t.TempDir(), "deploy-log-*.json")
	if err != nil {
		t.Fatalf("failed to create temp log file: %v", err)
	}
	logFile.Close()

	origLoadAWSConfig := loadAWSConfig
	origGetCfnClient := getCfnClient
	origCreateChangeset := createChangesetFunc
	origShowChangeset := showChangesetFunc
	origDeleteChangeset := deleteChangesetFunc
	origOsExit := osExitFunc
	origFlags := deployFlags
	defer func() {
		loadAWSConfig = origLoadAWSConfig
		getCfnClient = origGetCfnClient
		createChangesetFunc = origCreateChangeset
		showChangesetFunc = origShowChangeset
		deleteChangesetFunc = origDeleteChangeset
		osExitFunc = origOsExit
		deployFlags = origFlags
		viper.Reset()
	}()

	viper.Set("templates.prechecks", []string{"echo ok"})
	viper.Set("templates.stop-on-failed-prechecks", true)
	viper.Set("templates.directory", "../examples/templates")
	viper.Set("changeset.name-format", "changeset-$TIMESTAMP")
	viper.Set("logging.enabled", true)
	viper.Set("logging.filename", logFile.Name())

	mockClient := testutil.NewMockCFNClient()
	mockClient.WithError(errors.New("stack not found"))

	deployFlags = DeployFlags{
		StackName:       "test-stack-success",
		Template:        "../examples/templates/basicvpc.yaml",
		NonInteractive:  true,
		CreateChangeset: true, // stops after changeset creation without deletion
	}

	loadAWSConfig = func(_ context.Context, _ config.Config) (config.AWSConfig, error) {
		return config.AWSConfig{
			AccountID: "123456789012",
			Region:    "us-east-1",
		}, nil
	}
	getCfnClient = func(_ config.AWSConfig) lib.CloudFormationDescribeStacksAPI {
		return mockClient
	}

	// Track that we reach changeset creation (past the abort check)
	changesetReached := false
	createChangesetFunc = func(info *lib.DeployInfo, cfg config.AWSConfig) *lib.ChangesetInfo {
		changesetReached = true
		info.DeploymentEnd = time.Now()
		return &lib.ChangesetInfo{
			Status:       string(types.ChangeSetStatusCreateComplete),
			StatusReason: "test",
		}
	}
	showChangesetFunc = func(_ lib.ChangesetInfo, _ lib.DeployInfo, _ config.AWSConfig, _ ...*output.Builder) {}
	deleteChangesetFunc = func(_ lib.DeployInfo, _ config.AWSConfig) {}

	osExitFunc = func(code int) {
		t.Fatalf("os.Exit(%d) should not be called when prechecks pass", code)
	}

	// Run deployTemplate — it should proceed to changeset creation
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic: %v", r)
			}
		}()
		deployTemplate(nil, nil)
	}()

	if !changesetReached {
		t.Error("expected deploy flow to reach changeset creation after successful prechecks")
	}
}
