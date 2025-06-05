package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/spf13/viper"
)

type stubCfnClient struct {
	output cloudformation.DescribeStacksOutput
	err    error
}

func (s stubCfnClient) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &s.output, nil
}

// TestRunPrechecks verifies that runPrechecks updates the deployment log and deployment info correctly.
func TestRunPrechecks(t *testing.T) {
	info := lib.DeployInfo{TemplateRelativePath: "test"}
	logObj := lib.DeploymentLog{}
	outputsettings = settings.NewOutputSettings()
	viper.Set("templates.prechecks", []string{"echo hello"})

	out := runPrechecks(&info, config.AWSConfig{}, &logObj)
	if info.PrechecksFailed {
		t.Errorf("expected prechecks to pass")
	}
	if logObj.PreChecks != lib.DeploymentLogPreChecksPassed {
		t.Errorf("log status not updated")
	}
	if out == "" {
		t.Errorf("expected output to be returned")
	}
}

// TestRunPrechecksFail ensures failures are handled correctly without exiting.
func TestRunPrechecksFail(t *testing.T) {
	info := lib.DeployInfo{TemplateRelativePath: "test"}
	logObj := lib.DeploymentLog{}
	outputsettings = settings.NewOutputSettings()
	viper.Set("templates.prechecks", []string{"false"})
	viper.Set("templates.stop-on-failed-prechecks", false)

	out := runPrechecks(&info, config.AWSConfig{}, &logObj)
	if !info.PrechecksFailed {
		t.Errorf("expected prechecks to fail")
	}
	if logObj.PreChecks != lib.DeploymentLogPreChecksFailed {
		t.Errorf("log status not set to failed")
	}
	if out == "" {
		t.Errorf("expected output with failure details")
	}
}

// TestPrepareDeployment exercises success and error scenarios for prepareDeployment.
func TestPrepareDeployment(t *testing.T) {
	// stub AWS config loader and CloudFormation client
	originalLoad := loadAWSConfig
	originalClient := getCfnClient
	defer func() {
		loadAWSConfig = originalLoad
		getCfnClient = originalClient
	}()

	loadAWSConfig = func(c config.Config) (config.AWSConfig, error) {
		return config.AWSConfig{AccountID: "123", Region: "us-west-2"}, nil
	}

	viper.Set("logging.enabled", false)
	viper.Set("templates.directory", "../examples/templates")
	outputsettings = settings.NewOutputSettings()

	t.Run("new stack", func(t *testing.T) {
		deployFlags = DeployFlags{StackName: "test", Template: "../examples/templates/basicvpc.yaml"}
		getCfnClient = func(cfg config.AWSConfig) lib.CloudFormationDescribeStacksAPI {
			return stubCfnClient{err: errors.New("not found")}
		}
		info, _, err := prepareDeployment()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !info.IsNew {
			t.Errorf("expected new stack")
		}
	})

	t.Run("update not ready", func(t *testing.T) {
		deployFlags = DeployFlags{StackName: "test", Template: "../examples/templates/basicvpc.yaml"}
		getCfnClient = func(cfg config.AWSConfig) lib.CloudFormationDescribeStacksAPI {
			stack := types.Stack{StackStatus: types.StackStatusUpdateInProgress, StackName: strPtr("test")}
			return stubCfnClient{output: cloudformation.DescribeStacksOutput{Stacks: []types.Stack{stack}}}
		}
		_, _, err := prepareDeployment()
		if err == nil {
			t.Errorf("expected error when stack not ready")
		}
	})

	t.Run("update ready", func(t *testing.T) {
		deployFlags = DeployFlags{StackName: "test", Template: "../examples/templates/basicvpc.yaml"}
		getCfnClient = func(cfg config.AWSConfig) lib.CloudFormationDescribeStacksAPI {
			stack := types.Stack{StackStatus: types.StackStatusCreateComplete, StackName: strPtr("test")}
			return stubCfnClient{output: cloudformation.DescribeStacksOutput{Stacks: []types.Stack{stack}}}
		}
		info, _, err := prepareDeployment()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.IsNew {
			t.Errorf("expected existing stack")
		}
	})
}

// helper for pointer string
func strPtr(s string) *string { return &s }

