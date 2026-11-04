package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/ArjenSchwarz/fog/lib/testutil"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/spf13/viper"
)

// Regression tests for T-1679: when an explicitly requested output file cannot
// be written, the deploy result-rendering paths must fail the command instead
// of printing a warning and reporting success. Ancillary work (deployment log,
// cleanup of failed new stacks) must still happen.

// setUnwritableOutputFile configures viper with an output-file beneath a
// regular file so the file writer's os.MkdirAll fails (see T-1190 tests).
func setUnwritableOutputFile(t *testing.T) {
	t.Helper()
	setupViperDefaults()
	t.Cleanup(viper.Reset)

	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0600); err != nil {
		t.Fatalf("failed to create blocker file: %v", err)
	}
	viper.Set("output-file", filepath.Join(blocker, "sub", "result.json"))
	viper.Set("logging.enabled", false)
}

// stubDeployResultDeps replaces the AWS-facing function vars used by
// printDeploymentResults and returns a pointer reporting whether the failed new
// stack cleanup ran.
func stubDeployResultDeps(t *testing.T, status types.StackStatus, lookupErr error) *bool {
	t.Helper()
	origGetStack := getFreshStackFunc
	origShowFailed := showFailedEventsFunc
	origDeleteNew := deleteStackIfNewFunc
	origGetClient := getCfnClient
	t.Cleanup(func() {
		getFreshStackFunc = origGetStack
		showFailedEventsFunc = origShowFailed
		deleteStackIfNewFunc = origDeleteNew
		getCfnClient = origGetClient
	})

	deleted := false
	resultStack := testutil.NewStackBuilder("test-stack").WithStatus(status).Build()
	getFreshStackFunc = func(info *lib.DeployInfo, svc lib.CloudFormationDescribeStacksAPI) (types.Stack, error) {
		if lookupErr != nil {
			return types.Stack{}, lookupErr
		}
		return *resultStack, nil
	}
	showFailedEventsFunc = func(info lib.DeployInfo, cfg config.AWSConfig, prefixMessage string) []map[string]any {
		return nil
	}
	deleteStackIfNewFunc = func(info lib.DeployInfo, cfg config.AWSConfig) {
		deleted = true
	}
	getCfnClient = func(cfg config.AWSConfig) lib.CloudFormationDescribeStacksAPI {
		return testutil.NewMockCFNClient()
	}
	return &deleted
}

func TestPrintDeploymentResults_ReturnsOutputFileError(t *testing.T) {
	// Not parallel: viper and the stubbed function vars are global state.
	tests := map[string]struct {
		status       types.StackStatus
		lookupErr    error
		isNew        bool
		wantLog      lib.DeploymentLogStatus
		wantDeletion bool
	}{
		"successful deployment": {
			status:  types.StackStatusUpdateComplete,
			wantLog: lib.DeploymentLogStatusSuccess,
		},
		"rolled back new stack is still cleaned up": {
			status:       types.StackStatusRollbackComplete,
			isNew:        true,
			wantLog:      lib.DeploymentLogStatusFailed,
			wantDeletion: true,
		},
		"final stack lookup failure": {
			lookupErr: errors.New("lookup failed"),
			wantLog:   lib.DeploymentLogStatusFailed,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			setUnwritableOutputFile(t)
			deleted := stubDeployResultDeps(t, tc.status, tc.lookupErr)

			info := lib.DeployInfo{StackName: "test-stack", IsNew: tc.isNew}
			logObj := lib.DeploymentLog{}

			var err error
			_, _ = captureBothStreams(func() {
				err = printDeploymentResults(&info, config.AWSConfig{}, &logObj)
			})

			if err == nil {
				t.Fatal("expected an error when the output file cannot be written, got nil")
			}
			if !strings.Contains(err.Error(), "failed to generate output") {
				t.Errorf("expected output failure context in error, got %q", err)
			}
			if logObj.Status != tc.wantLog {
				t.Errorf("deployment log status = %q, want %q", logObj.Status, tc.wantLog)
			}
			if *deleted != tc.wantDeletion {
				t.Errorf("deleteStackIfNew called = %v, want %v", *deleted, tc.wantDeletion)
			}
		})
	}
}

func TestPrintDeploymentResults_NoErrorWithoutOutputFile(t *testing.T) {
	// Not parallel: viper and the stubbed function vars are global state.
	setupViperDefaults()
	t.Cleanup(viper.Reset)
	viper.Set("logging.enabled", false)
	stubDeployResultDeps(t, types.StackStatusCreateComplete, nil)

	info := lib.DeployInfo{StackName: "test-stack"}
	logObj := lib.DeploymentLog{}

	var err error
	_, _ = captureBothStreams(func() {
		err = printDeploymentResults(&info, config.AWSConfig{}, &logObj)
	})

	if err != nil {
		t.Fatalf("expected no error when no output file is configured, got %v", err)
	}
	if logObj.Status != lib.DeploymentLogStatusSuccess {
		t.Errorf("deployment log status = %q, want %q", logObj.Status, lib.DeploymentLogStatusSuccess)
	}
}

func TestReportNoChanges_ExitCodes(t *testing.T) {
	// Not parallel: viper, deployFlags and osExitFunc are global state.
	tests := map[string]struct {
		unwritableFile bool
		wantCode       int
		wantStderr     string
	}{
		"exits successfully when output renders": {
			wantCode: 0,
		},
		"fails when the output file cannot be written": {
			unwritableFile: true,
			wantCode:       1,
			wantStderr:     "failed to generate output",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			origFlags := deployFlags
			t.Cleanup(func() { deployFlags = origFlags })
			deployFlags = DeployFlags{Quiet: true}

			if tc.unwritableFile {
				setUnwritableOutputFile(t)
			} else {
				setupViperDefaults()
				t.Cleanup(viper.Reset)
			}

			deployment := &lib.DeployInfo{StackName: "test-stack"}
			code, called, _, stderr := runDeployWithInterceptedExit(t, func() {
				reportNoChanges(deployment)
			})

			if !called {
				t.Fatal("expected osExitFunc to be called")
			}
			if code != tc.wantCode {
				t.Errorf("exit code = %d, want %d", code, tc.wantCode)
			}
			if tc.wantStderr != "" && !strings.Contains(stderr, tc.wantStderr) {
				t.Errorf("expected stderr to contain %q, got %q", tc.wantStderr, stderr)
			}
			if strings.Contains(stderr, "Warning: Failed to generate output") {
				t.Errorf("output failure must not be downgraded to a warning, got %q", stderr)
			}
		})
	}
}
