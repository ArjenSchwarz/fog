package lib

import (
	"errors"
	"testing"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib/testutil"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestChangesetInfo_DeleteChangesetRefactored tests DeleteChangeset with modern patterns
func TestChangesetInfo_DeleteChangesetRefactored(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		changeset *ChangesetInfo
		setup     func(*testutil.MockCFNClient)
		want      bool
	}{
		"successful deletion": {
			changeset: &ChangesetInfo{
				Name:      "test-changeset",
				StackName: "test-stack",
			},
			setup: func(client *testutil.MockCFNClient) {
				// Mock will succeed by default
			},
			want: true,
		},
		"deletion fails with error": {
			changeset: &ChangesetInfo{
				Name:      "test-changeset",
				StackName: "test-stack",
			},
			setup: func(client *testutil.MockCFNClient) {
				client.WithError(&types.ChangeSetNotFoundException{
					Message: strPtr("ChangeSet not found"),
				})
			},
			want: false,
		},
		"deletion with validation error": {
			changeset: &ChangesetInfo{
				Name:      "protected-changeset",
				StackName: "protected-stack",
			},
			setup: func(client *testutil.MockCFNClient) {
				client.WithError(errors.New("Cannot delete changeset in EXECUTE_IN_PROGRESS status"))
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup
			mockClient := testutil.NewMockCFNClient()
			if tc.setup != nil {
				tc.setup(mockClient)
			}

			// Execute
			got := tc.changeset.DeleteChangeset(mockClient)

			// Assert
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestChangesetInfo_DeployChangesetRefactored tests DeployChangeset with modern patterns
func TestChangesetInfo_DeployChangesetRefactored(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		changeset *ChangesetInfo
		setup     func(*testutil.MockCFNClient)
		wantErr   bool
		errMsg    string
	}{
		"successful deployment": {
			changeset: &ChangesetInfo{
				Name:      "test-changeset",
				StackName: "test-stack",
				ID:        "arn:aws:cloudformation:us-west-2:123456789012:changeSet/test-changeset/abc123",
			},
			setup: func(client *testutil.MockCFNClient) {
				// Mock will succeed by default
			},
			wantErr: false,
		},
		"deployment with insufficient capabilities": {
			changeset: &ChangesetInfo{
				Name:      "test-changeset",
				StackName: "test-stack",
			},
			setup: func(client *testutil.MockCFNClient) {
				client.WithError(&types.InsufficientCapabilitiesException{
					Message: strPtr("Requires CAPABILITY_IAM"),
				})
			},
			wantErr: true,
			errMsg:  "Requires CAPABILITY_IAM",
		},
		"deployment with changeset not found": {
			changeset: &ChangesetInfo{
				Name:      "non-existent",
				StackName: "test-stack",
			},
			setup: func(client *testutil.MockCFNClient) {
				client.WithError(&types.ChangeSetNotFoundException{
					Message: strPtr("ChangeSet [non-existent] not found"),
				})
			},
			wantErr: true,
			errMsg:  "ChangeSet [non-existent] not found",
		},
		"deployment with invalid changeset status": {
			changeset: &ChangesetInfo{
				Name:      "failed-changeset",
				StackName: "test-stack",
			},
			setup: func(client *testutil.MockCFNClient) {
				client.WithError(&types.InvalidChangeSetStatusException{
					Message: strPtr("ChangeSet is in FAILED status"),
				})
			},
			wantErr: true,
			errMsg:  "ChangeSet is in FAILED status",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup
			mockClient := testutil.NewMockCFNClient()
			if tc.setup != nil {
				tc.setup(mockClient)
			}

			// Execute
			err := tc.changeset.DeployChangeset(mockClient)

			// Assert
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestChangesetInfo_AddChangeRefactored tests AddChange with modern patterns
func TestChangesetInfo_AddChangeRefactored(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		initial       *ChangesetInfo
		changeToAdd   ChangesetChanges
		wantLength    int
		wantHasModule bool
	}{
		"add first change without module": {
			initial: &ChangesetInfo{
				Changes:   nil,
				HasModule: false,
			},
			changeToAdd: ChangesetChanges{
				Action:      "Add",
				LogicalID:   "MyBucket",
				Type:        "AWS::S3::Bucket",
				ResourceID:  "",
				Replacement: "False",
				Module:      "",
			},
			wantLength:    1,
			wantHasModule: false,
		},
		"add change with module sets HasModule": {
			initial: &ChangesetInfo{
				Changes: []ChangesetChanges{
					{
						Action:    "Add",
						LogicalID: "MyBucket",
						Type:      "AWS::S3::Bucket",
					},
				},
				HasModule: false,
			},
			changeToAdd: ChangesetChanges{
				Action:    "Modify",
				LogicalID: "MyRole",
				Type:      "AWS::IAM::Role",
				Module:    "SecurityModule(LogicalId/Type)",
			},
			wantLength:    2,
			wantHasModule: true,
		},
		"add change to list with existing module": {
			initial: &ChangesetInfo{
				Changes: []ChangesetChanges{
					{
						Action:    "Add",
						LogicalID: "MyBucket",
						Type:      "AWS::S3::Bucket",
					},
					{
						Action:    "Modify",
						LogicalID: "MyRole",
						Type:      "AWS::IAM::Role",
						Module:    "SecurityModule",
					},
				},
				HasModule: true,
			},
			changeToAdd: ChangesetChanges{
				Action:      "Remove",
				LogicalID:   "MyFunction",
				Type:        "AWS::Lambda::Function",
				ResourceID:  "my-function-12345",
				Replacement: "False",
			},
			wantLength:    3,
			wantHasModule: true,
		},
		"add change with replacement": {
			initial: &ChangesetInfo{
				Changes: []ChangesetChanges{},
			},
			changeToAdd: ChangesetChanges{
				Action:      "Modify",
				LogicalID:   "MyDB",
				Type:        "AWS::RDS::DBInstance",
				ResourceID:  "mydb-instance",
				Replacement: "True",
				Details: []types.ResourceChangeDetail{
					{
						Evaluation: types.EvaluationTypeStatic,
						Target: &types.ResourceTargetDefinition{
							Attribute:          "Properties.DBInstanceIdentifier",
							RequiresRecreation: "Always",
						},
						CausingEntity: strPtr("DBInstanceIdentifier"),
					},
				},
			},
			wantLength:    1,
			wantHasModule: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Execute
			tc.initial.AddChange(tc.changeToAdd)

			// Assert
			assert.Len(t, tc.initial.Changes, tc.wantLength)
			assert.Equal(t, tc.wantHasModule, tc.initial.HasModule)

			// Verify the last added change
			if tc.wantLength > 0 {
				lastChange := tc.initial.Changes[tc.wantLength-1]
				assert.Equal(t, tc.changeToAdd.Action, lastChange.Action)
				assert.Equal(t, tc.changeToAdd.LogicalID, lastChange.LogicalID)
				assert.Equal(t, tc.changeToAdd.Type, lastChange.Type)
				assert.Equal(t, tc.changeToAdd.Module, lastChange.Module)
			}
		})
	}
}

// TestChangesetInfo_GetStackRefactored tests GetStack with modern patterns
func TestChangesetInfo_GetStackRefactored(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		changeset *ChangesetInfo
		setup     func(*testutil.MockCFNClient)
		wantStack types.Stack
		wantErr   bool
		errMsg    string
	}{
		"successful get stack": {
			changeset: &ChangesetInfo{
				StackID:   "arn:aws:cloudformation:us-west-2:123456789012:stack/test-stack/abc123",
				StackName: "test-stack",
			},
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusCreateComplete).
					WithDescription("Test stack").
					Build()
				// The GetStack function passes the StackID to DescribeStacks, so we need to key by that
				client.Stacks["arn:aws:cloudformation:us-west-2:123456789012:stack/test-stack/abc123"] = stack
			},
			wantStack: types.Stack{
				StackName:    strPtr("test-stack"),
				StackId:      strPtr("arn:aws:cloudformation:us-west-2:123456789012:stack/test-stack/12345678-1234-1234-1234-123456789012"),
				StackStatus:  types.StackStatusCreateComplete,
				Description:  strPtr("Test stack"),
				CreationTime: ptrTime(time.Now()),
			},
			wantErr: false,
		},
		"stack not found": {
			changeset: &ChangesetInfo{
				StackID:   "arn:aws:cloudformation:us-west-2:123456789012:stack/non-existent/abc123",
				StackName: "non-existent",
			},
			setup: func(client *testutil.MockCFNClient) {
				client.WithError(errors.New("Stack does not exist"))
			},
			wantErr: true,
			errMsg:  "Stack does not exist",
		},
		"stack with outputs and parameters": {
			changeset: &ChangesetInfo{
				StackID:   "arn:aws:cloudformation:us-west-2:123456789012:stack/complex-stack/abc123",
				StackName: "complex-stack",
			},
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("complex-stack").
					WithStatus(types.StackStatusUpdateComplete).
					WithParameter("Environment", "Production").
					WithParameter("Version", "2.0").
					WithOutput("ApiUrl", "https://api.example.com").
					WithTag("Owner", "TeamA").
					Build()
				// The GetStack function passes the StackID to DescribeStacks, so we need to key by that
				client.Stacks["arn:aws:cloudformation:us-west-2:123456789012:stack/complex-stack/abc123"] = stack
			},
			wantStack: types.Stack{
				StackName:   strPtr("complex-stack"),
				StackId:     strPtr("arn:aws:cloudformation:us-west-2:123456789012:stack/complex-stack/12345678-1234-1234-1234-123456789012"),
				StackStatus: types.StackStatusUpdateComplete,
				Parameters: []types.Parameter{
					{ParameterKey: strPtr("Environment"), ParameterValue: strPtr("Production")},
					{ParameterKey: strPtr("Version"), ParameterValue: strPtr("2.0")},
				},
				Outputs: []types.Output{
					{OutputKey: strPtr("ApiUrl"), OutputValue: strPtr("https://api.example.com")},
				},
				Tags: []types.Tag{
					{Key: strPtr("Owner"), Value: strPtr("TeamA")},
				},
				CreationTime: ptrTime(time.Now()),
			},
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup
			mockClient := testutil.NewMockCFNClient()
			if tc.setup != nil {
				tc.setup(mockClient)
			}

			// Execute
			got, err := tc.changeset.GetStack(mockClient)

			// Assert
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)

				// Use cmp.Diff for better comparison
				opts := []cmp.Option{
					cmpopts.IgnoreFields(types.Stack{}, "CreationTime"),
					cmpopts.IgnoreUnexported(types.Stack{}),
					cmpopts.IgnoreUnexported(types.Parameter{}),
					cmpopts.IgnoreUnexported(types.Output{}),
					cmpopts.IgnoreUnexported(types.Tag{}),
				}

				if diff := cmp.Diff(tc.wantStack, got, opts...); diff != "" {
					t.Errorf("GetStack() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// TestChangesetInfo_GenerateChangesetUrlRefactored tests URL generation
func TestChangesetInfo_GenerateChangesetUrlRefactored(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		changeset *ChangesetInfo
		config    config.AWSConfig
		want      string
	}{
		"generate URL for us-west-2": {
			changeset: &ChangesetInfo{
				ID:      "arn:aws:cloudformation:us-west-2:123456789012:changeSet/my-changeset/abc123",
				StackID: "arn:aws:cloudformation:us-west-2:123456789012:stack/my-stack/def456",
			},
			config: config.AWSConfig{
				Region: "us-west-2",
			},
			want: "https://console.aws.amazon.com/cloudformation/home?region=us-west-2#/stacks/changesets/changes?stackId=arn:aws:cloudformation:us-west-2:123456789012:stack/my-stack/def456&changeSetId=arn:aws:cloudformation:us-west-2:123456789012:changeSet/my-changeset/abc123",
		},
		"generate URL for eu-west-1": {
			changeset: &ChangesetInfo{
				ID:      "arn:aws:cloudformation:eu-west-1:987654321098:changeSet/eu-changeset/xyz789",
				StackID: "arn:aws:cloudformation:eu-west-1:987654321098:stack/eu-stack/ghi012",
			},
			config: config.AWSConfig{
				Region: "eu-west-1",
			},
			want: "https://console.aws.amazon.com/cloudformation/home?region=eu-west-1#/stacks/changesets/changes?stackId=arn:aws:cloudformation:eu-west-1:987654321098:stack/eu-stack/ghi012&changeSetId=arn:aws:cloudformation:eu-west-1:987654321098:changeSet/eu-changeset/xyz789",
		},
		"generate URL for ap-southeast-2": {
			changeset: &ChangesetInfo{
				ID:      "arn:aws:cloudformation:ap-southeast-2:111222333444:changeSet/ap-changeset/mno345",
				StackID: "arn:aws:cloudformation:ap-southeast-2:111222333444:stack/ap-stack/pqr678",
			},
			config: config.AWSConfig{
				Region: "ap-southeast-2",
			},
			want: "https://console.aws.amazon.com/cloudformation/home?region=ap-southeast-2#/stacks/changesets/changes?stackId=arn:aws:cloudformation:ap-southeast-2:111222333444:stack/ap-stack/pqr678&changeSetId=arn:aws:cloudformation:ap-southeast-2:111222333444:changeSet/ap-changeset/mno345",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Execute
			got := tc.changeset.GenerateChangesetUrl(tc.config)

			// Assert
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestChangesetChanges_GetDangerDetailsRefactored tests danger detail extraction
func TestChangesetChanges_GetDangerDetailsRefactored(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		changes ChangesetChanges
		want    []string
	}{
		"no danger - RequiresRecreation is Never": {
			changes: ChangesetChanges{
				Action:    "Modify",
				LogicalID: "MyBucket",
				Type:      "AWS::S3::Bucket",
				Details: []types.ResourceChangeDetail{
					{
						Evaluation: types.EvaluationTypeStatic,
						Target: &types.ResourceTargetDefinition{
							Attribute:          "Properties.VersioningConfiguration",
							RequiresRecreation: "Never",
						},
						CausingEntity: strPtr("VersioningConfiguration"),
					},
				},
			},
			want: []string{},
		},
		"danger - RequiresRecreation is Always": {
			changes: ChangesetChanges{
				Action:    "Modify",
				LogicalID: "MyDB",
				Type:      "AWS::RDS::DBInstance",
				Details: []types.ResourceChangeDetail{
					{
						Evaluation: types.EvaluationTypeStatic,
						Target: &types.ResourceTargetDefinition{
							Attribute:          "Properties.DBInstanceIdentifier",
							RequiresRecreation: "Always",
						},
						CausingEntity: strPtr("DBInstanceIdentifier"),
					},
				},
			},
			want: []string{"Static: Properties.DBInstanceIdentifier - DBInstanceIdentifier"},
		},
		"danger - RequiresRecreation is Conditionally": {
			changes: ChangesetChanges{
				Action:    "Modify",
				LogicalID: "MyInstance",
				Type:      "AWS::EC2::Instance",
				Details: []types.ResourceChangeDetail{
					{
						Evaluation: types.EvaluationTypeDynamic,
						Target: &types.ResourceTargetDefinition{
							Attribute:          "Properties.InstanceType",
							RequiresRecreation: "Conditionally",
						},
						CausingEntity: strPtr("InstanceType"),
					},
				},
			},
			want: []string{"Dynamic: Properties.InstanceType - InstanceType"},
		},
		"multiple details with mixed danger": {
			changes: ChangesetChanges{
				Action:    "Modify",
				LogicalID: "MyResource",
				Type:      "AWS::EC2::Instance",
				Details: []types.ResourceChangeDetail{
					{
						Evaluation: types.EvaluationTypeStatic,
						Target: &types.ResourceTargetDefinition{
							Attribute:          "Properties.AvailabilityZone",
							RequiresRecreation: "Always",
						},
						CausingEntity: strPtr("AvailabilityZone"),
					},
					{
						Evaluation: types.EvaluationTypeStatic,
						Target: &types.ResourceTargetDefinition{
							Attribute:          "Properties.Tags",
							RequiresRecreation: "Never",
						},
						CausingEntity: strPtr("Tags"),
					},
					{
						Evaluation: types.EvaluationTypeDynamic,
						Target: &types.ResourceTargetDefinition{
							Attribute:          "Properties.InstanceType",
							RequiresRecreation: "Conditionally",
						},
						CausingEntity: strPtr("InstanceType"),
					},
				},
			},
			want: []string{
				"Static: Properties.AvailabilityZone - AvailabilityZone",
				"Dynamic: Properties.InstanceType - InstanceType",
			},
		},
		"no details": {
			changes: ChangesetChanges{
				Action:    "Add",
				LogicalID: "NewResource",
				Type:      "AWS::S3::Bucket",
				Details:   []types.ResourceChangeDetail{},
			},
			want: []string{},
		},
		"detail with nil CausingEntity": {
			changes: ChangesetChanges{
				Action:    "Modify",
				LogicalID: "MyResource",
				Type:      "AWS::Lambda::Function",
				Details: []types.ResourceChangeDetail{
					{
						Evaluation: types.EvaluationTypeStatic,
						Target: &types.ResourceTargetDefinition{
							Attribute:          "Properties.Runtime",
							RequiresRecreation: "Always",
						},
						CausingEntity: nil,
					},
				},
			},
			want: []string{"Static: Properties.Runtime - "},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Execute
			got := tc.changes.GetDangerDetails()

			// Assert
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestGetStackAndChangesetFromURLRefactored tests URL parsing with modern patterns
func TestGetStackAndChangesetFromURLRefactored(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		changesetURL  string
		region        string
		wantStack     string
		wantChangeset string
	}{
		"valid URL with standard format": {
			changesetURL:  "https://console.aws.amazon.com/cloudformation/home?region=us-west-2#/stacks/changesets/changes?stackId=arn:aws:cloudformation:us-west-2:123456789012:stack/my-stack/abc123&changeSetId=arn:aws:cloudformation:us-west-2:123456789012:changeSet/my-changeset/def456",
			region:        "us-west-2",
			wantStack:     "arn:aws:cloudformation:us-west-2:123456789012:stack/my-stack/abc123",
			wantChangeset: "arn:aws:cloudformation:us-west-2:123456789012:changeSet/my-changeset/def456",
		},
		"URL with percent encoding": {
			changesetURL:  "https://console.aws.amazon.com/cloudformation/home?region=eu-central-1#/stacks/changesets/changes?stackId=arn%3Aaws%3Acloudformation%3Aeu-central-1%3A987654321098%3Astack%2Fencoded-stack%2Fxyz789&changeSetId=arn%3Aaws%3Acloudformation%3Aeu-central-1%3A987654321098%3AchangeSet%2Fencoded-changeset%2Fghi012",
			region:        "eu-central-1",
			wantStack:     "arn:aws:cloudformation:eu-central-1:987654321098:stack/encoded-stack/xyz789",
			wantChangeset: "arn:aws:cloudformation:eu-central-1:987654321098:changeSet/encoded-changeset/ghi012",
		},
		"URL with escaped backslashes": {
			changesetURL:  `https://console.aws.amazon.com/cloudformation/home?region=ap-southeast-1#/stacks/changesets/changes?stackId=arn\:aws\:cloudformation\:ap-southeast-1\:111222333444\:stack\/escaped-stack\/mno345&changeSetId=arn\:aws\:cloudformation\:ap-southeast-1\:111222333444\:changeSet\/escaped-changeset\/pqr678`,
			region:        "ap-southeast-1",
			wantStack:     "arn:aws:cloudformation:ap-southeast-1:111222333444:stack/escaped-stack/mno345",
			wantChangeset: "arn:aws:cloudformation:ap-southeast-1:111222333444:changeSet/escaped-changeset/pqr678",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Note: Not running in parallel because GetStackAndChangesetFromURL uses log.Fatal

			// Execute
			gotStack, gotChangeset := GetStackAndChangesetFromURL(tc.changesetURL, tc.region)

			// Assert
			assert.Equal(t, tc.wantStack, gotStack)
			assert.Equal(t, tc.wantChangeset, gotChangeset)
		})
	}
}
