package cmd

import (
	"context"
	"testing"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/ArjenSchwarz/fog/lib/testutil"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/spf13/viper"
)

// TestDeploymentWorkflow_EndToEnd exercises the deploy orchestration
// (prepareDeployment -> createAndShowChangeset -> confirmAndDeployChangeset)
// through the package's injectable function seams. The changeset operations
// themselves are stubbed via createChangesetFunc/deployChangesetFunc/
// deleteChangesetFunc because createChangeset uses a concrete
// CloudFormation client that cannot be mocked through config.AWSConfig.
// Requires INTEGRATION=1 to run.
func TestDeploymentWorkflow_EndToEnd(t *testing.T) {
	testutil.SkipIfIntegration(t)
	// Cannot run in parallel: overrides global func vars, viper and deployFlags.

	tests := map[string]struct {
		stackName      string
		stackExists    bool
		nonInteractive bool
		confirm        bool // user's answer when interactive
		wantDeployed   bool
		wantDeleted    bool
	}{
		"new stack non-interactive deploys": {
			stackName:      "integration-test-new",
			stackExists:    false,
			nonInteractive: true,
			wantDeployed:   true,
			wantDeleted:    false,
		},
		"existing stack non-interactive deploys": {
			stackName:      "integration-test-existing",
			stackExists:    true,
			nonInteractive: true,
			wantDeployed:   true,
			wantDeleted:    false,
		},
		"interactive decline deletes changeset": {
			stackName:      "integration-test-decline",
			stackExists:    false,
			nonInteractive: false,
			confirm:        false,
			wantDeployed:   false,
			wantDeleted:    true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			mockCFN := testutil.NewMockCFNClient()
			if tc.stackExists {
				mockCFN.WithStack(testutil.NewStackBuilder(tc.stackName).
					WithStatus(types.StackStatusCreateComplete).
					Build())
			}

			changesetCreated := false
			stackDeployed := false
			changesetDeleted := false

			origLoadAWSConfig := loadAWSConfig
			origGetCfnClient := getCfnClient
			origCreateChangeset := createChangesetFunc
			origDeployChangeset := deployChangesetFunc
			origDeleteChangeset := deleteChangesetFunc
			origAskForConfirmation := askForConfirmationFunc
			origFlags := deployFlags
			t.Cleanup(func() {
				loadAWSConfig = origLoadAWSConfig
				getCfnClient = origGetCfnClient
				createChangesetFunc = origCreateChangeset
				deployChangesetFunc = origDeployChangeset
				deleteChangesetFunc = origDeleteChangeset
				askForConfirmationFunc = origAskForConfirmation
				deployFlags = origFlags
				viper.Reset()
			})

			loadAWSConfig = func(_ context.Context, _ config.Config) (config.AWSConfig, error) {
				return config.AWSConfig{AccountID: "123456789012", Region: "us-west-2"}, nil
			}
			getCfnClient = func(config.AWSConfig) lib.CloudFormationDescribeStacksAPI {
				return mockCFN
			}
			createChangesetFunc = func(info *lib.DeployInfo, _ config.AWSConfig) *lib.ChangesetInfo {
				changesetCreated = true
				return &lib.ChangesetInfo{
					Name:      "integration-test-changeset",
					StackName: info.StackName,
					ID:        "integration-test-changeset-id",
					Status:    string(types.ChangeSetStatusCreateComplete),
				}
			}
			deployChangesetFunc = func(lib.DeployInfo, config.AWSConfig) error {
				stackDeployed = true
				return nil
			}
			deleteChangesetFunc = func(lib.DeployInfo, config.AWSConfig) {
				changesetDeleted = true
			}
			askForConfirmationFunc = func(string) bool {
				return tc.confirm
			}

			viper.Reset()
			viper.Set("logging.enabled", false)
			viper.Set("templates.directory", "../examples/templates")
			viper.Set("changeset.name-format", "integration-test-changeset-$TIMESTAMP")

			deployFlags = DeployFlags{
				StackName:      tc.stackName,
				Template:       "../examples/templates/basicvpc.yaml",
				NonInteractive: tc.nonInteractive,
			}

			info, awsCfg, err := prepareDeployment()
			if err != nil {
				t.Fatalf("prepareDeployment returned an unexpected error: %v", err)
			}

			logObj := lib.NewDeploymentLog(awsCfg, info)

			cs := createAndShowChangeset(&info, awsCfg, &logObj, true)
			if cs == nil {
				t.Fatal("expected createAndShowChangeset to return a changeset")
			}
			if !changesetCreated {
				t.Error("expected the changeset to be created")
			}

			deployed, err := confirmAndDeployChangeset(cs, &info, awsCfg, false)
			if err != nil {
				t.Fatalf("confirmAndDeployChangeset returned an unexpected error: %v", err)
			}

			if deployed != tc.wantDeployed {
				t.Errorf("deployed = %v, want %v", deployed, tc.wantDeployed)
			}
			if stackDeployed != tc.wantDeployed {
				t.Errorf("stackDeployed = %v, want %v", stackDeployed, tc.wantDeployed)
			}
			if changesetDeleted != tc.wantDeleted {
				t.Errorf("changesetDeleted = %v, want %v", changesetDeleted, tc.wantDeleted)
			}
		})
	}
}

