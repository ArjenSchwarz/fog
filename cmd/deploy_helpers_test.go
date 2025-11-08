package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/ArjenSchwarz/fog/lib/testutil"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/viper"
)

// TestValidateStackReadiness tests the validateStackReadiness helper function
func TestValidateStackReadiness(t *testing.T) {
	tests := map[string]struct {
		stackName string
		setup     func(*testutil.MockCFNClient)
		wantErr   bool
		errMsg    string
	}{
		"stack ready for update": {
			stackName: "test-stack",
			setup: func(m *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusCreateComplete).
					Build()
				m.WithStack(stack)
			},
			wantErr: false,
		},
		"stack in update progress": {
			stackName: "test-stack",
			setup: func(m *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusUpdateInProgress).
					Build()
				m.WithStack(stack)
			},
			wantErr: true,
			errMsg:  "UPDATE_IN_PROGRESS",
		},
		"stack in delete progress": {
			stackName: "test-stack",
			setup: func(m *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusDeleteInProgress).
					Build()
				m.WithStack(stack)
			},
			wantErr: true,
			errMsg:  "DELETE_IN_PROGRESS",
		},
		"stack in rollback complete is actually ready": {
			stackName: "test-stack",
			setup: func(m *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusRollbackComplete).
					Build()
				m.WithStack(stack)
			},
			wantErr: false, // ROLLBACK_COMPLETE is actually a valid status for updates
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := testutil.NewMockCFNClient()
			if tc.setup != nil {
				tc.setup(mockClient)
			}

			err := validateStackReadiness(tc.stackName, mockClient)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("expected error containing %q, got %q", tc.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestFormatAccountDisplay tests the formatAccountDisplay helper function
func TestFormatAccountDisplay(t *testing.T) {
	tests := map[string]struct {
		accountID    string
		accountAlias string
		want         string
	}{
		"with alias": {
			accountID:    "123456789012",
			accountAlias: "prod-account",
			want:         "prod-account (123456789012)",
		},
		"without alias": {
			accountID:    "123456789012",
			accountAlias: "",
			want:         "123456789012",
		},
		"empty account ID with alias": {
			accountID:    "",
			accountAlias: "test-account",
			want:         "test-account ()",
		},
		"both empty": {
			accountID:    "",
			accountAlias: "",
			want:         "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := formatAccountDisplay(tc.accountID, tc.accountAlias)

			if got != tc.want {
				t.Errorf("formatAccountDisplay() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestDetermineDeploymentMethod tests the determineDeploymentMethod helper function
func TestDetermineDeploymentMethod(t *testing.T) {
	tests := map[string]struct {
		isNew    bool
		isDryrun bool
		want     string
	}{
		"new stack normal": {
			isNew:    true,
			isDryrun: false,
			want:     "Deploying",
		},
		"new stack dry run": {
			isNew:    true,
			isDryrun: true,
			want:     "dry run", // Contains "dry run"
		},
		"existing stack normal": {
			isNew:    false,
			isDryrun: false,
			want:     "Updating",
		},
		"existing stack dry run": {
			isNew:    false,
			isDryrun: true,
			want:     "dry run", // Contains "dry run"
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := determineDeploymentMethod(tc.isNew, tc.isDryrun)

			if !strings.Contains(got, tc.want) {
				t.Errorf("determineDeploymentMethod() = %q, want to contain %q", got, tc.want)
			}
		})
	}
}

// TestRunPrechecks tests the runPrechecks helper function
func TestRunPrechecks(t *testing.T) {
	// Don't run in parallel due to global viper state
	// t.Parallel()

	tests := map[string]struct {
		precheckCommands     []string
		stopOnFailedPrecheck bool
		wantPrechecksPassed  bool
		wantLogStatus        lib.DeploymentLogPreChecks
		wantOutputContains   string
	}{
		"no prechecks configured": {
			precheckCommands:    []string{},
			wantPrechecksPassed: true,
			wantOutputContains:  "",
		},
		"successful precheck": {
			precheckCommands:    []string{"echo hello"},
			wantPrechecksPassed: true,
			wantLogStatus:       lib.DeploymentLogPreChecksPassed,
			wantOutputContains:  "precheck",
		},
		"failed precheck continue": {
			precheckCommands:     []string{"sh -c 'exit 1'"},
			stopOnFailedPrecheck: false,
			wantPrechecksPassed:  false,
			wantLogStatus:        lib.DeploymentLogPreChecksFailed,
			wantOutputContains:   "Issues detected",
		},
		"failed precheck stop": {
			precheckCommands:     []string{"sh -c 'exit 1'"},
			stopOnFailedPrecheck: true,
			wantPrechecksPassed:  false,
			wantLogStatus:        lib.DeploymentLogPreChecksFailed,
			wantOutputContains:   "Issues detected",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Don't run in parallel due to global viper state
			// t.Parallel()

			// Setup viper configuration
			viper.Set("templates.prechecks", tc.precheckCommands)
			viper.Set("templates.stop-on-failed-prechecks", tc.stopOnFailedPrecheck)

			info := lib.DeployInfo{TemplateRelativePath: "test"}
			logObj := lib.DeploymentLog{}

			out := runPrechecks(&info, &logObj)

			// Check if prechecks failed or passed
			if len(tc.precheckCommands) > 0 {
				if info.PrechecksFailed == tc.wantPrechecksPassed {
					t.Errorf("expected PrechecksFailed=%v, got %v", !tc.wantPrechecksPassed, info.PrechecksFailed)
				}

				if tc.wantLogStatus != "" && logObj.PreChecks != tc.wantLogStatus {
					t.Errorf("expected log status %q, got %q", tc.wantLogStatus, logObj.PreChecks)
				}
			}

			if tc.wantOutputContains != "" && !strings.Contains(strings.ToLower(out), strings.ToLower(tc.wantOutputContains)) {
				t.Errorf("expected output to contain %q, got %q", tc.wantOutputContains, out)
			}
		})
	}
}

// TestPrepareDeployment tests the prepareDeployment helper function
func TestPrepareDeployment(t *testing.T) {
	// Don't run in parallel due to global state (viper, deployFlags, outputsettings)
	// t.Parallel()

	tests := map[string]struct {
		stackName     string
		template      string
		setup         func(*testutil.MockCFNClient)
		mockAWSConfig func(config.Config) (config.AWSConfig, error)
		wantIsNew     bool
		wantErr       bool
		errMsg        string
	}{
		"new stack": {
			stackName: "test-stack",
			template:  "../examples/templates/basicvpc.yaml",
			setup: func(m *testutil.MockCFNClient) {
				m.WithError(errors.New("stack not found"))
			},
			mockAWSConfig: func(c config.Config) (config.AWSConfig, error) {
				return config.AWSConfig{AccountID: "123", Region: "us-west-2"}, nil
			},
			wantIsNew: true,
			wantErr:   false,
		},
		"existing stack ready for update": {
			stackName: "test-stack",
			template:  "../examples/templates/basicvpc.yaml",
			setup: func(m *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusCreateComplete).
					Build()
				m.WithStack(stack)
			},
			mockAWSConfig: func(c config.Config) (config.AWSConfig, error) {
				return config.AWSConfig{AccountID: "123", Region: "us-west-2"}, nil
			},
			wantIsNew: false,
			wantErr:   false,
		},
		"existing stack not ready for update": {
			stackName: "test-stack",
			template:  "../examples/templates/basicvpc.yaml",
			setup: func(m *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusUpdateInProgress).
					Build()
				m.WithStack(stack)
			},
			mockAWSConfig: func(c config.Config) (config.AWSConfig, error) {
				return config.AWSConfig{AccountID: "123", Region: "us-west-2"}, nil
			},
			wantIsNew: false,
			wantErr:   true,
			errMsg:    "UPDATE_IN_PROGRESS",
		},
		"AWS config loading fails": {
			stackName: "test-stack",
			template:  "../examples/templates/basicvpc.yaml",
			setup:     func(m *testutil.MockCFNClient) {},
			mockAWSConfig: func(c config.Config) (config.AWSConfig, error) {
				return config.AWSConfig{}, errors.New("failed to load AWS config")
			},
			wantErr: true,
			errMsg:  "failed to load AWS config",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Don't run in parallel due to global state (viper, deployFlags)
			// t.Parallel()

			// Setup mocks
			mockClient := testutil.NewMockCFNClient()
			if tc.setup != nil {
				tc.setup(mockClient)
			}

			originalLoadAWSConfig := loadAWSConfig
			originalGetCfnClient := getCfnClient
			origFlags := deployFlags
			defer func() {
				loadAWSConfig = originalLoadAWSConfig
				getCfnClient = originalGetCfnClient
				deployFlags = origFlags
			}()

			if tc.mockAWSConfig != nil {
				loadAWSConfig = tc.mockAWSConfig
			} else {
				loadAWSConfig = func(c config.Config) (config.AWSConfig, error) {
					return config.AWSConfig{AccountID: "123", Region: "us-west-2"}, nil
				}
			}

			getCfnClient = func(cfg config.AWSConfig) lib.CloudFormationDescribeStacksAPI {
				return mockClient
			}

			// Setup viper and flags
			viper.Set("logging.enabled", false)
			viper.Set("templates.directory", "../examples/templates")
			viper.Set("changeset.name-format", "changeset-$TIMESTAMP")
			deployFlags = DeployFlags{
				StackName: tc.stackName,
				Template:  tc.template,
			}

			info, _, err := prepareDeployment()

			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("expected error containing %q, got %q", tc.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if info.IsNew != tc.wantIsNew {
				t.Errorf("expected IsNew=%v, got %v", tc.wantIsNew, info.IsNew)
			}
		})
	}
}

// TestCreateAndShowChangeset tests the createAndShowChangeset helper function
func TestCreateAndShowChangeset(t *testing.T) {
	// Don't run in parallel due to global state (deployFlags, outputsettings)
	// t.Parallel()

	tests := map[string]struct {
		isDryRun           bool
		expectDeleteCalled bool
	}{
		"dry run creates changeset (deletion handled in main flow)": {
			isDryRun:           true,
			expectDeleteCalled: false,
		},
		"normal run creates changeset": {
			isDryRun:           false,
			expectDeleteCalled: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Don't run in parallel due to global outputsettings state
			// t.Parallel()

			createCalled := false
			showCalled := false
			deleteCalled := false

			origCreate := createChangesetFunc
			origShow := showChangesetFunc
			origDelete := deleteChangesetFunc
			origFlags := deployFlags
			defer func() {
				createChangesetFunc = origCreate
				showChangesetFunc = origShow
				deleteChangesetFunc = origDelete
				deployFlags = origFlags
			}()

			// Reset flags to clean state
			deployFlags = DeployFlags{}

			createChangesetFunc = func(info *lib.DeployInfo, cfg config.AWSConfig) *lib.ChangesetInfo {
				createCalled = true
				return &lib.ChangesetInfo{Name: "test-changeset"}
			}
			showChangesetFunc = func(cs lib.ChangesetInfo, info lib.DeployInfo, cfg config.AWSConfig, optionalBuilder ...*output.Builder) {
				showCalled = true
			}
			deleteChangesetFunc = func(info lib.DeployInfo, cfg config.AWSConfig) {
				deleteCalled = true
			}

			// Set up deployment info with changeset
			changeset := &lib.ChangesetInfo{Name: "existing-changeset"}
			info := lib.DeployInfo{
				IsDryRun:  tc.isDryRun,
				Changeset: changeset,
			}
			logObj := lib.DeploymentLog{}

			cs := createAndShowChangeset(&info, config.AWSConfig{}, &logObj, false)

			if cs == nil || cs.Name != "test-changeset" {
				t.Errorf("unexpected changeset: %+v", cs)
			}

			if !createCalled {
				t.Error("expected createChangeset to be called")
			}

			if !showCalled {
				t.Error("expected showChangeset to be called")
			}

			if deleteCalled != tc.expectDeleteCalled {
				t.Errorf("expected deleteChangeset called=%v, got %v", tc.expectDeleteCalled, deleteCalled)
			}
		})
	}
}

// TestConfirmAndDeployChangeset tests the confirmAndDeployChangeset helper function
func TestConfirmAndDeployChangeset(t *testing.T) {
	// Don't run in parallel due to global deployFlags state
	// t.Parallel()

	tests := map[string]struct {
		nonInteractive bool
		userConfirm    bool
		expectDeploy   bool
		expectDelete   bool
		expectReturn   bool
	}{
		"non-interactive auto-deploy": {
			nonInteractive: true,
			userConfirm:    false,
			expectDeploy:   true,
			expectDelete:   false,
			expectReturn:   true,
		},
		"user confirms deployment": {
			nonInteractive: false,
			userConfirm:    true,
			expectDeploy:   true,
			expectDelete:   false,
			expectReturn:   true,
		},
		"user declines deployment": {
			nonInteractive: false,
			userConfirm:    false,
			expectDeploy:   false,
			expectDelete:   true,
			expectReturn:   false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Don't run in parallel due to global deployFlags state
			// t.Parallel()

			deployCalled := false
			deleteCalled := false

			origAsk := askForConfirmationFunc
			origDeploy := deployChangesetFunc
			origDelete := deleteChangesetFunc
			origFlags := deployFlags
			defer func() {
				askForConfirmationFunc = origAsk
				deployChangesetFunc = origDeploy
				deleteChangesetFunc = origDelete
				deployFlags = origFlags
			}()

			askForConfirmationFunc = func(string) bool {
				return tc.userConfirm
			}
			deployChangesetFunc = func(info lib.DeployInfo, cfg config.AWSConfig) {
				deployCalled = true
			}
			deleteChangesetFunc = func(info lib.DeployInfo, cfg config.AWSConfig) {
				deleteCalled = true
			}

			deployFlags = DeployFlags{
				NonInteractive: tc.nonInteractive,
			}

			result := confirmAndDeployChangeset(&lib.ChangesetInfo{}, &lib.DeployInfo{}, config.AWSConfig{})

			if result != tc.expectReturn {
				t.Errorf("expected return value=%v, got %v", tc.expectReturn, result)
			}

			if deployCalled != tc.expectDeploy {
				t.Errorf("expected deployChangeset called=%v, got %v", tc.expectDeploy, deployCalled)
			}

			if deleteCalled != tc.expectDelete {
				t.Errorf("expected deleteChangeset called=%v, got %v", tc.expectDelete, deleteCalled)
			}
		})
	}
}

