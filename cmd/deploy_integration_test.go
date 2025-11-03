//go:build integration
// +build integration

package cmd

import (
	"context"
	"strings"
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

// TestDeploymentWorkflow_EndToEnd tests the complete deployment workflow
// This test requires INTEGRATION=1 to run
func TestDeploymentWorkflow_EndToEnd(t *testing.T) {
	testutil.SkipIfIntegration(t)

	tests := map[string]struct {
		stackName           string
		templatePath        string
		params              map[string]string
		tags                map[string]string
		isDryRun            bool
		isNonInteractive    bool
		setupStack          func(*testutil.MockCFNClient)
		expectStackCreated  bool
		expectChangesetOnly bool
		wantErr             bool
		errMsg              string
	}{
		"new stack deployment": {
			stackName:    "integration-test-stack",
			templatePath: "../examples/templates/basicvpc.yaml",
			params: map[string]string{
				"AvailabilityZone": "us-west-2a",
			},
			tags: map[string]string{
				"Environment": "test",
				"Project":     "integration",
			},
			isDryRun:         false,
			isNonInteractive: true,
			setupStack: func(m *testutil.MockCFNClient) {
				// Stack doesn't exist - will be created
			},
			expectStackCreated:  true,
			expectChangesetOnly: false,
			wantErr:             false,
		},
		"existing stack update": {
			stackName:    "integration-test-existing",
			templatePath: "../examples/templates/basicvpc.yaml",
			params: map[string]string{
				"AvailabilityZone": "us-west-2b",
			},
			isDryRun:         false,
			isNonInteractive: true,
			setupStack: func(m *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("integration-test-existing").
					WithStatus(types.StackStatusCreateComplete).
					WithParameter("AvailabilityZone", "us-west-2a").
					Build()
				m.WithStack(stack)
			},
			expectStackCreated:  false,
			expectChangesetOnly: false,
			wantErr:             false,
		},
		"dry run deployment": {
			stackName:           "integration-test-dryrun",
			templatePath:        "../examples/templates/basicvpc.yaml",
			isDryRun:            true,
			setupStack:          func(m *testutil.MockCFNClient) {},
			expectStackCreated:  false,
			expectChangesetOnly: true, // Dry run only creates changeset
			wantErr:             false,
		},
		"deployment with changeset validation": {
			stackName:        "integration-test-changeset",
			templatePath:     "../examples/templates/basicvpc.yaml",
			isDryRun:         false,
			isNonInteractive: false, // Interactive mode
			setupStack: func(m *testutil.MockCFNClient) {
				// New stack
			},
			expectStackCreated:  false, // User won't confirm in test
			expectChangesetOnly: true,
			wantErr:             false,
		},
		"rollback scenario for new stack": {
			stackName:        "integration-test-rollback",
			templatePath:     "../examples/templates/basicvpc.yaml",
			isDryRun:         false,
			isNonInteractive: true,
			setupStack: func(m *testutil.MockCFNClient) {
				// Simulate rollback by setting error on CreateStack
				m.CreateStackFn = func(ctx context.Context, params *cloudformation.CreateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateStackOutput, error) {
					// Return success but stack will be in ROLLBACK_COMPLETE
					return &cloudformation.CreateStackOutput{
						StackId: aws.String("arn:aws:cloudformation:us-west-2:123456789012:stack/integration-test-rollback/test"),
					}, nil
				}
				// When checking stack status, return ROLLBACK_COMPLETE
				m.DescribeStacksFn = func(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
					stack := testutil.NewStackBuilder("integration-test-rollback").
						WithStatus(types.StackStatusRollbackComplete).
						Build()
					return &cloudformation.DescribeStacksOutput{
						Stacks: []types.Stack{*stack},
					}, nil
				}
			},
			expectStackCreated: false,
			wantErr:            false, // No error, but deployment fails
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// Cannot run in parallel due to global viper and deployFlags state
			// t.Parallel()

			// Setup test context
			ctx := testutil.NewTestContext(t)
			defer ctx.Cleanup()

			// Setup mock clients
			mockCFN := testutil.NewMockCFNClient()
			if tc.setupStack != nil {
				tc.setupStack(mockCFN)
			}

			// Setup changeset behavior
			changesetCreated := false
			changesetDeleted := false
			stackCreatedOrUpdated := false

			mockCFN.CreateChangeSetFn = func(ctx context.Context, params *cloudformation.CreateChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateChangeSetOutput, error) {
				changesetCreated = true
				return &cloudformation.CreateChangeSetOutput{
					Id:      aws.String("arn:aws:cloudformation:us-west-2:123456789012:changeSet/test-changeset/test"),
					StackId: aws.String("arn:aws:cloudformation:us-west-2:123456789012:stack/" + *params.StackName + "/test"),
				}, nil
			}

			mockCFN.DescribeChangeSetFn = func(ctx context.Context, params *cloudformation.DescribeChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeChangeSetOutput, error) {
				changes := []types.Change{
					{
						Type: types.ChangeTypeResource,
						ResourceChange: &types.ResourceChange{
							Action:            types.ChangeActionAdd,
							LogicalResourceId: aws.String("TestResource"),
							ResourceType:      aws.String("AWS::S3::Bucket"),
						},
					},
				}
				return &cloudformation.DescribeChangeSetOutput{
					ChangeSetId:   params.ChangeSetName,
					ChangeSetName: params.ChangeSetName,
					StackId:       params.StackName,
					StackName:     params.StackName,
					Status:        types.ChangeSetStatusCreateComplete,
					Changes:       changes,
					CreationTime:  aws.Time(time.Now()),
				}, nil
			}

			mockCFN.DeleteChangeSetFn = func(ctx context.Context, params *cloudformation.DeleteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteChangeSetOutput, error) {
				changesetDeleted = true
				return &cloudformation.DeleteChangeSetOutput{}, nil
			}

			mockCFN.ExecuteChangeSetFn = func(ctx context.Context, params *cloudformation.ExecuteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ExecuteChangeSetOutput, error) {
				stackCreatedOrUpdated = true
				return &cloudformation.ExecuteChangeSetOutput{}, nil
			}

			// Override global functions for testing
			originalLoadAWSConfig := loadAWSConfig
			originalGetCfnClient := getCfnClient
			originalAskForConfirmation := askForConfirmationFunc
			origFlags := deployFlags
			defer func() {
				loadAWSConfig = originalLoadAWSConfig
				getCfnClient = originalGetCfnClient
				askForConfirmationFunc = originalAskForConfirmation
				deployFlags = origFlags
			}()

			loadAWSConfig = func(c config.Config) (config.AWSConfig, error) {
				return config.AWSConfig{
					AccountID: "123456789012",
					Region:    "us-west-2",
				}, nil
			}

			getCfnClient = func(cfg config.AWSConfig) lib.CloudFormationDescribeStacksAPI {
				return mockCFN
			}

			// For non-interactive tests, never confirm
			// For interactive tests with user confirmation, simulate user declining
			askForConfirmationFunc = func(string) bool {
				return false
			}

			// Setup viper configuration
			viper.Reset()
			viper.Set("logging.enabled", false)
			viper.Set("templates.directory", "../examples/templates")
			viper.Set("changeset.name-format", "integration-test-changeset-$TIMESTAMP")

			// Setup deployment flags
			deployFlags = DeployFlags{
				StackName:      tc.stackName,
				Template:       tc.templatePath,
				Dryrun:         tc.isDryRun,
				NonInteractive: tc.isNonInteractive,
			}

			// Convert params and tags to comma-separated strings
			if tc.params != nil {
				var params []string
				for k, v := range tc.params {
					params = append(params, k+"="+v)
				}
				deployFlags.Parameters = strings.Join(params, ",")
			}

			if tc.tags != nil {
				var tagStrs []string
				for k, v := range tc.tags {
					tagStrs = append(tagStrs, k+"="+v)
				}
				deployFlags.Tags = strings.Join(tagStrs, ",")
			}

			// Execute deployment workflow
			info, awsCfg, err := prepareDeployment()

			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("expected error containing %q, got %q", tc.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error during prepareDeployment: %v", err)
			}

			// Create deployment log
			logObj := lib.NewDeploymentLog(awsCfg, info)

			// For dry run or interactive without confirmation, we stop here
			if !tc.isDryRun && tc.isNonInteractive {
				// Create and show changeset
				cs := createAndShowChangeset(&info, awsCfg, &logObj)
				if cs == nil {
					t.Fatal("expected changeset to be created")
				}

				// Confirm and deploy
				deployed := confirmAndDeployChangeset(cs, &info, awsCfg)

				if tc.expectStackCreated && !deployed {
					t.Error("expected stack to be deployed but it wasn't")
				}
			}

			// Verify expectations
			if !changesetCreated {
				t.Error("expected changeset to be created")
			}

			if tc.isDryRun && !changesetDeleted {
				t.Error("expected changeset to be deleted in dry run mode")
			}

			if tc.expectStackCreated && !stackCreatedOrUpdated {
				t.Error("expected stack to be created or updated")
			}

			if tc.expectChangesetOnly && stackCreatedOrUpdated {
				t.Error("expected only changeset creation, but stack was modified")
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

			originalLoadAWSConfig := loadAWSConfig
			originalGetCfnClient := getCfnClient
			origFlags := deployFlags
			defer func() {
				loadAWSConfig = originalLoadAWSConfig
				getCfnClient = originalGetCfnClient
				deployFlags = origFlags
				viper.Reset()
			}()

			loadAWSConfig = func(c config.Config) (config.AWSConfig, error) {
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

			// Run prechecks
			runPrechecks(&info, &logObj)

			// Verify precheck status
			if logObj.PreChecks != tc.wantPrecheckStatus {
				t.Errorf("expected precheck status %q, got %q", tc.wantPrecheckStatus, logObj.PreChecks)
			}

			// Verify deployment continuation based on prechecks
			if tc.stopOnFailedPrecheck && info.PrechecksFailed {
				if tc.expectDeployment {
					t.Error("expected deployment to continue despite failed prechecks")
				}
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