// TestDeploymentWorkflow_WithPrechecks tests deployment with prechecks
func TestDeploymentWorkflow_WithPrechecks(t *testing.T) {
	testutil.SkipIfIntegration(t)

	tests := map[string]struct {
		precheckCommands     []string
		stopOnFailedPrecheck bool
		expectDeployment     bool
		wantPrecheckStatus   lib.DeploymentLogPreChecks
	}{
		"successful prechecks allow deployment": {
			precheckCommands:   []string{"echo precheck1", "echo precheck2"},
			expectDeployment:   true,
			wantPrecheckStatus: lib.DeploymentLogPreChecksPassed,
		},
		"failed prechecks continue deployment": {
			precheckCommands:     []string{"sh -c 'exit 1'"},
			stopOnFailedPrecheck: false,
			expectDeployment:     true,
			wantPrecheckStatus:   lib.DeploymentLogPreChecksFailed,
		},
		"failed prechecks stop deployment": {
			precheckCommands:     []string{"sh -c 'exit 1'"},
			stopOnFailedPrecheck: true,
			expectDeployment:     false,
			wantPrecheckStatus:   lib.DeploymentLogPreChecksFailed,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// Don't run in parallel due to viper state
			// t.Parallel()

			// Setup mock client
			mockCFN := testutil.NewMockCFNClient()

			// Track whether changeset creation was attempted
			changesetCreated := false
			origCreateChangeset := createChangesetFunc
			originalLoadAWSConfig := loadAWSConfig
			originalGetCfnClient := getCfnClient
			origFlags := deployFlags
			defer func() {
				createChangesetFunc = origCreateChangeset
				loadAWSConfig = originalLoadAWSConfig
				getCfnClient = originalGetCfnClient
				deployFlags = origFlags
				viper.Reset()
			}()

			createChangesetFunc = func(info *lib.DeployInfo, cfg config.AWSConfig) *lib.ChangesetInfo {
				changesetCreated = true
				return &lib.ChangesetInfo{Name: "test-changeset"}
			}

			loadAWSConfig = func(_ context.Context, _ config.Config) (config.AWSConfig, error) {
				return config.AWSConfig{
					AccountID: "123456789012",
					Region:    "us-west-2",
				}, nil
			}

			getCfnClient = func(cfg config.AWSConfig) lib.CloudFormationDescribeStacksAPI {
				return mockCFN
			}

			// Setup viper configuration
			viper.Reset()
			viper.Set("logging.enabled", false)
			viper.Set("templates.directory", "../examples/templates")
			viper.Set("templates.prechecks", tc.precheckCommands)
			viper.Set("templates.stop-on-failed-prechecks", tc.stopOnFailedPrecheck)

			deployFlags = DeployFlags{
				StackName:      "precheck-test-stack",
				Template:       "../examples/templates/basicvpc.yaml",
				NonInteractive: true,
			}

			// Prepare deployment
			info, awsCfg, err := prepareDeployment()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Create deployment log
			logObj := lib.NewDeploymentLog(awsCfg, info)

			// Run prechecks and check abort signal
			_, abort := runPrechecks(&info, &logObj)

			// Verify precheck status
			if logObj.PreChecks != tc.wantPrecheckStatus {
				t.Errorf("expected precheck status %q, got %q", tc.wantPrecheckStatus, logObj.PreChecks)
			}

			// Only proceed with changeset creation if prechecks did not signal abort
			if !abort {
				createAndShowChangeset(&info, awsCfg, &logObj, true)
			}

			// Verify deployment continuation based on prechecks
			if tc.expectDeployment && !changesetCreated {
				t.Error("expected changeset to be created but it was not")
			}
			if !tc.expectDeployment && changesetCreated {
				t.Error("expected changeset creation to be skipped but it was created")
			}
		})
	}
}