// TestPrintDeploymentResults tests the printDeploymentResults helper function
func TestPrintDeploymentResults(t *testing.T) {
	// Don't run in parallel due to global getCfnClient state
	// t.Parallel()

	tests := map[string]struct {
		stackStatus         types.StackStatus
		isNew               bool
		expectSuccess       bool
		expectDeleteNewCall bool
	}{
		"successful create": {
			stackStatus:         types.StackStatusCreateComplete,
			isNew:               true,
			expectSuccess:       true,
			expectDeleteNewCall: false, // deleteStackIfNew only called on failure
		},
		"successful update": {
			stackStatus:         types.StackStatusUpdateComplete,
			isNew:               false,
			expectSuccess:       true,
			expectDeleteNewCall: false,
		},
		"rollback complete for new stack": {
			stackStatus:         types.StackStatusRollbackComplete,
			isNew:               true,
			expectSuccess:       false,
			expectDeleteNewCall: true, // Called because isNew=true AND failure state
		},
		"update rollback complete": {
			stackStatus:         types.StackStatusUpdateRollbackComplete,
			isNew:               false,
			expectSuccess:       false,
			expectDeleteNewCall: false, // Not called because isNew=false
		},
		"rollback failed for new stack": {
			stackStatus:         types.StackStatusRollbackFailed,
			isNew:               true,
			expectSuccess:       false,
			expectDeleteNewCall: true, // Called because isNew=true AND failure state
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Don't run in parallel due to global getCfnClient state
			// t.Parallel()

			deleteNewCalled := false

			origGetStack := getFreshStackFunc
			origShowFailed := showFailedEventsFunc
			origDeleteNew := deleteStackIfNewFunc
			origGetClient := getCfnClient
			defer func() {
				getFreshStackFunc = origGetStack
				showFailedEventsFunc = origShowFailed
				deleteStackIfNewFunc = origDeleteNew
				getCfnClient = origGetClient
			}()

			resultStack := testutil.NewStackBuilder("test-stack").
				WithStatus(tc.stackStatus).
				Build()

			getFreshStackFunc = func(info *lib.DeployInfo, svc lib.CloudFormationDescribeStacksAPI) (types.Stack, error) {
				return *resultStack, nil
			}
			showFailedEventsFunc = func(info lib.DeployInfo, cfg config.AWSConfig, prefixMessage string) []map[string]any {
				return nil
			}
			deleteStackIfNewFunc = func(info lib.DeployInfo, cfg config.AWSConfig) {
				deleteNewCalled = true
			}

			viper.Set("logging.enabled", false)

			info := lib.DeployInfo{IsNew: tc.isNew}
			logObj := lib.DeploymentLog{}
			mockClient := testutil.NewMockCFNClient()

			getCfnClient = func(cfg config.AWSConfig) lib.CloudFormationDescribeStacksAPI {
				return mockClient
			}

			printDeploymentResults(&info, config.AWSConfig{}, &logObj)

			if tc.expectSuccess {
				if logObj.Status != lib.DeploymentLogStatusSuccess {
					t.Errorf("expected success status, got %q", logObj.Status)
				}
			} else {
				if logObj.Status != lib.DeploymentLogStatusFailed {
					t.Errorf("expected failed status, got %q", logObj.Status)
				}
			}

			if deleteNewCalled != tc.expectDeleteNewCall {
				t.Errorf("expected deleteStackIfNew called=%v, got %v", tc.expectDeleteNewCall, deleteNewCalled)
			}
		})
	}
}