// TestCreateAndShowChangeset verifies changeset creation and cleanup logic.
func TestCreateAndShowChangeset(t *testing.T) {
	origCreate := createChangesetFunc
	origShow := showChangesetFunc
	origDelete := deleteChangesetFunc
	defer func() {
		createChangesetFunc = origCreate
		showChangesetFunc = origShow
		deleteChangesetFunc = origDelete
	}()

	called := struct{ create, show, del bool }{}
	createChangesetFunc = func(info *lib.DeployInfo, cfg config.AWSConfig) *lib.ChangesetInfo {
		called.create = true
		cs := &lib.ChangesetInfo{Name: "cs"}
		return cs
	}
	showChangesetFunc = func(cs lib.ChangesetInfo, info lib.DeployInfo, cfg config.AWSConfig) { called.show = true }
	deleteChangesetFunc = func(info lib.DeployInfo, cfg config.AWSConfig) { called.del = true }

	info := lib.DeployInfo{IsDryRun: true}
	logObj := lib.DeploymentLog{}

	cs := createAndShowChangeset(&info, config.AWSConfig{}, &logObj)
	if cs == nil || cs.Name != "cs" {
		t.Fatalf("unexpected changeset: %+v", cs)
	}
	if !called.create || !called.show || !called.del {
		t.Errorf("expected helper functions invoked")
	}
}

// TestConfirmAndDeployChangeset covers confirmation paths for deployment.
func TestConfirmAndDeployChangeset(t *testing.T) {
	origAsk := askForConfirmationFunc
	origDeploy := deployChangesetFunc
	origDelete := deleteChangesetFunc
	defer func() {
		askForConfirmationFunc = origAsk
		deployChangesetFunc = origDeploy
		deleteChangesetFunc = origDelete
	}()

	cases := []struct {
		name           string
		createOnly     bool
		nonInteractive bool
		confirm        bool
		expectDeploy   bool
		expectDelete   bool
		expectReturn   bool
	}{
		{"create only", true, false, false, false, false, false},
		{"noninteractive", false, true, false, true, false, true},
		{"confirm yes", false, false, true, true, false, true},
		{"confirm no", false, false, false, false, true, false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			deployFlags.CreateChangeset = tt.createOnly
			deployFlags.NonInteractive = tt.nonInteractive
			calledDeploy := false
			calledDelete := false
			askForConfirmationFunc = func(string) bool { return tt.confirm }
			deployChangesetFunc = func(info lib.DeployInfo, cfg config.AWSConfig) { calledDeploy = true }
			deleteChangesetFunc = func(info lib.DeployInfo, cfg config.AWSConfig) { calledDelete = true }

			res := confirmAndDeployChangeset(&lib.ChangesetInfo{}, &lib.DeployInfo{}, config.AWSConfig{})
			if res != tt.expectReturn {
				t.Errorf("expected %v got %v", tt.expectReturn, res)
			}
			if calledDeploy != tt.expectDeploy {
				t.Errorf("deploy called=%v want %v", calledDeploy, tt.expectDeploy)
			}
			if calledDelete != tt.expectDelete {
				t.Errorf("delete called=%v want %v", calledDelete, tt.expectDelete)
			}
		})
	}
}

// TestPrintDeploymentResults validates success and failure paths for result handling.
func TestPrintDeploymentResults(t *testing.T) {
	origGetStack := getFreshStackFunc
	origShowFailed := showFailedEventsFunc
	origDeleteNew := deleteStackIfNewFunc
	defer func() {
		getFreshStackFunc = origGetStack
		showFailedEventsFunc = origShowFailed
		deleteStackIfNewFunc = origDeleteNew
	}()

	viper.Set("logging.enabled", false)

	successStack := types.Stack{StackStatus: types.StackStatusCreateComplete, StackName: strPtr("test")}
	failureStack := types.Stack{StackStatus: types.StackStatusRollbackComplete, StackName: strPtr("test")}

	t.Run("success", func(t *testing.T) {
		getFreshStackFunc = func(info *lib.DeployInfo, svc lib.CloudFormationDescribeStacksAPI) (types.Stack, error) {
			return successStack, nil
		}
		logObj := lib.DeploymentLog{}
		printDeploymentResults(&lib.DeployInfo{}, config.AWSConfig{}, &logObj)
		if logObj.Status != lib.DeploymentLogStatusSuccess {
			t.Errorf("expected success status")
		}
	})

	t.Run("failure new stack", func(t *testing.T) {
		calledDelete := false
		getFreshStackFunc = func(info *lib.DeployInfo, svc lib.CloudFormationDescribeStacksAPI) (types.Stack, error) {
			return failureStack, nil
		}
		showFailedEventsFunc = func(info lib.DeployInfo, cfg config.AWSConfig) []map[string]interface{} { return nil }
		deleteStackIfNewFunc = func(info lib.DeployInfo, cfg config.AWSConfig) { calledDelete = true }

		info := lib.DeployInfo{IsNew: true}
		logObj := lib.DeploymentLog{}
		printDeploymentResults(&info, config.AWSConfig{}, &logObj)
		if logObj.Status != lib.DeploymentLogStatusFailed {
			t.Errorf("expected failed status")
		}
		if !calledDelete {
			t.Errorf("expected deleteStackIfNew called")
		}
	})
}