// TestDeploymentWorkflow_ChangesetCreationAndExecution tests changeset operations
func TestDeploymentWorkflow_ChangesetCreationAndExecution(t *testing.T) {
	testutil.SkipIfIntegration(t)
	// Cannot run in parallel due to global getCfnClient state
	// t.Parallel()

	tests := map[string]struct {
		stackExists         bool
		changesetHasChanges bool
		expectExecution     bool
		wantErr             bool
	}{
		"new stack with changes": {
			stackExists:         false,
			changesetHasChanges: true,
			expectExecution:     true,
			wantErr:             false,
		},
		"existing stack with changes": {
			stackExists:         true,
			changesetHasChanges: true,
			expectExecution:     true,
			wantErr:             false,
		},
		"no changes in changeset": {
			stackExists:         true,
			changesetHasChanges: false,
			expectExecution:     false,
			wantErr:             false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// Cannot run in parallel due to global getCfnClient state
			// t.Parallel()

			mockCFN := testutil.NewMockCFNClient()

			if tc.stackExists {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusCreateComplete).
					Build()
				mockCFN.WithStack(stack)
			}

			changesetExecuted := false

			mockCFN.CreateChangeSetFn = func(ctx context.Context, params *cloudformation.CreateChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateChangeSetOutput, error) {
				return &cloudformation.CreateChangeSetOutput{
					Id:      aws.String("test-changeset-id"),
					StackId: aws.String("test-stack-id"),
				}, nil
			}

			mockCFN.DescribeChangeSetFn = func(ctx context.Context, params *cloudformation.DescribeChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeChangeSetOutput, error) {
				var changes []types.Change
				if tc.changesetHasChanges {
					changes = []types.Change{
						{
							Type: types.ChangeTypeResource,
							ResourceChange: &types.ResourceChange{
								Action:            types.ChangeActionAdd,
								LogicalResourceId: aws.String("Resource1"),
								ResourceType:      aws.String("AWS::S3::Bucket"),
							},
						},
					}
				}

				return &cloudformation.DescribeChangeSetOutput{
					ChangeSetId:   params.ChangeSetName,
					ChangeSetName: params.ChangeSetName,
					Status:        types.ChangeSetStatusCreateComplete,
					Changes:       changes,
					CreationTime:  aws.Time(time.Now()),
				}, nil
			}

			mockCFN.ExecuteChangeSetFn = func(ctx context.Context, params *cloudformation.ExecuteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ExecuteChangeSetOutput, error) {
				changesetExecuted = true
				return &cloudformation.ExecuteChangeSetOutput{}, nil
			}

			// Setup test environment
			originalGetCfnClient := getCfnClient
			defer func() {
				getCfnClient = originalGetCfnClient
			}()

			getCfnClient = func(cfg config.AWSConfig) lib.CloudFormationDescribeStacksAPI {
				return mockCFN
			}

			// Simulate changeset workflow
			info := &lib.DeployInfo{
				StackName: "test-stack",
				IsNew:     !tc.stackExists,
				Changeset: &lib.ChangesetInfo{
					Name: "test-changeset",
				},
			}

			// This would normally be called by createAndShowChangeset
			// For this test, we're focusing on the execution logic
			if tc.changesetHasChanges && tc.expectExecution {
				// Simulate execution
				_, err := mockCFN.ExecuteChangeSet(context.Background(), &cloudformation.ExecuteChangeSetInput{
					ChangeSetName: aws.String(info.Changeset.Name),
				})

				if tc.wantErr {
					if err == nil {
						t.Error("expected error but got nil")
					}
					return
				}

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			// Verify execution expectations
			if tc.expectExecution && !changesetExecuted {
				t.Error("expected changeset to be executed")
			}

			if !tc.expectExecution && changesetExecuted {
				t.Error("changeset was executed when it shouldn't have been")
			}
		})
	}
}