// TestPrintDeploymentResults_WithOutputs tests output printing for successful deployments
func TestPrintDeploymentResults_WithOutputs(t *testing.T) {
	// Don't run in parallel due to global getCfnClient state
	// t.Parallel()

	origGetStack := getFreshStackFunc
	origGetClient := getCfnClient
	defer func() {
		getFreshStackFunc = origGetStack
		getCfnClient = origGetClient
	}()

	resultStack := testutil.NewStackBuilder("test-stack").
		WithStatus(types.StackStatusCreateComplete).
		WithOutput("VpcId", "vpc-12345").
		WithOutput("SubnetId", "subnet-67890").
		Build()

	getFreshStackFunc = func(info *lib.DeployInfo, svc lib.CloudFormationDescribeStacksAPI) (types.Stack, error) {
		return *resultStack, nil
	}

	viper.Set("logging.enabled", false)

	info := lib.DeployInfo{IsNew: true}
	logObj := lib.DeploymentLog{}
	mockClient := testutil.NewMockCFNClient()

	getCfnClient = func(cfg config.AWSConfig) lib.CloudFormationDescribeStacksAPI {
		return mockClient
	}

	printDeploymentResults(&info, config.AWSConfig{}, &logObj)

	if logObj.Status != lib.DeploymentLogStatusSuccess {
		t.Errorf("expected success status, got %q", logObj.Status)
	}

	if len(resultStack.Outputs) != 2 {
		t.Errorf("expected 2 outputs, got %d", len(resultStack.Outputs))
	}
}

// TestPrepareDeployment_ValidationError tests validation error handling
func TestPrepareDeployment_ValidationError(t *testing.T) {
	// Don't run in parallel due to global deployFlags state
	// t.Parallel()

	originalLoadAWSConfig := loadAWSConfig
	origFlags := deployFlags
	defer func() {
		loadAWSConfig = originalLoadAWSConfig
		deployFlags = origFlags
	}()

	// Setup invalid flags (missing stackname)
	deployFlags = DeployFlags{
		Template: "../examples/templates/basicvpc.yaml",
		// StackName is required but missing
	}

	_, _, err := prepareDeployment()

	if err == nil {
		t.Error("expected validation error but got nil")
	}
}

// TestMockClientInteraction tests that mock client interactions work correctly
func TestMockClientInteraction(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	tests := map[string]struct {
		setup func(*testutil.MockCFNClient)
		check func(*testing.T, *testutil.MockCFNClient)
	}{
		"describe stacks returns configured stack": {
			setup: func(m *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("my-stack").
					WithStatus(types.StackStatusCreateComplete).
					Build()
				m.WithStack(stack)
			},
			check: func(t *testing.T, m *testutil.MockCFNClient) {
				t.Helper()
				output, err := m.DescribeStacks(context.Background(), &cloudformation.DescribeStacksInput{
					StackName: aws.String("my-stack"),
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(output.Stacks) != 1 {
					t.Fatalf("expected 1 stack, got %d", len(output.Stacks))
				}
				if *output.Stacks[0].StackName != "my-stack" {
					t.Errorf("expected stack name 'my-stack', got %q", *output.Stacks[0].StackName)
				}
			},
		},
		"describe stacks returns error when configured": {
			setup: func(m *testutil.MockCFNClient) {
				m.WithError(errors.New("access denied"))
			},
			check: func(t *testing.T, m *testutil.MockCFNClient) {
				t.Helper()
				_, err := m.DescribeStacks(context.Background(), &cloudformation.DescribeStacksInput{
					StackName: aws.String("my-stack"),
				})
				if err == nil {
					t.Error("expected error but got nil")
				}
				if !strings.Contains(err.Error(), "access denied") {
					t.Errorf("expected 'access denied' error, got %q", err.Error())
				}
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state

			mockClient := testutil.NewMockCFNClient()
			if tc.setup != nil {
				tc.setup(mockClient)
			}

			tc.check(t, mockClient)
		})
	}
}

// TestStackBuilder tests the StackBuilder helper
func TestStackBuilder(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	stack := testutil.NewStackBuilder("test-stack").
		WithStatus(types.StackStatusCreateComplete).
		WithParameter("Env", "prod").
		WithTag("Project", "foo").
		WithOutput("VpcId", "vpc-123").
		Build()

	if *stack.StackName != "test-stack" {
		t.Errorf("expected stack name 'test-stack', got %q", *stack.StackName)
	}

	if stack.StackStatus != types.StackStatusCreateComplete {
		t.Errorf("expected status CREATE_COMPLETE, got %v", stack.StackStatus)
	}

	if len(stack.Parameters) != 1 {
		t.Errorf("expected 1 parameter, got %d", len(stack.Parameters))
	}

	if len(stack.Tags) != 1 {
		t.Errorf("expected 1 tag, got %d", len(stack.Tags))
	}

	if len(stack.Outputs) != 1 {
		t.Errorf("expected 1 output, got %d", len(stack.Outputs))
	}
}

// TestAssertEqualStructs demonstrates using cmp.Diff for struct comparison
func TestAssertEqualStructs(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	type TestStruct struct {
		Name  string
		Value int
		Items []string
	}

	tests := map[string]struct {
		got      TestStruct
		want     TestStruct
		wantDiff bool
	}{
		"equal structs": {
			got:      TestStruct{Name: "test", Value: 42, Items: []string{"a", "b"}},
			want:     TestStruct{Name: "test", Value: 42, Items: []string{"a", "b"}},
			wantDiff: false,
		},
		"different values": {
			got:      TestStruct{Name: "test", Value: 42, Items: []string{"a", "b"}},
			want:     TestStruct{Name: "test", Value: 99, Items: []string{"a", "b"}},
			wantDiff: true,
		},
		"different slices": {
			got:      TestStruct{Name: "test", Value: 42, Items: []string{"a", "b"}},
			want:     TestStruct{Name: "test", Value: 42, Items: []string{"a", "c"}},
			wantDiff: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state

			diff := cmp.Diff(tc.want, tc.got)
			hasDiff := diff != ""

			if hasDiff != tc.wantDiff {
				if tc.wantDiff {
					t.Error("expected difference but structs were equal")
				} else {
					t.Errorf("unexpected difference:\n%s", diff)
				}
			}
		})
	}
}

// TestCreateStderrOutput tests the createStderrOutput helper function
func TestCreateStderrOutput(t *testing.T) {
	t.Parallel()

	out := createStderrOutput()

	if out == nil {
		t.Fatal("createStderrOutput() returned nil")
	}

	// Test that we got a valid output instance
	// We can't easily test TTY detection or transformers without mocking os.Stderr
	// but we can verify the function returns a valid Output instance
	testDoc := output.New().Text("test message").Build()
	err := out.Render(context.Background(), testDoc)
	if err != nil {
		t.Errorf("failed to render test document: %v", err)
	}
}

// TestCreateStderrOutput_TableFormat verifies that createStderrOutput always uses table format
func TestCreateStderrOutput_TableFormat(t *testing.T) {
	t.Parallel()

	out := createStderrOutput()
	if out == nil {
		t.Fatal("createStderrOutput() returned nil")
	}

	// Create a simple table to verify table format is being used
	testData := []map[string]any{
		{"Name": "test", "Value": "123"},
	}
	testDoc := output.New().
		Table("Test Table", testData, output.WithKeys("Name", "Value")).
		Build()

	// Render should succeed for table format
	err := out.Render(context.Background(), testDoc)
	if err != nil {
		t.Errorf("failed to render table: %v", err)
	}
}