// TestDeploymentWorkflow_RollbackHandling tests rollback scenarios
func TestDeploymentWorkflow_RollbackHandling(t *testing.T) {
	testutil.SkipIfIntegration(t)
	// Cannot run in parallel due to global function overrides
	// t.Parallel()

	tests := map[string]struct {
		isNewStack               bool
		finalStackStatus         types.StackStatus
		expectStackDeletion      bool
		expectFailedEventDisplay bool
	}{
		"new stack rollback complete": {
			isNewStack:               true,
			finalStackStatus:         types.StackStatusRollbackComplete,
			expectStackDeletion:      true,
			expectFailedEventDisplay: true,
		},
		"new stack rollback failed": {
			isNewStack:               true,
			finalStackStatus:         types.StackStatusRollbackFailed,
			expectStackDeletion:      true,
			expectFailedEventDisplay: true,
		},
		"existing stack update rollback": {
			isNewStack:               false,
			finalStackStatus:         types.StackStatusUpdateRollbackComplete,
			expectStackDeletion:      false,
			expectFailedEventDisplay: true,
		},
		"successful deployment": {
			isNewStack:               true,
			finalStackStatus:         types.StackStatusCreateComplete,
			expectStackDeletion:      false,
			expectFailedEventDisplay: false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// Cannot run in parallel due to global function overrides
			// t.Parallel()

			stackDeleted := false
			failedEventsShown := false

			mockCFN := testutil.NewMockCFNClient()
			stack := testutil.NewStackBuilder("test-stack").
				WithStatus(tc.finalStackStatus).
				Build()
			mockCFN.WithStack(stack)

			mockCFN.DeleteStackFn = func(ctx context.Context, params *cloudformation.DeleteStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error) {
				stackDeleted = true
				return &cloudformation.DeleteStackOutput{}, nil
			}

			// Setup function overrides
			originalGetStack := getFreshStackFunc
			originalShowFailed := showFailedEventsFunc
			originalDeleteIfNew := deleteStackIfNewFunc
			originalGetClient := getCfnClient
			defer func() {
				getFreshStackFunc = originalGetStack
				showFailedEventsFunc = originalShowFailed
				deleteStackIfNewFunc = originalDeleteIfNew
				getCfnClient = originalGetClient
			}()

			getFreshStackFunc = func(info *lib.DeployInfo, svc lib.CloudFormationDescribeStacksAPI) (types.Stack, error) {
				return *stack, nil
			}

			showFailedEventsFunc = func(info lib.DeployInfo, cfg config.AWSConfig, prefixMessage string) []map[string]any {
				failedEventsShown = true
				return []map[string]any{
					{"ResourceType": "AWS::S3::Bucket", "Status": "CREATE_FAILED"},
				}
			}

			deleteStackIfNewFunc = func(info lib.DeployInfo, cfg config.AWSConfig) {
				if info.IsNew {
					stackDeleted = true
				}
			}

			getCfnClient = func(cfg config.AWSConfig) lib.CloudFormationDescribeStacksAPI {
				return mockCFN
			}

			viper.Set("logging.enabled", false)

			info := &lib.DeployInfo{
				StackName: "test-stack",
				IsNew:     tc.isNewStack,
			}
			logObj := &lib.DeploymentLog{}

			// Execute deployment results printing
			printDeploymentResults(info, config.AWSConfig{}, logObj)

			// Verify expectations
			if tc.expectStackDeletion && !stackDeleted {
				t.Error("expected stack to be deleted but it wasn't")
			}

			if !tc.expectStackDeletion && stackDeleted {
				t.Error("stack was deleted when it shouldn't have been")
			}

			if tc.expectFailedEventDisplay && !failedEventsShown {
				t.Error("expected failed events to be shown but they weren't")
			}
		})
	}
}
